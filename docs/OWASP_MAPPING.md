# OWASP Kubernetes Top-10 Mapping

This document maps every CHA **fixer** (`internal/fix/*.go`) and every
security-relevant **analyzer / signal** (`internal/diagnose/*.go`) to the
[OWASP Kubernetes Top-10](https://owasp.org/www-project-kubernetes-top-ten/)
item(s) it relates to.

The distinction below is deliberate and load-bearing:

- **Fixers RESPECT** an OWASP item: they perform cluster mutations, so the
  claim is *negative* — "this fixer's mutation never weakens the cluster's
  posture against item K0x." That claim is not an aspiration; it is locked by
  `internal/fix/owasp_posture_test.go`, a posture-non-regression guard that
  inspects every mutation each fixer can emit and fails the build if any of
  them would add a privileged container, broaden RBAC, remove/weaken a
  NetworkPolicy, downgrade a TLS secret reference, or delete a resource in a
  protected namespace.

- **Analyzers DETECT** an OWASP item: they are *observational only*. They
  surface a gap so an operator (or the paid AI/approval tier) can react.
  **Detection is not enforcement.** CHA's OSS analyzers never apply a
  NetworkPolicy, never relabel a namespace for Pod Security Standards, never
  rewrite an RBAC rule. The `NetworkPolicyProposer` emits ready-to-apply YAML
  but explicitly does **not** apply it.

## OWASP Kubernetes Top-10 (reference)

| ID  | Item |
| --- | --- |
| K01 | Insecure Workload Configurations |
| K02 | Supply Chain Vulnerabilities |
| K03 | Overly Permissive RBAC |
| K04 | Lack of Centralized Policy Enforcement |
| K05 | Inadequate Logging and Monitoring |
| K06 | Broken Authentication |
| K07 | Missing Network Segmentation Controls |
| K08 | Secrets Management Failures |
| K09 | Misconfigured Cluster Components |
| K10 | Outdated and Vulnerable Components |

## Fixers — RESPECT (proven not to violate)

Each fixer is whitelisted, idempotent, namespace-protected, and GitOps-aware.
None of them ever writes a Secret, ConfigMap, or CRD (`pkg/fix` contract). The
"OWASP respected" column is the set of items the posture guard actively asserts
the fixer cannot regress.

| Fixer (`internal/fix/…`) | Mutation it performs | OWASP respected | Why it cannot weaken posture |
| --- | --- | --- | --- |
| `StaleErrorPods` | `Delete` of a `Failed`-phase Pod that is Job-owned or has no controller owner | **K01** (config), **K07** (segmentation) | Deletes only stuck/orphan Pods; never edits a PodSpec, so it can't add `privileged`, `hostPath`, `hostNetwork`, or capabilities. Deletes are namespace-protected and never touch a NetworkPolicy. |
| `StuckJobsWithBadSecretRef` | `Delete` of a frozen Job (bad Secret-key ref) whose parent CronJob still exists | **K01** (config), **K08** (secrets) | Deletes a frozen Job so the corrected CronJob template respawns; it never reads, writes, or weakens a Secret — it unblocks a *fixed* Secret reference. No PodSpec mutation. |
| `StuckRSPods` | `Patch` (strategic-merge) adding `kubectl.kubernetes.io/restartedAt` to a Deployment template, iff the live Deployment revision has already rolled past the stuck ReplicaSet | **K01** (config) | The only field written is a restart-timestamp annotation — equivalent to `kubectl rollout restart`. It cannot introduce `privileged`/`hostPath`/`hostNetwork`/capabilities and explicitly refuses when the failure is a missing Secret key (a rollout would just reproduce it). |
| `StuckCertificateRequests` | `Delete` of a terminally-failed cert-manager `CertificateRequest` / ACME `Order` | **K08** (secrets/TLS) | cert-manager immediately recreates the deleted CR and retries issuance; CHA never writes the resulting TLS Secret. Deleting a *failed* request cannot downgrade a live cert. Health-gated on the cert-manager controller being up. |
| `TLSSecretMismatch` *(opt-in)* | `Patch` (JSONPatch) repointing an Ingress `spec.tls[].secretName` from a stale Secret to a healthy cert-manager-managed Secret for the **same** host | **K08** (secrets/TLS), **K01** (config) | Only ever repoints *to* a cert-manager Certificate that is `Ready=True` and whose `dnsNames` include the Ingress host — i.e. an **upgrade**, never a downgrade. High-confidence match required (stale secret expired/expiring + better managed cert exists). GitOps-protected. |

### Posture guard coverage

`internal/fix/owasp_posture_test.go` enumerates **all five** fixers and, for
each, runs it against a fixture and asserts the produced mutation never:

- removes or weakens a NetworkPolicy (**K07**);
- adds `privileged: true`, `hostPath`, `hostNetwork`, or a capability add (**K01**);
- broadens RBAC — adds verbs/resources or widens a binding (**K03**);
- downgrades/removes a TLS secret reference or swaps to a weaker one (**K08**);
- deletes a resource in a protected namespace (**already enforced**, re-asserted).

The guard also runs a **meta-check**: it scans `internal/fix/*.go` for every
type implementing `Name() string` (the `Fixer` marker) and fails if any fixer
type is missing from the test table. **A new fixer cannot silently skip the
guard** — adding one without a posture-test entry breaks `go test ./...`.

## Analyzers — DETECT (observational, not enforcement)

| Analyzer / signal (`internal/diagnose/…`) | What it surfaces | OWASP detected |
| --- | --- | --- |
| `SecurityDrift` → PSS posture gap | User namespaces with no `pod-security.kubernetes.io/enforce` label, or `enforce=privileged` | **K01**, **K04** |
| `SecurityDrift` → mutable image tag (the "digest-pin proposer") | Pods whose containers reference images by tag only, not by `@sha256:` digest | **K02**, **K10** |
| `SecurityDrift` → NetworkPolicy coverage gap | User namespaces running pods with zero NetworkPolicies (on enforcing CNIs) | **K07** |
| `NetworkPolicyProposer` | Per-namespace ready-to-apply NetworkPolicy YAML for uncovered namespaces on enforcing CNIs (emits; never applies) | **K07**, **K04** |
| `RBACDrift` → out-of-band RBAC edit | Role/RoleBinding/ClusterRole/ClusterRoleBinding edited outside the deploy pipeline (diverges from `last-applied`) | **K03**, **K05** |
| `RBACDrift` → wildcard-verb role | Non-system Role/ClusterRole with a `verbs: ["*"]` rule | **K03** |
| `RBACDrift` → ServiceAccount with no RoleBinding | SA mounted to a Deployment but referenced by no binding | **K03**, **K06** |
| `TLSSecretMismatch` (diagnose) | Ingress serving an expired/expiring hand-made cert while a healthy cert-manager Certificate renews into a different, unreferenced Secret | **K08** |
| `CertExpiry` | TLS Secrets / Certificates expired or expiring within the window | **K08** |
| `FailingExternalSecrets` | ExternalSecret CRs in a failed sync state | **K08** |
| `SecretKeyMissing` / `ProactiveSecretKeyCheck` | Pods referencing a Secret key that does not exist (CreateContainerConfigError) | **K08** |
| `UnprovisionedSecret` | Workloads referencing a Secret that was never created | **K08** |
| `ImagePullAuth` | Image-pull auth failures (missing/invalid `imagePullSecrets`) | **K02**, **K06** |
| `VaultPathMissing` | Vault paths referenced by ExternalSecrets that don't resolve | **K08** |
| `LogPatternMatcher` | High-signal failure events incl. `Forbidden` (RBAC denials) | **K03**, **K05** |
| `ConfigDrift` | ConfigMap/Deployment drift from `last-applied` (out-of-band edits) | **K01**, **K05** |
| `GitOpsDrift` / `WorkloadStateDrift` | Resources diverging from their GitOps source of truth | **K04**, **K05** |

## Honesty notes

- The OSS tier is **detect-only** for K02/K03/K04/K06/K07/K10. The remediation
  path for those classes runs through the paid AI/approval tier
  (`ApprovalProposal` CR + Slack Approve/Deny), which is human- or
  policy-gated — not an autofixer.
- The five fixers above are the **entire** set of cluster-mutating actions CHA
  ships. They are narrow by design: delete a stuck Pod/Job/failed-cert-request,
  or patch a restart annotation / a TLS `secretName` to the correct value.
  None of them touches RBAC, NetworkPolicy, PodSpec security fields, or Secret
  contents — which is exactly why the posture guard's assertions pass today.
  The guard's value is **forward-looking**: it locks the posture so a future
  fixer cannot introduce a regression unnoticed.
