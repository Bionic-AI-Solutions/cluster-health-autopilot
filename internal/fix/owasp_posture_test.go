// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// This file is the OWASP Kubernetes Top-10 posture-non-regression guard
// (Task G2). It is the TESTED form of the invariant that Srenix's autofixers
// never weaken the cluster's security posture.
//
// For every fixer, it runs the fixer against a fixture that drives it to
// emit a real mutation, captures every Delete / Patch, and asserts that NO
// mutation:
//
//   - removes or weakens a NetworkPolicy                         (OWASP K07)
//   - adds privileged:true, hostPath, hostNetwork, or a cap add  (OWASP K01)
//   - broadens RBAC (adds verbs/resources, widens a binding)     (OWASP K03)
//   - downgrades/removes a TLS secret reference                  (OWASP K08)
//   - deletes a resource in a protected namespace               (re-asserted)
//
// Most Srenix fixers only delete a stuck Pod/Job/failed-cert-request or patch a
// restartedAt annotation / a TLS secretName to the CORRECT secret, so these
// assertions PASS today. The value is forward-looking: a FUTURE fixer cannot
// introduce a posture regression without this test going red.
//
// See docs/OWASP_MAPPING.md for the full fixer↔OWASP table.

// recordedMutation is one Delete or Patch captured during a fixer run.
type recordedMutation struct {
	op        string // "Delete" | "Patch"
	gvr       schema.GroupVersionResource
	ns, name  string
	patchType types.PatchType
	body      []byte // patch bytes, nil for Delete
}

// postureMutator captures the FULL mutation (including patch bodies) so the
// posture assertions can inspect what each fixer actually writes. It is a
// live Mutator (AsMutator returns non-nil) so fixers don't refuse.
type postureMutator struct {
	muts []recordedMutation
}

func (p *postureMutator) Delete(_ context.Context, gvr schema.GroupVersionResource, ns, name string) error {
	p.muts = append(p.muts, recordedMutation{op: "Delete", gvr: gvr, ns: ns, name: name})
	return nil
}

func (p *postureMutator) Patch(_ context.Context, gvr schema.GroupVersionResource, ns, name string, pt types.PatchType, patch []byte) error {
	p.muts = append(p.muts, recordedMutation{op: "Patch", gvr: gvr, ns: ns, name: name, patchType: pt, body: append([]byte(nil), patch...)})
	return nil
}

func (p *postureMutator) PatchStatus(_ context.Context, gvr schema.GroupVersionResource, ns, name string, pt types.PatchType, patch []byte) error {
	p.muts = append(p.muts, recordedMutation{op: "PatchStatus", gvr: gvr, ns: ns, name: name, patchType: pt, body: append([]byte(nil), patch...)})
	return nil
}

func (p *postureMutator) Create(_ context.Context, gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) error {
	// Fixers are contractually forbidden from Create (pkg/fix); a fixer that
	// calls this is itself a violation. Record it so assertNoPostureWeakening
	// can fail loudly.
	p.muts = append(p.muts, recordedMutation{op: "Create", gvr: gvr, ns: ns, name: obj.GetName()})
	return nil
}

// assertNoPostureWeakening inspects every captured mutation and fails t if any
// of them weakens OWASP K8s posture.
func assertNoPostureWeakening(t *testing.T, fixer string, muts []recordedMutation) {
	t.Helper()
	for _, m := range muts {
		switch m.op {
		case "Delete":
			// K-protected: a fixer must never delete in a protected namespace.
			if IsProtectedNamespace(m.ns) {
				t.Errorf("%s: deletes %s in PROTECTED namespace %q — posture regression",
					fixer, m.gvr.Resource, m.ns)
			}
			// K07: never delete a NetworkPolicy (removing segmentation).
			if m.gvr.Resource == "networkpolicies" {
				t.Errorf("%s: deletes a NetworkPolicy (%s/%s) — weakens network segmentation (K07)",
					fixer, m.ns, m.name)
			}
		case "Create":
			t.Errorf("%s: called Create on %s/%s/%s — fixers are forbidden from Create (pkg/fix contract)",
				fixer, m.gvr.Resource, m.ns, m.name)
		case "Patch", "PatchStatus":
			assertPatchDoesNotWeaken(t, fixer, m)
		}
	}
}

