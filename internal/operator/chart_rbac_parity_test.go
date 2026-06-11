// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

// P1.3 — permanent gate for the reader-RBAC drift bug class.
//
// The watcher's reader permissions are declared on THREE surfaces:
//
//  1. chart:    charts/.../templates/clusterrole-reader.yaml (helm installs)
//  2. operator: BuildReaderClusterRole() (operator-managed installs)
//  3. OLM:      same as (2) — the bundle ships NO static reader-role
//     manifest; under OLM the operator materializes
//     BuildReaderClusterRole() at runtime, which only works because
//     the CSV grants the operator `escalate` on clusterroles. The OLM
//     leg therefore asserts (a) that mechanism is intact and (b) that
//     no static ClusterRole has been added to bundle/manifests/ that
//     could drift from the builder.
//
// History: services+endpoints were added to the operator builder
// (v1.10.3, DNS-chain drift) but never to the chart — so on chart
// installs TraefikRoutes listed services cluster-wide into a
// `forbidden` (PROBE_FAILED noise every cycle on k3s) and KongRoutes'
// v1.Endpoints fallback 403'd. Fake clients don't enforce RBAC, so
// unit tests stayed green for months. This test makes the drift a
// compile-adjacent failure.
//
// Comparison scope: the probe/read surface. Two apiGroups are
// excluded because the chart intentionally splits them across sibling
// roles bound to the same ServiceAccount while the operator unifies
// them into the single reader role:
//
//   - cha.bionicaisolutions.com — chart splits read (reader role,
//     gated on driftReport.enabled) vs write (clusterrole-driftreport
//     .yaml, clusterrole-resolutionrecord.yaml, clusterrole-silence
//     .yaml)
//   - coordination.k8s.io — chart uses a namespaced, resourceName-
//     scoped Role (role-leader-election.yaml)
//
// Everything else MUST match verb-for-verb in both directions.

const readerChartPath = "../../charts/cluster-health-autopilot/templates/clusterrole-reader.yaml"

// parityExcludedGroups — see scope note above.
var parityExcludedGroups = map[string]bool{
	"cha.bionicaisolutions.com": true,
	"coordination.k8s.io":       true,
}

func TestReaderRBAC_ChartOperatorParity(t *testing.T) {
	chartSet := normalizeRules(t, chartReaderRules(t))
	builderSet := normalizeRules(t, BuildReaderClusterRole().Rules)

	missingInChart := diffSets(builderSet, chartSet)
	missingInBuilder := diffSets(chartSet, builderSet)

	for _, k := range missingInChart {
		t.Errorf("operator BuildReaderClusterRole() grants %s but the chart reader role does NOT — add it to %s (probes/analyzers silently 403 on chart installs)", k, readerChartPath)
	}
	for _, k := range missingInBuilder {
		t.Errorf("chart reader role grants %s but operator BuildReaderClusterRole() does NOT — add it to internal/operator/rbac_builders.go (probes/analyzers silently 403 on operator installs)", k)
	}
}

// TestReaderRBAC_OLMBundleParity is the third leg. The bundle carries
// no static reader role — reader RBAC under OLM is the operator
// builder applied at runtime. Two invariants keep that true:
//
//  1. The CSV must grant the operator create+escalate on clusterroles
//     and create on clusterrolebindings, or the runtime apply of
//     BuildReaderClusterRole() fails RBAC escalation prevention and
//     OLM installs get NO reader permissions at all.
//  2. If someone ever adds a static ClusterRole manifest to
//     bundle/manifests/, it becomes a fourth drift surface — it must
//     then match the builder on the probe surface (compared here so
//     the day it appears, parity is enforced automatically).
func TestReaderRBAC_OLMBundleParity(t *testing.T) {
	// (1) CSV grants the runtime-apply mechanism.
	csvRules := csvClusterPermissionRules(t)
	for _, want := range []struct{ resource, verb string }{
		{"clusterroles", "create"},
		{"clusterroles", "escalate"},
		{"clusterrolebindings", "create"},
	} {
		if !roleGrants(csvRules, "rbac.authorization.k8s.io", want.resource, want.verb) {
			t.Errorf("CSV clusterPermissions missing %q on rbac.authorization.k8s.io/%s — under OLM the operator cannot materialize BuildReaderClusterRole() and the watcher gets no reader RBAC",
				want.verb, want.resource)
		}
	}

	// (2) Any static ClusterRole shipped in bundle/manifests/ must
	// match the builder's probe surface.
	builderSet := normalizeRules(t, BuildReaderClusterRole().Rules)
	for _, cr := range bundleStaticClusterRoles(t) {
		set := normalizeRules(t, cr.Rules)
		for _, k := range diffSets(builderSet, set) {
			t.Errorf("bundle static ClusterRole %q is missing %s relative to BuildReaderClusterRole() — keep bundle manifests in lockstep with the builder", cr.Name, k)
		}
		for _, k := range diffSets(set, builderSet) {
			t.Errorf("bundle static ClusterRole %q grants %s that BuildReaderClusterRole() does not — keep bundle manifests in lockstep with the builder", cr.Name, k)
		}
	}
}

