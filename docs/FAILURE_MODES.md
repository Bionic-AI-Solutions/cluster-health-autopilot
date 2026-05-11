# Failure-Mode Catalog

The default catalog (v0.9.5) covers **four auto-fixers** and **eight analyzers**.
Each is a separate Go function with a published unit-test corpus under
[`internal/`](../internal/). The "real example" column is a real incident from
the cluster the product was built on.

---

## A. Auto-fixers (whitelist; default-OFF; opt-in via `remediation.enabled=true`)

Each fixer is gated by the **Mutator interface** at the type-system level: `snapshot.File` doesn't implement `Mutator`, so fixers physically cannot run against a captured snapshot. Live mode only. The remediator ClusterRole grants five narrow verbs (`pods/delete`, `jobs/delete`, `deployments/patch`, `certificaterequests/delete`, `orders/delete`); no Secret, ConfigMap, or generic CRD writes are possible regardless of how the binary is called.

### 1. `StaleErrorPods`

| | |
|---|---|
| **Symptom** | `kubectl get pods -A` shows `Error` / `Failed` pods sitting around for hours/days. |
| **Root cause class** | Manual `kubectl debug` leftovers, `kubectl run --restart=Never` scratch pods, Job pods that crashed without a retry policy. The pod IS terminal — its presence in listings is just visual noise, but it skews `Pods/cluster` metrics and clutters incident reviews. |
| **What it does** | Filters pods to `status.phase == "Failed"`, owned by a Job or unowned. Skips RS/Deployment/StatefulSet/DaemonSet-owned pods (let the controller recover). Skips protected namespaces. Calls `pods.delete`. |
| **Why it's safe** | Job-owned pods that were going to retry already have a successor; the dead one is a tombstone. Unowned pods don't have a controller to recover them — they're already lost. The only "data loss" is one pod's container logs, which the Job retains via `backoffLimit`. |
| **Real example** | 3× `node-debugger-gpu-*` pods stuck in `Error` for 11–12 days from a node debug session that was never cleaned up. Caught and removed on the first tick of the bash-version predecessor; would be the same outcome with this fixer. |
| **Source** | [`internal/fix/stale_error_pods.go`](../internal/fix/stale_error_pods.go) |

### 2. `StuckJobsWithBadSecretRef`

| | |
|---|---|
| **Symptom** | A pod has been in `CreateContainerConfigError` for days; the parent CronJob is `concurrencyPolicy: Forbid` so the next tick can't run; the cron's effect (Slack alert, backup, scrub) silently stops working. |
| **Root cause class** | Someone updated the CronJob template to reference a Secret key that was renamed (case mismatch, ESO refactor). The CronJob template is correct *now*, but the in-flight Job's pod template is **immutable** — Kubernetes never rewrites it. The Job retries CreateContainer forever. |
| **What it does** | Finds pods in CCE with kubelet event message containing "couldn't find key", confirms the parent is a Job whose owner is a still-existing CronJob, calls `jobs.delete` on the frozen Job. The CronJob's next scheduled tick spawns a fresh Job using the corrected template. Skips Jobs without a CronJob owner — one-off Jobs deserve human attention. |
| **Why it's safe** | Deleting the Job kills the dead pod and lets the cron schedule do its thing. The Job's history is gone, but the Job had been failing anyway — there was no successful run to lose. |
| **Real example** | `gpu-docker-monitor` cron's Slack-webhook Secret got renamed `webhook-url` → `WEBHOOK_URL`. The frozen Job retried for **26 days** before being caught. With this fixer wired, it would be resolved on the next tick after the rename. |
| **Source** | [`internal/fix/stuck_jobs.go`](../internal/fix/stuck_jobs.go) |

### 3. `StuckRSPods`

