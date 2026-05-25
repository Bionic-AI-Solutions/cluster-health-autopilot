# Failure-Mode Catalog

The default catalog (v1.5.2) covers **four default auto-fixers + one opt-in**
and **eight OSS analyzers**. All analyzer source code ships under Apache 2.0.
`VaultPathMissing` requires a constructed Vault client to register — the OSS
ships it unwired (you supply the client); the paid CHA Enterprise binary
auto-wires it from your Vault configuration. Each is a separate Go function
with a published unit-test corpus under [`internal/`](../internal/). The
"real example" column is a real incident from the cluster the product was
built on.

Since v1.5, every CRITICAL finding also passes through a **Layer-2 Investigator**
(rule-based in OSS, optional LLM-backed in CHA-com) that attaches a root-cause
summary before the alert reaches Slack / Alertmanager / DriftReport. See §C
at the bottom of this document.

---

## A. Auto-fixers (whitelist; default-OFF; opt-in via `remediation.enabled=true`)

Each fixer is gated by the **Mutator interface** at the type-system level: `snapshot.File` doesn't implement `Mutator`, so fixers physically cannot run against a captured snapshot. Live mode only. The remediator ClusterRole grants five narrow verbs (`pods/delete`, `jobs/delete`, `deployments/patch`, `certificaterequests/delete`, `orders/delete`); no Secret, ConfigMap, or generic CRD writes are possible regardless of how the binary is called. The opt-in `TLSSecretMismatch` fixer (§5) conditionally adds `ingresses/patch` to that role.

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

### 5. `TLSSecretMismatch` (opt-in, added in v1.3)

| | |
|---|---|
| **Symptom** | An `Ingress.spec.tls[].secretName` points at a TLS Secret whose cert is expired or expiring soon, while a healthy cert-manager `Certificate` exists in the same namespace targeting a *different* Secret name with the same host in `dnsNames`. The wires were crossed at install time — cert-manager is renewing into a Secret nobody uses; Kong serves the stale cert from the Secret the Ingress points at. |
| **Root cause class** | Two-Secret naming drift. A hand-crafted Secret (`foo-secret`) gets the Ingress wired to it early; later cert-manager is added with a Certificate targeting a different name (`foo-tls`). The Ingress is never updated. Operator sees "cert-manager works" + "Ingress works" — both true in isolation. Discovery of the issue typically waits until the original cert expires and traffic breaks. |
| **What it does** | Patches `Ingress.spec.tls[N].secretName` from the stale Secret to the cert-manager-managed Secret via JSON patch. Verifies the candidate is `Ready=True` and has the matching host in `dnsNames`. **Skips** protected namespaces. **Skips** GitOps-managed Ingresses (annotations: `argocd.argoproj.io/instance`, `argocd.argoproj.io/tracking-id`, `kustomize.toolkit.fluxcd.io/{name,namespace}`, `meta.helm.sh/release-{name,namespace}`; label `app.kubernetes.io/managed-by` ∈ {helm, argocd, flux, fluxcd}) — patching those would fight the reconcile loop. |
| **Why it's safe** | The patch is narrow (one field, one Ingress), reversible (operator can patch it back), and only fires when there is independent evidence (a healthy Certificate CR) that the destination Secret is the right one. GitOps-managed Ingresses are refused because the right fix for those lives in the source repo. |
| **Why opt-in** | Default off because GitOps adoption varies. Enable with Helm value `fixers.tlsSecretMismatch.enabled=true`, which: flips env var `CHA_FIXER_TLS_SECRET_MISMATCH=true`, registers the fixer in the catalog, and conditionally adds `networking.k8s.io/ingresses [patch]` to the remediator ClusterRole. |
| **Real example** | `pg.bionicaisolutions.com` Ingress was wired to `pg-bionicaisolutions-com-secret` (hand-crafted, expired 2026-01-07). cert-manager Certificate `pg-bionicaisolutions-com` had been dutifully renewing into `pg-bionicaisolutions-com-tls` (the `-tls` suffix) for months. Once the cert expired, traffic broke. With this fixer enabled, the next reconcile would have repointed the Ingress before the cert went stale. |
| **Source** | [`internal/fix/tls_secret_mismatch.go`](../internal/fix/tls_secret_mismatch.go) |

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

### 8. `TLSSecretMismatch` (added in v1.3)

