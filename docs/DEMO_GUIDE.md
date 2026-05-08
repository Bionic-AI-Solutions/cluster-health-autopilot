# Cluster Health Autopilot — Demo Script & Instruction Guide

**Audience**: SRE / Platform Engineer evaluating CHA as a design partner or pilot customer.  
**Time**: ~45 minutes end-to-end; zero-trust section alone takes 5 minutes.  
**What you need**: macOS or Linux laptop with `jq` installed; `kubectl` context to a real cluster only for Parts 3 and 4 (optional).

---

## Part 1 — Zero-Trust Demo (No Install, No RBAC, 5 Minutes)

> The selling point: the audience sees real cluster diagnostics before trusting you with any
> credentials. This is the "aha" moment.

### 1.1 — Download the binary

```bash
# macOS arm64 (Apple Silicon)
curl -L https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_darwin_arm64.tar.gz \
  | tar xz
chmod +x cha

# macOS amd64 (Intel)
curl -L https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_darwin_amd64.tar.gz \
  | tar xz
chmod +x cha

# Linux amd64
curl -L https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_linux_amd64.tar.gz \
  | tar xz
chmod +x cha

# Linux arm64
curl -L https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_linux_arm64.tar.gz \
  | tar xz
chmod +x cha
```

**Windows** (PowerShell):
```powershell
Invoke-WebRequest -Uri "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_windows_amd64.zip" -OutFile cha.zip
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

Diagnostics (6):
  🔎 Secret `billing/billing-svc-secrets` missing key `STRIPE_API_KEY` (SecretKeyMissing)
  🔎 ExternalSecret `billing/billing-svc-secrets` not Ready: cannot find secret data for key: "stripe_api_key"
  🔎 ExternalSecret `billing/old-payment-gateway` not Ready: vault path not found
  🔎 Pod `monitoring/metrics-exporter-5c7d8b-abc12` container "exporter" cannot pull image — auth failure: 401 unauthorized
  🔎 Certificate `monitoring/grafana-tls` is not Ready: ACME rate-limited (too many certificates issued)
  🔎 Certificate `production/api-server-tls` EXPIRED at 2025-02-28 00:00 UTC

Total findings: 0, diagnostics: 6
```

**Talking points**:
- Five probes ran across storage, nodes, database, PVCs, and services — all green.
- Six diagnostics from four different analyzers: secret key mismatch, two failing ExternalSecrets, a registry auth failure, a cert-manager ACME rate-limit, and an expired TLS certificate.
- The pod `billing/billing-svc-d3e4f-new1` is stuck in `CreateContainerConfigError` — CHA traced the root cause to a Vault key name mismatch before anyone filed a ticket.
- **Nothing was connected to your cluster**. Zero RBAC. Zero trust required.

### 1.4 — Switch to JSON output (for pipeline demos)

```bash
./cha diagnose --snapshot examples/sample-cluster --format json | jq .
```

The structured output is designed for fleet-console pipelines: each diagnostic carries `kind`, `name`, `namespace`, `message`, and `analyzer`.

---

## Part 2 — Failure Showcase (Sample Fixture Walk-Through)

> All four failures below are visible from the same `examples/sample-cluster/` snapshot — no
> live cluster or `kubectl` needed. The `jq` queries let you "open the hood" and show the
> audience the raw data CHA reasoned over.

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

**Show the raw data**:
```bash
# See the pod stuck
cat examples/sample-cluster/core-pods.json \
  | jq '.items[] | select(.status.containerStatuses[0].state.waiting.reason == "CreateContainerConfigError")
        | {pod: .metadata.name, ns: .metadata.namespace, reason: .status.containerStatuses[0].state.waiting.reason}'
```

**Root cause chain**: `stripe_api_key` (Vault key) → `STRIPE_API_KEY` (pod env ref) → name mismatch → pod cannot start.

---

### Failure 2: FailingExternalSecrets

**What happened in the fixture**:
- `billing/billing-svc-secrets`: ESO fetched the secret but the key name in the Vault response didn't match the `remoteRef.property` in the ExternalSecret spec.
- `billing/old-payment-gateway`: ESO tried to sync but the Vault path `secret/t6-apps/billing/old-payment` no longer exists (deleted during a Vault cleanup).

**CHA detection**:
```
🔎 ExternalSecret `billing/billing-svc-secrets` not Ready: cannot find secret data for key: "stripe_api_key"
🔎 ExternalSecret `billing/old-payment-gateway` not Ready: vault path not found
```

