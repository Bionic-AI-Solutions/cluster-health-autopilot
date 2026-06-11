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
		logListFailure("externalsecrets", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	// Build the Vault-backed-store filter. Without this, a cluster running
	// ESOs against AWS Secrets Manager / GCP Secret Manager / etc. would
	// have every non-Vault path queried against Vault and return 404 or
	// permission-denied → noisy false positives. We resolve each ESO's
	// secretStoreRef to a SecretStore or ClusterSecretStore and check
	// whether spec.provider.vault is set.
	//
	// On clusters where ESO CRDs are not installed we get zero stores and
	// the analyzer falls through to the v0.2 behavior (treat every ESO
	// as Vault-backed). That's the conservative choice — better to emit
	// "vault path not found" than to silently skip a real drift event.
	vaultStoreNS := loadVaultBackedStores(ctx, src)

	// When the reader ClusterRole lacks secretstores/clustersecretstores RBAC
	// (e.g. binary upgraded without `helm upgrade`), emit one informational
	// diagnostic rather than silently degrading to the all-ESOs-are-Vault mode.
	var rbacHintOut []Diagnostic
	if vaultStoreNS.rbacHint != "" {
		rbacHintOut = []Diagnostic{{
			Subject: "vault-store-rbac-missing",
			Message: fmt.Sprintf(
				"VaultPathMissing: could not list SecretStore/ClusterSecretStore objects (%s). "+
					"Falling back to treating ALL ExternalSecrets as Vault-backed — expect false-positive diagnostics "+
					"on non-Vault stores. Run `helm upgrade` to apply the updated reader ClusterRole.",
				vaultStoreNS.rbacHint,
			),
		}}
	}

	// Build the requirement map: vault-path → set of required keys.
	// Each requirement also tracks the set of consuming ExternalSecrets
	// so the diagnostic can name a culpable resource.
	requirements := map[string]*vaultPathReq{}
	for i := range list.Items {
		eso := list.Items[i]
		if !isVaultBackedESO(eso, vaultStoreNS) {
			continue
		}
		collectVaultRefs(eso, requirements)
	}

	// Query Vault once per unique path. If the path is missing or a key
	// is absent, emit a Diagnostic. Path-level cache is implicit since
	// the requirements map already deduped on path.
	//
	// Transport-level errors (Vault unreachable, auth expired, etc.) are
	// collected separately and rendered in batch at the end so a single
	// outage doesn't spam one diagnostic per ESO path. See vaultOutageThreshold.
	var out []Diagnostic
	seen := map[string]struct{}{}
	transportErrs := map[string][]string{} // err.Error() → paths

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
			// Defer transport errors — emit in batch after the loop.
			transportErrs[err.Error()] = append(transportErrs[err.Error()], p)
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

	out = append(out, renderTransportErrors(transportErrs)...)
	return append(rbacHintOut, out...)
}

// vaultOutageThreshold collapses N per-path errors of the same shape into
// a single summary diagnostic when N reaches this count. Below the
// threshold we keep emitting per-path so an isolated misconfiguration
// (one wrong path-prefix on one ESO) still surfaces visibly.
const vaultOutageThreshold = 3