| | |
|---|---|
| **Symptom** | Deployment shows `1/2 ready`. One pod is in `CreateContainerConfigError`. The "1" pod is from an older ReplicaSet revision; the "0/1" pod is the new revision that can't start. Rolling update is wedged. |
| **Root cause class** | A transient condition that has since cleared (image pull retry, scheduler quirk) prevented the new pod from starting at the moment of the rollout, and the ReplicaSet hasn't retried because `progressDeadlineSeconds` hasn't fired. |
| **What it does** | Detects a CCE pod owned by a ReplicaSet whose `revision` annotation differs from the live Deployment's `revision`. Patches the Deployment template's `kubectl.kubernetes.io/restartedAt` annotation — the same patch `kubectl rollout restart` produces. Kubernetes spawns a new ReplicaSet, the old wedged one ages out. **Refuses** to restart when the failure is "couldn't find key" — rollout would reproduce the same error against the same Secret; that needs human action and is handled by the analyzer below. |
| **Why it's safe** | `kubectl rollout restart` is the standard SRE remedy for stuck rollouts; the patched annotation is the documented mechanism. Kubernetes itself decides which pods get replaced and in what order. |
| **Real example** | `frontend-deploy` revision 7 stuck because revision 6's pod template had a different env layout that took longer than `progressDeadlineSeconds=600` to converge. Manual `kubectl rollout restart` from on-call cleared it; this fixer does the same automatically when the revision diff says it's safe. |
| **Source** | [`internal/fix/stuck_rs_pods.go`](../internal/fix/stuck_rs_pods.go) |

### 4. `StuckCertificateRequests` (added in v0.9.1)

| | |
|---|---|
| **Symptom** | `kubectl get certificaterequest -A` shows requests stuck in `Ready=False`/`reason=Failed` for hours/days; cert-manager won't retry because the failed CR is still present. ACME `Order` CRs in `state=errored` or `state=invalid` keep the parent CR wedged. |
| **Root cause class** | Transient ACME failure (rate limit, DNS-01 propagation slowness, account-key rotation) that the CR captured permanently. cert-manager creates one CR per renewal attempt — once a CR is terminally failed, the next renewal cycle stalls behind it. |
| **What it does** | Filters CertificateRequest CRs to terminally-failed states (`Ready=False` with `reason=Failed`, or `status.failureTime` set). Filters ACME Orders to `state=errored` or `state=invalid`. Calls `certificaterequests.delete` / `orders.delete`. cert-manager recreates the CR within seconds and retries. |
| **Why it's safe** | The fixer only touches **terminally-failed** CRs — never pending or in-progress issuance. cert-manager's recovery model is "delete the failed CR; I'll make a new one" — this fixer just automates that step. No private-key material is touched (Secrets are untouched). |
| **Real example** | `dashboard-baisoln-com-tls` Certificate showed `Ready=False` for 11 hours because a CertificateRequest from the previous renewal hit an ACME rate-limit and never cleared. Once deleted, cert-manager issued a fresh CR and renewed within 90 seconds. |
| **Source** | [`internal/fix/stuck_cert_requests.go`](../internal/fix/stuck_cert_requests.go) |

---

## B. Analyzers (read-only; never act; surface diagnostics for human action)

Analyzers run in both `cha diagnose` and `cha remediate`. They produce structured diagnostics that go into the Slack report's `Diagnostics` section, alongside any auto-fixes that were applied.

### 1. `SecretKeyMissing`

| | |
|---|---|
| **Symptom** | `couldn't find key X in Secret ns/name` on pod events. Pod stuck in CCE. The Deployment's env wires a key that doesn't exist on the live Secret. |
| **What it produces** | One Diagnostic line per `(secret_ns, secret_name, missing_key)` tuple, naming all four entities involved: missing key + Secret + consuming Deployment (resolved via `Pod → ReplicaSet → Deployment` owner chain) + owning ExternalSecret (resolved by matching `spec.target.name`). |
| **Why no auto-fix** | The fix is to either (a) edit the ExternalSecret template to expose the key, or (b) edit the Deployment env to reference an existing key. Both are git-side changes; an automated edit would conflict with version control and could fan out to other Deployments referencing the same Secret. |
| **Output (verbatim)** | `🔎 Secret mcp/mcp-openproject-secrets missing key openproject-url (referenced by Deployment/mcp-openproject-server in ns mcp). Owning ExternalSecret: mcp/mcp-openproject-secrets — add data/template entry exposing openproject-url, or remove the env reference if unused.` |
| **Source** | [`internal/diagnose/secret_key_missing.go`](../internal/diagnose/secret_key_missing.go) |

### 2. `FailingExternalSecrets`

