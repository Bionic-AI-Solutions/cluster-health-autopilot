# Cluster Health Autopilot — Demo Script & Instruction Guide

**Audience**: SRE / Platform Engineer evaluating CHA as a design partner or pilot customer.  
**Time**: ~45 minutes end-to-end; zero-trust section alone takes 5 minutes.  
**What you need**: macOS or Linux laptop; `kubectl` context to a real cluster for the in-cluster sections (optional).

> **Note on release naming**: GoReleaser includes the version in asset filenames — e.g. `cluster-health-autopilot_0.8.0_darwin_arm64.tar.gz`. Replace `0.8.0` with the current release version as needed.

---

## Part 1 — Zero-Trust Demo (No Install, No RBAC, 5 Minutes)

> The selling point: the audience sees real cluster diagnostics before trusting you with any
> credentials. This is the "aha" moment.

### 1.1 — Download the binary

Set `VERSION` to the latest release (check [Releases](https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases)):

```bash
VERSION=0.8.0

# macOS arm64 (Apple Silicon)
curl -L "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/download/v${VERSION}/cluster-health-autopilot_${VERSION}_darwin_arm64.tar.gz" \
  | tar xz
chmod +x cha

# macOS amd64 (Intel)
curl -L "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/download/v${VERSION}/cluster-health-autopilot_${VERSION}_darwin_amd64.tar.gz" \
  | tar xz
chmod +x cha

# Linux amd64
curl -L "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/download/v${VERSION}/cluster-health-autopilot_${VERSION}_linux_amd64.tar.gz" \
  | tar xz
chmod +x cha
```

**Windows** (PowerShell):
```powershell
$VERSION = "0.8.0"
Invoke-WebRequest -Uri "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/download/v$VERSION/cluster-health-autopilot_${VERSION}_windows_amd64.zip" -OutFile cha.zip
Expand-Archive cha.zip -DestinationPath .
# Binary is cha.exe — run as: .\cha.exe diagnose --snapshot examples\sample-cluster
```

No Docker, no Go, no runtime dependencies. Single static binary.

### 1.2 — Clone the repo to get the example snapshot

```bash
git clone https://github.com/Bionic-AI-Solutions/cluster-health-autopilot.git
cd cluster-health-autopilot
```

The snapshot at `examples/sample-cluster/` is a real anonymized capture from a production Kubernetes cluster with deliberately injected failures.

### 1.3 — Run diagnostics

```bash
./cha diagnose --snapshot examples/sample-cluster
```

**Expected output** (the exact output your audience will see):

```
• Ceph Storage:          🟢 HEALTHY — 1 cluster(s): rook-ceph@rook-ceph OK (11.5% used)
• Cluster Nodes:         🟢 HEALTHY — All 4 nodes ready
• PostgreSQL:            🟢 HEALTHY — 1 CNPG cluster(s): main@data (3/3 ready, primary=main-1)
• Storage Claims:        🟢 HEALTHY — All 3 PVCs bound
• Critical Services:     🟢 HEALTHY — All 0 critical services operational

Diagnostics (3):
  🔎 Secret `billing/billing-svc-secrets` missing key `STRIPE_API_KEY` (SecretKeyMissing)
  🔎 ExternalSecret `billing/billing-svc-secrets` not Ready: cannot find secret data for key: "stripe_api_key"
  🔎 ExternalSecret `billing/old-payment-gateway` not Ready: vault path not found

Total findings: 0, diagnostics: 3
```

**Talking points**:
- Five probes ran across storage, nodes, database, PVCs, and services — all green.
- Three diagnostics surfaced from analyzers: a secret key mismatch between the Vault path and what the pod expects, and a stale ExternalSecret pointing at a Vault path that no longer exists.
- The pod `billing/billing-svc-d3e4f-new1` is currently stuck in `CreateContainerConfigError` — CHA detected the root cause (STRIPE_API_KEY missing from the secret) before anyone filed a ticket.
- **Nothing was connected to your cluster**. Zero RBAC. Zero trust required.