**Show the raw data**:
```bash
cat examples/sample-cluster/external-secrets.io-externalsecrets.json \
  | jq '.items[] | select(.status.conditions[0].status == "False")
        | {name: .metadata.name, ns: .metadata.namespace,
           ready: .status.conditions[0].status,
           message: .status.conditions[0].message}'
```

---

### Failure 3: ImagePullAuth

**What happened in the fixture**:
- The `monitoring/metrics-exporter` pod references a private GHCR image.
- No `imagePullSecret` is configured — the kubelet's pull attempt returns HTTP 401.
- Pod state: `ImagePullBackOff`.

**CHA detection**:
```
🔎 Pod `monitoring/metrics-exporter-5c7d8b-abc12` container "exporter" cannot pull image
   "ghcr.io/myorg/metrics-exporter:v2.1.0": auth failure — 401 unauthorized: authentication required
```

**Show the raw data** (two queries — pod state, then the auth event):
```bash
# Pod stuck in ImagePullBackOff
cat examples/sample-cluster/core-pods.json \
  | jq '.items[] | select(.status.containerStatuses[0].state.waiting.reason == "ImagePullBackOff")
        | {pod: .metadata.name, ns: .metadata.namespace,
           image: .status.containerStatuses[0].image,
           reason: .status.containerStatuses[0].state.waiting.reason}'

# The kubelet event carrying the 401 error
cat examples/sample-cluster/core-events.json \
  | jq '.items[] | select(.reason == "Failed" and (.message | test("unauthorized|401")))
        | {pod: .involvedObject.name, ns: .metadata.namespace, message: .message}'
```

**Why CHA ignores non-auth pull failures**: `manifest unknown`, `image not found`, and other deployment errors are noise — the team already knows their image tags. CHA only surfaces auth-signal keywords (`401`, `unauthorized`, `denied`, `authentication required`, `pull access denied`) so the on-call engineer doesn't have to grep events manually.

---

### Failure 4: CertExpiry

**What happened in the fixture**:
- `monitoring/grafana-tls`: cert-manager hit an ACME rate-limit and cannot renew. The certificate is not Ready.
- `production/api-server-tls`: The certificate expired on 2025-02-28. TLS secrets are still mounted in running pods but will break on the next pod restart.

**CHA detection**:
```
🔎 Certificate `monitoring/grafana-tls` is not Ready: ACME rate-limited (too many certificates issued)
🔎 Certificate `production/api-server-tls` EXPIRED at 2025-02-28 00:00 UTC
```

**Show the raw data**:
```bash
cat examples/sample-cluster/cert-manager.io-certificates.json \
  | jq '.items[] | {name: .metadata.name, ns: .metadata.namespace,
                    ready: (.status.conditions[0] | {status: .status, message: .message}),
                    notAfter: .status.notAfter}'
```

**Three conditions CHA flags** (explain to the audience):
1. `Ready: False` — renewal stalled (ACME error, issuer down, DNS misconfiguration)
2. `notAfter` in the past — certificate already expired
3. `notAfter` within 14 days — cert-manager renewal has likely stalled (sits above the default 2/3-of-validity renewal point)

---

## Part 3 — In-Cluster Install (Live Cluster Demo)

### Prerequisites

```bash
# Verify Helm is installed
helm version

# Verify kubectl context
kubectl config current-context

# Add the chart repo (one-time setup)
helm repo add cha https://bionic-ai-solutions.github.io/cluster-health-autopilot
helm repo update
```

### 3.1 — Install (scheduled diagnostics + optional real-time watcher)

CHA has two complementary operational layers. Deploy one or both depending on what you want to show:

| Layer | Command | Latency | Best for |
|---|---|---|---|
| **CronJob** `diagnose` | daily at 09:00 UTC | minutes | scheduled audit, WS-C JSONL evidence |
| **Watcher** `watch --live` | event-driven, ~10 s debounce | seconds | on-call alerting, live demos |

**Minimal install (scheduled diagnostics only)**:
```bash
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --create-namespace
```

This deploys:
- One `diagnose` CronJob (runs daily at 09:00 UTC)
- A `ServiceAccount` + `ClusterRole` (read-only + `watch` verb: pods, nodes, pvcs, secrets key-names, externalsecrets, cnpg, ceph, certs)
- DriftReport CRD writing enabled

**Recommended for live demos — also enable the watcher**:
```bash
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --create-namespace \
  --set watcher.enabled=true \
  --set slack.enabled=true \
  --set slack.webhookSecretName=cha-slack-webhook
```