| | |
|---|---|
| **Symptom** | Kong serves an expired (or soon-expiring) cert on a host even though cert-manager has been renewing a fresh cert in the same namespace. Both states look healthy in isolation — the Secret exists with a cert; the Certificate CR is `Ready=True`. The bug is that the Ingress points at the wrong Secret name. |
| **What it produces** | Walks every `networking.k8s.io/v1` Ingress, parses x509 from `Secret.data.tls.crt` for each `tls[].secretName`, and when the served cert is expired or within 14 days of expiry, checks for a Certificate CR in the same namespace whose `spec.dnsNames` covers the host AND whose `spec.secretName` is a different name AND that is `Ready=True`. Emits a Diagnostic with the exact `kubectl patch` command. |
| **Why no auto-fix in the analyzer** | The matching `TLSSecretMismatch` fixer (§A.5) IS the auto-fix path — opt-in, with a GitOps escape hatch. The analyzer always runs; the fixer is gated. |
| **Output (verbatim)** | `🔎 Ingress pg/kong-pgadmin-ingress host pg.bionicaisolutions.com serves expired cert from Secret pg-bionicaisolutions-com-secret while cert-manager is renewing a healthy cert for the same host into Secret pg-bionicaisolutions-com-tls in the same namespace. Wires crossed — Kong is serving the wrong Secret. Remediation: kubectl -n pg patch ingress kong-pgadmin-ingress --type=json -p '[{"op":"replace","path":"/spec/tls/0/secretName","value":"pg-bionicaisolutions-com-tls"}]'` |
| **Source** | [`internal/diagnose/tls_secret_mismatch.go`](../internal/diagnose/tls_secret_mismatch.go) |

> **Note**: an earlier `IngressCoverage` analyzer (v0.9.x) was REMOVED in v1.2.
> It warned about Ingress hosts not present in `probe.DefaultEndpointTargets()`.
> v1.2 added auto-discovery — every Ingress host is now probed automatically
> (with per-Ingress opt-out via `cha.bionicaisolutions.com/probe-disable=true`),
> so the gap that analyzer warned about no longer exists.

---

## C. Layer-2 Investigator (added in v1.5; runs on CRITICAL findings)

When a Finding or Diagnostic reaches `SeverityCritical`, a registered Investigator
runs a read-only deep-dive and attaches a one-line root-cause Summary to the
record before it surfaces. Renderers display the Summary as a 🔬 block in Slack
and Alertmanager; the `DriftReport` CR persists it under `spec.investigation`.

The OSS catalog registers a deterministic **rule-based Investigator**
([`internal/investigator/rules.go`](../internal/investigator/rules.go)) by
default. The paid CHA-com binary may replace it with an LLM-backed
implementation that uses the same closed-enum `Environment` surface:

| Tool | What it sees | Used by which rule |
|---|---|---|
| `DNSLookup` | A / AAAA records, resolve latency | Connection-failure, slow-DNS classification |
| `HTTPProbe` | One follow-up request with optional `InsecureSkipVerify` | Transient-recovery confirmation, status-mismatch reproduction |
| `TLSInspect` | Cert chain, SANs, validity, issuer | TLS-failure root-cause classification |
| `Describe` | Read-only Kubernetes resource snapshot (no Secret values) | ExternalSecret / Secret / Certificate diagnostic enrichment |
| `GetEvents` | Recent events involving a specific object | Backing-context for failing controllers |

**Rule coverage today:**

| Pattern | Tools | Conclusion |
|---|---|---|
| `TLS verification failed` | `TLSInspect` + `DNSLookup` | Classifies expired / SAN-mismatch / fallback-cert |
| `connection failed — no such host` | `DNSLookup` | Surfaces DNS root cause |
| `connection failed — context deadline / EOF` | `DNSLookup` + `HTTPProbe` (strict & insecure) | Likely-transient (if retry succeeds), or backend-down |
| Slow DNS (>1.5s on follow-up) | `DNSLookup` | CoreDNS contention root cause |
| `HTTP <X> (expected <Y>)` | `HTTPProbe` | Confirms cleared / 5xx / auth-walled |
| `ExternalSecret/<ns>/<name>` not Ready | `Describe` + `GetEvents` | Vault path / property hint |
| `missing-secret`, `missing-key` patterns | `Describe` | Confirms absence / case-mismatch |
| `cert-expiry/<ns>/<name>` | `Describe` + `GetEvents` | Surfaces issuer/order state |

Hard contract:
- **Read-only by interface.** `Environment` exposes no mutation methods.
- **No new RBAC.** Reuses the watcher's existing read access.
- **20-second wall-clock cap** across all critical findings in one cycle.
- **Soft-fail per item.** Investigation is additive; the original finding always surfaces.
- **Disable with `CHA_INVESTIGATOR=off`** (env var on the watcher Deployment).

Design rationale in [`docs/design/2026-05-investigator-agent.md`](design/2026-05-investigator-agent.md).

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