### 1.4 — Switch to JSON output (for pipeline demos)

```bash
./cha diagnose --snapshot examples/sample-cluster --format json | jq .
```

The structured output is designed for fleet-console pipelines: each diagnostic carries `kind`, `name`, `namespace`, `message`, and `analyzer`.

---

## Part 2 — Failure Showcase (Sample Fixture Walk-Through)

### Failure 1: SecretKeyMissing

**What happened in the fixture**:
- Vault holds the ExternalSecret path with a key named `stripe_api_key` (lowercase).
- The pod's `envFrom` references `billing-svc-secrets` and expects the key `STRIPE_API_KEY` (uppercase).
- Kubernetes copied the lowercase key into the Secret; the uppercase reference is missing.
- Pod state: `CreateContainerConfigError`.

**CHA detection**:
```
🔎 Secret `billing/billing-svc-secrets` missing key `STRIPE_API_KEY` (SecretKeyMissing)
```

**What to show the audience**:
```bash
# See the pod stuck
cat examples/sample-cluster/core-pods.json | jq '.items[] | select(.status.containerStatuses[0].state.waiting.reason == "CreateContainerConfigError") | {name: .metadata.name, ns: .metadata.namespace, reason: .status.containerStatuses[0].state.waiting.reason}'
```

**Root cause chain**: `stripe_api_key` (Vault key) → `STRIPE_API_KEY` (pod env ref) → name mismatch → pod cannot start.

---

### Failure 2: FailingExternalSecrets

**What happened in the fixture**:
- `billing/billing-svc-secrets`: ESO fetched the secret but the key name in the Vault response didn't match the `remoteRef.property` in the ExternalSecret spec.
- `billing/old-payment-gateway`: ESO tried to sync but the Vault path `secret/t6-apps/billing/old-payment` no longer exists (someone deleted it during a Vault cleanup).

**CHA detection**:
```
🔎 ExternalSecret `billing/billing-svc-secrets` not Ready: cannot find secret data for key: "stripe_api_key"
🔎 ExternalSecret `billing/old-payment-gateway` not Ready: vault path not found
```

**What to show the audience**:
```bash
cat examples/sample-cluster/external-secrets.io-externalsecrets.json | jq '.items[] | {name: .metadata.name, ns: .metadata.namespace, ready: (.status.conditions[] | select(.type == "Ready") | .status), message: (.status.conditions[] | select(.type == "Ready") | .message)}'
```

---

### Failure 3: ImagePullAuth (analyzer, inject to fixture or live)

**What it detects**: Registry authentication failures — 401 Unauthorized, credential errors, `pull access denied`.  
**What it ignores**: Legitimate "image not found" errors (`manifest unknown`, `not found`).

**Simulated scenario** (explain, or inject into a live cluster):
```yaml
# Create a pod that pulls from a private registry with bad credentials
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: bad-creds-demo
  namespace: default
spec:
  containers:
  - name: app
    image: ghcr.io/myorg/private-app:latest
    # No imagePullSecrets — will fail with 401
EOF
```

After a minute, CHA will surface:
```
🔎 Pod default/bad-creds-demo container app image ghcr.io/myorg/private-app:latest — pull auth failure: unauthorized: authentication required
```

**Note**: CHA reads the Kubernetes Events for the pod and filters on auth-signal keywords (`unauthorized`, `401`, `denied`, `pull access denied`, `no basic auth credentials`). Non-auth pull failures are intentionally excluded to avoid noise.

---

### Failure 4: CertExpiry (analyzer)

