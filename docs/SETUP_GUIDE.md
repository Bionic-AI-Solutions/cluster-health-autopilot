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
14. [TLSSecretMismatch fixer (opt-in)](#14-tlssecretmismatch-fixer)
15. [Endpoint probe: auto-discovery, flake suppression, Investigator](#15-endpoint-probe-features)

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
docker pull docker4zerocool/cluster-health-autopilot:v1.5.2
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
cephclusters.ceph.rook.io, ingresses, secrets — read-only, never writes). Output is a tarball.

> v1.2 added `Ingress` to the snapshot loader's kind map; previously the loader silently
> dropped Ingresses, so older snapshots may not exercise the `TLSSecretMismatch` analyzer
> or the auto-discovered endpoint targets. Recapture with v1.2+ for full coverage.

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
  --set image.tag=v1.5.2
```

With Alertmanager as hub (recommended):

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set image.tag=v1.5.2 \
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
  --set image.tag=v1.5.2 \
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
  tag: v1.5.2

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
    repeatInterval: 6h
    criticalRepeatInterval: "2h"   # Criticals stay loud (0 = fall back to repeatInterval)

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
  patch deployment annotations, delete cert-manager CertificateRequests and Orders. When
  `fixers.tlsSecretMismatch.enabled=true`, the chart additionally grants
  `networking.k8s.io/ingresses [patch]` on this role; otherwise the verb is omitted
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
    postOnResolved: true        # Post when a diagnostic resolves
    repeatInterval: 6h          # Re-post still-active warning/info issues at this cadence (0 = never)
    criticalRepeatInterval: "2h" # Re-post still-active criticals (default 2h; 0 = fall back to repeatInterval)
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
  --slack-repeat-interval=6h \
  --slack-critical-repeat-interval=2h \
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
| `--slack-repeat-interval` | `6h` | Re-post still-active warning/info diagnostics; `0` disables |
| `--slack-critical-repeat-interval` | `2h` (= criticals stay loud; `0` falls back to `--slack-repeat-interval`) | Per-severity override for **critical** diagnostics. Defaults to `2h` so unresolved criticals keep re-posting while warnings calm down at `--slack-repeat-interval` (`6h`). New in v1.6.1. |
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

### Endpoint probe reports `[transient, 1/2]` warning, then escalates next cycle

This is the v1.4 default. The endpoint probe applies a two-strikes streak counter to
**transient** failures (context deadline, connection reset, EOF, `no such host`, i/o
timeout). The first cycle emits SeverityWarning with a `[transient, 1/2]` tag; a second
consecutive transient on the same target escalates to SeverityCritical. **Deterministic**
failures (TLS error, HTTP status mismatch, invalid URL) bypass the counter and fire as
critical on the first cycle. See §15.2 for the rationale.

### Investigator block (🔬) missing from Slack/Alertmanager output

The OSS rule-based Investigator is on by default. It is silently skipped if:
- `CHA_INVESTIGATOR=off` is set on the watcher Deployment (intentional disable)
- No Investigator rule matches the finding (no DNS/TLS/HTTP/secret/cert pattern)

To force-enable for diagnosis, ensure the env var is not set to `off`, then `kubectl
rollout restart deployment/cha-cluster-health-autopilot-watcher`. No RBAC change is
required — the Investigator only uses verbs the reader role already grants.

### TLSSecretMismatch fixer refuses to patch a GitOps-managed Ingress

By design. The fixer skips any Ingress carrying ArgoCD
(`argocd.argoproj.io/instance` or `tracking-id`), Flux
(`kustomize.toolkit.fluxcd.io/name|namespace`), or Helm
(`meta.helm.sh/release-name|namespace`) ownership annotations, as well as any object
labeled `app.kubernetes.io/managed-by` ∈ `{helm, argocd, flux, fluxcd}`. The analyzer
still emits the diagnostic; only the in-cluster patch is suppressed. Fix the desired
state in the GitOps source of truth instead.

---

## 14. TLSSecretMismatch fixer

CHA v1.3.0 introduced the `TLSSecretMismatch` analyzer (always on in OSS) and a matching
opt-in fixer. The analyzer scans every Ingress with `spec.tls[].secretName` and verifies
the referenced Secret exists in the same namespace, is type
`kubernetes.io/tls`, and contains usable cert+key data. When the analyzer finds a
mismatch — wrong name, missing object, or wrong type — it emits a diagnostic with the
precise `kubectl patch` command to correct `spec.tls[].secretName`.

The fixer is **disabled by default**. Enable it with:

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --reuse-values \
  --set fixers.tlsSecretMismatch.enabled=true
```

What this flag does, mechanically:

1. Flips the env var `CHA_FIXER_TLS_SECRET_MISMATCH=true` on the watcher and remediate
   pods, activating the fixer at runtime.
2. Conditionally renders the `networking.k8s.io/ingresses [patch]` verb into the
   remediator ClusterRole. With the flag disabled, the verb is absent — even a
   compromised watcher pod cannot patch any Ingress.

### Safety constraints

- **Protected namespaces** are skipped. The compiled-in floor is `kube-system`,
  `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, and
  `cnpg-system`; operators can APPEND namespaces via `protectedNamespaces.extra`
  (Helm) / `spec.protectedNamespacesExtra` (operator CR) /
  `CHA_PROTECTED_NAMESPACES_EXTRA` (env, comma-separated). The floor itself can
  never be removed.
- **GitOps-managed Ingresses are skipped** so the fixer never fights ArgoCD, Flux, or
  Helm. Detected via:
  - ArgoCD: `argocd.argoproj.io/instance` or `argocd.argoproj.io/tracking-id` annotation
  - Flux: `kustomize.toolkit.fluxcd.io/name` or `…/namespace` annotation
  - Helm: `meta.helm.sh/release-name` or `…/release-namespace` annotation
  - Label `app.kubernetes.io/managed-by` ∈ `{helm, argocd, flux, fluxcd}`
- The analyzer always emits the diagnostic regardless. The escape hatch only suppresses
  the in-cluster mutation.

### Verify the fixer is loaded

```sh
kubectl -n cluster-health-autopilot get deploy \
  cha-cluster-health-autopilot-watcher -o jsonpath='{.spec.template.spec.containers[0].env}' \
  | jq '.[] | select(.name=="CHA_FIXER_TLS_SECRET_MISMATCH")'
# → {"name":"CHA_FIXER_TLS_SECRET_MISMATCH","value":"true"}

kubectl get clusterrole cha-cluster-health-autopilot-remediator -o yaml \
  | grep -A2 '"ingresses"'
# → verbs: ["patch"]   (only if the fixer is enabled)
```

---

## 15. Endpoint probe features

The `Endpoints` probe gained three operator-visible behaviors across v1.2–v1.5. None
require new RBAC; all are surfaced as separate sections here because they change what
operators should expect to see in their alerts.

### 15.1 Auto-discovery of Ingress hosts (v1.2)

The endpoint probe now auto-discovers every Ingress host in the cluster at Run time —
operators no longer need to maintain a static target list. Each discovered host becomes
a probe target with default settings (HTTP GET, 10 s timeout, 2xx/3xx accepted).

- Protected namespaces are skipped.
- Opt-out per-Ingress is supported via the annotation
  `cha.bionicaisolutions.com/probe-disable: "true"` on the Ingress object. Useful for
  Ingresses that intentionally return non-2xx (e.g. webhook receivers) or for hosts not
  reachable from inside the cluster.
- The OSS `IngressCoverage` analyzer (which used to warn "this Ingress is not in the
  probe target list") was removed in v1.2 — the gap it warned about no longer exists.
- Latent bug fix in the same release: the snapshot loader's kind map was missing
  `Ingress`, so `cha diagnose --snapshot` had been silently dropping Ingresses on
  load. Snapshot mode now sees the same Ingress set as `--live`.

Disable a noisy or intentionally-unreachable Ingress:

```sh
kubectl annotate ingress <name> -n <ns> \
  cha.bionicaisolutions.com/probe-disable="true"
```

### 15.2 Layer-1 flake suppression (v1.4)

The endpoint probe applies two layers of network-flake suppression so transient
upstream blips do not page operators:

1. **In-cycle retry.** When the first probe returns a transient error
   (context deadline, connection reset, EOF, `no such host`, i/o timeout), the probe
   retries once in the same cycle with `1.5 × timeout`. Only the retry result is
   surfaced for that cycle.
2. **N-of-M consecutive failures before SeverityCritical.** The first transient failure
   (after retry) is tagged `[transient, 1/2]` at SeverityWarning. A second consecutive
   transient on the **same target** escalates to SeverityCritical. The default threshold
   is 2; success at any point resets the counter.

**Deterministic** failures (TLS handshake error, HTTP status outside the accepted set,
invalid URL) bypass the streak counter — these fire as critical on the first cycle
because they are not flakes and waiting buys nothing.

There is no Helm value to tune the threshold in this release. The 2-of-2 default was
chosen because each cycle is already a debounced full diagnose pass (default 10 m
resync), so the warning-then-critical pattern adds at most one cycle of latency for a
real outage while suppressing virtually all single-cycle flakes seen in production. If
your traffic profile demands a different threshold, file an issue with the use case.

Operators should expect: the first time a new endpoint goes down, Slack/AM shows a
`[transient, 1/2]` SeverityWarning. If the issue is real, the next cycle escalates. If
it was a flake, the warning resolves and never returns. This is the new normal — do not
treat the warning as a regression.

New programmatic constructor for embedders:

```go
endpoints := probe.NewEndpoints(targets, discovery)  // use this, not bare struct literal
```

### 15.3 Layer-2 Investigator (v1.5)

The Investigator is a deterministic, read-only root-cause classifier that runs after a
probe or analyzer emits a Finding/Diagnostic. It pattern-matches the failure mode and
invokes a small set of read-only tools (`DNSLookup`, `HTTPProbe`, `TLSInspect`,
`Describe`, `GetEvents`), then attaches a one-paragraph Summary to the diagnostic. The
Summary surfaces in Slack and Alertmanager as a `🔬` block beneath the regular finding,
and in `DriftReport.spec.investigation` (maxLength 1024).

**OSS ships the rule-based implementation** in
`internal/investigator/rules.go` and it is **on by default**. Rule coverage:

- TLS certificate expiry
- TLS SAN/hostname mismatch
- Connection failure (with transient vs. persistent classification)
- HTTP status mismatch
- Slow-DNS root-cause classification (resolver vs. upstream)
- ExternalSecret diagnostic enrichment
- Secret missing / missing-key
- cert-manager Certificate expiry

The Investigator requires **no new RBAC** — it only uses verbs already on the reader
ClusterRole.

To disable:

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --reuse-values \
  --set 'env.CHA_INVESTIGATOR=off'
```

Or set the env var directly via `extraEnv` if your values schema uses that key. The
binary checks `CHA_INVESTIGATOR=off` (case-sensitive) at startup; any other value, or
unset, leaves the Investigator enabled.

**LLM-backed Investigator (CHA-com).** The same `Investigator` interface is implemented
by an LLM-backed engine in the commercial CHA-com binary; it consults the rule-based
implementation first, falls back to the LLM only when the rules return no Summary, and
respects the same `CHA_INVESTIGATOR=off` kill switch. Operators running CHA-com see
richer free-form root-cause writeups for failure modes the rule-based version does not
cover. Design and rollout details: see
[`docs/design/2026-05-investigator-agent.md`](design/2026-05-investigator-agent.md).

### Sprint 2 — Per-probe opt-out env vars (v1.6.0)

Each new probe is independently disablable via env var on the watcher
Deployment (or CronJob, for diagnose/remediate). Set to `off`:

| Env var | Disables |
|---|---|
| `CHA_PROBE_NODE_PRESSURE` | NodePressure probe |
| `CHA_PROBE_DAEMONSETS` | System DaemonSets probe |
| `CHA_PROBE_PENDING_PODS` | PendingPods probe |
| `CHA_PROBE_CRASHLOOP` | Generic CrashLoopBackOff probe |
| `CHA_PROBE_ETCD` | ETCD probe (use this for k3s / managed control plane) |
| `CHA_PROBE_FAILED_MOUNTS` | FailedMounts probe |

The six base probes are also independently disablable (set to `off`; Helm:
`probes.<name>.enabled: false`):

| Env var | Disables | Helm value |
|---|---|---|
| `CHA_PROBE_CEPH` | Ceph Storage probe | `probes.ceph.enabled` |
| `CHA_PROBE_NODES` | Cluster Nodes probe | `probes.nodes.enabled` |
| `CHA_PROBE_POSTGRES` | PostgreSQL probe | `probes.postgres.enabled` |
| `CHA_PROBE_PVCS` | Storage Claims (PVC) probe | `probes.pvcs.enabled` |
| `CHA_PROBE_CRITICAL_WORKLOADS` | Critical Services (workload list) probe | `probes.criticalWorkloads.enabled` |
| `CHA_PROBE_ENDPOINTS` | External Endpoints probe | `probes.endpoints.enabled` |

The seven core analyzers (secret-chain / cert / image-auth) default ON and
are independently disablable the same way (set to `off`; Helm:
`analyzers.<name>.enabled: false`):

| Env var | Disables | Helm value |
|---|---|---|
| `CHA_ANALYZER_SECRET_KEY_MISSING` | SecretKeyMissing analyzer | `analyzers.secretKeyMissing.enabled` |
| `CHA_ANALYZER_FAILING_EXTERNAL_SECRETS` | FailingExternalSecrets analyzer | `analyzers.failingExternalSecrets.enabled` |
| `CHA_ANALYZER_PROACTIVE_SECRET_KEY_CHECK` | ProactiveSecretKeyCheck analyzer | `analyzers.proactiveSecretKeyCheck.enabled` |
| `CHA_ANALYZER_UNPROVISIONED_SECRET` | UnprovisionedSecret analyzer | `analyzers.unprovisionedSecret.enabled` |
| `CHA_ANALYZER_IMAGE_PULL_AUTH` | ImagePullAuth analyzer | `analyzers.imagePullAuth.enabled` |
| `CHA_ANALYZER_CERT_EXPIRY` | CertExpiry analyzer | `analyzers.certExpiry.enabled` |
| `CHA_ANALYZER_TLS_SECRET_MISMATCH` | TLSSecretMismatch analyzer (the opt-in auto-fixer is gated separately by `fixers.tlsSecretMismatch.enabled` / `CHA_FIXER_TLS_SECRET_MISMATCH`) | `analyzers.tlsSecretMismatch.enabled` |

The 30 cloud probes default ON when their provider is enabled
(`cloud.<provider>.enabled`) and are independently disablable the same way
(set to `off`; Helm: `cloud.<provider>.probes.<name>: false`). The
`eks` / `gke` / `aks` toggles each gate BOTH the control-plane and
node-pool probes:

| Env var | Disables | Helm value |
|---|---|---|
| `CHA_CLOUD_PROBE_AWS_RDS` | AWS RDS probe | `cloud.aws.probes.rds` |
| `CHA_CLOUD_PROBE_AWS_EBS` | AWS EBS volumes probe | `cloud.aws.probes.ebs` |
| `CHA_CLOUD_PROBE_AWS_EKS` | AWS EKS control-plane + node-group probes | `cloud.aws.probes.eks` |
| `CHA_CLOUD_PROBE_AWS_IAM` | AWS IAM roles probe | `cloud.aws.probes.iam` |
| `CHA_CLOUD_PROBE_AWS_ALB` | AWS ALB target-health probe | `cloud.aws.probes.alb` |
| `CHA_CLOUD_PROBE_AWS_ACM` | AWS ACM cert-expiry probe | `cloud.aws.probes.acm` |
| `CHA_CLOUD_PROBE_AWS_KMS` | AWS KMS keys probe | `cloud.aws.probes.kms` |
| `CHA_CLOUD_PROBE_AWS_S3` | AWS S3 public-access probe | `cloud.aws.probes.s3` |
| `CHA_CLOUD_PROBE_AWS_VPC` | AWS VPC subnets probe | `cloud.aws.probes.vpc` |
| `CHA_CLOUD_PROBE_GCP_CLOUDSQL` | GCP Cloud SQL probe | `cloud.gcp.probes.cloudsql` |
| `CHA_CLOUD_PROBE_GCP_DISKS` | GCP persistent disks probe | `cloud.gcp.probes.disks` |
| `CHA_CLOUD_PROBE_GCP_GKE` | GCP GKE control-plane + node-pool probes | `cloud.gcp.probes.gke` |
| `CHA_CLOUD_PROBE_GCP_IAM` | GCP IAM service-accounts probe | `cloud.gcp.probes.iam` |
| `CHA_CLOUD_PROBE_GCP_SUBNETS` | GCP subnets probe | `cloud.gcp.probes.subnets` |
| `CHA_CLOUD_PROBE_GCP_LB` | GCP LB backends probe | `cloud.gcp.probes.lb` |
| `CHA_CLOUD_PROBE_GCP_CERTS` | GCP managed-certificates probe | `cloud.gcp.probes.certs` |
| `CHA_CLOUD_PROBE_GCP_GCS` | GCP GCS public-access probe | `cloud.gcp.probes.gcs` |
| `CHA_CLOUD_PROBE_GCP_KMS` | GCP KMS probe | `cloud.gcp.probes.kms` |
| `CHA_CLOUD_PROBE_AZURE_SQL` | Azure SQL databases probe | `cloud.azure.probes.sql` |
| `CHA_CLOUD_PROBE_AZURE_DISKS` | Azure disks probe | `cloud.azure.probes.disks` |
| `CHA_CLOUD_PROBE_AZURE_AKS` | Azure AKS control-plane + node-pool probes | `cloud.azure.probes.aks` |
| `CHA_CLOUD_PROBE_AZURE_IDENTITIES` | Azure managed-identities probe | `cloud.azure.probes.identities` |
| `CHA_CLOUD_PROBE_AZURE_SUBNETS` | Azure subnets probe | `cloud.azure.probes.subnets` |
| `CHA_CLOUD_PROBE_AZURE_APPGW` | Azure App Gateway backends probe | `cloud.azure.probes.appgw` |
| `CHA_CLOUD_PROBE_AZURE_CERTS` | Azure certificates probe | `cloud.azure.probes.certs` |
| `CHA_CLOUD_PROBE_AZURE_STORAGE` | Azure storage public-access probe | `cloud.azure.probes.storage` |
| `CHA_CLOUD_PROBE_AZURE_KEYVAULTS` | Azure Key Vaults probe | `cloud.azure.probes.keyvaults` |

Tuning knob (not a disable toggle): `CHA_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX` sets the capacity-only GCP subnets probe's small-prefix threshold (an unmeasured subnet with a primary CIDR smaller than `/<threshold>` is flagged). Rendered from `cloud.gcp.subnetsSmallPrefixThreshold`; 0 / unset = the compiled-in `/26` default.

Critical-workloads list configuration (Sprint 2.6):

| Env var | Effect |
|---|---|
| `CHA_CRITICAL_SERVICES` | Semicolon-separated `ns/selector\|Display` pairs; appended to compiled-in defaults |
| `CHA_CRITICAL_SERVICES_REPLACE=true` | Replace defaults entirely (clusters with no Bionic services) |
| Annotation `cha.bionicaisolutions.com/probe-critical: "true"` | On any Deployment / StatefulSet, opts it into the Services probe |
| Annotation `cha.bionicaisolutions.com/probe-display: "..."` | Friendly display name for the annotated workload |

Leader election (Sprint 4.3):

| Env var | Effect |
|---|---|
| `CHA_LEADER_ELECTION=off` | Disable lease acquisition (single-pod dev / non-K8s) |
| `MY_POD_NAMESPACE` | Lease namespace (downward-API; chart sets automatically) |
| `MY_POD_NAME` | Lease holder identity (downward-API; chart sets automatically) |

### DriftReport CR change

`DriftReport.spec.investigation` (string, maxLength 1024) carries the Investigator
Summary. Both Create and Update reconcile paths now refresh severity, message,
remediation, **and investigation** — older CRs created before v1.5 will pick up the
field on the next watcher cycle.

---

## 16. AWS cloud-probe setup (opt-in)

CHA ships 10 AWS probes (RDS, EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC) that
authenticate via [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
— the watcher ServiceAccount assumes a per-cluster IAM role. No long-lived
AWS credentials in CHA.

### Step 1 — Create the IAM policy

Attach this minimum-permissions policy to a new IAM role (rename to match
your tenant). The probes only read; no `*:Delete*`, `*:Put*`, `*:Create*`
verbs are requested.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "RDSRead",
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances",
        "rds:DescribeDBClusters",
        "rds:DescribeDBSnapshots"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2EBSVPCRead",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVolumes",
        "ec2:DescribeSnapshots",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs",
        "ec2:DescribeAvailabilityZones"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EKSRead",
      "Effect": "Allow",
      "Action": [
        "eks:DescribeCluster",
        "eks:DescribeNodegroup",
        "eks:ListNodegroups",
        "eks:DescribeAddon",
        "eks:ListAddons"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMRead",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole",
        "iam:ListRoles",
        "iam:GetRolePolicy",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ALBRead",
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeTargetHealth"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ACMRead",
      "Effect": "Allow",
      "Action": ["acm:ListCertificates", "acm:DescribeCertificate"],
      "Resource": "*"
    },
    {
      "Sid": "KMSRead",
      "Effect": "Allow",
      "Action": [
        "kms:ListKeys",
        "kms:DescribeKey",
        "kms:GetKeyRotationStatus",
        "kms:ListAliases"
      ],
      "Resource": "*"
    },
    {
      "Sid": "S3Read",
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:GetBucketPolicyStatus",
        "s3:GetBucketAcl",
        "s3:GetBucketLocation"
      ],
      "Resource": "*"
    }
  ]
}
```

### Step 2 — IRSA trust policy

Bind the role to the cha ServiceAccount via the cluster's OIDC provider:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::<ACCOUNT_ID>:oidc-provider/<OIDC_PROVIDER_HOST>"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "<OIDC_PROVIDER_HOST>:sub": "system:serviceaccount:cluster-health-autopilot:cha-cluster-health-autopilot-sa",
          "<OIDC_PROVIDER_HOST>:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

### Step 3 — Enable in the Helm chart

```yaml
cloud:
  aws:
    enabled: true
    region: us-east-1
    roleArn: arn:aws:iam::123456789012:role/cha-cloud-readonly  # IRSA role
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/cha-cloud-readonly
```

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reset-then-reuse-values \
  -f aws-values.yaml
```

