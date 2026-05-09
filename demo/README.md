# CHA Demo Guide

**Cluster:** test-cluster1 (ap-south-1, EKS)  
**CHA version:** 0.9.2 (watcher with pre-fix Slack diff fix)  
**Slack channel:** #aws-alerts

---

## Prerequisites

### Step 1 — Connect to the remote AWS cluster

All scripts target whichever cluster the `KUBE_CONTEXT` environment variable points to.
Set it once before running anything:

```bash
# Option A: Add the EKS cluster to your kubeconfig (requires AWS CLI + IAM access)
aws eks update-kubeconfig --name test-cluster1 --region ap-south-1

# Find the context name just added
kubectl config get-contexts
# Copy the context name (looks like: arn:aws:eks:ap-south-1:ACCOUNT:cluster/test-cluster1)

# Export it for the demo session
export KUBE_CONTEXT="arn:aws:eks:ap-south-1:ACCOUNT:cluster/test-cluster1"

# Option B: If kubectl is already configured for the remote cluster, just export it
export KUBE_CONTEXT=$(kubectl config current-context)

# Verify you're on the right cluster
kubectl --context "$KUBE_CONTEXT" get nodes
# Expected: 3 EKS nodes (ap-south-1a/b/c)
```

> **Note:** scripts default to `kubectl config current-context` when `KUBE_CONTEXT` is unset,
> which is the **local k3s cluster** on this machine — not the AWS cluster. Always set
> `KUBE_CONTEXT` explicitly for the demo.

### Step 2 — Verify CHA is running

```bash
kubectl --context "$KUBE_CONTEXT" get pods -n cha
# Expected: cha-watcher-... Running  (with --remedy flag)
```

### Step 3 — Open Slack #aws-alerts in browser
Webhook is stored in Vault at `secret/t6-apps/cha/config` → `aws_slack_webhook`

### Step 4 — Make scripts executable (one-time)
```bash
chmod +x demo/simulate/*.sh demo/fix-scripts/*.sh demo/run-demo.sh
```

---

## Run the Full Demo (Scripted)

```bash
cd /home/skadam/cluster-health-autopilot
export KUBE_CONTEXT="arn:aws:eks:ap-south-1:ACCOUNT:cluster/test-cluster1"
bash demo/run-demo.sh
```

The script pauses at each section for narration. Press ENTER to advance.

---

## Run Individual Demos

### 1. Stale Pod Auto-Fix (< 30 seconds end-to-end)
```bash
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/01-stale-error-pods.sh demo-app
# Watch Slack for: 🔴 Active Issues → 🔧 Fixes Applied → ✅ Resolved
```

### 2. Missing Secret Key (Report + Manual Fix)
```bash
# Simulate failure
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/02-missing-secret-key.sh demo-app

# Watch Slack for: SecretKeyMissing alert with Deployment/ESO context
# Then fix it
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/fix-scripts/fix-missing-secret-key.sh \
  demo-app database-credentials DB_PASSWORD 'demo-pw'

# Watch Slack for: ✅ Resolved
```

### 3. Failing ExternalSecret
```bash
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/03-failing-externalsecret.sh demo-app
# Shows: FailingExternalSecrets alert with exact Vault property name
# Fix:
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/fix-scripts/fix-failing-externalsecret.sh demo-app demo-externalsecret
```

### 4. Stuck CronJob Auto-Fix
```bash
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/04-stuck-job-bad-secret.sh demo-app
# CHA: detects frozen Job → deletes it → CronJob unblocks
```

### 5. ImagePull Auth Failure
```bash
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/05-image-pull-auth-failure.sh demo-app
# Shows: ImagePullAuth alert with registry auth failure context
# Fix (get pod name first):
POD=$(kubectl --context "$KUBE_CONTEXT" get pods -n demo-app -l demo=image-pull-auth -o name | head -1 | cut -d/ -f2)
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/fix-scripts/fix-image-pull-secret.sh demo-app "$POD"
```

### Cleanup everything
```bash
KUBE_CONTEXT="$KUBE_CONTEXT" bash demo/simulate/06-cleanup-all.sh demo-app
```

---

## Key Talking Points