**What it detects**: cert-manager `Certificate` resources that are:
1. Not Ready (any reason — cert-manager hasn't issued yet, or issuance failed)
2. Already expired (`notAfter` in the past)
3. Expiring within 14 days

**Simulated scenario** (explain, or on a live cluster with cert-manager):
```bash
# Check any near-expiry certs
kubectl get certificates -A -o json | jq '.items[] | {name: .metadata.name, ns: .metadata.namespace, ready: (.status.conditions[] | select(.type == "Ready") | .status), notAfter: .status.notAfter}'
```

CHA output for an expiring cert:
```
🔎 Certificate monitoring/grafana-tls expiring in 8d (CertExpiry)
```

CHA output for a not-Ready cert:
```
🔎 Certificate staging/api-tls not Ready: BackoffLimitExceeded (CertExpiry)
```

---

## Part 3 — In-Cluster Install (Live Cluster Demo)

### Prerequisites

```bash
# Verify Helm is installed
helm version

# Verify kubectl context
kubectl config current-context

# Add the chart repo (or use the local checkout)
helm repo add cha https://bionic-ai-solutions.github.io/cluster-health-autopilot
helm repo update
```

### 3.1 — Minimal install (diagnostics only, daily 09:00 UTC)

```bash
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --create-namespace
```

This deploys:
- One `diagnose` CronJob (runs daily at 09:00 UTC)
- A `ServiceAccount` + `ClusterRole` (read-only: pods, nodes, pvcs, secrets key-names, externalsecrets, cnpg, ceph, certs)
- DriftReport CRD writing enabled

Verify the install:
```bash
kubectl get all -n cluster-health-autopilot
kubectl get clusterrole | grep cha
kubectl get driftreports -A   # empty until first run
```

### 3.2 — Enable Slack reporting

```bash
# First, create a Secret with your webhook URL
kubectl create secret generic cha-slack-webhook \
  --from-literal=WEBHOOK_URL="https://hooks.slack.com/services/..." \
  --namespace cluster-health-autopilot

# Upgrade with Slack enabled
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set slack.enabled=true \
  --set slack.webhookSecretName=cha-slack-webhook
```

### 3.3 — Trigger a manual run (don't wait for the cron)

```bash
# The cronjob name includes the release name prefix
kubectl create job --from=cronjob/cha-cluster-health-autopilot-diagnose cha-diagnose-manual \
  -n cluster-health-autopilot
kubectl logs -f job/cha-diagnose-manual -n cluster-health-autopilot
```

You will see the same probe + diagnostic output as the snapshot mode, but against the live cluster.

---

## Part 4 — Auto-Remediation Showcase

> Always demo remediation in dry-run first. The whitelist is narrow and intentional.

### 4.1 — Enable the remediate CronJob (dry-run first)

```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set remediation.enabled=true \
  --set remediation.dryRun=true
```

Trigger a manual remediation run:
```bash
kubectl create job --from=cronjob/cha-cluster-health-autopilot-remediate cha-remediate-dryrun \
  -n cluster-health-autopilot
kubectl logs -f job/cha-remediate-dryrun -n cluster-health-autopilot
```

The output will show what *would* be fixed, without touching the cluster:
```
[DRY-RUN] Would delete pod staging/old-deploy-abc123 (StaleErrorPods: pod is Error state, owned by completed Job)
[DRY-RUN] Would delete Job billing/billing-sync-1704067200 (StuckJobsWithBadSecretRef: Job frozen, CronJob has newer run pending)
```

### 4.2 — The three whitelisted fixers

| Fixer | What it fixes | Safety constraint |
|---|---|---|
| **StaleErrorPods** | Pods in `Error`/`OOMKilled` state that are owned by a completed `Job` | Only deletes if the owning Job is already complete — never touches live Job pods |
| **StuckJobsWithBadSecretRef** | A `Job` frozen due to `CreateContainerConfigError` on a bad Secret ref, when a newer CronJob run is already pending | Only deletes if: (1) Job is CronJob-owned, (2) Job is frozen (no active pods, no succeeded pods), (3) a newer run exists |
| **StuckRSPods** | Pods owned by an old `ReplicaSet` that the `Deployment` has already moved past | Only restarts if the RS's revision is behind the current Deployment revision |

**Safety properties** (explain to the audience):
1. All three fixers are **snapshot-mode-refused at compile time** via Go's type system — they cannot be called in `--snapshot` mode, only `--live`.
2. All three are **whitelisted** — there is no "auto-fix everything" mode.
3. The fix decision is re-evaluated fresh each run from the live cluster state — no persistent decisions.

### 4.3 — Staged live remediation demo

**Inject the StaleErrorPods scenario**:
```bash
# Create a Job that will produce an Error pod
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: crash-demo
  namespace: default
spec:
  template:
    spec:
      containers:
      - name: crasher
        image: busybox
        command: ["sh", "-c", "exit 1"]
      restartPolicy: Never
  backoffLimit: 0
EOF
```

Wait for it to fail (30–60 seconds):
```bash
kubectl get pods -n default -l job-name=crash-demo
# NAME                    READY   STATUS   RESTARTS   AGE
# crash-demo-abc12        0/1     Error    0          45s
```

Now run `cha remediate --dry-run` to confirm detection:
```bash
kubectl create job --from=cronjob/cha-cluster-health-autopilot-remediate cha-remediate-check \
  -n cluster-health-autopilot
kubectl logs -f job/cha-remediate-check -n cluster-health-autopilot
# [DRY-RUN] Would delete pod default/crash-demo-abc12 (StaleErrorPods)
```

Switch to live mode and confirm:
```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set remediation.dryRun=false

kubectl create job --from=cronjob/cha-cluster-health-autopilot-remediate cha-remediate-live \
  -n cluster-health-autopilot
kubectl logs -f job/cha-remediate-live -n cluster-health-autopilot
# Deleted pod default/crash-demo-abc12 (StaleErrorPods)

# Verify pod is gone
kubectl get pods -n default -l job-name=crash-demo
# No resources found.
```

---

## Part 5 — Snapshot Capture (Your Own Cluster, Zero-Trust)

Use this when a prospect wants to see CHA run against their cluster without giving you access.

### 5.1 — They capture, you analyze

Send the prospect this one-liner (they run it; you never see their kubeconfig):

```bash
# They run this on their workstation with kubectl configured
./cha snapshot capture --tar /tmp/my-cluster-$(date +%Y%m%d).tgz
```

They send you the `.tgz`. You analyze it:

```bash
./cha diagnose --snapshot /tmp/my-cluster-20260504.tgz
```

**What's in the snapshot**: pods, nodes, PVCs, events, deployments, replicasets, jobs, cronjobs, externalsecrets, cnpg clusters, ceph clusters, cert-manager certificates.

**What's NOT in the snapshot**: Secret values (never captured to disk — the tool deliberately excludes them). See `internal/snapshot/capture.go` for the explicit exclusion comment.

### 5.2 — DriftReport CRDs (kubectl-queryable diagnostics)

After the in-cluster CronJob runs:
```bash
kubectl get driftreports -A
# NAMESPACE   NAME                              AGE
# billing     secretkeymissing-billing-svc...   2m
# billing     failingexternalsecret-billing...  2m

kubectl describe driftreport secretkeymissing-billing-svc-secrets -n billing
```

DriftReports are Kubernetes objects — they integrate with any existing alerting that watches for CRD events (Prometheus, Datadog operator, Grafana k8sevents).

---

## Part 6 — Nightly Run Pipeline (WS-C Evidence)

> This section is for demos after Gate G3 (week 8 onward).

```bash
# View the accumulating run history
ls -la runs/
cat runs/SUMMARY.md
```

The `SUMMARY.md` is auto-generated nightly by the GitHub Actions workflow in Mode A (self-hosted in-cluster runner):
```
## Run Summary (last 30 days)

| Date       | Resources Scanned | Findings | Diagnostics | Auto-Fixed |
|------------|-------------------|----------|-------------|------------|
| 2026-05-04 | 487               | 0        | 3           | 1          |
| 2026-05-03 | 485               | 0        | 3           | 0          |
| ...        | ...               | ...      | ...         | ...        |

### Analyzer Breakdown (30-day totals)
- SecretKeyMissing:       12 occurrences, 12 resolved
- FailingExternalSecrets: 8 occurrences, 7 resolved, 1 ongoing
- ImagePullAuth:          3 occurrences, 3 resolved
- CertExpiry:             1 occurrence, 1 resolved (cert renewed by cert-manager)
```

**Talking point**: "This is 30 days of real cluster data, anonymized and public. Every incident class that CHA caught is catalogued — including the one we almost missed when someone rotated a Vault key and forgot to update the ExternalSecret property name."

---

## Part 7 — Design-Partner Pitch Close

After the demo, hand the prospect three things:

1. **Their own snapshot analyzed** — run `cha diagnose --snapshot` against the `.tgz` they captured. Show them their cluster's actual state.

2. **`helm install --dry-run` against their cluster** — proves the chart is non-invasive, shows exactly what RBAC it requests before they approve it.

3. **The `runs/SUMMARY.md` link** — live evidence that this runs daily in production.

The ask: "Let us deploy the Helm chart to one non-prod namespace, let the CronJob run for two weeks, and compare the results to what your team found manually in the same period."

---

## Appendix A — Troubleshooting Common Demo Issues

| Symptom | Fix |
|---|---|
| `cha: permission denied` | `chmod +x cha` |
| `cha diagnose --snapshot` shows no diagnostics on sample-cluster | Verify you're using the repo's `examples/sample-cluster/` directory, not a custom snapshot |
| Helm install fails: `no matches for kind "ExternalSecret"` | ESO not installed; skip the runner ExternalSecret section or install ESO first |
| Runner pod stays `Pending` | `kubectl describe pod -n cha -l app=cha-runner` — likely imagePullBackOff on `myoung34/github-runner:ubuntu-jammy` |
| DriftReports not appearing | Check `kubectl logs -n cha job/<latest-diagnose-job>`; DriftReport CRD may need manual install: `kubectl apply -f charts/cluster-health-autopilot/crds/` |
| `cha remediate --live` refuses in snapshot mode | Expected — fixers are type-system-gated. Must use `--live` flag with valid kubeconfig |

## Appendix B — Full Analyzer + Probe Catalog (v0.8.0)

**Probes** (read cluster state, report findings):
| Probe | What it checks |
|---|---|
| Ceph | `CephCluster` CRD `.status.ceph.health` |
| Nodes | NotReady, MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable |
| CNPG / Spilo | CloudNativePG `Cluster` CRD, falls back to Spilo pods if CNPG absent |
| PVCs | Pending PVCs, Lost PVCs |
| Services | Pods in CrashLoopBackOff, OOMKilled, Error with no restart budget |

**Analyzers** (cross-resource correlation, emit diagnostics):
| Analyzer | What it detects |
|---|---|
| SecretKeyMissing | Pod `envFrom`/`env.valueFrom.secretKeyRef` references a key absent from the Secret object |
| FailingExternalSecrets | ExternalSecret with `Ready: False` condition |
| ProactiveSecretKeyCheck | ESO-managed Secret where a key referenced by a pod is present in the Secret but the Vault value returns empty |
| ImagePullAuth | ImagePullBackOff with auth-signal event messages (401, unauthorized, denied) |
| CertExpiry | cert-manager Certificate: not-Ready, expired, or expiring within 14 days |

**Fixers** (mutation, whitelist-only, refused in snapshot mode):
| Fixer | What it does |
|---|---|
| StaleErrorPods | Deletes Error-state pods whose owning Job is complete |
| StuckJobsWithBadSecretRef | Deletes frozen CronJob-owned Jobs so the next run can start |
| StuckRSPods | Rollout-restarts Deployments with pods stuck on old ReplicaSets |
