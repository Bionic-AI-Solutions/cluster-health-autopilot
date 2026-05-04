# Adversarial analysis — Cluster Health Autopilot v0.2

This document is the cha team's red-team writeup of v0.2 (the release that
adds the three capabilities the [Vault → Pod Drift solution
brief](./vault_pod_drift_solution_brief.docx.pdf) called out). It is
deliberately written from the *attacker / paranoid SRE* point of view. Each
finding is rated **Severity** (impact if true) × **Likelihood** (how hard
it is to provoke) and resolved as one of:

- **MUST-FIX** — blocking; v0.2 should not ship without this.
- **WILL-FIX** — accepted as a v0.2 limitation, scheduled for v0.3.
- **DOCUMENT** — accepted forever; surfaced in SECURITY.md or README.

The new code surface under review:

- `internal/diagnose/proactive_secret_key_check.go` (Gap 3, PR #14)
- `internal/diagnose/vault_path_missing.go` + `internal/vault/` (Gap 1, PR #16)
- `internal/report/driftreport.go` + the CRD (Gap 2, PR #15)

---

## 1. False-positive surface

### 1.1 VaultPathMissing reports phantom drift on non-Vault SecretStores
**Severity: medium · Likelihood: high · Resolution: DOCUMENT (v0.3 fix)**

The analyzer walks every ExternalSecret cluster-wide and queries the
configured Vault for each path. If the cluster also runs ESOs backed by
AWS Secrets Manager, GCP Secret Manager, etc., those refs will return 404
from Vault and the analyzer will emit `missing-vault-path` diagnostics
that are entirely false.

**Why we accept it for v0.2**: the brief and the design-partner profile
target Vault-exclusive clusters. The OSS user with mixed providers will
see noise but not data corruption — they can disable the probe with
`vaultProbe.enabled=false` until v0.3 ships the SecretStore-provider
filter.

**Mitigation in v0.2**: README + values.yaml comment explicitly call out
"Vault-exclusive cluster only — disable on mixed-provider deployments."

**Fix for v0.3**: list `SecretStore`/`ClusterSecretStore`, build a set of
store names whose `spec.provider.vault` is non-nil, filter
ExternalSecrets by `spec.secretStoreRef.name` ∈ that set.

### 1.2 Vault outage spams 1× diagnostic per path
**Severity: low · Likelihood: high · Resolution: WILL-FIX (v0.3)**

If Vault is unreachable, every reference path emits a `vault-error/<path>`
diagnostic. A cluster with 200 ExternalSecrets each pointing at 1 unique
path will emit 200 near-identical "connection refused" diagnostics. The
DriftReport reconciler will faithfully create 200 CRs, then delete them
all when Vault comes back. Operationally noisy.

**Mitigation in v0.2**: dedupe map exists (subject-keyed) so the *same*
path-error doesn't repeat within a single tick. But each path still fires
its own diagnostic.

**Fix for v0.3**: accumulate transport errors, emit a single
"Vault probe could not reach `<addr>`: N paths affected" diagnostic when
≥3 paths return identical transport-level error.

### 1.3 ProactiveSecretKeyCheck misses `envFrom.secretRef`
**Severity: low · Likelihood: medium · Resolution: WILL-FIX (v0.3)**

The analyzer walks `env[].valueFrom.secretKeyRef` only. Containers using
`envFrom: [secretRef: {name: foo}]` (whole-secret import) are not
inspected. Operators using this pattern get a false negative — drift
will only surface after the pod restarts (existing reactive
SecretKeyMissing analyzer catches it).

**Why this is low-priority**: `envFrom` consumers don't reference
specific keys, so "missing key X" can't be precomputed; the analyzer's
contract degrades to "Secret exists at all" — already covered by the
"missing-secret" branch.

**Fix for v0.3**: add `envFrom.secretRef` walk that emits the
"missing-secret" diagnostic when the referenced Secret is absent.

### 1.4 DriftReport churn on flapping probes
**Severity: medium · Likelihood: medium · Resolution: DOCUMENT**

A probe that flaps (e.g., a Service-probe target that times out 30% of
the time) will emit a finding on bad ticks and not on good ones. The
reconciler creates a DriftReport CR on bad ticks, deletes it on good
ticks → cluster sees create/delete churn at cron cadence (every 30 min
for diagnose; or every cron schedule period).

**Mitigation in v0.2**: `observationCount` on the CR's `.status` lets a
human spot the flap. Status-only patches are cheap.

**Why we accept it**: the alternative (debounce, trailing-edge logic) is
state we'd have to persist across runs. The cron-shaped architecture
makes this a v0.4+ feature when there's a control-plane component.

---

## 2. Privacy / RBAC

### 2.1 Reader role now grants `secrets get,list` cluster-wide
**Severity: HIGH · Likelihood: medium (insider; or compromised SA token)
· Resolution: DOCUMENT + mitigation in chart**

To enable ProactiveSecretKeyCheck, the reader ClusterRole now includes:

```yaml
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
```

The CODE never reads byte values — `for k := range secret.Data` only
iterates names. But the API token has the permission. An attacker who
exfiltrates the cha ServiceAccount token can dump every Secret in every
namespace.

**Mitigation in v0.2**:
- Reader ClusterRole comment block points at the analyzer code that
  enforces the privacy contract.
- The CronJob has `runAsNonRoot=true`, distroless+nonroot image, no
  shell — token theft requires either (a) a worker-node compromise or
  (b) a malicious chart change. The branch-protection rules on `main`
  raise (b)'s cost.

**Fix for v0.3**: split the analyzer into a separate Pod with its own
ServiceAccount + restricted ClusterRole; the diagnose pod itself doesn't
need cluster-wide secret read.

### 2.2 Vault role bound to cha SA grants more than necessary
**Severity: MEDIUM · Likelihood: medium · Resolution: DOCUMENT**

The kubernetes-auth Vault role bound to the cha SA grants read on every
Vault path the SA might query — which is "every path any ExternalSecret
references." A malicious operator who can `kubectl edit externalsecret`
could:

1. Add a remote ref to a sensitive Vault path (e.g. `secret/admin/root`)
2. Wait for cha to query it
3. Read the diagnostic that reveals whether the path exists and what
   keys it contains (key NAMES leak)

**Mitigation in v0.2**:
- The Vault role is operator-supplied, not chart-installed. The
  README + values.yaml comment recommends scoping the role to **only
  the paths used by ExternalSecrets in this cluster**.
- The privacy contract (`vault.Client.ListKeys` returns `[]string`) means
  byte values never leak — but key names DO leak.

**Why we accept it**: a malicious operator who can edit ESOs can already
exfiltrate via the legitimate ESO refresh path; the cha diagnostic is
not a novel exfil channel. Documented in SECURITY.md.

### 2.3 DriftReport CRs are cluster-scoped + readable by anyone
**Severity: low · Likelihood: low · Resolution: DOCUMENT**

DriftReport CRs are cluster-scoped. Any user/SA with `driftreports list`
sees the full set of active issues — useful operationally, but it's
"what's broken across every namespace" intel that may be sensitive in
multi-tenant clusters.

**Mitigation in v0.2**: `kubectl get crd driftreports.cha.bionicaisolutions.com`
is admin-only by default; reader role on the CR is explicit only for the
cha SA itself. Operators who want broader visibility opt in by RBAC.

---

## 3. Performance / scale

### 3.1 ProactiveSecretKeyCheck pulls all Secret bytes over the wire
**Severity: medium · Likelihood: high (large clusters) · Resolution:
WILL-FIX (v0.3)**

`src.List(ctx, GVRSecret, "")` lists every Secret cluster-wide. The
client receives full Secret objects (data field bytes included), then
the analyzer iterates `for k := range data` and discards the values.
On a 10k-Secret cluster, this is potentially 100s of MB of bandwidth
**per cron tick** for data we never use.

**Mitigation in v0.2**: clusters of this scale should set the cron
schedule to a slower cadence (default is daily). Bandwidth cost is
amortized.

**Fix for v0.3**: use the K8s `partial-object-metadata` accept header
(or `metadata.k8s.io/v1`) to fetch metadata only — but that doesn't
give us key names. Alternative: list Secrets with a server-side projection
once that's available. Tracked.

### 3.2 VaultPathMissing issues N HTTP calls per tick
**Severity: low · Likelihood: low · Resolution: ACCEPT**

For 200 ExternalSecrets with ~5 unique Vault paths each, ~1000 GETs/tick.
Vault default rate limit is 8000 req/s — well under. Daily cron =
0.012 req/s amortized. Acceptable.

### 3.3 DriftReport reconcile is O(N) on every tick
**Severity: low · Likelihood: low · Resolution: ACCEPT**

The reconciler lists all driftreports, builds the desired set from
findings + diagnostics + actions, and computes the diff. A cluster with
1000 active issues + 1000 stale CRs = 2000 GETs in the list, ~2000
patches/creates/deletes. Cron cadence makes this irrelevant.

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

The CRD has `helm.sh/resource-policy: keep`. Uninstalling cha leaves the
CRD + every DriftReport CR behind. Operator must:

```
kubectl delete crd driftreports.cha.bionicaisolutions.com
```

manually. Documented in NOTES.txt + README cleanup section.

### 4.3 Two cha pods running concurrently → racy reconcile
**Severity: low · Likelihood: very low · Resolution: ACCEPT**

CronJob `concurrencyPolicy: Forbid` is the default. If an operator
overrides to `Allow`, two diagnose pods can race on the reconciler →
last-writer-wins on each CR. No data corruption (CRs are idempotent
on subject), but observationCount may double-increment.

---

## 5. Auth / token handling

### 5.1 Vault token (when method=token) visible in pod env
**Severity: medium · Likelihood: low (kubectl describe needs RBAC) ·
Resolution: DOCUMENT**

The `token` auth method injects `$VAULT_TOKEN` from a Secret into the
pod env. Anyone with `pods get` on the cha namespace + `pods/exec` (to
read /proc/$pid/environ) can read the token.

**Mitigation in v0.2**: `kubernetes` auth is the default in values.yaml
documentation — the kubernetes-auth path uses the SA JWT (rotates with
the pod, never sits in env).

### 5.2 SA JWT login token has no refresh
**Severity: low · Likelihood: low · Resolution: DOCUMENT**

`buildVaultClient` performs the kubernetes-auth login once at probe
init. If the token TTL is shorter than the cron tick (e.g. TTL=15min,
cron=hourly), subsequent `ListKeys` calls fail with 403 and emit
`vault-error/<path>` diagnostics. The next cron tick recovers (fresh
client, fresh login).

**Mitigation in v0.2**: Vault role TTL guidance in SECURITY.md — set
TTL ≥ cron schedule.

**Fix for v0.3**: if 403 received, attempt one re-login.

---

## 6. Threat model — net assessment

| Threat | v0.1 | v0.2 | Comment |
|---|---|---|---|
| Reactive-only secret-drift detection | ⚠️ | ✅ | Closed by ProactiveSecretKeyCheck (Gap 3) |
| L1 stale-Ready window invisible | ⚠️ | ✅ | Closed by VaultPathMissing (Gap 1) |
| No kubectl-queryable diagnostic surface | ⚠️ | ✅ | Closed by DriftReport CRD (Gap 2) |
| Cluster-wide Secret read | ✅ | ⚠️ NEW | Code-level privacy contract; documented |
| Vault key-name leak via diagnostic | n/a | ⚠️ NEW | Operator scopes Vault role; documented |
| Mixed-provider ESO false-positive noise | n/a | ⚠️ NEW | Disable probe; v0.3 will filter |
| Diagnostic output flood on Vault outage | n/a | ⚠️ NEW | v0.3 will collapse |
| CRD schema instability | n/a | ⚠️ NEW | v1alpha1; documented |

**Net**: v0.2 closes the three brief-defined gaps without introducing
any **MUST-FIX** issue. Five **DOCUMENT** items, three **WILL-FIX**
items scheduled for v0.3.

---

## 7. Pre-tag checklist

- [x] All MUST-FIX items resolved (none).
- [x] All DOCUMENT items captured in SECURITY.md / README / values.yaml comments.
- [ ] `helm template --set vaultProbe.enabled=true …` rendered against a real Vault role; smoke test on the production cluster.
- [ ] `kubectl get driftreports -A` round-tripped on the production cluster (CRD installs, CRs created, deleted on resolve).
- [ ] WhisperLive STT silence-hallucination work is *not* in this release scope (already shipped separately).
