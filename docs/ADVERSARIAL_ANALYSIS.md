# Adversarial Analysis — Cluster Health Autopilot

This document is the cha team's red-team writeup of the **current shipping
release (v0.9.5)**. It is deliberately written from the *attacker / paranoid
SRE* point of view. Each finding is rated **Severity** (impact if true) ×
**Likelihood** (how hard it is to provoke) and resolved as one of:

- **MUST-FIX** — blocking; the release should not ship without this.
- **WILL-FIX** — accepted as a limitation, scheduled for a future release.
- **DOCUMENT** — accepted forever; surfaced in SECURITY.md or README.

**Scope reviewed:**

- `internal/diagnose/` — 8 analyzers (secret_key_missing, failing_externalsecrets,
  proactive_secret_key_check, unprovisioned_secret, vault_path_missing,
  cert_expiry, image_pull_auth, ingress_coverage)
- `internal/probe/` — 6 probes (Ceph, Nodes, Postgres, PVCs, Services, Endpoints)
- `internal/fix/` — 4 fixers (stale_error_pods, stuck_jobs, stuck_rs_pods,
  stuck_cert_requests)
- `internal/watcher/` — long-running event-driven engine
- `internal/report/` — `routing.go` (three-channel Slack), `daily.go`
  (DriftReport-history digest), `alertmanager.go` (direct AM API hub)
- `charts/cluster-health-autopilot/` — Helm chart shape, RBAC, secret wiring
- Self-hosted GitHub Actions runner Deployment (WS-C publish pipeline)

---

## 1. False-positive surface

### 1.1 VaultPathMissing on non-Vault SecretStores
**Severity: medium · Likelihood: high · Resolution: ✅ FIXED in v0.3**

The analyzer used to query Vault for every ESO regardless of store provider,
emitting `missing-vault-path` for AWS/GCP-backed ESOs. v0.3 resolves each
ESO's `spec.secretStoreRef` to a SecretStore/ClusterSecretStore and only
queries Vault for ESOs whose backing store has `spec.provider.vault` set.

### 1.2 Vault-outage diagnostic flood
**Severity: low · Likelihood: high · Resolution: ✅ FIXED in v0.3**

Transport errors are now accumulated and grouped by error string. When a
group has ≥3 paths, a summary diagnostic fires with up to 3 sample paths
and a "+N more" suffix. Below the threshold, per-path diagnostics still
fire so isolated misconfigurations stay visible.

### 1.3 ProactiveSecretKeyCheck missed envFrom
**Severity: low · Likelihood: medium · Resolution: ✅ FIXED in v0.3**

The analyzer now walks `container.envFrom[].secretRef` too. Whole-secret
imports referencing a non-existent Secret emit the same `missing-secret`
diagnostic with a message distinguishing "envFrom whole-secret import"
from "env key". `optional: true` is honored.

### 1.4 DriftReport churn on flapping probes
**Severity: medium · Likelihood: medium · Resolution: DOCUMENT**

A probe that flaps (Service-probe target that times out 30% of the time)
will emit a finding on bad ticks and not good ones. The reconciler
creates a CR on bad ticks, deletes on good — cluster sees create/delete
churn at watcher resync cadence (default 10 min).

**Mitigation**: `observationCount` on the CR's `.status` lets a human
spot the flap. `lastObserved` timestamps differentiate stable issues
from flapping ones. Status-only patches are cheap.

### 1.5 IngressCoverage emits findings for legitimately uncovered hosts
**Severity: low · Likelihood: high · Resolution: DOCUMENT (NEW in v0.9.x)**

`IngressCoverage` walks every `networking.k8s.io/v1` Ingress and flags
each `spec.rules[].host` that is NOT in `probe.DefaultEndpointTargets()`.
On a cluster with many internal-only hosts (admin UIs, dev tools), this
fires per-host until an operator either (a) adds the host to the
endpoint list or (b) adds the ingress to an explicit ignore list.

**Mitigation in v0.9.5**: the diagnostic explicitly tells operators
which Go file to edit (`internal/probe/endpoints.go`) and notes that
removal requires explicit operator action — never auto-removed. There
is intentionally no chart-level ignore list (would mask probe gaps).