### "Why not just use Prometheus/PagerDuty?"
CHA doesn't alert on metrics thresholds — it reads Kubernetes object state and event text. It knows *which deployment* caused the failure, *which ExternalSecret* is broken, and *which Vault property* is missing. That context is in the Slack message, not in a generic alert that pages someone at 2am.

### "What makes it safe to auto-fix?"
Four whitelisted fixers, all with explicit safety gates:
- **StaleErrorPods**: only deletes `Failed`-phase pods with no controller that would restart them. Never touches Running pods.
- **StuckJobsWithBadSecretRef**: verifies parent CronJob exists before deleting the frozen Job. Never deletes standalone Jobs.
- **StuckCertificateRequests**: only deletes terminal-failure cert CRs. cert-manager immediately recreates — fully idempotent.
- **StuckRSPods**: triggers a rollout restart — same as `kubectl rollout restart`. Never if the root cause is a missing Secret key.

### "What about false positives?"
CHA uses fingerprint deduplication (`SHA-256(subject|severity|message)`) — a second pass with the same diagnostic won't re-post to Slack. On pod restart, DriftReports pre-populate the dedup state so there's no Slack flood after rollouts.

### "Does it need cluster-admin?"
No. Two ClusterRoles:
- **diagnose**: `get/list/watch` on the 14 resource types it probes.
- **remediate** (opt-in): adds `delete` for pods, jobs, cert CRs; `patch` for deployments.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                    │
│                                                         │
│  Watch API (14 GVRs)                                    │
│       │ 10s debounce                                    │
│       ▼                                                 │
│  ┌─────────┐    ┌────────────────────────────────────┐ │
│  │ Probes  │    │          Analyzers                  │ │
│  │ • Ceph  │    │ • SecretKeyMissing                  │ │
│  │ • Nodes │    │ • FailingExternalSecrets            │ │
│  │ • PG    │    │ • ProactiveSecretKeyCheck           │ │
│  │ • PVCs  │    │ • UnprovisionedSecret               │ │
│  │ • Svcs  │    │ • ImagePullAuth                     │ │
│  └────┬────┘    │ • CertExpiry                        │ │
│       │         │ • VaultPathMissing (paid)           │ │
│       └────┬────┘                                     │ │
│            │ pre-fix state captured here              │ │
│            ▼                                          │ │
│  ┌─────────────────┐                                  │ │
│  │     Fixers      │  (whitelist-only, opt-in)        │ │
│  │ • StaleErrorPods│                                  │ │
│  │ • StuckJobs     │                                  │ │
│  │ • StuckCertReqs │                                  │ │
│  │ • StuckRSPods   │                                  │ │
│  └────┬────────────┘                                  │ │
│       │ re-diagnose post-fix                         │ │
│       ▼                                               │ │
│  ┌────────────┐  ┌───────────────────┐               │ │
│  │ DriftReport│  │  Slack #aws-alerts│               │ │
│  │    CRDs    │  │  (deduplicated)   │               │ │
│  └────────────┘  └───────────────────┘               │ │
└─────────────────────────────────────────────────────────┘
```

---

## Files in This Demo Package

```
demo/
├── README.md                          ← this file
├── CAPABILITIES.md                    ← full capabilities reference with sample output
├── run-demo.sh                        ← master demo script (scripted walkthrough)
├── simulate/
│   ├── 01-stale-error-pods.sh         ← auto-fixed by CHA
│   ├── 02-missing-secret-key.sh       ← reported; manual fix required
│   ├── 03-failing-externalsecret.sh   ← reported; manual fix required
│   ├── 04-stuck-job-bad-secret.sh     ← auto-fixed by CHA
│   ├── 05-image-pull-auth-failure.sh  ← reported; manual fix required
│   └── 06-cleanup-all.sh              ← reset cluster after demo
└── fix-scripts/
    ├── fix-missing-secret-key.sh      ← patch Secret + restart Deployment
    ├── fix-failing-externalsecret.sh  ← fix Vault data + force ESO re-sync
    ├── fix-image-pull-secret.sh       ← create imagePullSecret + patch SA
    ├── fix-certificate-renewal.sh     ← delete failed CertReqs + verify Issuer
    └── fix-pvc-stuck-pending.sh       ← diagnose and guide PVC provisioner fix
```