// forbiddenPatchSubstrings are JSON fragments that, if a fixer patch ever
// introduces them, represent a posture regression. The check is intentionally
// substring-on-compacted-JSON: it is conservative (a fixer that merely sets a
// restartedAt annotation can never match), and it is what makes the guard
// red the instant someone adds a privileged/hostNetwork/RBAC/TLS-downgrade
// mutation.
var forbiddenPatchPatterns = []struct {
	owasp  string
	reason string
	// matchers run against the compacted-JSON form of the patch body.
	match func(compact string, m recordedMutation) bool
}{
	{
		owasp:  "K01",
		reason: "adds privileged:true to a container securityContext",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"privileged":true`)
		},
	},
	{
		owasp:  "K01",
		reason: "adds hostNetwork:true",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"hostNetwork":true`)
		},
	},
	{
		owasp:  "K01",
		reason: "adds hostPID:true",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"hostPID":true`)
		},
	},
	{
		owasp:  "K01",
		reason: "adds hostIPC:true",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"hostIPC":true`)
		},
	},
	{
		owasp:  "K01",
		reason: "introduces a hostPath volume",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"hostPath"`)
		},
	},
	{
		owasp:  "K01",
		reason: "adds a Linux capability (capabilities.add)",
		match: func(c string, _ recordedMutation) bool {
			// match "add":[ ... ] appearing alongside a capabilities key.
			return strings.Contains(c, `"capabilities"`) && strings.Contains(c, `"add"`)
		},
	},
	{
		owasp:  "K01",
		reason: "sets allowPrivilegeEscalation:true",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"allowPrivilegeEscalation":true`)
		},
	},
	{
		owasp:  "K03",
		reason: "writes RBAC rules/subjects/roleRef (broadens RBAC)",
		match: func(c string, m recordedMutation) bool {
			switch m.gvr.Resource {
			case "roles", "clusterroles", "rolebindings", "clusterrolebindings":
				return true
			}
			// even outside an RBAC GVR, a patch carrying rules/roleRef is suspect.
			return strings.Contains(c, `"roleRef"`) ||
				(strings.Contains(c, `"rules"`) && strings.Contains(c, `"verbs"`))
		},
	},
	{
		owasp:  "K08",
		reason: "removes a TLS secretName (sets it to empty/null)",
		match: func(c string, _ recordedMutation) bool {
			return strings.Contains(c, `"secretName":""`) ||
				strings.Contains(c, `"secretName":null`)
		},
	},
	{
		owasp:  "K07",
		reason: "patches a NetworkPolicy (could weaken segmentation)",
		match: func(_ string, m recordedMutation) bool {
			return m.gvr.Resource == "networkpolicies"
		},
	},
}

func assertPatchDoesNotWeaken(t *testing.T, fixer string, m recordedMutation) {
	t.Helper()
	compact := compactJSON(m.body)
	for _, p := range forbiddenPatchPatterns {
		if p.match(compact, m) {
			t.Errorf("%s: patch on %s/%s/%s %s — posture regression (OWASP %s)\n  patch: %s",
				fixer, m.gvr.Resource, m.ns, m.name, p.reason, p.owasp, compact)
		}
	}
}

// compactJSON returns the body with insignificant whitespace removed when it
// parses as JSON, else the raw string. Compacting makes the substring matchers
// above robust to formatting.
func compactJSON(body []byte) string {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(body)
	}
	return string(out)
}

// ---- the fixer table -------------------------------------------------------

// runFixer is the shared driver: load a read-only Source from the fixture
// files, run the fixer with a capturing Mutator, return the mutations.
func runFixer(t *testing.T, f Fixer, files map[string]string) []recordedMutation {
	t.Helper()
	src := loadSrc(t, files)
	mut := &postureMutator{}
	res := f.Run(context.Background(), src, mut)
	if res.Refused != "" {
		t.Fatalf("%s refused unexpectedly: %s", f.Name(), res.Refused)
	}
	return mut.muts
}