**Why we accept it**: every uncovered host is a real TLS/DNS/Kong-route
blind spot. The pattern is "you actively decided not to probe X" and
that decision is best made in code review, not Helm values.

---

## 2. Privacy / RBAC

### 2.1 Reader role grants `secrets get,list,watch` cluster-wide
**Severity: HIGH · Likelihood: medium (insider; or compromised SA token)
· Resolution: DOCUMENT + mitigation in chart**

To enable `ProactiveSecretKeyCheck` and `UnprovisionedSecret`, the reader
ClusterRole includes:

```yaml
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
```

The `watch` verb was added in v0.9.0 for the long-running watcher. The
CODE never reads byte values — `for k := range secret.Data` only
iterates names. But the API token has the permission.

**Mitigation**:
- Reader ClusterRole comment block points at the analyzer code that
  enforces the privacy contract.
- Watcher Deployment uses `runAsNonRoot=true`, distroless+nonroot image,
  no shell — token theft requires either (a) a worker-node compromise
  or (b) a malicious chart change.
- Branch-protection rules on `main` raise (b)'s cost.

**Future**: split the secret-name iteration into a separate Pod with its
own ServiceAccount + restricted ClusterRole.

### 2.2 Vault role scope is operator's responsibility
**Severity: MEDIUM · Likelihood: medium · Resolution: DOCUMENT**

The kubernetes-auth Vault role bound to the cha SA grants read on every
Vault path the SA queries. A malicious operator who can `kubectl edit
externalsecret` can:

1. Add a remote ref to a sensitive Vault path
2. Wait for cha to query it
3. Read the diagnostic that reveals key NAMES at the path

**Mitigation**:
- The Vault role is operator-supplied, not chart-installed.
- SETUP_GUIDE.md §7 recommends scoping the role to **only paths
  referenced by ExternalSecrets in this cluster**.
- Privacy contract (`vault.Client.ListKeys` returns `[]string` of names)
  means byte values never leak.

**Why we accept it**: a malicious operator who can edit ESOs can
exfiltrate via the legitimate ESO refresh path anyway; the cha
diagnostic is not a novel exfil channel.

### 2.3 DriftReport CRs are cluster-scoped + readable by anyone
**Severity: low · Likelihood: low · Resolution: DOCUMENT**

DriftReport CRs are cluster-scoped. Any user/SA with `driftreports list`
sees the full set of active issues — useful operationally, but "what's
broken across every namespace" intel that may be sensitive in
multi-tenant clusters.

**Mitigation**: CRD access is admin-only by default; reader role on the
CR is explicit only for the cha SA. Operators who want broader
visibility opt in by RBAC.

### 2.4 Alertmanager API has no auth in default install
**Severity: medium · Likelihood: medium · Resolution: DOCUMENT (NEW in v0.9.5)**

CHA posts to `http://alertmanager.<ns>.svc.cluster.local:9093/api/v2/alerts`
on every watcher cycle. The Alertmanager API in a default
kube-prometheus-stack install accepts un-authenticated writes from
anything that can reach the Service ClusterIP. An attacker with a pod
in any namespace can:

- Inject fake `cha_issue` alerts to mask real ones
- Inject `cha_fixer_acted` alerts to make a real attack look like
  routine self-healing
- Flood Alertmanager to exhaust dispatch budget

**Mitigation**:
- This is an Alertmanager surface, not a CHA surface — CHA's payload
  has no special trust.
- Production Alertmanager deployments should front the API with
  NetworkPolicy (or an authenticating proxy) to allow only CHA's
  ServiceAccount-scoped pod to write.
- CHA never reads from Alertmanager — only writes. There is no
  feedback loop an attacker can exploit through CHA.

**Why we accept it**: Alertmanager's auth model is the cluster
operator's choice. Documenting the NetworkPolicy recommendation in
SETUP_GUIDE.md §5.

### 2.5 Slack webhook URLs in three separate Secrets
**Severity: low · Likelihood: low · Resolution: DOCUMENT (NEW in v0.9.4)**

The three-channel routing requires three Kubernetes Secrets:
`cha-slack-ceph-alerts`, `cha-slack-ceph-critical`, `cha-slack-healthinfo`.
Each carries a Slack incoming-webhook URL. Anyone with `secrets get`
on the `cluster-health-autopilot` namespace can read all three.

