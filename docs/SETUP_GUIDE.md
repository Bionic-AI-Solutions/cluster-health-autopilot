# Cluster Health Autopilot — Full Setup Guide

Self-healing operational layer for Kubernetes: **detect → fix → re-verify → report**, on a
schedule, without dashboards or pagers. This guide covers every step to install and operate
CHA on a **brand-new cluster** — from first prerequisites to live Alertmanager routing.

---

## Table of contents

1. [Prerequisites](#1-prerequisites)
2. [Install the binary (local / offline use)](#2-install-the-binary)
3. [Zero-trust offline mode (no install, no RBAC)](#3-zero-trust-offline-mode)
4. [In-cluster install via Helm](#4-in-cluster-install-via-helm)
5. [Alertmanager integration (recommended hub)](#5-alertmanager-integration)
6. [Slack three-channel routing](#6-slack-three-channel-routing)
7. [Vault probe (closes the L1 stale-Ready window)](#7-vault-probe)
8. [ESO / Vault provisioning analyzers](#8-eso--vault-provisioning-analyzers)
9. [DriftReport CRD (kubectl-queryable diagnostics)](#9-driftreport-crd)
10. [Watcher mode (event-driven, real-time)](#10-watcher-mode)
11. [Retiring old bash health CronJobs](#11-retiring-old-bash-health-cronjobs)
12. [Verification checklist](#12-verification-checklist)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Prerequisites

### Hard requirements (must exist before installing CHA)

| Requirement | Minimum version | Notes |
|---|---|---|
| Kubernetes cluster | 1.27 | Any distribution — EKS, GKE, bare-metal, k3s, RKE2 |
| `kubectl` | any | In your PATH; kubeconfig pointing at the target cluster |
| `helm` | 3.x | For in-cluster install |
| Container image pull access | — | Cluster must be able to pull from Docker Hub (`docker.io`) |

### Soft requirements (enable specific features)

| Feature | What is needed |
|---|---|
| Slack alerts | Three pre-existing Kubernetes Secrets (one per channel) — create manually or via ESO |
| Alertmanager routing | A running Alertmanager instance reachable from inside the cluster (e.g. `kube-prometheus-stack`) |
| Vault probe | HashiCorp Vault KV-v2 + a Vault role bound to the CHA ServiceAccount via kubernetes-auth |
| ESO analyzers | External Secrets Operator installed in the cluster |
| Auto-remediation | `remediation.enabled: true` in Helm values (opt-in) |
| Daily run-log publishing | Self-hosted GitHub Actions runner + GitHub PAT in Vault |

### What CHA does NOT require

- Prometheus / Grafana — CHA is self-contained; it reads the Kubernetes API directly
- MinIO / object storage — required only for the optional run-log publishing pipeline
- Persistent storage — the watcher Deployment is stateless; DriftReport history lives in etcd as CRs
- Elevated privileges — CHA runs as a read-only ServiceAccount; the remediator role is narrowly scoped

> **For zero-trust offline mode (section 3) you need none of the above.** Just the `cha` binary
> and a `kubectl get -o json` snapshot.

---

## 2. Install the binary

### Download a release binary (recommended)

```sh
# Linux amd64
curl -sSL https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_Linux_x86_64.tar.gz \
  | tar xz && sudo mv cha /usr/local/bin/

# macOS arm64
curl -sSL https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_Darwin_arm64.tar.gz \
  | tar xz && sudo mv cha /usr/local/bin/

# Verify
cha version
```

Pre-built binaries for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`.

Each release includes a `checksums.txt`:

```sh
sha256sum -c checksums.txt --ignore-missing
```

### Build from source

```sh
git clone https://github.com/Bionic-AI-Solutions/cluster-health-autopilot.git
cd cluster-health-autopilot
go build -o cha ./cmd/cha   # requires Go 1.22+
```

### Container image

```sh
docker pull docker4zerocool/cluster-health-autopilot:latest
# Pin to a specific version:
docker pull docker4zerocool/cluster-health-autopilot:v0.9.5
```

> **Image registry**: `docker4zerocool/cluster-health-autopilot` on Docker Hub.
> The chart's default `image.repository` already points here.

---

## 3. Zero-trust offline mode

No cluster access needed. Capture a snapshot from any machine with `kubectl`, then
analyze it from any machine.

### Step 1 — capture a snapshot

```sh
# From a machine with kubectl + cluster access:
cha snapshot capture --tar my-cluster.tgz
```

Runs `kubectl get -o json` for each supported GVR (pods, events, deployments, replicasets,
statefulsets, jobs, cronjobs, nodes, pvcs, externalsecrets, clusters.postgresql.cnpg.io,
cephclusters.ceph.rook.io, secrets — read-only, never writes). Output is a tarball.

### Step 2 — diagnose offline

```sh
# From any machine — no cluster access, no credentials:
cha diagnose --snapshot my-cluster.tgz
# or from the sample fixture:
cha diagnose --snapshot examples/sample-cluster
```

### Step 3 — JSON output for automation

```sh
cha diagnose --snapshot my-cluster.tgz --format json | jq '.diagnostics'
```

---

## 4. In-cluster install via Helm

### Step 1 — add the Helm repository

```sh
helm repo add cha https://bionic-ai-solutions.github.io/cluster-health-autopilot
helm repo update
```

### Step 2 — create the namespace

```sh
kubectl create namespace cluster-health-autopilot
```

### Step 3 — create Slack secrets (if using Slack; see section 6)

```sh
# Three channels — create whichever you need:
kubectl create secret generic cha-slack-ceph-alerts \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...

kubectl create secret generic cha-slack-ceph-critical \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...

kubectl create secret generic cha-slack-healthinfo \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...
```

### Step 4 — (optional) Docker Hub image pull secret

If your cluster does not have public Docker Hub access configured globally:

```sh
kubectl create secret docker-registry dockerhub-pull-secret \
  --namespace cluster-health-autopilot \
  --docker-username=<user> \
  --docker-password=<token>
```

Then add to your `values.yaml`:

```yaml
image:
  pullSecrets:
    - name: dockerhub-pull-secret
```

### Step 5 — install

Minimal install (diagnose CronJob only, no Slack):

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set image.tag=v0.9.5
```

With Alertmanager as hub (recommended):

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set image.tag=v0.9.5 \
  --set alertmanager.enabled=true \
  --set alertmanager.url=http://alertmanager.pg.svc.cluster.local:9093 \
  --set alertmanager.clusterName=my-cluster \
  --set watcher.enabled=true \
  --set watcher.remedy.enabled=true
```

Full install with Alertmanager + three-channel Slack + watcher:

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set image.tag=v0.9.5 \
  --set alertmanager.enabled=true \
  --set alertmanager.url=http://alertmanager.pg.svc.cluster.local:9093 \
  --set "alertmanager.clusterName=my-cluster" \
  --set watcher.enabled=true \
  --set watcher.remedy.enabled=true \
  --set slack.alerts.enabled=true \
  --set slack.alerts.secretName=cha-slack-ceph-alerts \
  --set slack.critical.enabled=true \
  --set slack.critical.secretName=cha-slack-ceph-critical \
  --set slack.healthinfo.enabled=true \
  --set slack.healthinfo.secretName=cha-slack-healthinfo
```

Or use a `values.yaml` file (recommended for production):

```yaml
# cha-values.yaml
image:
  tag: v0.9.5

alertmanager:
  enabled: true
  url: "http://alertmanager.pg.svc.cluster.local:9093"
  clusterName: "my-cluster"

watcher:
  enabled: true
  debounce: 10s
  resyncPeriod: 10m
  remedy:
    enabled: true
    dryRun: false
  slack:
    postOnResolved: true
    repeatInterval: 4h

slack:
  alerts:
    enabled: true
    secretName: cha-slack-ceph-alerts
  critical:
    enabled: true
    secretName: cha-slack-ceph-critical
  healthinfo:
    enabled: true
    secretName: cha-slack-healthinfo

diagnose:
  schedule: "0 9 * * *"
  format: daily

driftReport:
  enabled: true
```

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  -f cha-values.yaml
```

### What the chart creates

- `ServiceAccount/cha-cluster-health-autopilot`
- `ClusterRole/cha-cluster-health-autopilot-reader` — read-only on pods, nodes, pvcs, events,
  namespaces, deployments, replicasets, statefulsets, daemonsets, jobs, cronjobs, externalsecrets,
  secretstores, clustersecretstores, secrets, clusters (CNPG), cephclusters, certificates,
  certificaterequests, orders (ACME), driftreports, ingresses, services, endpoints
- `ClusterRole/cha-cluster-health-autopilot-remediator` — narrow write: delete pods/jobs,
  patch deployment annotations, delete cert-manager CertificateRequests and Orders
- `ClusterRole/cha-cluster-health-autopilot-driftreport-writer` — create/patch/delete DriftReports
- `CronJob/cha-cluster-health-autopilot-diagnose` — daily `cha diagnose --live` (default: `0 9 * * *`)
- `CronJob/cha-cluster-health-autopilot-remediate` — opt-in auto-fixers (off by default)
- `Deployment/cha-cluster-health-autopilot-watcher` — real-time event-driven engine (if enabled)

### Verify install

```sh
kubectl -n cluster-health-autopilot get all
kubectl -n cluster-health-autopilot get cronjobs

# Trigger an immediate diagnose run:
kubectl -n cluster-health-autopilot create job \
  --from=cronjob/cha-cluster-health-autopilot-diagnose cha-test-$(date +%s)

# Stream logs:
kubectl -n cluster-health-autopilot logs \
  -l job-name=cha-test-<timestamp> --follow
```

---

## 5. Alertmanager integration

Alertmanager is the **recommended hub** for CHA's issue routing. Rather than posting directly
to Slack, CHA posts all active issues to Alertmanager as alerts on every watcher cycle.
Alertmanager handles: deduplication, grouping, silencing, repeat intervals, and fan-out to
all configured receivers (Slack, PagerDuty, Teams, email, webhook, …).

### Why Alertmanager over direct Slack

| Concern | Direct Slack | Alertmanager |
|---|---|---|
| Dedup | CHA does its own seen-map | Alertmanager does it cluster-wide |
| Silencing | Not supported | Full silence UI |
| Multi-receiver | Manual per-channel code | Config-file routes |
| Repeat interval | Hardcoded in CHA | Configurable per route |
| History | Slack history only | Alertmanager history + silence log |

### Prerequisites

A running Alertmanager instance reachable from inside the cluster. The most common way is
`kube-prometheus-stack` (creates an Alertmanager in the monitoring or pg namespace).

```sh
# Verify Alertmanager is reachable from within the cluster:
kubectl -n cluster-health-autopilot run am-test --image=curlimages/curl --rm -it \
  --restart=Never -- curl -s http://alertmanager.pg.svc.cluster.local:9093/api/v2/status | jq '.versionInfo'
```

### Enable in Helm

```yaml
alertmanager:
  enabled: true
  url: "http://alertmanager.pg.svc.cluster.local:9093"
  clusterName: "my-cluster"   # stamps `cluster` label on every alert
```

### How it works

CHA posts to `/api/v2/alerts` on every watcher cycle. Each active diagnostic becomes an
alert with:

```yaml
labels:
  alertname: cha_issue           # or cha_fixer_acted when CHA fixed something
  severity: critical|warning|info
  subject: "pod/mynamespace/mypod"   # truncated to 256 chars
  source: "SecretKeyMissing"         # probe/analyzer source
  cluster: "my-cluster"
annotations:
  message: "full diagnostic text"
  remediation: "suggested action"
```

Alerts auto-expire after `2 × resyncPeriod + 1 min` (default TTL ≈ 21 min). When CHA resolves
an issue, the alert disappears at the next cycle — no explicit resolve needed.

### Configure Alertmanager routes for CHA

Add CHA-specific routes at the **top** of your Alertmanager config's `route.routes` list so
they take priority over generic severity routes:

```yaml
route:
  routes:
    # CHA auto-fixed an issue → #ceph-alerts (informational)
    - match:
        alertname: cha_fixer_acted
      receiver: 'slack-ceph-alerts'
      group_wait: 10s
      repeat_interval: 1h
      continue: false

    # All other CHA issues → #ceph-critical (needs human attention)
    - match_re:
        alertname: "cha_.*"
      receiver: 'slack-ceph-critical'
      group_wait: 30s
      repeat_interval: 4h
      continue: false

    # ... your existing routes below ...
```

Receivers for Slack (example):

```yaml
receivers:
  - name: 'slack-ceph-alerts'
    slack_configs:
      - api_url_file: /etc/alertmanager/secrets/slack-ceph-alerts/webhook_url
        channel: '#ceph-alerts'
        title: 'CHA Auto-Fixed | {{ .CommonLabels.cluster }}'
        text: |
          {{ range .Alerts }}
          *{{ .Labels.subject }}* — {{ .Annotations.message }}
          {{ end }}

  - name: 'slack-ceph-critical'
    slack_configs:
      - api_url_file: /etc/alertmanager/secrets/slack-ceph-critical/webhook_url
        channel: '#ceph-critical'
        title: '🔴 CHA Issues | {{ .CommonLabels.cluster }}'
        text: |
          {{ range .Alerts }}
          *[{{ .Labels.severity | toUpper }}]* {{ .Labels.subject }}
          {{ .Annotations.message }}
          {{ end }}
```

### Alertmanager filesystem permissions (common gotcha)

If your Alertmanager pod runs as a non-root user (e.g. `nobody`, uid=65534) and the
`/alertmanager/` data directory is owned by root, the notification log and silences
database will fail to write — silently stalling the dispatch pipeline.

Fix by ensuring `fsGroup` is set on the Alertmanager pod:

```yaml
# For kube-prometheus-stack Alertmanager CR:
securityContext:
  fsGroup: 65534
  runAsNonRoot: true
  runAsUser: 65534
```

Verify the dispatch pipeline is healthy:

```sh
kubectl -n <alertmanager-namespace> logs -l app.kubernetes.io/name=alertmanager | \
  grep -i "nflog\|silences\|permission denied"
# Should be empty — any "permission denied" means fsGroup is not set correctly.
```

---

## 6. Slack three-channel routing

CHA routes Slack messages across three dedicated channels:

| Channel | Content | When |
|---|---|---|
| `#ceph-alerts` | Issues CHA **acted on** (auto-fixed) | Event-driven (watcher cycle) |
| `#ceph-critical` | Issues CHA **cannot fix** (needs human) | Event-driven (watcher cycle) |
| `#healthinfo` | Full daily cluster health digest | Daily CronJob (default 09:00 UTC) |

> **If you use Alertmanager** (section 5), `#ceph-alerts` and `#ceph-critical` are routed
> via Alertmanager's receiver config — no direct webhook needed for those. `#healthinfo` is
> always posted directly by the diagnose CronJob.

### Create the Slack secrets

```sh
# Each secret holds a single key: WEBHOOK_URL
kubectl create secret generic cha-slack-ceph-alerts \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...

kubectl create secret generic cha-slack-ceph-critical \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...

kubectl create secret generic cha-slack-healthinfo \
  --namespace cluster-health-autopilot \
  --from-literal=WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...
```

### Option B — ExternalSecret from Vault (recommended for production)

Store webhook URLs in Vault at `secret/t6-apps/cha/config`:

```sh
vault kv patch secret/t6-apps/cha/config \
  slack_alerts_webhook_url="https://hooks.slack.com/services/..." \
  slack_critical_webhook_url="https://hooks.slack.com/services/..." \
  slack_healthinfo_webhook_url="https://hooks.slack.com/services/..."
```

Create ExternalSecrets:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: cha-slack-ceph-alerts
  namespace: cluster-health-autopilot
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: cha-slack-ceph-alerts
    template:
      data:
        WEBHOOK_URL: "{{ .slack_alerts_webhook_url }}"
  data:
    - secretKey: slack_alerts_webhook_url
      remoteRef:
        key: t6-apps/cha/config
        property: slack_alerts_webhook_url
---
# Repeat for cha-slack-ceph-critical and cha-slack-healthinfo
```

### Enable in Helm values

```yaml
slack:
  alerts:
    enabled: true
    secretName: cha-slack-ceph-alerts   # K8s secret name
    secretKey: "WEBHOOK_URL"            # key within the secret
  critical:
    enabled: true
    secretName: cha-slack-ceph-critical
    secretKey: "WEBHOOK_URL"
  healthinfo:
    enabled: true
    secretName: cha-slack-healthinfo
    secretKey: "WEBHOOK_URL"
```

### Daily digest format

When `diagnose.format: daily` (the default), the diagnose CronJob posts to `#healthinfo`
with a structured digest:

```
📊 Daily Cluster Health — my-cluster — 2026-05-11 09:00 UTC

Probes: 12 passed, 0 failed
Active issues (28): 20 persistent, 8 new today
Auto-fixed today: 5
```

Change the schedule:

```yaml
diagnose:
  schedule: "0 9 * * *"  # cron — default is 09:00 UTC daily
  format: daily           # daily | text | json
```

---

## 7. Vault probe

Closes the **L1 stale-Ready window**: detects Vault path/key drift BEFORE the ESO
controller's next refresh cycle, while the ExternalSecret is still reporting `Ready=True`.

### Prerequisites

- Vault KV-v2 mount (default: `secret`)
- A Vault kubernetes-auth role bound to the `cha` ServiceAccount

### Step 1 — create the Vault policy

```hcl
# cha-diagnose-read.hcl
# Scope to only the paths referenced by your ExternalSecrets.
path "secret/data/team/*" {
  capabilities = ["read"]
}
path "secret/data/shared/*" {
  capabilities = ["read"]
}
```

```sh
vault policy write cha-diagnose-read cha-diagnose-read.hcl
```

### Step 2 — create the kubernetes-auth role

```sh
vault write auth/kubernetes/role/cha-diagnose \
  bound_service_account_names=cha-cluster-health-autopilot \
  bound_service_account_namespaces=cluster-health-autopilot \
  policies=cha-diagnose-read \
  ttl=1h
```

### Step 3 — enable in Helm

```yaml
vaultProbe:
  enabled: true
  address: "https://vault.svc.cluster.local:8200"
  kvMount: "secret"
  auth:
    method: kubernetes
    role: "cha-diagnose"
```

### Step 4 — verify

```sh
kubectl -n cluster-health-autopilot create job \
  --from=cronjob/cha-cluster-health-autopilot-diagnose vault-test-$(date +%s)
kubectl -n cluster-health-autopilot logs -l "job-name=vault-test-*" --follow
# Look for "VaultPathMissing: X paths checked, Y missing"
```

> **Security note:** CHA never reads Vault byte values — only key NAMES at each path.
> Scope the Vault policy to only the paths used by your ExternalSecrets.

---

## 8. ESO / Vault Provisioning Analyzers

Three analyzers detect the full class of "Secret not reachable" failures.

### UnprovisionedSecret

Walks every Deployment, StatefulSet, and CronJob and checks whether each Secret referenced
via `envFrom.secretRef` or `volumes.secret.secretName` either exists in the cluster **or**
has an ExternalSecret configured to provision it.

```
Secret `playground/playground-agent-secrets` referenced by Deployment/playground-agent
has no ExternalSecret provisioning it. Create an ExternalSecret with
spec.target.name=playground-agent-secrets pointing to Vault path
`secret/t6-apps/playground/config`.
```

### ProactiveSecretKeyCheck near-miss hint

When a `secretKeyRef` references a missing key, CHA checks whether a case/format variant
exists and appends: `Key GITHUB_TOKEN is a case/format variant — possible naming mismatch.`

### FailingExternalSecrets t6 path hint

When an ESO reports `Ready: False`, CHA checks whether the Vault path follows the expected
hierarchy and suggests the canonical form if not.

---

## 9. DriftReport CRD (kubectl-queryable diagnostics)

CHA creates and maintains `DriftReport` cluster-scoped CRs — one per active issue.

### CRD installation

The CRD is installed automatically by the Helm chart (with `resource-policy: keep` so
`helm uninstall` does NOT remove it). To install manually:

```sh
kubectl apply -f charts/cluster-health-autopilot/crds/driftreports.yaml
```

### Query diagnostics

```sh
# All active issues:
kubectl get driftreports

# Watch live:
kubectl get driftreports --watch

# All critical issues:
kubectl get driftreports -o json | jq '.items[] | select(.spec.severity=="critical")'

# Full detail on one issue:
kubectl describe driftreport <name>
```

### DriftReport fields

```yaml
spec:
  subject:     "missing-key/billing/billing-svc-secrets/STRIPE_API_KEY"
  severity:    "critical"
  source:      "analyzer:SecretKeyMissing"
  message:     "Secret `billing/billing-svc-secrets` is missing key ..."
  remediation: "Add the key to the ExternalSecret template ..."
  category:    "secret-drift"
  resourceRef:
    kind:      "Secret"
    namespace: "billing"
    name:      "billing-svc-secrets"
status:
  firstObserved:    "2026-05-01T09:00:00Z"
  lastObserved:     "2026-05-11T09:00:00Z"
  observationCount: 10
  runID:            "20260511-090001"
```

### Cleanup after helm uninstall

```sh
kubectl delete crd driftreports.cha.bionicaisolutions.com
# This also deletes all DriftReport CRs.
```

---

## 10. Watcher mode

Watcher mode deploys a long-running `Deployment` that holds open Kubernetes watches for all
resource types CHA analyzes. When any resource changes, a short debounce fires, then the
full probe+analyzer stack runs — sub-15 seconds vs waiting for the next cron tick.

**Slack deduplication**: each diagnostic is fingerprinted by subject + severity + message.
Slack posts only when a diagnostic is new, changes, or resolves. The seen-map is seeded from
existing DriftReport CRs on pod startup to avoid a flood after a rolling update.

### Enable via Helm

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.enabled=true \
  --set watcher.remedy.enabled=true
```

### Helm values reference

```yaml
watcher:
  enabled: false           # Deploy the watcher Deployment
  debounce: 10s            # Wait after the last event before running diagnostics
  resyncPeriod: 10m        # Full re-diagnose regardless of events
  slack:
    postOnResolved: true   # Post when a diagnostic resolves
    repeatInterval: 4h     # Re-post still-active issues at this cadence (0 = never)
  remedy:
    enabled: false         # Run auto-fixers after each diagnose cycle
    dryRun: false          # Evaluate fixers without cluster mutation
  resources:
    limits:
      cpu: 500m
      memory: 256Mi
    requests:
      cpu: 50m
      memory: 64Mi
```

### Run manually (without Helm)

```sh
# Alertmanager as hub (recommended):
cha watch --live \
  --debounce=10s \
  --resync-period=10m \
  --alertmanager-url=http://alertmanager:9093 \
  --cluster-name=my-cluster \
  --write-driftreports=true \
  --remedy

# Direct Slack (fallback / simpler setups):
cha watch --live \
  --debounce=10s \
  --resync-period=10m \
  --slack-alerts=$SLACK_ALERTS_URL \
  --slack-critical=$SLACK_CRITICAL_URL \
  --slack-post-on-resolved=true \
  --slack-repeat-interval=4h \
  --write-driftreports=true
```

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `--live` | — | Required; runs against the live cluster |
| `--kubeconfig` | in-cluster / `$KUBECONFIG` | Path to kubeconfig |
| `--debounce` | `10s` | Debounce window after a Kubernetes event |
| `--resync-period` | `10m` | Periodic full re-diagnose interval |
| `--alertmanager-url` | `$ALERTMANAGER_URL` | Alertmanager API URL (preferred hub) |
| `--cluster-name` | `$CLUSTER_NAME` or `cluster` | Stamped as `cluster` label on AM alerts |
| `--slack-alerts` | — | Slack webhook for `#ceph-alerts` (CHA-fixed issues) |
| `--slack-critical` | — | Slack webhook for `#ceph-critical` (unfixable issues) |
| `--slack-post-on-resolved` | `true` | Post when a diagnostic resolves |
| `--slack-repeat-interval` | `4h` | Re-post still-active diagnostics; `0` disables |
| `--write-driftreports` | `true` | Upsert DriftReport CRs on every cycle |
| `--remedy` | `false` | Run auto-fixers after each diagnose cycle |
| `--dry-run` | `false` | With `--remedy`: evaluate without mutating |

### Coexistence with the diagnose CronJob

Running both is safe and recommended:
- The watcher provides sub-minute reactive alerting via Alertmanager
- The CronJob provides the full daily digest to `#healthinfo` and creates WS-C run evidence

---

## 11. Retiring old bash health CronJobs

If you previously ran manual bash-based cluster health CronJobs (common pattern before CHA),
clean them up to avoid duplicate `#healthinfo` posts:

```sh
# Example: old rook-ceph namespace bash health report
kubectl -n rook-ceph delete cronjob cluster-health-report 2>/dev/null || true
kubectl -n rook-ceph delete configmap cluster-health-report-script 2>/dev/null || true
kubectl -n rook-ceph delete secret health-report-slack-webhook 2>/dev/null || true

# Example: old gpu-monitor namespace scripts
kubectl -n gpu-monitor delete cronjob gpu-health-report gpu-docker-monitor 2>/dev/null || true
kubectl -n gpu-monitor delete configmap gpu-monitor-script gpu-docker-monitor-script 2>/dev/null || true
kubectl -n gpu-monitor delete secret gpu-monitor-slack 2>/dev/null || true

# Old single-webhook CHA secret (replaced by three-channel secrets)
kubectl -n cluster-health-autopilot delete secret cha-slack-webhook 2>/dev/null || true
```

After retiring old CronJobs, verify no duplicate Slack messages arrive in `#healthinfo`
on the next daily cycle.

---

## 12. Verification checklist

### A. Namespace and resources

```sh
kubectl -n cluster-health-autopilot get all
# Expected: watcher Deployment (1/1 Running), runner Deployment (if enabled),
#           two CronJobs (diagnose + remediate)
```

### B. Immediate diagnose run

```sh
kubectl -n cluster-health-autopilot create job \
  --from=cronjob/cha-cluster-health-autopilot-diagnose verify-$(date +%s)

# Wait for completion (usually <30s):
kubectl -n cluster-health-autopilot get jobs --watch

# Stream logs:
kubectl -n cluster-health-autopilot logs -l "job-name=verify-*" --follow
```

### C. DriftReport CRD

```sh
kubectl get crd driftreports.cha.bionicaisolutions.com
kubectl get driftreports   # shows active issues; empty if cluster is healthy
```

### D. Watcher health

```sh
kubectl get deployment -n cluster-health-autopilot | grep watcher
kubectl logs -f deployment/cha-cluster-health-autopilot-watcher \
  -n cluster-health-autopilot | head -20
# Expected: "watcher: initial diagnose cycle" then "watcher: cycle complete N diagnostics"
```

### E. Alertmanager

```sh
# Check alerts are flowing:
curl -s http://alertmanager.pg.svc.cluster.local:9093/api/v2/alerts | \
  jq '[.[] | select(.labels.alertname | startswith("cha_"))] | length'
# Should return the number of active CHA issues (>0 if any issues exist)
```

Or from inside the cluster:

```sh
kubectl -n cluster-health-autopilot run am-check --image=curlimages/curl --rm -it \
  --restart=Never -- \
  curl -s http://alertmanager.pg.svc.cluster.local:9093/api/v2/alerts | \
  jq '[.[] | select(.labels.alertname | startswith("cha_"))] | length'
```

### F. Slack

After a diagnose job completes with `format: daily`, check `#healthinfo` for the daily
digest. For watcher Slack posts, inject a failure (see DEMO_GUIDE §5.3) and verify a
deduped message arrives within ~15 seconds.

### G. Zero-trust mode

```sh
cha diagnose --snapshot examples/sample-cluster
# Expected: 9 diagnostics spanning secret drift, unprovisioned Secrets,
#           key-name mismatches, image pull auth, and cert expiry.
```

---

## 13. Troubleshooting

### `cha diagnose --live` returns no results

- Check kubeconfig: `kubectl get pods -A` should work first.
- Check RBAC: `kubectl auth can-i list pods --as=system:serviceaccount:cluster-health-autopilot:cha-cluster-health-autopilot`

### DriftReports not created

- Ensure the CRD is installed: `kubectl get crd driftreports.cha.bionicaisolutions.com`
- Check logs for `driftreport reconcile partial failure`.
- Confirm `--write-driftreports=true` (default) is not overridden.

### Watcher pod crash-loops with `watch requires --live`

The Helm template always injects `--live`; this error means you ran `cha watch` manually
without the flag. Add `--live`.

### Watcher Slack floods after pod restart

This happens when DriftReport CRD is absent — the watcher cannot seed its seen-map.
Install the CRD first: `kubectl apply -f charts/cluster-health-autopilot/crds/driftreports.yaml`

### Watcher `watch … no matches for kind` log lines

Normal for CRDs not installed in this cluster (e.g. CNPG absent on a plain cluster). The
watcher exits that GVR goroutine silently and continues watching all others.

### Alertmanager: CHA alerts not routing to Slack

1. Verify alerts arrive: `curl -s http://alertmanager:9093/api/v2/alerts | jq '. | length'`
2. Check Alertmanager routes are in the right order (CHA routes must be BEFORE generic severity routes)
3. Check Alertmanager logs for `permission denied` on nflog/silences — this means `fsGroup` is not set (see §5)
4. Verify Alertmanager receivers reference the correct Slack secret keys

### VaultPathMissing false positives on non-Vault stores

- Ensure `clusterrole-reader` includes `secretstores`/`clustersecretstores` RBAC.
- Run `helm upgrade` to apply the updated ClusterRole.

### Vault authentication failures (403 in logs)

- Check that the Vault role TTL ≥ the CronJob schedule (TTL=1h for daily is fine).
- Verify `bound_service_account_names=cha-cluster-health-autopilot` (not just `cha`).
- Check `bound_service_account_namespaces=cluster-health-autopilot`.

### Image pull failures

- Verify the cluster can reach Docker Hub: `docker.io/docker4zerocool/cluster-health-autopilot`
- If using a private registry proxy, set `image.repository` to your internal mirror.
- Ensure `imagePullSecrets` is set if Docker Hub rate limits apply.

### Watcher RBAC: `cannot watch pods`

The `watch` verb was added to `clusterrole-reader` in v0.9.0. If upgrading from an earlier chart:

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --reuse-values
```

### `path contains traversal component` error in Vault logs

An ESO has `remoteRef.key` containing `..` or `.` path segments. Fix the ExternalSecret
spec to use a clean path.

---

## Appendix — Key Helm values reference

```yaml
image:
  repository: docker4zerocool/cluster-health-autopilot   # Docker Hub
  tag: ""         # defaults to Chart.appVersion (e.g. v0.9.5)
  pullPolicy: IfNotPresent
  pullSecrets: [] # [{name: dockerhub-pull-secret}]

diagnose:
  enabled: true
  schedule: "0 9 * * *"
  format: daily   # daily | text | json

remediation:
  enabled: false
  schedule: "*/30 * * * *"
  dryRun: false

slack:
  alerts:
    enabled: false
    secretName: ""      # K8s secret with key WEBHOOK_URL
    secretKey: "WEBHOOK_URL"
  critical:
    enabled: false
    secretName: ""
    secretKey: "WEBHOOK_URL"
  healthinfo:
    enabled: false
    secretName: ""
    secretKey: "WEBHOOK_URL"

alertmanager:
  enabled: false
  url: ""           # e.g. http://alertmanager.monitoring.svc.cluster.local:9093
  clusterName: "cluster"

watcher:
  enabled: false
  debounce: 10s
  resyncPeriod: 10m
  slack:
    postOnResolved: true
    repeatInterval: 4h
  remedy:
    enabled: false
    dryRun: false
  resources:
    limits: {cpu: 500m, memory: 256Mi}
    requests: {cpu: 50m, memory: 64Mi}

vaultProbe:
  enabled: false
  address: ""
  kvMount: "secret"
  auth:
    method: kubernetes
    role: ""

driftReport:
  enabled: true

rbac:
  create: true

serviceAccount:
  create: true

podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
```
