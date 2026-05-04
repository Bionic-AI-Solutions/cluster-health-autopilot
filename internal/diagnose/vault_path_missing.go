// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/vault"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// VaultPathMissing closes the L1 stale-Ready window described in the brief
// — the case where Vault has been edited (path renamed, key removed) but
// the ExternalSecret hasn't refreshed yet. FailingExternalSecrets only
// catches drift AFTER the ESO controller's next refresh marks itself
// not-Ready; this analyzer queries Vault directly and catches the same
// drift while the ESO is still reporting Ready=True.
//
// Privacy contract: this analyzer reads from Vault but only ever inspects
// the SET OF KEY NAMES at each path (vault.Client.ListKeys returns []string,
// not map[string][]byte). Byte values are never read, never logged, never
// included in diagnostic output. Same posture as ProactiveSecretKeyCheck
// has for Kubernetes Secrets.
type VaultPathMissing struct {
	// Client is the Vault read client. When nil the analyzer is a no-op
	// (vault probe disabled in helm or VAULT_ADDR not set).
	Client vault.Client
}

// Name returns the analyzer's identifier.
func (VaultPathMissing) Name() string { return "VaultPathMissing" }

// Run produces the diagnostic set.
func (a VaultPathMissing) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	if a.Client == nil {
		return nil
	}
	// Snapshot mode can't reach Vault from a captured directory — the
	// whole point of zero-trust is that the analyst doesn't need Vault
	// credentials. Skip silently.
	if src.Mode() != snapshot.ModeLive {
		return nil
	}

	list, err := src.List(ctx, snapshot.GVRExtSecret, "")
	if err != nil || len(list.Items) == 0 {
		return nil
	}

	// Build the requirement map: vault-path → set of required keys.
	// Each requirement also tracks the set of consuming ExternalSecrets
	// so the diagnostic can name a culpable resource.
	requirements := map[string]*vaultPathReq{}
	for i := range list.Items {
		eso := list.Items[i]
		collectVaultRefs(eso, requirements)
	}

	// Query Vault once per unique path. If the path is missing or a key
	// is absent, emit a Diagnostic. Path-level cache is implicit since
	// the requirements map already deduped on path.
	var out []Diagnostic
	seen := map[string]struct{}{}

	// Iterate in deterministic order for stable test output.
	paths := make([]string, 0, len(requirements))
	for p := range requirements {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		req := requirements[p]
		actualKeys, err := a.Client.ListKeys(ctx, p)
		if errors.Is(err, vault.ErrPathNotFound) {
			dedupe := "missing-vault-path/" + p
			if _, dup := seen[dedupe]; dup {
				continue
			}
			seen[dedupe] = struct{}{}
			out = append(out, Diagnostic{
				Subject: dedupe,
				Message: fmt.Sprintf(
					"Vault path `%s` does NOT exist (referenced by %s). "+
						"Either restore the path in Vault, or update/remove the ExternalSecret. "+
						"Existing pods are still alive (the K8s Secret was last cached when the path existed); "+
						"NEXT pod restart will hit CreateContainerConfigError.",
					p, formatConsumers(req.consumers),
				),
			})
			continue
		}
		if err != nil {
			// Transport / auth error — emit one diagnostic per path so the
			// operator sees coverage gaps but a single Vault outage doesn't
			// drown the report. (Throttled by the dedupe map.)
			dedupe := "vault-error/" + p
			if _, dup := seen[dedupe]; dup {
				continue
			}
			seen[dedupe] = struct{}{}
			out = append(out, Diagnostic{
				Subject: dedupe,
				Message: fmt.Sprintf(
					"VaultPathMissing analyzer could not query path `%s`: %s. "+
						"Check Vault address, network reachability, and the cha ServiceAccount's Vault role binding.",
					p, truncate(err.Error(), 200),
				),
			})
			continue
		}
		actual := make(map[string]struct{}, len(actualKeys))
		for _, k := range actualKeys {
			actual[k] = struct{}{}
		}
		// For data[].remoteRef refs we know the specific keys required.
		// For dataFrom[].extract refs we just verified the path exists.
		for _, requiredKey := range sortedSet(req.requiredKeys) {
			if _, ok := actual[requiredKey]; ok {
				continue
			}
			dedupe := "vault-missing-key/" + p + "/" + requiredKey
			if _, dup := seen[dedupe]; dup {
				continue
			}
			seen[dedupe] = struct{}{}
			haveSummary := strings.Join(actualKeys, ", ")
			if len(haveSummary) > 100 {
				haveSummary = haveSummary[:97] + "..."
			}
			out = append(out, Diagnostic{
				Subject: dedupe,
				Message: fmt.Sprintf(
					"Vault path `%s` exists but is missing key `%s` (required by %s). "+
						"K8s Secret derived from this path is stale-Ready; pod restart will fail. "+
						"Existing keys at the path: [%s].",
					p, requiredKey, formatConsumers(req.consumers), haveSummary,
				),
			})
		}
	}
	return out
}

// vaultPathReq tracks what an ExternalSecret cluster-wide requires from
// a given Vault KV-v2 path.
type vaultPathReq struct {
	// requiredKeys is the union of keys explicitly named via
	// spec.data[].remoteRef.property. Empty for paths only consumed via
	// spec.dataFrom[].extract (path-level existence is the only check).
	requiredKeys map[string]struct{}
	// consumers names "ns/name" of each ExternalSecret that references
	// this path. Used to build the diagnostic's "referenced by …" line.
	consumers map[string]struct{}
}

// collectVaultRefs walks an ExternalSecret's spec and adds its references
// into the requirements map.
func collectVaultRefs(eso unstructured.Unstructured, requirements map[string]*vaultPathReq) {
	consumer := eso.GetNamespace() + "/" + eso.GetName()

	// spec.data[]: property-level granularity.
	data, _, _ := unstructured.NestedSlice(eso.Object, "spec", "data")
	for _, d := range data {
		dm, ok := d.(map[string]any)
		if !ok {
			continue
		}
		remoteRef, _ := dm["remoteRef"].(map[string]any)
		if remoteRef == nil {
			continue
		}
		key, _ := remoteRef["key"].(string)
		property, _ := remoteRef["property"].(string)
		if key == "" {
			continue
		}
		req := requirements[key]
		if req == nil {
			req = &vaultPathReq{
				requiredKeys: map[string]struct{}{},
				consumers:    map[string]struct{}{},
			}
			requirements[key] = req
		}
		if property != "" {
			req.requiredKeys[property] = struct{}{}
		}
		req.consumers[consumer] = struct{}{}
	}

	// spec.dataFrom[].extract: path-level (whole-secret) reference.
	dataFrom, _, _ := unstructured.NestedSlice(eso.Object, "spec", "dataFrom")
	for _, df := range dataFrom {
		dfm, ok := df.(map[string]any)
		if !ok {
			continue
		}
		extract, _ := dfm["extract"].(map[string]any)
		if extract == nil {
			continue
		}
		key, _ := extract["key"].(string)
		if key == "" {
			continue
		}
		req := requirements[key]
		if req == nil {
			req = &vaultPathReq{
				requiredKeys: map[string]struct{}{},
				consumers:    map[string]struct{}{},
			}
			requirements[key] = req
		}
		req.consumers[consumer] = struct{}{}
	}
}

func formatConsumers(set map[string]struct{}) string {
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, "ExternalSecret/"+c)
	}
	sort.Strings(out)
	if len(out) > 3 {
		return strings.Join(out[:3], ", ") + fmt.Sprintf(" (+%d more)", len(out)-3)
	}
	return strings.Join(out, ", ")
}

func sortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
