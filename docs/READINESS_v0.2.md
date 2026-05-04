# v0.2 readiness — vs. the Vault → Pod Drift solution brief

This document is the cha team's updated readiness assessment of the
[Vault → Pod Drift solution brief](./vault_pod_drift_solution_brief.docx.pdf)
post-v0.2. The brief defined a five-layer detection stack (L1–L5);
v0.1 covered three of them, partially. v0.2 closes the remaining three
gaps and tightens the partial coverage to full. Read alongside
[ADVERSARIAL_ANALYSIS_v0.2.md](./ADVERSARIAL_ANALYSIS_v0.2.md).

## 1. Brief's five-layer stack — coverage matrix

| Layer | Brief's intent | v0.1 coverage | v0.2 coverage | How |
|---|---|---|---|---|
| **L1** Vault stale-Ready window | Catch Vault edits BEFORE the ESO controller refreshes | ❌ none | ✅ **NEW** | `VaultPathMissing` analyzer queries Vault directly |
| **L2** Failing ExternalSecret detection | Catch ESOs reporting `Ready=False` | ✅ `FailingExternalSecrets` | ✅ same | unchanged |
| **L3** Failing Pod with bad Secret ref | Catch CCE pods, name the missing key | ✅ `SecretKeyMissing` | ✅ same | unchanged |
| **L4** Proactive bipartite-graph drift | Walk Deployment+SS env refs vs. live Secret keys, flag drift before pod restart | ❌ none | ✅ **NEW** | `ProactiveSecretKeyCheck` analyzer |
| **L5** kubectl-queryable diagnostic objects | Each diagnostic visible as a CR with status + history | ❌ none | ✅ **NEW** | `DriftReport` CRD + reconciler |

## 2. New code shipped in v0.2

| PR | Surface | Test count | Risk |
|---|---|---|---|
| #14 | `internal/diagnose/proactive_secret_key_check.go` | 7 | low — pure read-only; privacy contract enforced by code shape |
| #14 | reader ClusterRole `+secrets get,list` | n/a | **medium — see adversarial §2.1** |
| #15 | `charts/.../crd-driftreport.yaml` (CRD) | n/a | low — v1alpha1, schema explicitly unstable |
| #15 | `charts/.../clusterrole-driftreport.yaml` (writer role) | n/a | low — separate role, single responsibility |
| #15 | `internal/report/driftreport.go` (reconciler) | 5 | low — Mutator interface gate keeps snapshot mode read-only |
| #15 | `internal/snapshot/mutator.go` (+Create method) | n/a | low — interface extension, all impls updated |
| #16 | `internal/diagnose/vault_path_missing.go` | 9 | low — opt-in; nil-Client = no-op |
| #16 | `internal/vault/{client,auth,client_test}.go` | 6 | low — hand-rolled HTTP, no SDK dep |
| #16 | `charts/.../values.yaml` (vaultProbe block) | n/a | low — opt-in; default off |

**Aggregate**: 27 new tests, 0 new code dependencies, 1 new CRD, 1 new
ClusterRole, 1 RBAC expansion (Secret read).

## 3. Capability deltas vs. the brief

| Brief capability | v0.1 had it? | v0.2 has it? |
|---|---|---|
| Detect Vault path deletion before pod restart | no | **yes** |
| Detect Vault key removal before pod restart | no | **yes** |
| Detect ExternalSecret/Vault drift with no error in ESO yet | no | **yes** |
| Detect Deployment env reference to missing K8s Secret key (pre-restart) | no | **yes** |
| Detect Deployment env reference to nonexistent K8s Secret (pre-restart) | no | **yes** |
| Surface diagnostics as queryable cluster objects | no (Slack/JSON only) | **yes** (DriftReport CR) |
| Diagnostic objects show first-observed / last-observed / observation count | no | **yes** (CRD `.status` subresource) |
| Auto-cleanup resolved diagnostics | no | **yes** (reconciler deletes CRs whose subjects no longer reported) |
| Run in zero-trust snapshot mode | yes | yes (no Vault probe in snapshot — by design) |
| Run live with kubernetes-auth Vault role | n/a | **yes** |
| Run live with VAULT_TOKEN | n/a | **yes** (dev posture) |
| OSS Apache 2.0 engine | yes | yes |
| Helm chart with toggleable probes | yes (diagnose, remediate) | **yes** (+ DriftReport, vaultProbe) |

## 4. Gaps that remain

These are **not** addressed in v0.2 and **not** in the brief; surfaced
here so design partners aren't surprised.

| Gap | Rationale | Target |
|---|---|---|
| Multi-Vault / multi-SecretStore support | Brief assumed one Vault per cluster | v0.4 |
| envFrom.secretRef walk in proactive analyzer | `envFrom` whole-secret import doesn't reference specific keys | v0.3 |
| Vault-outage diagnostic dedupe | Single-issue summary instead of per-path diagnostics | v0.3 |
| SecretStore-provider filter (skip non-Vault ESOs in VaultPathMissing) | Mixed-provider clusters get noisy false-positives | v0.3 |
| Self-hosted DriftReport viewer | Currently kubectl + grep; a tiny web UI is post-fundraise | v1.0+ |
| Trend / time-series storage | DriftReport `.status.observationCount` is the closest thing | v1.0+ (Fleet Console scope) |
| Cross-cluster aggregation | Single-cluster scope; multi-cluster is the commercial wedge | post-fundraise |

## 5. Net readiness

> **Are we ready to take the brief to a design-partner conversation?**

**Yes.** v0.2 closes every L1–L5 gap the brief specifies. The
[adversarial analysis](./ADVERSARIAL_ANALYSIS_v0.2.md) flagged zero
must-fix items, three will-fix items (all scheduled for v0.3, all with
clear resolutions), and five document items (all already captured in
SECURITY.md / values.yaml / README).

The honest disclosures we'd make to the design partner:

1. **CRD is v1alpha1.** We will change the schema before v1beta1.
   Consumer scripts should pin on `additionalPrinterColumns` rather
   than full JSON paths.
2. **Vault probe is Vault-exclusive.** Mixed-provider clusters should
   leave it disabled until v0.3 ships the SecretStore filter.
3. **Reader role grants cluster-wide Secret read.** The code-level
   privacy contract (analyzer iterates `for k := range secret.Data`
   only) is documented in the role manifest. Partners with strict
   data-governance constraints can disable `ProactiveSecretKeyCheck`
   and revoke the rule.
4. **Vault role scoping is the operator's responsibility.** The chart
   doesn't install a Vault role; the partner's security team must
   author one scoped to the paths their ExternalSecrets reference.

## 6. Pre-launch checklist

- [x] PR #14 merged (Gap 3)
- [x] PR #15 merged (Gap 2)
- [ ] PR #16 merged (Gap 1) — CI in progress
- [ ] `v0.2.0` tag pushed
- [ ] GoReleaser workflow green (4 binary tarballs + multi-arch image)
- [ ] Helm chart `0.2.0` `helm install --dry-run` clean against the prod cluster
- [ ] Smoke test on the prod cluster — `vaultProbe.enabled=true`, observe DriftReport CRs round-trip

## 7. What "ready" does NOT mean

- It does **not** mean we have product-market fit. We have a technically
  defensible product against the brief's specific scenario.
- It does **not** mean we are SOC 2 ready. SOC 2 is post-fundraise scope.
- It does **not** mean we have a paying customer. Design-partner outreach
  starts in WS-D week 11 of the productization plan.
- It does **not** mean v0.2 is feature-complete. It means v0.2 is the
  smallest cha that can credibly answer the brief's stated needs.