func renderTransportErrors(errs map[string][]string) []Diagnostic {
	if len(errs) == 0 {
		return nil
	}
	// Iterate error groups in deterministic order — sorted by error string.
	keys := make([]string, 0, len(errs))
	for k := range errs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []Diagnostic
	for _, errStr := range keys {
		paths := errs[errStr]
		sort.Strings(paths)
		if len(paths) >= vaultOutageThreshold {
			samplePaths := paths
			if len(samplePaths) > 3 {
				samplePaths = append([]string{}, paths[:3]...)
			}
			out = append(out, Diagnostic{
				Subject: "vault-outage/" + truncate(errStr, 80),
				Message: fmt.Sprintf(
					"VaultPathMissing analyzer could not query Vault: %d paths failed with `%s`. "+
						"Sample paths: %s%s. "+
						"Check Vault address, network reachability, and the cha ServiceAccount's Vault role binding.",
					len(paths),
					truncate(errStr, 200),
					strings.Join(samplePaths, ", "),
					func() string {
						if len(paths) > len(samplePaths) {
							return fmt.Sprintf(" (+%d more)", len(paths)-len(samplePaths))
						}
						return ""
					}(),
				),
			})
			continue
		}
		// Below threshold — emit per-path so individual misconfigs are visible.
		for _, p := range paths {
			out = append(out, Diagnostic{
				Subject: "vault-error/" + p,
				Message: fmt.Sprintf(
					"VaultPathMissing analyzer could not query path `%s`: %s. "+
						"Check Vault address, network reachability, and the cha ServiceAccount's Vault role binding.",
					p, truncate(errStr, 200),
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

// vaultStoreSet identifies which SecretStore/ClusterSecretStore objects
// are configured with a Vault provider.
//
// The two halves are queried separately because SecretStore is namespaced
// (and identified by ns+name) while ClusterSecretStore is cluster-scoped
// (identified by name alone). At ESO-resolution time, secretStoreRef.kind
// determines which set to consult.
type vaultStoreSet struct {
	// namespaced[ns][name] = true when the SecretStore is Vault-backed.
	namespaced map[string]map[string]bool
	// cluster[name] = true when the ClusterSecretStore is Vault-backed.
	cluster map[string]bool
	// hasAnyStore is true when at least one SecretStore or ClusterSecretStore
	// was discovered. When false, the ESO CRD's resolution chain isn't
	// known to this analyzer and we fall back to the v0.2 "treat all as
	// Vault" behavior so we don't silently drop real drift detection.
	hasAnyStore bool
	// rbacHint is non-empty when a SecretStore/ClusterSecretStore list call
	// failed (typically RBAC forbidden). It is emitted as one informational
	// diagnostic so operators know to run `helm upgrade` to apply the updated
	// ClusterRole rather than discovering the fallback mode silently.
	rbacHint string
}

func loadVaultBackedStores(ctx context.Context, src snapshot.Source) *vaultStoreSet {
	out := &vaultStoreSet{
		namespaced: map[string]map[string]bool{},
		cluster:    map[string]bool{},
	}

	var listErr error
	if list, err := src.List(ctx, snapshot.GVRClusterSecretStore, ""); err == nil {
		for _, css := range list.Items {
			out.hasAnyStore = true
			if hasVaultProvider(css) {
				out.cluster[css.GetName()] = true
			}
		}
	} else {
		listErr = err
	}
	if list, err := src.List(ctx, snapshot.GVRSecretStore, ""); err == nil {
		for _, ss := range list.Items {
			out.hasAnyStore = true
			if !hasVaultProvider(ss) {
				continue
			}
			ns := ss.GetNamespace()
			if out.namespaced[ns] == nil {
				out.namespaced[ns] = map[string]bool{}
			}
			out.namespaced[ns][ss.GetName()] = true
		}
	} else if listErr == nil {
		listErr = err
	}

	// If we couldn't list any store CRDs and got an error, record a hint so
	// the caller can emit one informational diagnostic. The fallback behavior
	// (treat all ESOs as Vault-backed) is still correct but now visible.
	if !out.hasAnyStore && listErr != nil {
		out.rbacHint = truncate(listErr.Error(), 200)
	}
	return out
}

// hasVaultProvider returns true if the (Cluster)SecretStore's spec.provider
// has a non-nil `vault` key. ESO encodes provider type as the only present
// child key under spec.provider, so presence of `vault` is the signal.
func hasVaultProvider(store unstructured.Unstructured) bool {
	provider, found, _ := unstructured.NestedMap(store.Object, "spec", "provider")
	if !found {
		return false
	}
	_, ok := provider["vault"]
	return ok
}

// isVaultBackedESO reports whether an ExternalSecret's secretStoreRef
// resolves to a Vault-backed (Cluster)SecretStore. When no stores were
// loaded at all (ESO CRDs absent / RBAC denied), we degrade gracefully
// to v0.2 behavior and return true — preserving false-negative-free
// detection at the cost of false-positive noise on mixed-provider clusters
// where the operator hasn't given cha access to the SecretStore CRDs.
func isVaultBackedESO(eso unstructured.Unstructured, set *vaultStoreSet) bool {
	if !set.hasAnyStore {
		return true
	}
	storeName, _, _ := unstructured.NestedString(eso.Object, "spec", "secretStoreRef", "name")
	if storeName == "" {
		return false
	}
	storeKind, _, _ := unstructured.NestedString(eso.Object, "spec", "secretStoreRef", "kind")
	if storeKind == "" {
		// ESO default is SecretStore.
		storeKind = "SecretStore"
	}
	switch storeKind {
	case "ClusterSecretStore":
		return set.cluster[storeName]
	case "SecretStore":
		ns := eso.GetNamespace()
		return set.namespaced[ns] != nil && set.namespaced[ns][storeName]
	default:
		return false
	}
}