**Mitigation**:
- Default install does NOT create these Secrets — operator-supplied,
  scoped to the install namespace.
- ExternalSecrets Operator + Vault is the recommended production
  pattern (SETUP_GUIDE.md §6 Option B).
- Slack webhook URLs are post-only — leakage allows spam to those
  channels, not read access to channel history.

---

## 3. Performance / scale

### 3.1 ProactiveSecretKeyCheck pulls all Secret bytes over the wire
**Severity: medium · Likelihood: high (large clusters) · Resolution: WILL-FIX**

`src.List(ctx, GVRSecret, "")` lists every Secret cluster-wide with full
data field bytes (which the analyzer immediately discards). On a
10k-Secret cluster this is potentially 100s of MB per watcher cycle.

**Mitigation in v0.9.5**:
- Watcher `resyncPeriod` defaults to 10 min — at 10-minute cadence,
  bandwidth is amortized.
- Operators on large clusters can disable the watcher and rely on the
  daily CronJob (`resyncPeriod` effectively becomes 24 h).

**Future**: switch to `partial-object-metadata` accept header once
the K8s API offers a key-names-only projection. Tracked.

### 3.2 Watcher Deployment increases write load on the apiserver
**Severity: low · Likelihood: medium · Resolution: ACCEPT (NEW in v0.9.0)**

Each watcher cycle issues N patches/creates/deletes against DriftReport
CRs (one per active diagnostic). On a cluster with 100 active issues at
the 10-min resync cadence, that's 600 etcd writes/hour.

**Why we accept it**: etcd writes to a single CRD are well within
default kube-apiserver throughput budgets. CRD update events are not
fanned out to LIST-WATCH clients except those explicitly watching
`driftreports.cha.bionicaisolutions.com`.

### 3.3 Alertmanager `/api/v2/alerts` POST every cycle
**Severity: low · Likelihood: low · Resolution: ACCEPT (NEW in v0.9.5)**

Each watcher cycle POSTs the full active issue set to Alertmanager.
At 100 issues × 10-min resync, that's ~6 POSTs/hour with ~25 KB
payloads. Alertmanager's dedup keeps memory bounded.

### 3.4 DriftReport reconcile is O(N) per tick
**Severity: low · Likelihood: low · Resolution: ACCEPT**

The reconciler lists all driftreports, builds the desired set from
findings + diagnostics + actions, computes the diff. A cluster with
1000 active issues = 2000 GETs in the list, ~2000 patches/creates/deletes
per tick. Watcher cadence makes this irrelevant for normal scale.

---

## 4. Blast radius / supply chain

### 4.1 CRD is v1alpha1 — schema is explicitly unstable
**Severity: medium · Likelihood: certain · Resolution: DOCUMENT**

The DriftReport CRD is `v1alpha1`. We will change the schema before
moving to `v1beta1`. Consumer scripts that `kubectl get driftreports
-o jsonpath=...` will break.

**Mitigation**: README + values.yaml call this out. The
`additionalPrinterColumns` are stable surface (column names won't
change without a major bump).

### 4.2 `helm uninstall` does not remove DriftReport CRs
**Severity: low · Likelihood: high · Resolution: DOCUMENT**

The CRD has `helm.sh/resource-policy: keep`. Uninstalling cha leaves
the CRD + every DriftReport CR behind. Operator must:

```
kubectl delete crd driftreports.cha.bionicaisolutions.com
```

manually. Documented in NOTES.txt + SETUP_GUIDE.md §9.

### 4.3 Watcher continuous remediation widens blast radius vs cron
**Severity: medium · Likelihood: medium · Resolution: ACCEPT + DOCUMENT (NEW in v0.9.0)**

In `cha watch --live --remedy`, fixers run after every diagnose cycle
(default 10 min). Compared to the daily CronJob (24-hour blast budget),
a bug in a fixer can mutate the cluster 144× more often before someone
notices.

**Mitigation**:
- Fixers are the same Go code in both paths — increase in *frequency*,
  not *capability*.
