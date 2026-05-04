// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ProactiveSecretKeyCheck — bipartite-graph walk of every Deployment's and
// StatefulSet's `env.valueFrom.secretKeyRef` against the live Secret's keys.
//
// This is the "pre-crash" complement to SecretKeyMissing: it catches Secret-
// key drift BEFORE a pod restart turns it into a CreateContainerConfigError.
// Specifically, the case where a referenced Secret exists but the named key
// has been removed from it (e.g. ExternalSecret template refactor, manual
// `kubectl edit secret` that dropped a key).
//
// Privacy contract: this analyzer reads Secret resources but ONLY inspects
// `metadata.name` and the SET OF KEY NAMES via `for k := range secret.Data`.
// It never reads the byte values stored under each key, never logs them,
// never includes them in Diagnostic output. Adding `secrets get,list` to
// the reader role is a deliberate position — same scope the brief's L4
// proposes, with the value-read defended by code convention.
type ProactiveSecretKeyCheck struct{}

// Name returns the analyzer's identifier.
func (ProactiveSecretKeyCheck) Name() string { return "ProactiveSecretKeyCheck" }

// Run produces the diagnostic set.
func (ProactiveSecretKeyCheck) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	// Build the Secret-key inventory: { (ns, name) → set-of-key-names }.
	secrets, err := src.List(ctx, snapshot.GVRSecret, "")
	if err != nil {
		// In snapshot mode, Secrets are intentionally absent (see capture.go).
		// Without the inventory we can't compute drift, so the analyzer is a no-op.
		return nil
	}
	keysBySecret := make(map[string]map[string]struct{}, len(secrets.Items))
	for _, s := range secrets.Items {
		key := s.GetNamespace() + "/" + s.GetName()
		data, _, _ := unstructured.NestedMap(s.Object, "data")
		set := make(map[string]struct{}, len(data))
		for k := range data {
			set[k] = struct{}{}
		}
		keysBySecret[key] = set
	}

	// Walk all Deployments + StatefulSets and resolve every secretKeyRef.
	// Dedupe diagnostics by (secret-ns, secret-name, missing-key) so a
	// Deployment with 5 replicas → 1 diagnostic.
	seen := map[string]struct{}{}
	var out []Diagnostic

	check := func(workloadKind string, items []unstructured.Unstructured) {
		for i := range items {
			obj := items[i]
			ns := obj.GetNamespace()
			name := obj.GetName()
			containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
			initContainers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "initContainers")
			allContainers := append(containers, initContainers...)
			for _, c := range allContainers {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				// Per-key references via env[].valueFrom.secretKeyRef.
				envs, _, _ := unstructured.NestedSlice(cm, "env")
				for _, e := range envs {
					em, ok := e.(map[string]any)
					if !ok {
						continue
					}
					vf, _ := em["valueFrom"].(map[string]any)
					if vf == nil {
						continue
					}
					skr, _ := vf["secretKeyRef"].(map[string]any)
					if skr == nil {
						continue
					}
					secretName, _ := skr["name"].(string)
					keyName, _ := skr["key"].(string)
					optional, _ := skr["optional"].(bool)
					if secretName == "" || keyName == "" {
						continue
					}
					secretKey := ns + "/" + secretName
					keys, exists := keysBySecret[secretKey]
					if !exists {
						// Secret doesn't exist at all. The brief's L5 catches
						// this AFTER the pod fails; we catch it BEFORE.
						// Optional refs are skipped — the kubelet honors
						// `optional: true` and the workload runs fine.
						if optional {
							continue
						}
						dedupe := "missing-secret/" + secretKey
						if _, dup := seen[dedupe]; dup {
							continue
						}
						seen[dedupe] = struct{}{}
						out = append(out, Diagnostic{
							Subject: dedupe,
							Message: fmt.Sprintf(
								"Secret `%s` does NOT exist (referenced by %s/%s in ns %s, env key `%s`). "+
									"Pod will fail to start on next restart. Create the Secret or remove the env reference.",
								secretKey, workloadKind, name, ns, keyName,
							),
						})
						continue
					}
					if _, hasKey := keys[keyName]; !hasKey {
						if optional {
							continue
						}
						dedupe := "missing-key/" + secretKey + "/" + keyName
						if _, dup := seen[dedupe]; dup {
							continue
						}
						seen[dedupe] = struct{}{}
						haveKeys := sortedKeys(keys)
						haveSummary := strings.Join(haveKeys, ", ")
						if len(haveSummary) > 100 {
							haveSummary = haveSummary[:97] + "..."
						}
						out = append(out, Diagnostic{
							Subject: dedupe,
							Message: fmt.Sprintf(
								"Secret `%s` exists but is missing key `%s` (referenced by %s/%s in ns %s). "+
									"Pod will hit CreateContainerConfigError on next restart. Existing keys: [%s].",
								secretKey, keyName, workloadKind, name, ns, haveSummary,
							),
						})
					}
				}

				// Whole-secret import via envFrom[].secretRef. Pre-restart we
				// can only verify the SECRET exists — there's no key list to
				// validate against because envFrom imports every key. Missing
				// the Secret entirely will hard-fail pod start regardless of
				// which keys downstream consumers actually use.
				envFroms, _, _ := unstructured.NestedSlice(cm, "envFrom")
				for _, ef := range envFroms {
					efm, ok := ef.(map[string]any)
					if !ok {
						continue
					}
					sref, _ := efm["secretRef"].(map[string]any)
					if sref == nil {
						continue
					}
					secretName, _ := sref["name"].(string)
					if secretName == "" {
						continue
					}
					optional, _ := sref["optional"].(bool)
					secretKey := ns + "/" + secretName
					if _, exists := keysBySecret[secretKey]; exists {
						continue
					}
					if optional {
						continue
					}
					dedupe := "missing-secret/" + secretKey
					if _, dup := seen[dedupe]; dup {
						continue
					}
					seen[dedupe] = struct{}{}
					out = append(out, Diagnostic{
						Subject: dedupe,
						Message: fmt.Sprintf(
							"Secret `%s` does NOT exist (referenced by %s/%s in ns %s, envFrom whole-secret import). "+
								"Pod will fail to start on next restart. Create the Secret or remove the envFrom entry.",
							secretKey, workloadKind, name, ns,
						),
					})
				}
			}
		}
	}

	deploys, _ := src.List(ctx, snapshot.GVRDeployment, "")
	if deploys != nil {
		check("Deployment", deploys.Items)
	}
	statefulsets, _ := src.List(ctx, snapshot.GVRStatefulSet, "")
	if statefulsets != nil {
		check("StatefulSet", statefulsets.Items)
	}
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