| | |
|---|---|
| **Symptom** | An ExternalSecret has `Ready=False`. The Kubernetes Secret it owns still exists with a stale-but-cached value. Running pods don't notice; the next pod restart will. |
| **What it produces** | One Diagnostic line per failing ESO, with the most recent `UpdateFailed` event message — which carries the precise missing Vault property name. Falls back to the condition message when no events are in the snapshot. |
| **Why no auto-fix** | Resolution requires writing to Vault — not a place an automated tool should reach. The diagnostic surfaces the exact path + property name so the operator can `vault kv put` in seconds. |
| **Output (verbatim)** | `🔎 ExternalSecret mail/mail-service-api-key not Ready: error processing spec.data[0] (key: t6-apps/mail/config), err: cannot find secret data for key: "mail_service_api_key". Check Vault path / property names. Vault path 'counsellor/config' does not follow t6 hierarchy; expected: 'secret/t6-apps/mail/config'.` |
| **Source** | [`internal/diagnose/failing_externalsecrets.go`](../internal/diagnose/failing_externalsecrets.go) |

### 3. `ProactiveSecretKeyCheck`

| | |
|---|---|
| **Symptom** | A Deployment env reference points at a Secret key the live Secret object does NOT contain — the cluster is **silently primed to fail** on the next pod restart. The current pod is happy because its env was resolved at start. The next rollout will hit CreateContainerConfigError. |
| **What it produces** | Walks every Deployment/StatefulSet/CronJob `env[].valueFrom.secretKeyRef` and `envFrom[].secretRef`. For each reference, checks the live Secret. If the key is absent, emits a diagnostic naming the workload + Secret + missing key. Appends a **case/format variant hint** when a normalized form of the key exists (e.g. `github-token` referenced but Secret has `GITHUB_TOKEN`). |
| **Why no auto-fix** | Same as SecretKeyMissing — fix is to either edit the workload manifest or the ESO template. Both are git-side actions. |
| **Output (verbatim)** | `🔎 Secret repomind/repomind-secrets exists but is missing key github-token (referenced by Deployment/repomind in ns repomind). Pod will hit CreateContainerConfigError on next restart. Existing keys: [GITHUB_TOKEN, REDIS_URL]. Key 'GITHUB_TOKEN' is a case/format variant — possible naming mismatch.` |
| **Source** | [`internal/diagnose/proactive_secret_key_check.go`](../internal/diagnose/proactive_secret_key_check.go) |

### 4. `UnprovisionedSecret` (added in v0.9.1)

| | |
|---|---|
| **Symptom** | A workload references a Secret via `envFrom` or volume that has NO ExternalSecret provisioning it. The Secret is absent and there is no mechanism by which it will appear — pods will fail on next restart. |
| **What it produces** | Walks every Deployment/StatefulSet/CronJob and checks each Secret reference for either (a) live existence or (b) an ExternalSecret targeting the same Secret name. When neither is true, emits a diagnostic suggesting the canonical Vault path `secret/t6-apps/<namespace>/config`. |
| **Why no auto-fix** | Creating an ExternalSecret requires choosing a Vault path AND populating Vault — both human decisions. |
| **Output (verbatim)** | `🔎 Secret playground/playground-agent-secrets referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path 'secret/t6-apps/playground/config'.` |
| **Source** | [`internal/diagnose/unprovisioned_secret.go`](../internal/diagnose/unprovisioned_secret.go) |

### 5. `VaultPathMissing`

| | |
|---|---|
| **Symptom** | Vault has been edited (path deleted, key renamed, KV mount remounted) but the ESO controller's next refresh hasn't fired yet. The ExternalSecret is still `Ready=True` from its last successful sync — but the next pod restart will fail with `couldn't find key` because ESO will fail to repopulate the K8s Secret. |
| **What it produces** | Queries Vault directly for every path referenced by ExternalSecrets in the cluster. Reads key NAMES only (privacy contract enforced in code — see [`pkg/registry`](../pkg/registry/) for byte-vs-name boundary). Skips non-Vault SecretStores via provider check (v0.3+). Groups Vault outages with ≥3 affected paths into a single summary diagnostic. |
| **Why no auto-fix** | Resolution requires writing to Vault — not a place an automated tool reaches. |
| **Output (verbatim)** | `🔎 VaultPathMissing: 3 paths referenced by ExternalSecrets are not in Vault: t6-apps/playground/config, t6-apps/missing/config, secret/legacy/old (+0 more).` |
| **Source** | [`internal/diagnose/vault_path_missing.go`](../internal/diagnose/vault_path_missing.go) |

### 6. `CertExpiry`