### Step 4 — Verify

The watcher pod should log `cloud: aws enabled (region=us-east-1)` on startup,
and the next diagnose cycle will include AWS sections in the report. If
authentication fails, the AWS probes surface a single `cloud_aws/<service>/auth-failed`
finding rather than crashing the diagnose cycle.

### Non-AWS clusters

CHA on GKE, AKS, bare-metal, k3s, etc. — leave `cloud.aws.enabled: false`
(the chart's default). The cloud-probe layer is a complete no-op when no
provider is enabled. **GCP and Azure equivalents are scoped for M2 (v1.7+);
setting `--cloud-gcp-enabled` or `--cloud-azure-enabled` today errors at
binary startup.**

---

## Upgrading from v1.5.x → v1.6.0

Two operational gotchas worth knowing about:

### `helm upgrade --reuse-values` no longer works for v1.6.0

The v1.6.0 chart added new value blocks (`watcher.leaderElection`,
`diagnose.backoffLimit`, `diagnose.activeDeadlineSeconds`,
`remediation.backoffLimit`, `remediation.activeDeadlineSeconds`).
Helm's `--reuse-values` flag carries *only* user-set values from the
prior release; chart-default blocks fall back to nil and the
templates panic. Use `--reset-then-reuse-values` (Helm 3.14+) instead:

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reset-then-reuse-values \
  --set image.tag=v1.6.0
```

### Mutable tags + IfNotPresent — pin to a digest or use a unique tag

If you push the same mutable tag (e.g. `latest`, `v1.6.0`) twice with
different image content, the kubelet's `IfNotPresent` cache will use
the stale digest. The v1.6.0 chart adds a `cha.pullPolicy` helper that
detects mutable tags (`latest`, `*-latest`, `main`, `dev`) and forces
`imagePullPolicy: Always`. Semver-style tags continue to use
`IfNotPresent`. In production, prefer pinning the chart to an
immutable tag or a digest reference.

### New v1.6.0 capabilities

After the upgrade, verify the new probes are firing:

```sh
kubectl exec -n cluster-health-autopilot \
  deploy/cha-cluster-health-autopilot-watcher \
  -- /usr/local/bin/cha diagnose --live --format text
```

Twelve probes should appear (Ceph / Postgres / Critical Services /
Cluster Nodes / Storage Claims / External Endpoints **+ Node Pressure /
System DaemonSets / Pending Pods / CrashLoopBackOff / ETCD / Failed
Mounts**).

The watcher pod will log lease acquisition on startup:
```
watcher: acquired lease cluster-health-autopilot/cha-watcher as "<pod-name>"
```

---

## Appendix — Key Helm values reference

```yaml
image:
  # Docker Hub is the canonical publish target. GHCR mirror at
  # ghcr.io/bionic-ai-solutions/cluster-health-autopilot is published by
  # GoReleaser on every release for operators who prefer it.
  repository: docker4zerocool/cluster-health-autopilot
  tag: ""           # defaults to Chart.appVersion (e.g. v1.6.0)
  pullPolicy: ""    # auto: Always for mutable tags, IfNotPresent for semver
  pullSecrets: []   # [{name: dockerhub-pull-secret}]

diagnose:
  enabled: true
  schedule: "0 9 * * *"
  format: daily              # daily | text | json
  backoffLimit: 1            # v1.6: cap Job retries (K8s default was 6)
  activeDeadlineSeconds: 120 # v1.6: cap Job wall-clock (K8s default was unlimited)

remediation:
  enabled: false
  schedule: "*/30 * * * *"
  dryRun: false
  backoffLimit: 1            # v1.6
  activeDeadlineSeconds: 120 # v1.6

# Sprint 4.3 — Lease-based leader election. Default on so multi-replica
# watcher deployments are safe. Disable for single-pod dev / non-K8s runs.
watcher:
  leaderElection:
    enabled: true
    leaseName: cha-watcher
    leaseDuration: 30s
    renewDeadline: 20s
    retryPeriod: 5s

fixers:
  # Opt-in OSS fixer added in v1.3. When true:
  #   - sets CHA_FIXER_TLS_SECRET_MISMATCH=true on watcher + remediate pods
  #   - adds `networking.k8s.io/ingresses [patch]` to the remediator ClusterRole
  # Skips protected namespaces and GitOps-managed Ingresses (ArgoCD/Flux/Helm/
  # managed-by label). The analyzer always emits the diagnostic regardless.
  tlsSecretMismatch:
    enabled: false

# Protected namespaces (v1.26.0). The compiled-in floor — kube-system,
# kube-public, kube-node-lease, rook-ceph, vault, external-secrets,
# cnpg-system — is NEVER touched by any fixer or AI-proposed action and
# cannot be shrunk. `extra` APPENDS namespaces to that floor: rendered as
# CHA_PROTECTED_NAMESPACES_EXTRA on the watcher, diagnose, remediate, and
# aiwatch containers so the fixer guard AND the AI-action validator extend
# the same boundary. Operator-managed installs use the equivalent CR field
# `spec.protectedNamespacesExtra`. Append-only: garbage or duplicate
# entries are ignored; nothing in this list can un-protect a floor entry.
protectedNamespaces:
  extra: []          # e.g. [prod-payments, tenant-a]

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
    repeatInterval: 6h
    criticalRepeatInterval: "2h"
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

# Layer-2 Investigator (v1.5). On by default in OSS (rule-based).
# Set CHA_INVESTIGATOR=off to disable cluster-wide. No new RBAC required.
# The CHA-com binary swaps in the LLM-backed implementation behind the
# same interface and respects the same kill switch.
env:
  CHA_INVESTIGATOR: ""   # "" = on (default), "off" = disabled

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

---

## 17. CHA-com paid binary — install on top of OSS

The OSS chart templates an **approval-server** Deployment under
`approval.enabled=true`. That Deployment runs the paid CHA-com binary
(`cha-com approval-server`), which provides the AI-tier approval
webhook + Ed25519 signing flow. The OSS engine continues to handle
probes / analyzers / fixers; the paid binary is a *sidecar* that
augments alerts with AI features.

### Step 1 — Generate the approval signing key

```sh
# Run once, persists into a K8s Secret the chart references.
kubectl create namespace cluster-health-autopilot --dry-run=client -o yaml | kubectl apply -f -
kubectl run cha-com-keygen \
  --namespace cluster-health-autopilot \
  --image=docker4zerocool/cha-com:v1.0.0 \
  --restart=Never \
  --command -- /usr/local/bin/cha-com gen-signing-key \
    --secret-namespace=cluster-health-autopilot \
    --secret-name=cha-com-signing-key
kubectl delete pod cha-com-keygen -n cluster-health-autopilot
```

### Step 2 — Enable approval-server in Helm values

```yaml
approval:
  enabled: true
  image:
    repository: docker4zerocool/cha-com
    tag: v1.0.0
  signingKey:
    secretName: cha-com-signing-key
  baseURL: https://cha-approve.example.com   # external URL operators see
ai:
  enabled: true
  tier: t0      # t0 / t1 / t2 / t3 / off
  endpoint: https://gpu-ai.example.com/v1   # OpenAI-compatible
  model: qwen3.6-35b-a3b-fp8                # in-cluster vLLM recommended
  apiKey:
    secretName: cha-com-llm-key
    secretKey:  api-key
```

### Step 3 — Helm upgrade

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reset-then-reuse-values \
  -f cha-com-values.yaml
```

The watcher pod starts emitting AI-enriched DriftReports; the
approval-server pod handles signed click-to-fix URLs.

### What's shipped today vs roadmap

CHA-com v1.0.0 ships:

- **Approval-server with Ed25519 signing + JTI replay protection** (real)
- **`gen-signing-key`** utility (real)
- **Paid catalog plumbing** — `PaidBoundaryAnalyzer` boundary-check today;
  real paid analyzers ship in G2 increments
- **AI tier code** in `ai/` package (enricher, fix-proposer, planner, vault-runbook, audit hash-chain) — used by the approval-server; the `cha-com diagnose/watch` subcommands that drive T0–T4 enrichment in-process are G3 work, not yet wired into the binary's CLI surface

See [`docs/design/2026-05-cha-com-publishing-gap.md`](design/2026-05-cha-com-publishing-gap.md) for the
G1/G2/G3 work breakdown.

---

## Appendix — GHCR package visibility (one-time setup)

OSS GoReleaser publishes to both Docker Hub and GHCR on every release.
The first push to GHCR creates the package as **private** by default. To
make it pullable by non-org-members:

1. Open https://github.com/orgs/Bionic-AI-Solutions/packages/container/cluster-health-autopilot/settings
2. Under "Danger Zone" → "Change package visibility" → **Public**
3. Repeat for `cha-com` package once the first CHA-com release publishes

This is a one-time step per package — subsequent releases inherit the
visibility setting. Docker Hub images are public-by-default and don't
need this step.

---

## 14. AI tier setup (CHA-com only)

AI tiers (T0 narration through T3 break-glass Vault runbook) ship in
the **commercial CHA-com binary**. The OSS `cha` binary remains AI-free
regardless of these settings. See [AI_USAGE.md](AI_USAGE.md) for the
positioning rationale and [AI_TIERS.md](AI_TIERS.md) for the tier
capability specification.

### 14.1 Prerequisites

In addition to the base prerequisites (§1):

| Tier | Additional prerequisite |
|---|---|
| **T0** | LLM endpoint reachable from inside the cluster (in-cluster vLLM/Ollama, or operator-approved SaaS with `--ai-allow-saas`) |
| **T1** | T0 + Ingress with HTTPS + OIDC-aware reverse proxy (oauth2-proxy or Kong key-auth + jwt) sitting in front of the approval-server |
| **T2** | T1 + SREs trained on multi-step approval workflow |
| **T3** | T2 + RBAC group `cha.io/approver` with ≥2 distinct members + Vault path allowlist defined |
| **All** | Optional but recommended: OPA Gatekeeper installed for defense-in-depth admission |

### 14.2 Container image

```sh
docker pull docker4zerocool/cha-com:v1.0.0
```

### 14.3 T0 install (narration only) — start here

```sh
# Optional: API key Secret (skip if your LLM endpoint has no auth)
kubectl create secret generic cha-ai-llm-key \
  --namespace cluster-health-autopilot \
  --from-literal=API_KEY=<your-key>

helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --reuse-values \
  --set image.repository=docker4zerocool/cha-com \
  --set image.tag=v1.0.0 \
  --set ai.enabled=true \
  --set ai.tier=t0 \
  --set ai.endpoint=http://your-llm-endpoint:8000/v1 \
  --set ai.apiKey.secretName=cha-ai-llm-key
```

Verify Slack/AM messages now include `🤖 _<narrative>_` blocks under
each diagnostic. Kubernetes Events: look for `AIEnrichmentApplied`.

### 14.4 T1 install (one-click fixes)

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --reuse-values \
  --set ai.tier=t1 \
  --set approval.enabled=true \
  --set approval.image.repository=docker4zerocool/cha-com \
  --set approval.image.tag=v1.0.0 \
  --set approval.ingress.enabled=true \
  --set approval.ingress.host=cha-approve.your-domain.com \
  --set approval.ingress.ingressClassName=kong
```

The chart's pre-install hook runs `cha-com gen-signing-key` to create
the `cha-approval-signing-key` Secret (Ed25519 keypair, signing-key
mounted to approval-server only — watcher SA cannot read it).

Configure your Ingress controller to require OIDC for `/approve` paths
(verified `X-Forwarded-User` header is read by the approval-server).

Optional defense-in-depth — install OPA Gatekeeper:

```sh
helm install gatekeeper gatekeeper/gatekeeper -n gatekeeper-system --create-namespace
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set gatekeeper.install=true
```

### 14.5 T2 / T3 install

```sh
# T2 — multi-step plans (per-step approval state machine)
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.tier=t2

# T3 — Vault recovery runbooks (dual-approval, 30-min audit window)
helm upgrade cha cha/cluster-health-autopilot --reuse-values \
  --set ai.tier=t3 \
  --set 'ai.t3.allowedPathPrefixes={secret/t6-apps/,secret/shared/}'
```

### 14.6 End-to-end verification

```sh
# Inject a stale Error pod
kubectl create job crash-demo --from=cronjob/cha-cluster-health-autopilot-diagnose -n default 2>/dev/null
# (or use the demo job from DEMO_GUIDE Part 9)

# Within ~15s: Slack #ceph-critical shows an Apply Fix button.
# Click the URL — approval-server logs ai.approval.granted, then
# ai.action.applied with post_apply_verified=true.
```

Negative tests:
- Click same URL twice → 409 Conflict (replay rejected)
- Wait 16 min, then try in a fresh tab → 410 Gone (expired)
- Trigger Error pod in `kube-system` → no Apply Fix button emitted (proposer refuses protected NS)

### 14.7 Disabling AI tiers (downshift)

```sh
# Downshift to T0 — keeps narration, disables Apply Fix buttons
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.tier=t0

# Full disable — returns the cluster to OSS behavior
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.enabled=false
```

In-flight T2 plans and T3 runbooks remain valid until their TTL
expires; downshift never strands pending approvals.

### 14.8 Operational runbooks

- **LLM endpoint down**: deterministic diagnostics continue, `🤖` block
  omitted, `AIEnrichmentFailed` events recorded. No operator action
  required; enrichment resumes when the endpoint comes back.
- **Circuit breaker tripped**: `ai.circuit_breaker.tripped` (Warning)
  Event routed to oncall via Alertmanager. Investigate the
  consecutive post-apply failures; reset via operator endpoint when
  confident.
- **Approver out of office (T3)**: T3 runbooks expire after 60 min
  awaiting second approval. A new runbook is generated on the next
  cycle. Update the `cha.io/approver` group membership to include
  more approvers.
- **Rate limit hit**: `ai.rate_limited` Events expose `retry_after_ms`.
  Raise `ai.rateLimit.actionsPerHour` if sustained limits indicate
  the budget is too tight.

### 14.9 Audit-trail review

```sh
# Recent AI events (Kubernetes Events sink, default)
kubectl -n cluster-health-autopilot get events --sort-by=lastTimestamp | grep -E "AI(Enrichment|Proposal|Approval|Action|Runbook)"

# Filter by tier via annotation
kubectl -n cluster-health-autopilot get events -o json | \
  jq '.items[] | select(.metadata.annotations."cha.bionicaisolutions.com/audit-tier" == "t1")'
```

For production retention (Kubernetes Events GC at 1h), configure a
Loki or OTLP sink — see [AI_AUDIT_TRAIL.md](AI_AUDIT_TRAIL.md) for the
event schema and production sink wiring.