- `watcher.remedy.dryRun=true` is the recommended first-week posture.
- All four fixers refuse mutation in `--snapshot` mode at the type-
  system level (Mutator interface).
- The remediator ClusterRole is unchanged: `pods/delete`,
  `jobs/delete`, `deployments/patch`, `certificaterequests/delete`,
  `orders/delete`. No new verbs added in v0.9.0+.

### 4.4 Self-hosted GitHub Actions runner runs as root
**Severity: medium · Likelihood: low · Resolution: DOCUMENT (NEW in v0.9.x)**

`runner.enabled=true` deploys `myoung34/github-runner` as root. The
container holds:

- A GitHub PAT with `repo` scope (via ExternalSecret from Vault)
- The cha ServiceAccount token (via projected volume) — same RBAC
  as the watcher

A workflow run that executes attacker-controlled code in this runner
inherits both credentials.

**Mitigation**:
- `runner.enabled` is **off by default**.
- The runner is opt-in for the WS-C publish-runs pipeline only.
- The PAT is `repo` scope (not org-admin); blast radius is limited
  to the cluster-health-autopilot repo.
- GitHub Actions branch-protection rules on `main` prevent merging
  workflow changes without review.

**Why root**: `myoung34/github-runner` upstream requires root. Future:
investigate `actions/runner-controller` (ARC) which supports rootless.

### 4.5 Endpoint probe — egress, redirects, TLS
**Severity: low · Likelihood: low · Resolution: ✅ DOCUMENTED IN CODE (NEW in v0.9.x)**

The endpoint probe issues HTTP GET against each `DefaultEndpointTargets`
URL. Probe failure modes considered:

- **Redirects**: follows up to 10 by default (Go `http.Client` behavior).
  Accepted — Kong commonly issues 308 → HTTPS.
- **TLS verification**: `InsecureSkipVerify=false`. A self-signed cert
  produces a "TLS handshake error" probe failure — desirable.
- **Timeout**: 10 seconds per target. Probe failure surfaces as
  diagnostic; does not block other probes.
- **Outbound from cluster**: targets are public hostnames; probe
  traffic exits via the cluster's default egress (NAT/SNAT). Network
  Policy operators can rate-limit if needed.

### 4.6 Two cha pods concurrent → racy reconcile
**Severity: low · Likelihood: very low · Resolution: ACCEPT**

CronJob `concurrencyPolicy: Forbid` is default. Watcher Deployment uses
`strategy: Recreate` with `replicas: 1`. If an operator overrides to
`Allow` / scales replicas >1, two reconcilers can race → last-writer-
wins on each CR. No data corruption (CRs are idempotent on subject),
but `observationCount` may double-increment.

**Future**: leader-election (controller-runtime style) is part of the
Operator migration plan.

---

## 5. Auth / token handling

### 5.1 Vault token (when method=token) visible in pod env
**Severity: medium · Likelihood: low (requires pods/exec) · Resolution: DOCUMENT**

The `token` auth method injects `$VAULT_TOKEN` from a Secret into the
pod env. Anyone with `pods get` + `pods/exec` can read `/proc/$pid/environ`.

**Mitigation**: `kubernetes` auth is the documented default. The SA
JWT rotates with the pod and never sits in env.

### 5.2 SA JWT login token has no refresh
**Severity: low · Likelihood: low · Resolution: DOCUMENT**

`buildVaultClient` performs the kubernetes-auth login once at probe
init. If the token TTL is shorter than the resync cadence, subsequent
`ListKeys` calls fail with 403 and emit `vault-error/<path>` diagnostics.
The next cycle recovers (fresh client, fresh login).

**Mitigation**: SETUP_GUIDE.md §7 — set Vault role TTL ≥ watcher
resync period (default 10 min; recommend ≥ 1 h).

**Future**: if 403 received, attempt one re-login in the same cycle.

### 5.3 GitHub PAT in runner Secret rotates manually
**Severity: low · Likelihood: medium · Resolution: DOCUMENT (NEW in v0.9.x)**

The runner's GH PAT in `cha-runner-token` Secret (via ExternalSecret
from Vault path `secret/t6-apps/cha/config:github_pat`) does not rotate
automatically. If the PAT is revoked, the runner enters a fail-loop
until the operator updates Vault.