| | |
|---|---|
| **Symptom** | A cert-manager `Certificate` CR has `Ready=False` (renewal stalled), or `status.notAfter` is in the past (expired), or `status.notAfter` is within 14 days (cert-manager should have renewed at 2/3-of-validity — if it hasn't, something's wrong). |
| **What it produces** | One Diagnostic line per problem certificate naming the CR + namespace + reason. For `Ready=False` includes the condition message (carries ACME rate-limit details if applicable). |
| **Why no auto-fix** | The underlying issue is usually upstream (ACME issuer, DNS-01 propagation, account key). `StuckCertificateRequests` fixer addresses the proximate symptom (a stuck CR) but does not auto-respond to expiry itself. |
| **Output (verbatim)** | `🔎 Certificate monitoring/grafana-tls is not Ready: ACME rate-limited (too many certificates issued).` |
| **Source** | [`internal/diagnose/cert_expiry.go`](../internal/diagnose/cert_expiry.go) |

### 7. `ImagePullAuth`

| | |
|---|---|
| **Symptom** | A pod is in `ImagePullBackOff` and the kubelet event message contains an authentication signal — `401`, `unauthorized`, `denied`, `authentication required`, or `pull access denied`. Other pull failures (`manifest unknown`, `image not found`) are intentionally ignored — those are not auth issues. |
| **What it produces** | Filters pods in `ImagePullBackOff`, scans the kubelet events for auth-signal substrings, emits one Diagnostic per affected pod + image. |
| **Why no auto-fix** | Resolution requires creating/fixing an `imagePullSecret` — a credential write that CHA's RBAC explicitly excludes. |
| **Output (verbatim)** | `🔎 Pod monitoring/metrics-exporter container "exporter" cannot pull image 'ghcr.io/myorg/metrics-exporter:v2.1.0': auth failure — 401 unauthorized: authentication required.` |
| **Source** | [`internal/diagnose/image_pull_auth.go`](../internal/diagnose/image_pull_auth.go) |

### 8. `IngressCoverage` (added in v0.9.x)

| | |
|---|---|
| **Symptom** | An `Ingress` exposes a public hostname for which CHA has no endpoint probe target — so TLS faults, missing Kong routes, and DNS failures go undetected. This is a **probe coverage gap**, not a runtime failure. |
| **What it produces** | Walks every `networking.k8s.io/v1` Ingress and computes the set difference: `{ingress hosts} − {probe.DefaultEndpointTargets()}`. Emits one Diagnostic per uncovered host with the exact Go file + function to edit. |
| **Why no auto-fix** | Adding a new probe target is a code change with operator intent — it requires choosing a canonical display name and confirming the host is meant to be monitored. Auto-adding everything would hide intentional ingress-only-not-monitored decisions. |
| **Output (verbatim)** | `🔎 Ingress nextcloud/nextcloud exposes host 'nextcloud.baisoln.com' with no endpoint probe — TLS faults, missing Kong routes, and DNS failures will go undetected. Add '{URL: "https://nextcloud.baisoln.com", Name: "<display name>"}' to probe.DefaultEndpointTargets() in internal/probe/endpoints.go (removal requires explicit operator action — never auto-removed).` |
| **Source** | [`internal/diagnose/ingress_coverage.go`](../internal/diagnose/ingress_coverage.go) |

---

## How the catalog grows

New entries land via PR. Two design rules:

- **A fixer must be reversible.** `kubectl delete pod` reschedules within seconds; `kubectl rollout restart` is part of every operator's vocabulary. We don't add fixers whose mistake recovery requires more than a minute of human action.
- **A fixer must be idempotent.** Running the same fixer against a cluster that already had it run produces no actions and no errors. Repeated execution is a no-op when nothing matches.

Analyzers have one rule: **the diagnostic must name the resolution.** A line that says "X is broken" without telling the operator *what to do about it* is alert-fatigue fuel. Every analyzer in the catalog produces a hint that ends in the kubectl/vault/git command that fixes it.

## Failure-mode tracking for product evolution

The Verified Signature Library — the commercial wedge per the product's pricing model — grows by codifying recurring incident patterns from real customer fleets. Each new failure mode added to the library:

1. Has a unit test against an anonymized fixture from the original incident
2. Has a documented "what an SRE would do manually" runbook
3. Either ships as an auto-fixer (when reversible + idempotent + safe) OR an analyzer (when human judgment is needed)

The goal is a defensible breadth of failure-mode coverage that is hard for a competitor to match without operating real production fleets — the catalog **is** the moat.