The watcher is a long-running `Deployment` that reacts within seconds of a cluster event — no manual job triggers needed (see Part 5). The CronJob and watcher run concurrently and serve different purposes.

Verify the install:
```bash
kubectl get all -n cluster-health-autopilot
kubectl get clusterrole | grep cha
```

### 3.2 — On-demand run (one-shot audit without waiting for the cron)

For an immediate audit snapshot — useful when you want to confirm the current cluster state before or after a change:

```bash
kubectl create job --from=cronjob/cha-cluster-health-autopilot-diagnose cha-diagnose-manual \
  -n cluster-health-autopilot
kubectl logs -f job/cha-diagnose-manual -n cluster-health-autopilot
```

You will see the same probe + diagnostic output as the snapshot mode, but against the live cluster.

> **Continuous alternative**: Once the watcher is deployed (Part 5), you no longer need to trigger manual jobs for real-time detection. The watcher reacts within seconds of each Kubernetes event and keeps DriftReports up to date continuously. Use the manual CronJob trigger for on-demand audits or compliance snapshots.

### 3.3 — Inspect DriftReport CRDs (kubectl-queryable diagnostics)

After the CronJob, manual job, or watcher cycle runs:
```bash
kubectl get driftreports -A
# NAMESPACE   NAME                              AGE
# billing     secretkeymissing-billing-svc...   2m
# billing     failingexternalsecret-billing...  2m

kubectl describe driftreport secretkeymissing-billing-svc-secrets -n billing
```

DriftReports are Kubernetes objects — they integrate with any existing alerting that watches for CRD events (Prometheus, Datadog operator, Grafana k8sevents).

### 3.4 — Enable Slack reporting

Create the Secret. Use `printf` (not `echo`) to avoid embedding a trailing newline in
the URL — a hidden newline causes a `parse` error at post time:

```bash
printf '%s' 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL' \
  | kubectl create secret generic cha-slack-webhook \
      --from-file=WEBHOOK_URL=/dev/stdin \
      -n cluster-health-autopilot
```

Verify the value looks clean (no `$` before the bash prompt means no trailing newline):

```bash
kubectl get secret cha-slack-webhook -n cluster-health-autopilot \
  -o jsonpath='{.data.WEBHOOK_URL}' | base64 -d | cat -A
```

Enable Slack in the Helm release:

```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set slack.enabled=true \
  --set slack.webhookSecretName=cha-slack-webhook
```

---

## Part 4 — Auto-Remediation Showcase

> Always demo remediation in dry-run first. The whitelist is narrow and intentional.

CHA offers two remediation paths — both use the same whitelisted fixers:

| Path | Trigger | Latency | When to use |
|---|---|---|---|
| **Watcher `--remedy`** | event-driven (recommended) | seconds | live demos, on-call automation |
| **Remediate CronJob** | scheduled (opt-in) | daily or on-demand | scheduled batch cleanup |

For live demos, the watcher with `--remedy` is the best story: a failure appears, CHA detects and fixes it within seconds, Slack confirms the resolution — no manual job triggers.

### 4.1 — Enable dry-run remediation (verify what would be fixed)

**Primary: watcher in dry-run mode**

```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.enabled=true \
  --set watcher.remedy.enabled=true \
  --set watcher.remedy.dryRun=true
```

Watch the logs to see dry-run output on the next event:
```bash
kubectl logs -f deployment/cha-cluster-health-autopilot-watcher \
  -n cluster-health-autopilot
# [DRY-RUN] Would delete pod staging/old-deploy-abc123 (StaleErrorPods)
# [DRY-RUN] Would delete Job billing/billing-sync-1704067200 (StuckJobsWithBadSecretRef)
```

**Alternative: one-off CronJob trigger (scheduled or on-demand)**

Enable the remediate CronJob for batch/scheduled cleanup:
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

### 4.2 — The three whitelisted fixers

| Fixer | What it fixes | Safety constraint |
|---|---|---|
| **StaleErrorPods** | Pods in `Error`/`OOMKilled` state that are owned by a completed `Job` | Only deletes if the owning Job is already complete — never touches live Job pods |
| **StuckJobsWithBadSecretRef** | A `Job` frozen due to `CreateContainerConfigError` on a bad Secret ref, when a newer CronJob run is already pending | Only deletes if: (1) Job is CronJob-owned, (2) Job is frozen (no active pods, no succeeded pods), (3) a newer run exists |
| **StuckRSPods** | Pods owned by an old `ReplicaSet` that the `Deployment` has already moved past | Only restarts if the RS's revision is behind the current Deployment revision |
| **StuckCertificateRequests** | `CertificateRequest` CRs with `Ready=False/reason=Failed` or `failureTime` set; ACME `Order` CRs in state `errored` or `invalid` | Only deletes terminally-failed CRs — never touches pending/in-progress issuance; cert-manager recreates the CR immediately and retries |