// postureCases enumerates EVERY fixer in internal/fix. A new fixer MUST be
// added here — TestOWASPPosture_EveryFixerCovered fails otherwise, so a fixer
// cannot silently skip the posture guard.
func postureCases() []struct {
	fixerType string
	run       func(t *testing.T) []recordedMutation
} {
	return []struct {
		fixerType string
		run       func(t *testing.T) []recordedMutation
	}{
		{
			fixerType: "StaleErrorPods",
			run: func(t *testing.T) []recordedMutation {
				// Two deletable Failed pods (one orphan, one Job-owned) + one in a
				// protected ns that must NOT be deleted.
				pods := `{"apiVersion":"v1","kind":"PodList","items":[
				  {"apiVersion":"v1","kind":"Pod","metadata":{"name":"orphan","namespace":"demo"},"status":{"phase":"Failed"}},
				  {"apiVersion":"v1","kind":"Pod","metadata":{"name":"jobpod","namespace":"demo","ownerReferences":[{"kind":"Job","name":"j1"}]},"status":{"phase":"Failed"}},
				  {"apiVersion":"v1","kind":"Pod","metadata":{"name":"vault-failed","namespace":"vault"},"status":{"phase":"Failed"}}
				]}`
				return runFixer(t, StaleErrorPods{}, map[string]string{"pods.json": pods})
			},
		},
		{
			fixerType: "StuckJobsWithBadSecretRef",
			run: func(t *testing.T) []recordedMutation {
				pods := `{"apiVersion":"v1","kind":"PodList","items":[
				  {"apiVersion":"v1","kind":"Pod","metadata":{"name":"p1","namespace":"demo","ownerReferences":[{"kind":"Job","name":"j1"}]},
				   "status":{"containerStatuses":[{"state":{"waiting":{"reason":"CreateContainerConfigError","message":"couldn't find key API_KEY in Secret demo/creds"}}}]}}
				]}`
				jobs := `{"apiVersion":"batch/v1","kind":"JobList","items":[
				  {"apiVersion":"batch/v1","kind":"Job","metadata":{"name":"j1","namespace":"demo","ownerReferences":[{"kind":"CronJob","name":"cj1"}]}}
				]}`
				cronjobs := `{"apiVersion":"batch/v1","kind":"CronJobList","items":[
				  {"apiVersion":"batch/v1","kind":"CronJob","metadata":{"name":"cj1","namespace":"demo"},"spec":{"suspend":false}}
				]}`
				return runFixer(t, StuckJobsWithBadSecretRef{}, map[string]string{
					"pods.json": pods, "jobs.json": jobs, "cronjobs.json": cronjobs,
				})
			},
		},
		{
			fixerType: "StuckRSPods",
			run: func(t *testing.T) []recordedMutation {
				// CCE pod (not a missing-key error) under an RS whose Deployment
				// has rolled forward → fixer emits a restartedAt patch.
				pods := `{"apiVersion":"v1","kind":"PodList","items":[
				  {"apiVersion":"v1","kind":"Pod","metadata":{"name":"p1","namespace":"demo","ownerReferences":[{"kind":"ReplicaSet","name":"rs1"}]},
				   "status":{"containerStatuses":[{"state":{"waiting":{"reason":"CreateContainerConfigError","message":"config error"}}}]}}
				]}`
				rs := `{"apiVersion":"apps/v1","kind":"ReplicaSetList","items":[
				  {"apiVersion":"apps/v1","kind":"ReplicaSet","metadata":{"name":"rs1","namespace":"demo","ownerReferences":[{"kind":"Deployment","name":"d1"}],"annotations":{"deployment.kubernetes.io/revision":"1"}}}
				]}`
				deploys := `{"apiVersion":"apps/v1","kind":"DeploymentList","items":[
				  {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"d1","namespace":"demo","annotations":{"deployment.kubernetes.io/revision":"2"}}}
				]}`
				return runFixer(t, StuckRSPods{}, map[string]string{
					"pods.json": pods, "replicasets.json": rs, "deployments.json": deploys,
				})
			},
		},
		{
			fixerType: "StuckCertificateRequests",
			run: func(t *testing.T) []recordedMutation {
				crs := `{"apiVersion":"cert-manager.io/v1","kind":"CertificateRequestList","items":[
				  {"apiVersion":"cert-manager.io/v1","kind":"CertificateRequest","metadata":{"name":"cr1","namespace":"demo"},
				   "status":{"conditions":[{"type":"Ready","status":"False","reason":"Failed"}]}}
				]}`
				orders := `{"apiVersion":"acme.cert-manager.io/v1","kind":"OrderList","items":[
				  {"apiVersion":"acme.cert-manager.io/v1","kind":"Order","metadata":{"name":"o1","namespace":"demo"},"status":{"state":"errored"}}
				]}`
				return runFixer(t, StuckCertificateRequests{}, map[string]string{
					"certificaterequests.json": crs, "orders.json": orders,
				})
			},
		},
		{
			fixerType: "TLSSecretMismatch",
			run: func(t *testing.T) []recordedMutation {
				host := "app.example.com"
				// Ingress points at a stale (expired) secret; a Ready cert-manager
				// Certificate for the same host targets a different healthy secret.
				ing := `{"apiVersion":"networking.k8s.io/v1","kind":"IngressList","items":[
				  {"apiVersion":"networking.k8s.io/v1","kind":"Ingress","metadata":{"name":"ing1","namespace":"demo"},
				   "spec":{"tls":[{"hosts":["` + host + `"],"secretName":"stale-tls"}]}}
				]}`
				expired := makeCertB64(t, host, time.Now().Add(-48*time.Hour))
				secrets := `{"apiVersion":"v1","kind":"SecretList","items":[
				  {"apiVersion":"v1","kind":"Secret","type":"kubernetes.io/tls","metadata":{"name":"stale-tls","namespace":"demo"},"data":{"tls.crt":"` + expired + `"}}
				]}`
				certs := `{"apiVersion":"cert-manager.io/v1","kind":"CertificateList","items":[
				  {"apiVersion":"cert-manager.io/v1","kind":"Certificate","metadata":{"name":"managed","namespace":"demo"},
				   "spec":{"secretName":"managed-tls","dnsNames":["` + host + `"]},
				   "status":{"conditions":[{"type":"Ready","status":"True"}]}}
				]}`
				return runFixer(t, TLSSecretMismatch{}, map[string]string{
					"ingresses.json": ing, "secrets.json": secrets, "certificates.json": certs,
				})
			},
		},
	}
}