**Mitigation**: ESO refresh interval is 1 h — recovery is bounded.
PAT scope is `repo` only, limiting blast radius of compromise.

---

## 6. Threat model — net assessment

| Threat | v0.1 | v0.9.5 | Comment |
|---|---|---|---|
| Reactive-only secret-drift detection | ⚠️ | ✅ | Closed by ProactiveSecretKeyCheck (v0.2) |
| L1 stale-Ready window invisible | ⚠️ | ✅ | Closed by VaultPathMissing (v0.2) |
| No kubectl-queryable diagnostic surface | ⚠️ | ✅ | DriftReport CRD (v0.2) |
| Detection latency (CronJob = minutes) | ⚠️ | ✅ | Closed by Watcher mode (v0.9.0) — seconds |
| Slack noise on stable cluster | ⚠️ | ✅ | Closed by fingerprint dedup + DriftReport seed (v0.9.0) |
| Auto-remediation requires manual trigger | ⚠️ | ✅ | Watcher --remedy runs fixers each cycle (v0.9.0) |
| Single-channel alert routing | ⚠️ | ✅ | Three-channel routing (v0.9.4) |
| Alert dedup/silencing not supported | ⚠️ | ✅ | Alertmanager-as-hub integration (v0.9.5) |
| Ingress hosts have no reachability monitor | ⚠️ | ✅ | IngressCoverage + Endpoints probe (v0.9.x) |
| Stuck cert-manager renewal | ⚠️ | ✅ | StuckCertificateRequests fixer + CertExpiry analyzer |
| Cluster-wide Secret read | n/a | ⚠️ | Code-level privacy contract; documented (§2.1) |
| Vault key-name leak via diagnostic | n/a | ⚠️ | Operator scopes Vault role (§2.2) |
| CRD schema instability | n/a | ⚠️ | v1alpha1 documented (§4.1) |
| Alertmanager API unauthenticated | n/a | ⚠️ NEW | NetworkPolicy recommendation (§2.4) |
| Watcher continuous fix blast radius | n/a | ⚠️ NEW | dryRun-first posture documented (§4.3) |
| GH Actions runner root + PAT | n/a | ⚠️ NEW | Opt-in only; scoped PAT (§4.4) |

**Net**: v0.9.5 closes every functional gap from v0.1 through v0.9.4 with
zero **MUST-FIX** items. Six **DOCUMENT** items (three carried forward
from v0.2, three new in v0.9.x), one **WILL-FIX** (Secret list bandwidth).

The new surface added in v0.5–v0.9.5 (watcher Deployment, Alertmanager
integration, three-channel Slack, self-hosted runner) does not introduce
novel privilege escalation paths — all new code reuses existing RBAC and
the privacy contracts established in v0.2.

---

## 7. Pre-release checklist (per tag)

- [ ] All MUST-FIX items resolved (none in v0.9.5).
- [ ] All DOCUMENT items captured in SECURITY.md / SETUP_GUIDE.md / values.yaml comments.
- [ ] `helm template --set …` rendered against a real production cluster smoke-test.
- [ ] `kubectl get driftreports -A` round-trips on the production cluster.
- [ ] Watcher `cycle complete` log line within `resyncPeriod + debounce` of pod start.
- [ ] Alertmanager `/api/v2/alerts` shows `cha_issue` alerts within 1 cycle.
- [ ] Three-channel Slack — at least #healthinfo receives a daily digest.
- [ ] Image size budget: distroless+static, multi-arch, <20 MB compressed.
  Current v0.9.5: 13 MB.

---

## Historical context

This document supersedes the v0.2 / v0.3 / v0.4 adversarial writeups. The
findings from those earlier reviews that remain valid are folded into the
sections above with their original severity ratings preserved. Findings
that were closed by subsequent code changes (false-positive surface §1.1
through §1.3) are retained as a paper trail rather than deleted, so future
reviewers can see the resolution arc.

Findings introduced in v0.9.0 (watcher mode), v0.9.4 (three-channel Slack)
and v0.9.5 (Alertmanager hub) are explicitly marked **NEW in v0.X.Y** to
distinguish them from carried-forward analysis.
