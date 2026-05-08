# Cluster Health Autopilot — Full Setup Guide

This guide covers every step to install and operate `cha` on a brand-new
Kubernetes cluster — from downloading the binary to publishing anonymized run
logs. Sections are ordered by complexity; stop at the level you need.

---

## Table of contents

1. [Prerequisites](#1-prerequisites)
2. [Install the binary](#2-install-the-binary)
3. [Zero-trust offline mode (no install, no RBAC)](#3-zero-trust-offline-mode)
4. [In-cluster install via Helm](#4-in-cluster-install-via-helm)
5. [Slack alerts](#5-slack-alerts)
6. [Vault probe (optional — closes the L1 stale-Ready window)](#6-vault-probe)
7. [ESO / Vault provisioning analyzers (UnprovisionedSecret + hints)](#7-eso--vault-provisioning-analyzers)
8. [DriftReport CRD (kubectl-queryable diagnostics)](#8-driftreport-crd)
9. [Watcher mode (event-driven, real-time)](#9-watcher-mode)
10. [Run-log publishing pipeline (WS-C)](#10-run-log-publishing-pipeline)
11. [Verification checklist](#11-verification-checklist)
12. [Troubleshooting](#12-troubleshooting)

---

## 1. Prerequisites

| Requirement | Minimum version | Notes |
|---|---|---|
| Kubernetes cluster | 1.27 | Any distribution (EKS, GKE, bare-metal, k3s) |
| `kubectl` | any | In your PATH; kubeconfig pointing at the target cluster |
| `helm` | 3.x | For in-cluster install only |
| Vault (HashiCorp) | 1.x KV-v2 | For the L1 Vault probe only |
| Go toolchain | 1.22 | Only if building from source |

> **For the zero-trust offline mode (section 3) you need none of the
> above.** Just the `cha` binary and a `kubectl get -o json` output.

---

## 2. Install the binary

### Download a release binary (recommended)

```sh
# macOS arm64
curl -sSL https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/releases/latest/download/cluster-health-autopilot_$(uname -s)_$(uname -m).tar.gz \
  | tar xz && mv cha /usr/local/bin/

# Verify
cha version
```

Pre-built binaries are published for:
- `darwin/amd64`, `darwin/arm64`
- `linux/amd64`, `linux/arm64`

Each release includes a `checksums.txt`. Verify before running on production:

```sh
sha256sum -c checksums.txt --ignore-missing
```

### Build from source

```sh
git clone https://github.com/Bionic-AI-Solutions/cluster-health-autopilot.git
cd cluster-health-autopilot
go build -o cha ./cmd/cha
```

### Container image

```sh
docker pull ghcr.io/bionic-ai-solutions/cluster-health-autopilot:latest
# Or pin to a specific version:
docker pull ghcr.io/bionic-ai-solutions/cluster-health-autopilot:0.5.0
```

---

## 3. Zero-trust offline mode

No Kubernetes access needed. Capture a snapshot from any machine with
`kubectl`, then diagnose from any machine.

### Step 1 — capture a snapshot

```sh
# From a machine with kubectl + cluster access:
cha snapshot capture --tar my-cluster.tgz
```

This runs `kubectl get -o json` for each supported GVR (pods, events,
deployments, replicasets, statefulsets, jobs, cronjobs, nodes, pvcs,
externalsecrets, clusters.postgresql.cnpg.io, cephclusters.ceph.rook.io,
secrets — read-only, never writes). Output is a tarball you can share.

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

### Step 1 — add the Helm repo (once published)

```sh
helm repo add cha https://bionic-ai-solutions.github.io/cluster-health-autopilot
helm repo update
```

> Until the repo is published at launch, install from the local chart:
> ```sh
> helm install cha ./charts/cluster-health-autopilot \
>   --namespace cluster-health-autopilot --create-namespace
> ```

### Step 2 — create the Slack secret (optional; see section 5)

```sh
kubectl create secret generic cha-slack-webhook \
  --namespace cluster-health-autopilot \
  --from-literal=url=https://hooks.slack.com/services/T.../B.../...
```

### Step 3 — install

```sh
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --create-namespace \
  --set slackWebhookSecretName=cha-slack-webhook
```

This creates:
- `ServiceAccount/cha` in the `cluster-health-autopilot` namespace
- `ClusterRole/cha-reader` — read-only (`get`, `list`, `watch`) on: pods,
  nodes, pvcs, events, namespaces, deployments, replicasets, statefulsets,
  daemonsets, jobs, cronjobs, externalsecrets, secretstores,
  clustersecretstores, secrets, clusters (CNPG), cephclusters,
  certificates, certificaterequests, orders (ACME), driftreports
- `ClusterRole/cha-remediator` — narrowly-scoped write access for the
  whitelisted fixers: delete pods/jobs, patch deployment annotations,
  delete cert-manager CertificateRequests and ACME Orders
- `ClusterRole/cha-driftreport-writer` — create/patch/delete DriftReport
  CRs only
- `CronJob/cha-diagnose` — runs `cha diagnose --live` on a schedule
  (default: `0 6 * * *` — 06:00 UTC daily)
- `CronJob/cha-remediate` — opt-in; disabled by default
  (`values.yaml`: `remediate.enabled: false`)

### Verify

```sh
kubectl -n cluster-health-autopilot get cronjobs
kubectl -n cluster-health-autopilot create job --from=cronjob/cha-diagnose test-run
kubectl -n cluster-health-autopilot logs -l job-name=test-run --follow
```

### Key Helm values

```yaml
# charts/cluster-health-autopilot/values.yaml (abbreviated)

diagnose:
  schedule: "0 6 * * *"   # cron schedule for cha diagnose

remediate:
  enabled: false            # opt-in; set true to also run fixers
  schedule: "30 6 * * *"

slackWebhookSecretName: ""  # K8s secret holding the Slack URL

image:
  repository: ghcr.io/bionic-ai-solutions/cluster-health-autopilot
  tag: ""  # defaults to Chart.appVersion

vaultProbe:
  enabled: false            # see section 6
```

---

## 5. Slack alerts

`cha` posts a formatted Slack message at the end of each run. It mirrors
the attachment shape of the in-cluster bash version (component status
blocks + diagnostics section).

### Option A — static webhook URL (simpler)

```sh
kubectl create secret generic cha-slack-webhook \
  --namespace cluster-health-autopilot \
  --from-literal=url=https://hooks.slack.com/services/T.../B.../...

helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set slackWebhookSecretName=cha-slack-webhook
```

### Option B — ExternalSecret from Vault (recommended for production)

Store the webhook URL in Vault at `secret/cha/config` under key
`slack_webhook_url`, then create an ExternalSecret:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: cha-slack-webhook
  namespace: cluster-health-autopilot
spec:
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: cha-slack-webhook
  data:
    - secretKey: url
      remoteRef:
        key: secret/cha/config
        property: slack_webhook_url
```

The Helm chart will pick up the secret automatically if
`slackWebhookSecretName: cha-slack-webhook` is set.

---

## 6. Vault probe

The Vault probe closes the **L1 stale-Ready window**: it detects Vault
path/key drift BEFORE the ESO controller's next refresh cycle, while the
ExternalSecret is still reporting `Ready=True`.

### Prerequisites

- Vault KV-v2 mount (default: `secret`)
- A Vault role bound to the `cha` ServiceAccount via kubernetes auth

### Step 1 — create the Vault kubernetes-auth role

```sh
# In Vault:
vault write auth/kubernetes/role/cha-diagnose \
  bound_service_account_names=cha \
  bound_service_account_namespaces=cluster-health-autopilot \
  policies=cha-diagnose-read \
  ttl=1h
```

Create the policy (`cha-diagnose-read`):

```hcl
# Allow read on all KV-v2 paths referenced by ExternalSecrets in the cluster.
# Scope this to only the paths actually used — don't grant wildcard read.
path "secret/data/team/*" {
  capabilities = ["read"]
}
path "secret/data/shared/*" {
  capabilities = ["read"]
}
```

> **Security note:** The `cha` SA token grants read on every Vault path it
> can query. A malicious operator with `kubectl edit externalsecret` could
> add a path to an ESO spec and have `cha` query it, learning whether the
> path exists and what keys it contains (key NAMES only — byte values are
> never returned or logged). Scope the Vault policy to the paths used by
> your ExternalSecrets and no broader. See `docs/ADVERSARIAL_ANALYSIS_v0.2.md §2.2`.

### Step 2 — enable vaultProbe in Helm

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --set vaultProbe.enabled=true \
  --set vaultProbe.address=https://vault.example.com \
  --set vaultProbe.kvMount=secret \
  --set vaultProbe.auth.role=cha-diagnose
```

Or in `values.yaml`:

```yaml
vaultProbe:
  enabled: true
  address: https://vault.example.com
  kvMount: secret     # KV-v2 mount path
  auth:
    method: kubernetes
    role: cha-diagnose
```

### Step 3 — verify

```sh
kubectl -n cluster-health-autopilot create job --from=cronjob/cha-diagnose vault-test
kubectl -n cluster-health-autopilot logs -l job-name=vault-test --follow
# Look for "VaultPathMissing: X paths checked, Y missing" in the output.
```

### Token auth (alternative — development only)

For local testing without kubernetes auth:

```sh
kubectl create secret generic cha-vault-token \
  --namespace cluster-health-autopilot \
  --from-literal=token=hvs.XXXXXX

helm upgrade cha ... \
  --set vaultProbe.enabled=true \
  --set vaultProbe.address=https://vault.example.com \
  --set vaultProbe.auth.method=token \
  --set vaultProbe.auth.tokenSecretRef.name=cha-vault-token \
  --set vaultProbe.auth.tokenSecretRef.key=token
```

> **Do not use token auth in production.** The kubernetes auth method uses
> the in-cluster SA JWT (rotates with the pod, never sits in env vars).

---

## 7. ESO / Vault Provisioning Analyzers

Three analyzers work together to detect the full class of "Secret not reachable" failures — each adding a layer of detail to the diagnostic.

### UnprovisionedSecret (new in v0.9.1)

Walks every Deployment, StatefulSet, and CronJob and checks whether each Secret
referenced via `envFrom.secretRef` or `volumes.secret.secretName` either:
- exists in the cluster, **or**
- has an ExternalSecret (ESO) configured to provision it.

When neither is true, the diagnostic emits:

```
Secret `playground/playground-agent-secrets` referenced by Deployment/playground-agent
has no ExternalSecret provisioning it. Create an ExternalSecret with
spec.target.name=playground-agent-secrets pointing to Vault path
`secret/t6-apps/playground/config`.
```

The suggested t6 path is derived from the workload's namespace (`secret/t6-apps/<namespace>/config`),
which is the canonical hierarchy in this cluster. If your Vault layout differs, use the suggestion
as a starting point and adjust the path.

**Works in both live and snapshot mode** — in snapshot mode without Secrets captured, the analyzer
falls back to ESO-coverage-only: any workload with no backing ESO is reported regardless of whether
the Secret exists on-cluster.

### ProactiveSecretKeyCheck near-miss hint (enhanced in v0.9.1)

When a `secretKeyRef` references a key that is absent from the Secret, CHA now checks whether a
case/format variant of that key exists. If so, it appends:

```
Key `GITHUB_TOKEN` is a case/format variant — possible naming mismatch.
```

Normalization: lowercase + replace `-` and `.` with `_`. This catches the common pattern where
a Deployment manifest uses `github-token` and the ESO template produces `GITHUB_TOKEN`.

### FailingExternalSecrets t6 path hint (enhanced in v0.9.1)

When an ESO reports `Ready: False`, CHA now checks whether its `spec.data[].remoteRef.key`
(or `spec.dataFrom[].extract.key`) starts with `t6-apps/`. If not, it appends:

```
Vault path `counsellor/config` does not follow t6 hierarchy;
expected: `secret/t6-apps/livekit-agents/config`.
```

This immediately distinguishes "wrong Vault path layout" from "correct path, missing property".

### None of these are auto-fixable

All three are **diagnostic-only**. The underlying issues require human action:
- Creating a Vault KV path and seeding it with the required keys
- Creating an ExternalSecret resource pointing to that path
- Patching a Deployment manifest to use the correct key name

CHA surfaces the exact action needed so the fix takes minutes, not a support ticket.

---

## 8. DriftReport CRD (kubectl-queryable diagnostics)

`cha` creates and maintains `DriftReport` cluster-scoped CRs — one per
active issue. This gives `kubectl` users a live view of the cluster's health
without reading logs.

### Install the CRD

The CRD is installed automatically by the Helm chart. To install manually:

```sh
kubectl apply -f charts/cluster-health-autopilot/crds/driftreports.yaml
```

### Query diagnostics with kubectl

```sh
# All active issues:
kubectl get driftreports

# Watch live:
kubectl get driftreports --watch

# Full detail on one issue:
kubectl describe driftreport <name>

# All critical issues:
kubectl get driftreports -o json | jq '.items[] | select(.spec.severity=="critical")'
```

### DriftReport fields

```yaml
spec:
  subject:      "missing-key/billing/billing-svc-secrets/STRIPE_API_KEY"
  severity:     "critical"
  source:       "analyzer:SecretKeyMissing"
  message:      "Secret `billing/billing-svc-secrets` is missing key ..."
  remediation:  "Add the key to the ExternalSecret template ..."
  category:     "secret-drift"
  resourceRef:
    kind:       "Secret"
    namespace:  "billing"
    name:       "billing-svc-secrets"
status:
  firstObserved:    "2026-05-01T06:00:00Z"
  lastObserved:     "2026-05-04T06:00:00Z"
  observationCount: 4
  runID:            "20260504-060001"
```

### Cleanup after helm uninstall

The CRD has `helm.sh/resource-policy: keep` so `helm uninstall` does NOT
remove it (this preserves your history). To remove completely:

```sh
kubectl delete crd driftreports.cha.bionicaisolutions.com
# This also deletes all DriftReport CRs.
```

---

## 9. Watcher mode

Watcher mode deploys a long-running `Deployment` (instead of a CronJob) that
holds open Kubernetes watches for all resource types CHA analyzes. When any
resource changes, a short debounce fires, then the full probe+analyzer stack
runs — exactly like `cha diagnose --live` but in under 15 seconds instead of
waiting for the next cron tick.

**Slack deduplication**: each diagnostic is fingerprinted by subject + severity
+ message. Slack only posts when a diagnostic is new, changes, or resolves. The
seen-map is seeded from existing DriftReport CRs on pod startup to avoid a Slack
flood after a rolling update.

### Enable via Helm

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.enabled=true
```

With Slack:

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set watcher.enabled=true \
  --set slack.enabled=true \
  --set slack.webhookSecretName=cha-slack-webhook
```

With auto-remediation (fixers run after each cycle, post-fix state reported):

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
  debounce: 10s            # Wait this long after the last event before running diagnostics
  resyncPeriod: 10m        # Full re-diagnose regardless of events (catches non-event drift)
  slack:
    postOnResolved: true   # Post when a diagnostic disappears
    repeatInterval: 4h     # Re-post still-active issues at this cadence (0 = never)
  remedy:
    enabled: false         # Run auto-fixers after each diagnose cycle
    dryRun: false          # Evaluate fixers without cluster mutation
  resources:               # Resource requests/limits for the watcher pod
    limits:
      cpu: 500m
      memory: 256Mi
    requests:
      cpu: 50m
      memory: 64Mi
```

### Run manually (without Helm)

```sh
cha watch --live \
  --debounce=10s \
  --resync-period=10m \
  --slack-webhook=$(SLACK_WEBHOOK_URL) \
  --slack-post-on-resolved=true \
  --slack-repeat-interval=4h \
  --write-driftreports=true

# With remediation:
cha watch --live --remedy --slack-webhook=$(SLACK_WEBHOOK_URL)

# All flags:
cha watch --help
```

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `--live` | — | Required; runs against the live cluster |
| `--kubeconfig` | in-cluster / `$KUBECONFIG` | Path to kubeconfig |
| `--debounce` | `10s` | Debounce window after a Kubernetes event |
| `--resync-period` | `10m` | Periodic full re-diagnose interval |
| `--slack-webhook` | — | Slack webhook URL; omit to disable Slack |
| `--slack-post-on-resolved` | `true` | Post when a diagnostic resolves |
| `--slack-repeat-interval` | `4h` | Re-post still-active diagnostics; `0` disables |
| `--write-driftreports` | `true` | Upsert DriftReport CRs on every cycle |
| `--remedy` | `false` | Run auto-fixers after each diagnose cycle |
| `--dry-run` | `false` | With `--remedy`: evaluate without mutating |
| `--vault-addr` | `$VAULT_ADDR` | Vault endpoint (enables VaultPathMissing analyzer) |
| `--vault-kv-mount` | `secret` | Vault KV-v2 mount path |
| `--vault-k8s-role` | `$VAULT_K8S_ROLE` | Vault kubernetes-auth role |

### RBAC note

The chart's `clusterrole-reader` already includes `watch` verb on all monitored
resources (added in v0.9.0). If you are upgrading from an earlier version, run
`helm upgrade` to apply the updated ClusterRole — otherwise the watcher will
fail to open watches and fall back to no-op reconnect loops.

```sh
kubectl get clusterrole <release>-reader -o yaml | grep -A2 "verbs:"
# Should include: ["get", "list", "watch"]
```

### Coexistence with the diagnose CronJob

Running both the watcher Deployment and the daily diagnose CronJob is safe:
- The CronJob provides a full-state snapshot each day (WS-C evidence, anonymized JSONL).
- The watcher provides sub-minute reactive alerting.
- Both write to DriftReport CRs; the upsert logic is idempotent (update wins).

---

## 10. Run-log publishing pipeline

WS-C publishes anonymized daily run logs to `runs/` in the public repo so
design partners and the community can see the analyzer's track record over
time.

**MinIO is NOT a hard dependency for core CHA.** The diagnostic engine,
Vault probe, auto-fixers, and DriftReport CRD all work without object storage.
MinIO is only needed for Mode B of this pipeline.

### Choose your mode

| | **Mode A — Private cluster** | **Mode B — Public cluster** |
|---|---|---|
| **Runner** | Self-hosted, inside the cluster | `ubuntu-latest` (GitHub cloud) |
| **MinIO** | Not needed | Required (must be internet-reachable) |
| **GH Actions secrets** | None | `VAULT_TOKEN` only |
| **How it works** | Runner calls `cha diagnose --live` directly | Runner fetches Vault creds, pulls JSON runs from MinIO |
| **Use when** | Cluster has no public IP / private network | Cluster exposes MinIO publicly (e.g. cloud S3) |

**This cluster uses Mode A** (bare-metal, private IP `192.168.0.x`).

Switch modes by toggling the `if:` lines in
`.github/workflows/publish-runs.yml`:

```yaml
# Mode A active (current):
publish-self-hosted:
  if: true
publish-minio:
  if: false

# To switch to Mode B:
publish-self-hosted:
  if: false
publish-minio:
  if: true
```

---

### Mode A — Private cluster setup (self-hosted runner)

The nightly workflow runs inside the cluster as a Kubernetes pod. No MinIO,
no external secrets — the runner has direct in-cluster API access via the
`cha` ServiceAccount.

#### Architecture

```
GitHub Actions
  └─ dispatches job to runner pod (self-hosted,cluster)
       └─ runs inside cluster-health-autopilot namespace
            ├─ go build ./cmd/cha              (build fresh binary)
            ├─ ./cha diagnose --live           (in-cluster API calls via cha SA)
            ├─ ./cha anonymize                 (SHA-256 hashing)
            └─ git commit + push runs/*.jsonl  (GitHub PAT from Vault)
```

#### Prerequisites

- ESO ClusterSecretStore `vault-backend` configured (see §6).
- GitHub PAT stored in Vault (already done — see below).
- `runner.enabled: true` in Helm values.

#### Vault secret (already populated)

`secret/t6-apps/cha/config` holds:

```
github_pat        → GitHub OAuth/PAT token (repo scope)
minio_*           → MinIO service account (Mode B only; ignored in Mode A)
```

To update or rotate the PAT:

```sh
vault kv patch secret/t6-apps/cha/config \
  github_pat="ghp_YOURNEWTOKENHERE"
```

#### Step A-1 — enable the runner via Helm

```sh
helm upgrade cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot \
  --reuse-values \
  --set runner.enabled=true \
  --set runner.repoUrl=https://github.com/Bionic-AI-Solutions/cluster-health-autopilot \
  --set runner.labels=self-hosted,cluster \
  --set runner.name=cha-cluster-runner
```

Or add to `values.yaml`:

```yaml
runner:
  enabled: true
  repoUrl: "https://github.com/Bionic-AI-Solutions/cluster-health-autopilot"
  labels: "self-hosted,cluster"
  name: "cha-cluster-runner"
```

This creates:
- **ExternalSecret** `cha-runner-token` — pulls `github_pat` from Vault
- **Deployment** `cha-runner` — `myoung34/github-runner:ubuntu-22.04` pod
  using the `cha` ServiceAccount for in-cluster API access

#### Step A-2 — verify runner is registered

```sh
kubectl -n cluster-health-autopilot get deployment cha-cluster-health-autopilot-runner
kubectl -n cluster-health-autopilot logs -l app.kubernetes.io/component=runner | tail -10
# Look for: "Runner successfully added" and "Listening for Jobs"
```

Also verify in GitHub: **Settings → Actions → Runners** — the runner
`cha-cluster-runner` should appear as **Idle**.

#### Step A-3 — trigger a first manual run

```sh
gh workflow run publish-runs.yml --field date=$(date -u -d 'yesterday' +%Y-%m-%d)
# Check result:
gh run list --workflow=publish-runs.yml --limit=1
git pull && ls runs/
```

---

### Mode B — Public cluster setup (GitHub-hosted runner + MinIO)

Use this mode when the cluster exposes MinIO at a public internet address
(e.g. AWS S3, GCS, or an on-prem MinIO behind a real public IP — **not** a
private IP like `192.168.0.x`, which GitHub runners cannot reach).

#### Prerequisites

- MinIO accessible at a public URL from GitHub runner IPs.
- `vault.baisoln.com` reachable from the internet (already true).
- `VAULT_TOKEN` set as a GH Actions secret (already done).

#### Vault secret

`secret/t6-apps/cha/config` must have (already populated; update endpoint
if your public MinIO URL changes):

```sh
vault kv patch secret/t6-apps/cha/config \
  minio_endpoint="https://YOUR_PUBLIC_S3_ENDPOINT"
```

> For this cluster: the in-cluster MinIO at `192.168.0.x` is **not** reachable
> from GitHub runners. Mode B would require a Cloudflare Tunnel or cloud S3.

#### Vault policy (already created)

```hcl
# policy: cha-publish-runs
path "secret/data/t6-apps/cha/config" {
  capabilities = ["read"]
}
```

Recreate if needed:

```sh
vault policy write cha-publish-runs /tmp/cha-policy.hcl
```

#### Step B-1 — mint a GH Actions token

```sh
vault token create \
  -policy=cha-publish-runs \
  -ttl=87600h \
  -display-name="cha-gh-actions" \
  -no-default-policy
# Add the printed token to: Settings → Secrets → Actions → VAULT_TOKEN
```

#### Step B-2 — wire the CronJob to write JSON to MinIO

The CronJob must upload each run to `cha-runs/runs/YYYY-MM-DD/` before
02:00 UTC when the nightly action fires. Override the diagnose command in
`values.yaml`:

```yaml
diagnose:
  command:
    - /bin/sh
    - -c
    - |
      cha diagnose --live --format json > /tmp/run.json
      mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
      mc cp /tmp/run.json minio/${MINIO_BUCKET}/runs/$(date +%Y-%m-%d)/run-$(date +%H%M%S).json
  envFrom:
    - secretRef:
        name: cha-minio-creds
```

Supply the env vars via Vault → ExternalSecret (already in Vault):

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: cha-minio-creds
  namespace: cluster-health-autopilot
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: cha-minio-creds
  data:
    - secretKey: MINIO_ENDPOINT
      remoteRef: { key: t6-apps/cha/config, property: minio_endpoint }
    - secretKey: MINIO_ACCESS_KEY
      remoteRef: { key: t6-apps/cha/config, property: minio_access_key }
    - secretKey: MINIO_SECRET_KEY
      remoteRef: { key: t6-apps/cha/config, property: minio_secret_key }
    - secretKey: MINIO_BUCKET
      remoteRef: { key: t6-apps/cha/config, property: minio_bucket }
```

---

### What the workflow does (both modes)

`.github/workflows/publish-runs.yml` fires at 02:00 UTC daily:

| Step | Mode A | Mode B |
|---|---|---|
| Get run data | `cha diagnose --live --format json` (in-cluster) | Pull JSON files from MinIO |
| Anonymize | `cha anonymize --run-id ... --timestamp ...` | Same |
| Append | `runs/YYYY-MM-DD.jsonl` | Same |
| Summarize | `cha summarize runs/` → `runs/SUMMARY.md` | Same |
| Commit | `git commit + push` | Same |

No secrets appear in committed files — only deterministic SHA-256 hashes.

### Link SUMMARY.md from the README

```markdown
## Live run data

[`runs/SUMMARY.md`](runs/SUMMARY.md) — anonymized daily diagnostics from
the production cluster, updated nightly.
```

### Trigger manually (first run)

```sh
gh workflow run publish-runs.yml --field date=2026-05-04
```

---

## 11. Verification checklist

Work through this after completing each section.

### Zero-trust mode

```sh
cha diagnose --snapshot examples/sample-cluster
# Expected: 9 diagnostics spanning secret drift, unprovisioned Secrets,
#           key-name mismatches, image pull auth, and cert expiry.
```

### In-cluster mode

```sh
# Trigger a one-off run:
kubectl -n cluster-health-autopilot create job --from=cronjob/cha-diagnose verify-$(date +%s)
kubectl -n cluster-health-autopilot wait --for=condition=complete job/verify-...
kubectl -n cluster-health-autopilot logs -l job-name=verify-... | tail -20
```

### Watcher mode

```sh
kubectl get deployment -n cluster-health-autopilot | grep watcher
kubectl logs -f deployment/cha-cluster-health-autopilot-watcher \
  -n cluster-health-autopilot
# Expected: "watcher: initial diagnose cycle" then idle log lines
```

### DriftReport CRD

```sh
kubectl get crd driftreports.cha.bionicaisolutions.com
kubectl get driftreports   # empty if cluster is healthy
```

### Vault probe

```sh
# Temporarily rename a Vault path your ESO references, trigger a run,
# then check the output for a "missing-vault-path" diagnostic.
# Restore the path afterward.
```

### Slack

Check the configured Slack channel for the formatted summary attachment
after a job run. For the watcher, inject a failure (see DEMO_GUIDE §5.3)
and verify a deduped message arrives within ~15 seconds.

### Run-log pipeline

```sh
gh workflow run publish-runs.yml
# Wait ~2 min, then:
git pull && ls runs/
```

---

## 12. Troubleshooting

### `cha diagnose --live` returns no results

- Check kubeconfig: `kubectl get pods -A` should work first.
- Check RBAC: `kubectl auth can-i list pods --as=system:serviceaccount:cluster-health-autopilot:cha`

### DriftReports not created

- Ensure `--write-driftreports=true` (default) is not overridden.
- Ensure the CRD is installed: `kubectl get crd driftreports.cha.bionicaisolutions.com`
- Check logs for `driftreport reconcile partial failure`.

### VaultPathMissing emitting false positives on non-Vault stores

- The `clusterrole-reader` must include `secretstores`/`clustersecretstores` RBAC.
- Run `helm upgrade` to apply the updated ClusterRole from v0.3+.
- Look for `vault-store-rbac-missing` in the diagnostics output — this means
  the ClusterRole needs updating.

### Vault authentication failures (403 in logs)

- Check that the Vault role TTL is ≥ the CronJob schedule (e.g. TTL=1h for
  an hourly cron). See `docs/ADVERSARIAL_ANALYSIS_v0.2.md §5.2`.
- Verify the kubernetes auth mount path: default is `auth/kubernetes`.
- Check that `bound_service_account_names` and
  `bound_service_account_namespaces` match exactly.

### `helm install` fails with "vaultProbe.auth.role is required"

- Set `--set vaultProbe.auth.role=<your-vault-role>` or add it to
  `values.yaml` before enabling `vaultProbe.enabled=true`.

### Watcher pod crash-loops with `watch requires --live`

- The Helm template always injects `--live`; this error means you are running
  `cha watch` manually without the flag. Add `--live`.

### Watcher Slack posts firing on every resync

- The repeat interval defaults to `4h`. Set `watcher.slack.repeatInterval: 0`
  in values to disable repeat posts entirely, or increase the interval.

### Watcher Slack floods Slack after pod restart

- This happens when DriftReport CRD is absent — the watcher cannot seed its
  seen-map. Install the CRD first:
  ```sh
  kubectl apply -f charts/cluster-health-autopilot/templates/crd-driftreport.yaml
  ```

### Watcher `watch … no matches for kind` log lines

- Normal for CRDs not installed in this cluster (e.g. CNPG absent on a plain
  cluster). The watcher exits the goroutine for that GVR silently and continues
  watching all others.

### Watcher RBAC: `cannot watch pods`

- The `watch` verb was added to `clusterrole-reader` in v0.9.0. If you
  installed an earlier chart, run `helm upgrade` to apply the updated role:
  ```sh
  helm upgrade cha cha/cluster-health-autopilot \
    --namespace cluster-health-autopilot --reuse-values
  ```

### `path contains traversal component` error in Vault logs

- An ESO has `remoteRef.key` containing `..` or `.` path segments.
- This is rejected by the Vault client as a security measure (v0.4+).
- Fix the ExternalSecret spec to use a clean path.

### Run-log pipeline: "No run files found for DATE" (Mode B only)

- The in-cluster CronJob must be writing JSON to the `cha-runs` MinIO bucket
  before the 02:00 UTC GH Action fires. Check the CronJob schedule and:
  ```sh
  mc ls minio/cha-runs/runs/YYYY-MM-DD/
  ```
- Verify the `cha-minio-creds` ExternalSecret is synced:
  ```sh
  kubectl get externalsecret cha-minio-creds -n cluster-health-autopilot
  ```
- Manually trigger: `gh workflow run publish-runs.yml --field date=YYYY-MM-DD`

### Run-log pipeline: Vault credentials fetch fails in GH Action

- Verify `VAULT_TOKEN` is set in Settings → Secrets and variables → Actions.
- The token must have the `cha-publish-runs` policy. Check:
  ```sh
  vault token lookup <token>
  # "policies" should include "cha-publish-runs"
  ```
- `vault.baisoln.com` must be reachable from GitHub's runner network (it is
  public; verify with `curl https://vault.baisoln.com/v1/sys/health`).