// chartReaderRules parses the reader ClusterRole's rules out of the
// helm template. The template is static YAML except for {{ ... }}
// lines (metadata name/labels + the driftReport.enabled conditional);
// stripping those lines leaves a parseable rules block. The
// conditional driftreports rule survives the strip and is parsed —
// fine, because cha.bionicaisolutions.com is excluded from the
// comparison scope anyway.
func chartReaderRules(t *testing.T) []rbacv1.PolicyRule {
	t.Helper()
	raw, err := os.ReadFile(readerChartPath)
	if err != nil {
		t.Fatalf("read chart reader role: %v", err)
	}
	// First document only (the file is ClusterRole --- ClusterRoleBinding).
	doc := strings.SplitN(string(raw), "\n---", 2)[0]

	// Keep only the rules block, dropping any line that carries Helm
	// templating.
	var b strings.Builder
	inRules := false
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "rules:") {
			inRules = true
		}
		if !inRules || strings.Contains(line, "{{") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	var parsed struct {
		Rules []rbacv1.PolicyRule `json:"rules"`
	}
	if err := yaml.Unmarshal([]byte(b.String()), &parsed); err != nil {
		t.Fatalf("parse chart reader rules block: %v\nblock:\n%s", err, b.String())
	}
	if len(parsed.Rules) == 0 {
		t.Fatalf("parsed 0 rules from %s — template shape changed; update chartReaderRules()", readerChartPath)
	}
	return parsed.Rules
}

// csvClusterPermissionRules returns every rule under the CSV's
// spec.install.spec.clusterPermissions (all service accounts).
func csvClusterPermissionRules(t *testing.T) []rbacv1.PolicyRule {
	t.Helper()
	raw, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	var csv struct {
		Spec struct {
			Install struct {
				Spec struct {
					ClusterPermissions []struct {
						Rules []rbacv1.PolicyRule `json:"rules"`
					} `json:"clusterPermissions"`
				} `json:"spec"`
			} `json:"install"`
		} `json:"spec"`
	}
	if err := yaml.Unmarshal(raw, &csv); err != nil {
		t.Fatalf("decode CSV: %v", err)
	}
	var out []rbacv1.PolicyRule
	for _, cp := range csv.Spec.Install.Spec.ClusterPermissions {
		out = append(out, cp.Rules...)
	}
	if len(out) == 0 {
		t.Fatal("CSV has no clusterPermissions rules — shape changed; update csvClusterPermissionRules()")
	}
	return out
}

// bundleStaticClusterRoles returns any standalone ClusterRole objects
// shipped in bundle/manifests/ (multi-document aware). Today: none.
func bundleStaticClusterRoles(t *testing.T) []rbacv1.ClusterRole {
	t.Helper()
	entries, err := filepath.Glob(filepath.Dir(csvPath) + "/*.yaml")
	if err != nil {
		t.Fatalf("glob bundle manifests: %v", err)
	}
	var out []rbacv1.ClusterRole
	for _, p := range entries {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		for _, doc := range strings.Split(string(raw), "\n---") {
			if !strings.Contains(doc, "kind: ClusterRole") {
				continue
			}
			var cr rbacv1.ClusterRole
			if err := yaml.Unmarshal([]byte(doc), &cr); err != nil {
				continue // not a bare ClusterRole doc (e.g. CSV prose mentioning the kind)
			}
			if cr.Kind == "ClusterRole" {
				out = append(out, cr)
			}
		}
	}
	return out
}

// normalizeRules flattens rules into a set of "group/resource verb"
// keys, dropping the apiGroups excluded from the parity scope.
func normalizeRules(t *testing.T, rules []rbacv1.PolicyRule) map[string]bool {
	t.Helper()
	out := make(map[string]bool)
	for _, r := range rules {
		for _, g := range r.APIGroups {
			if parityExcludedGroups[g] {
				continue
			}
			for _, res := range r.Resources {
				for _, v := range r.Verbs {
					out[fmt.Sprintf("%s/%s %s", orCore(g), res, v)] = true
				}
			}
		}
	}
	return out
}

// diffSets returns the sorted keys present in a but absent from b.
func diffSets(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
