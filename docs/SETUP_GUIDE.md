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
7. [DriftReport CRD (kubectl-queryable diagnostics)](#7-driftreport-crd)
8. [Run-log publishing pipeline (WS-C)](#8-run-log-publishing-pipeline)
9. [Verification checklist](#9-verification-checklist)
10. [Troubleshooting](#10-troubleshooting)

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
- `ClusterRole/cha-reader` — read-only access to the resource set the
  probes and analyzers need (pods, nodes, pvcs, events, deployments,
  replicasets, statefulsets, jobs, cronjobs, externalsecrets,
  secretstores, clustersecretstores, secrets, clusters, cephclusters,
  driftreports)
- `ClusterRole/cha-remediator` — narrowly-scoped write access for the
  three whitelisted fixers (delete pods/jobs, patch deployment annotations)
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

## 7. DriftReport CRD

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

## 8. Run-log publishing pipeline

WS-C publishes anonymized daily run logs to `runs/` in the public repo so
design partners and the community can see the analyzer's track record over
time.

**MinIO is NOT a hard dependency for the core CHA system.** The diagnostic
engine, Vault probe, auto-fixers, and DriftReport CRD all work without any
object storage. MinIO is only used by this pipeline (WS-C) as intermediate
storage between the in-cluster CronJob and the GitHub Actions runner.

### Pipeline modes

Two modes are available in `.github/workflows/publish-runs.yml`:

| Mode | Runner | MinIO needed? | When to use |
|---|---|---|---|
| **A — self-hosted** | Your own runner inside the cluster | No | Cleanest; runner can call `cha diagnose --live` directly |
| **B — GitHub-hosted** | `ubuntu-latest` (GitHub cloud) | Yes | Use when no in-cluster runner is available |

Switch modes by editing the `if:` condition on each job in the workflow file.

---

### Mode A — self-hosted runner (no MinIO)

The runner executes `cha diagnose --live` directly inside the cluster.
No intermediate object storage required.

**Step A-1 — register a self-hosted runner in the cluster**

```sh
# In GitHub: Settings → Actions → Runners → New self-hosted runner
# Follow the instructions to download and configure the runner binary.
# Label it with: self-hosted,cluster
```

**Step A-2 — enable Mode A in the workflow**

In `.github/workflows/publish-runs.yml`, change the `if:` on each job:

```yaml
publish-self-hosted:
  if: true   # was: false
  ...
publish-minio:
  if: false  # was: true
```

No GitHub Actions secrets are required for Mode A.

---

### Mode B — GitHub-hosted runner + MinIO (creds from Vault)

All MinIO credentials are stored in Vault and fetched at runtime by the
`hashicorp/vault-action` step. Only **one** GitHub Actions secret is needed:
`VAULT_TOKEN`.

#### Vault secret (already created)

The secret at `secret/t6-apps/cha/config` was created with:

```sh
vault kv put secret/t6-apps/cha/config \
  minio_endpoint="http://minio-tenant-hl.minio.svc.cluster.local:9000" \
  minio_access_key="<CHA_SERVICE_ACCOUNT_KEY>" \
  minio_secret_key="<CHA_SERVICE_ACCOUNT_SECRET>" \
  minio_bucket="cha-runs"
```

The Vault address is `https://vault.baisoln.com` (publicly reachable from
GitHub-hosted runners).

#### Vault policy (already created)

Policy `cha-publish-runs` is already applied:

```hcl
path "secret/data/t6-apps/cha/config" {
  capabilities = ["read"]
}
```

To update or recreate it:

```sh
cat > /tmp/cha-policy.hcl << 'EOF'
path "secret/data/t6-apps/cha/config" {
  capabilities = ["read"]
}
EOF
vault policy write cha-publish-runs /tmp/cha-policy.hcl
```

#### Step B-1 — mint a token for GH Actions

Create a token bound to the `cha-publish-runs` policy:

```sh
vault token create \
  -policy=cha-publish-runs \
  -ttl=87600h \
  -display-name="cha-gh-actions" \
  -no-default-policy
# Copy the token value.
```

#### Step B-2 — add the single GH Actions secret

In the repo's **Settings → Secrets and variables → Actions**, add:

| Secret | Value |
|---|---|
| `VAULT_TOKEN` | token from Step B-1 |

That is the only secret needed. All MinIO credentials are fetched from Vault
at workflow runtime — they are never stored in GitHub.

#### Step B-3 — configure the in-cluster CronJob to write JSON runs

Override the diagnose command in `values.yaml` so the CronJob writes each
run as JSON to the `cha-runs` MinIO bucket before the 02:00 UTC GH Action
fires:

```yaml
diagnose:
  command:
    - /bin/sh
    - -c
    - |
      cha diagnose --live --format json > /tmp/run.json
      mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
      mc cp /tmp/run.json minio/${MINIO_BUCKET}/runs/$(date +%Y-%m-%d)/run-$(date +%H%M%S).json
```

Supply `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, and
`MINIO_BUCKET` to the CronJob pod via Vault → ExternalSecret — **not**
plaintext in `values.yaml`:

```yaml
# ExternalSecret for the CronJob's MinIO env vars
apiVersion: external-secrets.io/v1beta1
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

Then reference it in `values.yaml`:

```yaml
diagnose:
  envFrom:
    - secretRef:
        name: cha-minio-creds
```

---

### What the nightly workflow does (both modes)

The workflow (`.github/workflows/publish-runs.yml`) fires at 02:00 UTC daily
after the CronJob has run:

1. **Mode A**: runs `cha diagnose --live --format json | cha anonymize` directly
   **Mode B**: pulls the day's JSON run(s) from MinIO, runs `cha anonymize` on each
2. Appends anonymized JSONL to `runs/YYYY-MM-DD.jsonl`
3. Runs `cha summarize runs/` to regenerate `runs/SUMMARY.md`
4. Commits and pushes

No secrets appear in the committed files — only deterministic SHA-256 hashes.

### Step — link SUMMARY.md from README

Add to your README:

```markdown
## Live run data

[`runs/SUMMARY.md`](runs/SUMMARY.md) — anonymized daily diagnostics from
the production cluster, updated nightly. Demonstrates false-positive-free
detection over X+ days.
```

### Trigger manually (first run)

```sh
gh workflow run publish-runs.yml --field date=2026-05-04
```

---

## 9. Verification checklist

Work through this after completing each section.

### Zero-trust mode

```sh
cha diagnose --snapshot examples/sample-cluster
# Expected: 3 diagnostics (see README demo output)
```

### In-cluster mode

```sh
# Trigger a one-off run:
kubectl -n cluster-health-autopilot create job --from=cronjob/cha-diagnose verify-$(date +%s)
kubectl -n cluster-health-autopilot wait --for=condition=complete job/verify-...
kubectl -n cluster-health-autopilot logs -l job-name=verify-... | tail -20
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
after a job run.

### Run-log pipeline

```sh
gh workflow run publish-runs.yml
# Wait ~2 min, then:
git pull && ls runs/
```

---

## 10. Troubleshooting

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