**Safety properties** (explain to the audience):
1. All fixers are **snapshot-mode-refused at compile time** via Go's type system — they cannot be called in `--snapshot` mode, only `--live`.
2. All fixers are **whitelisted** — there is no "auto-fix everything" mode.
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

**Primary path — watcher detects and fixes automatically**:

The watcher fires within ~10–15 seconds of the pod entering `Error` state. Watch the logs:
```bash
kubectl logs -f deployment/cha-cluster-health-autopilot-watcher \
  -n cluster-health-autopilot
# watcher: event-triggered cycle
# [DRY-RUN] Would delete pod default/crash-demo-abc12 (StaleErrorPods)
```

Switch the watcher to live mode:
```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.remedy.dryRun=false
```

The next event cycle deletes the pod automatically. Slack receives a combined report: what was fixed + the post-fix diagnostic state.

**Alternative path — on-demand CronJob trigger**:

If you prefer to demonstrate the CronJob flow instead:
```bash
# Confirm detection (dry-run)
kubectl create job --from=cronjob/cha-cluster-health-autopilot-remediate cha-remediate-check \
  -n cluster-health-autopilot
kubectl logs -f job/cha-remediate-check -n cluster-health-autopilot
# [DRY-RUN] Would delete pod default/crash-demo-abc12 (StaleErrorPods)

# Switch to live and run
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set remediation.dryRun=false

kubectl create job --from=cronjob/cha-cluster-health-autopilot-remediate cha-remediate-live \
  -n cluster-health-autopilot
kubectl logs -f job/cha-remediate-live -n cluster-health-autopilot
# Deleted pod default/crash-demo-abc12 (StaleErrorPods)
```

Verify the pod is gone:
```bash
kubectl get pods -n default -l job-name=crash-demo
# No resources found.
```

---

## Part 5 — Event-Driven Watcher (Real-Time Alerts)

> **New in v0.9.0.** Instead of waiting for a CronJob tick, the watcher reacts
> within seconds of a Kubernetes event. Great for showing how CHA integrates into
> an on-call workflow.

### 5.1 — Enable the watcher via Helm

```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.enabled=true \
  --set slack.enabled=true \
  --set slack.webhookSecretName=cha-slack-webhook
```

This deploys a long-running `Deployment` (single replica, `Recreate` strategy)
that watches all resource types CHA already analyzes.

Verify it's running:

```bash
kubectl get deployment -n cluster-health-autopilot
# NAME                             READY   UP-TO-DATE   AVAILABLE
# cha-...-watcher                  1/1     1            1

kubectl logs -f deployment/cha-cluster-health-autopilot-watcher \
  -n cluster-health-autopilot
# watcher: pre-populated seen map with N DriftReports
# watcher: initial diagnose cycle
# watcher: driftreports: 0 created, N updated, 0 deleted
```

### 5.2 — Slack deduplication behavior

The Slack channel stays quiet as long as cluster state doesn't change:

| Condition | Slack posts? |
|---|---|
| Diagnostic first appears | ✅ Yes |
| Severity or message changes | ✅ Yes |
| Diagnostic resolves | ✅ Yes (if `--slack-post-on-resolved=true`) |
| Repeat interval expires (default 4 h) | ✅ Yes (reminder) |
| Same diagnostic, same fingerprint | ❌ Silently skipped |

Fingerprint = `SHA-256(subject | severity | message)`. DriftReport CRs are the
durable source of truth; after a pod restart, the seen-map is seeded from
existing DriftReport status so there is no Slack flood on rollout.

### 5.3 — Live demo: inject a failure and watch Slack fire

```bash
# Inject a bad ExternalSecret to trigger a FailingExternalSecrets diagnostic
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: watcher-demo-bad-es
  namespace: default
spec:
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: watcher-demo-bad-es
  data:
    - secretKey: some-key
      remoteRef:
        key: secret/nonexistent-path
        property: nonexistent-property
EOF
```

Within ~10–15 seconds (debounce + one diagnose cycle), Slack receives:

```
*Cluster Health Autopilot — Watch* — 2026-05-07 09:42:01 UTC

*🔔 Active Issues (1):*
• ⚠️ *ExternalSecret/default/watcher-demo-bad-es*
  ExternalSecret `default/watcher-demo-bad-es` not Ready: …
```