// TestOWASPPosture runs the posture guard for every fixer.
func TestOWASPPosture(t *testing.T) {
	for _, tc := range postureCases() {
		t.Run(tc.fixerType, func(t *testing.T) {
			muts := tc.run(t)
			if len(muts) == 0 {
				t.Fatalf("%s emitted no mutation — fixture does not exercise the fixer, so the posture guard would be vacuous; fix the fixture",
					tc.fixerType)
			}
			assertNoPostureWeakening(t, tc.fixerType, muts)
		})
	}
}

// nameMethodRE matches `func (T) Name() string` and `func (f T) Name() string`,
// capturing the receiver type T. This is how we discover every fixer type
// implementing the Fixer interface in this package.
var nameMethodRE = regexp.MustCompile(`func\s*\(\s*(?:[A-Za-z_]\w*\s+)?([A-Za-z_]\w*)\s*\)\s*Name\(\)\s*string`)

// TestOWASPPosture_EveryFixerCovered is the meta-check: it scans the package's
// non-test .go files for every type with a `Name() string` method (the Fixer
// marker) and fails if any such type is absent from postureCases(). This makes
// it IMPOSSIBLE to add a new fixer without also adding it to the posture
// guard — a new fixer cannot silently skip the OWASP non-regression test.
func TestOWASPPosture_EveryFixerCovered(t *testing.T) {
	covered := map[string]struct{}{}
	for _, tc := range postureCases() {
		covered[tc.fixerType] = struct{}{}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	discovered := map[string]struct{}{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, mm := range nameMethodRE.FindAllStringSubmatch(string(data), -1) {
			discovered[mm[1]] = struct{}{}
		}
	}

	if len(discovered) == 0 {
		t.Fatal("discovered zero fixer types — the Name() scan is broken; the meta-check would be vacuous")
	}

	var missing []string
	for typ := range discovered {
		if _, ok := covered[typ]; !ok {
			missing = append(missing, typ)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("fixer type(s) %v have NO entry in postureCases() — every fixer MUST be added to the OWASP posture guard (internal/fix/owasp_posture_test.go). See docs/OWASP_MAPPING.md.",
			missing)
	}

	// Symmetry: a stale postureCases() entry naming a type that no longer
	// exists is also a bug worth catching.
	var stale []string
	for typ := range covered {
		if _, ok := discovered[typ]; !ok {
			stale = append(stale, typ)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Fatalf("postureCases() references unknown fixer type(s) %v — remove the stale entry", stale)
	}
}