Clean up:

```bash
kubectl delete externalsecret watcher-demo-bad-es -n default
# Slack gets a ✅ Resolved message within the next cycle.
```

### 5.4 — With --remedy: immediate fix + post-fix report

Enable remediation on the watcher:

```bash
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.remedy.enabled=true
```

Now on each cycle, after the diagnose pass the watcher:
1. Runs the whitelisted fixers (`StaleErrorPods`, `StuckJobsWithBadSecretRef`, `StuckRSPods`)
2. Re-diagnoses post-fix to capture the accurate cluster state
3. Posts a combined Slack message: what was fixed + remaining active issues

**Talking points for the audience**:
- Detection latency drops from minutes (CronJob) to seconds (watch event + 10 s debounce).
- Slack remains quiet for stable clusters — no alert fatigue.
- Remediation is the same whitelist as `cha remediate --live`; no new risk surface.
- Pod restart does not re-flood Slack — DriftReport CRs serve as the durable state.

---

## Part 6 — Snapshot Capture (Your Own Cluster, Zero-Trust)

Use this when a prospect wants to see CHA run against their cluster without giving you access.

### 6.1 — They capture, you analyze

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

---

## Part 7 — Nightly Run Pipeline (WS-C Evidence)

> This section is for demos after Gate G3 (week 8 onward).
>
> The watcher (Part 5) handles real-time alerting. The nightly CronJob pipeline here is the **compliance evidence layer** — it produces immutable JSONL run logs, a rolling `SUMMARY.md`, and the `cha diagnose --format jsonl` output that WS-C requires. Both layers run concurrently; they serve different audiences.

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

## Part 8 — Design-Partner Pitch Close

After the demo, hand the prospect three things:

1. **Their own snapshot analyzed** — run `cha diagnose --snapshot` against the `.tgz` they captured. Show them their cluster's actual state.

2. **`helm install --dry-run` against their cluster** — proves the chart is non-invasive, shows exactly what RBAC it requests before they approve it.

3. **The `runs/SUMMARY.md` link** — live evidence that this runs daily in production.

The ask: "Let us deploy the Helm chart to one non-prod namespace with the watcher enabled. The watcher gives you immediate, event-driven alerts within seconds — no configuration beyond a Slack webhook. The CronJob runs in parallel to accumulate daily audit evidence. Two weeks of data, zero operator time, and we compare the results to what your team found manually in the same period."

**CronJob vs. Watcher in one sentence**: the CronJob is your compliance ledger; the watcher is your on-call colleague that never sleeps.

---

## Appendix A — Troubleshooting Common Demo Issues

| Symptom | Fix |
|---|---|
| `cha: permission denied` | `chmod +x cha` |
| `cha diagnose --snapshot` shows no diagnostics on sample-cluster | Verify you're using the repo's `examples/sample-cluster/` directory, not a custom snapshot |
| Helm install fails: `no matches for kind "ExternalSecret"` | ESO not installed; the diagnose CronJob still runs — it simply skips ExternalSecret probes |
| Runner pod stays `Pending` | `kubectl describe pod -n cluster-health-autopilot -l app=cha-runner` — likely imagePullBackOff on `myoung34/github-runner:ubuntu-jammy` |
| DriftReports not appearing | Check `kubectl logs -n cluster-health-autopilot job/<latest-diagnose-job>`; DriftReport CRD may need manual install: `kubectl apply -f charts/cluster-health-autopilot/crds/` |
| `cha remediate --live` refuses in snapshot mode | Expected — fixers are type-system-gated. Must use `--live` flag with valid kubeconfig |
| Watcher pod restarts in a loop | Check `kubectl logs deployment/cha-…-watcher`; likely kubeconfig missing or SA lacks `watch` verb. Run `helm upgrade` to pick up the updated ClusterRole |
| Slack flooded after watcher pod restart | Expected only if DriftReport CRD is absent (seen-map cannot be seeded). Install the CRD: `kubectl apply -f charts/cluster-health-autopilot/templates/crd-driftreport.yaml` |
| Watcher posts Slack on every resync | `--slack-repeat-interval` defaults to 4 h; reduce alert volume with `watcher.slack.repeatInterval: 0` to disable repeats |
| Watcher not firing on CRD resources (e.g. ExternalSecrets) | Normal if the CRD is not installed in this cluster — the watcher skips the watch silently. Check logs for `watch … no matches for kind` |

## Appendix B — Full Analyzer + Probe Catalog (v0.9.0)

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
