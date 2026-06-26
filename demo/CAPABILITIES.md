# Agentic SRE — Capabilities Reference

**Version:** 0.9.2 | **Date:** 2026-05-09 | **Cluster:** test-cluster1 (ap-south-1)

---

## Architecture Overview

```
Kubernetes Cluster
       │
       ▼
  [Watch API] ─── 14 resource types (pods, events, secrets, certs…)
       │  10s debounce
       ▼
  [Probes] ──────── Ceph, Nodes, Postgres, PVCs, Services
  [Analyzers] ───── 7 pattern-based diagnostics
       │
       ▼
  [Fixers] ──────── 4 whitelisted auto-remediation actions
       │
       ▼
  [DriftReport CRDs] + [Slack #aws-alerts]
```

Srenix runs a continuous detect → fix → re-verify → report loop.  
In `--remedy` mode the watcher: **(1)** probes pre-fix, **(2)** runs fixers, **(3)** re-probes post-fix, **(4)** posts Slack with both the Active Issues context AND the "Fixes Applied" block.

---

## What Srenix Monitors

### Probes — Infrastructure Health

| Probe | What It Checks | Severity | Resources Scanned |
|---|---|---|---|
| **Ceph** | CephCluster `.status.ceph.health` (HEALTH_OK / HEALTH_WARN / HEALTH_ERR), active alerts | critical / warning | `clusters.ceph.rook.io` |
| **Nodes** | Node Ready condition, disk/memory/PID pressure, cordoned status, kernel taints | critical | `nodes` |
| **Postgres** | CNPG Cluster `.status.phase`, replica lag, switchover state; Spilo StatefulSet ready replicas as fallback | critical / warning | `clusters.postgresql.cnpg.io`, StatefulSets |
| **PVCs** | PVC `.status.phase` != Bound (Pending, Lost, Unknown) | warning | `persistentvolumeclaims` |
| **Services** | Endpoint slice health for 30 named services; detects services with no ready endpoints | warning | `services`, `endpointslices` |

### Analyzers — Application-Layer Diagnostics

| Analyzer | What It Detects | Severity | Auto-Fix? |
|---|---|---|---|
| **SecretKeyMissing** | Pods in `CreateContainerConfigError` because a required Secret key is absent; traces back to the owning Deployment and ExternalSecret | warning | No |
| **FailingExternalSecrets** | ExternalSecrets in `Ready=False`; extracts exact missing Vault property from controller event | warning | No |
| **ProactiveSecretKeyCheck** | Jobs/CronJobs whose `envFrom` or `env` reference a Secret key that doesn't exist in the current Secret — before they fail | warning | No |
| **UnprovisionedSecret** | Pods that reference a Secret which doesn't exist at all (not just a missing key) | warning | No |
| **ImagePullAuth** | Pods in `ImagePullBackOff` / `ErrImagePull` due to registry auth failures (401, denied, no credentials); ignores non-auth pull errors | warning | No |
| **CertExpiry** | cert-manager Certificates that are: not Ready, expired, or expiring within 14 days | warning / critical | Partial* |
| **VaultPathMissing** *(paid)* | ExternalSecrets referencing a Vault path that returns 404 — finds Vault-side gaps before ESO even tries | warning | No |

*CertExpiry: Srenix auto-fixes stuck CertificateRequests/Orders that block renewal, but cannot fix CA/issuer misconfiguration.

---

## What Srenix Can Auto-Fix

| Fixer | Trigger Condition | Action | Safety Gates |
|---|---|---|---|
| **StaleErrorPods** | Pod in `Failed` phase, owned by a Job or unowned (debug/manual pods) | `kubectl delete pod` | Skips Deployment/StatefulSet/DaemonSet-owned pods; skips protected namespaces |
| **StuckJobsWithBadSecretRef** | Job-owned pod with `waiting.message` containing "couldn't find key", when the parent CronJob exists. Designed for the case where the CronJob template has been updated (root cause fixed) but the OLD frozen Job is blocking new runs via `concurrencyPolicy: Forbid` | Deletes the frozen Job; CronJob's next tick spawns a new Job using the corrected template | Requires parent CronJob to exist (won't delete standalone Jobs); skips protected namespaces |
| **StuckCertificateRequests** | cert-manager `CertificateRequest` or ACME `Order` in permanently-failed state (Ready=False/reason=Failed, state=errored/invalid) | Deletes failed CR/Order → cert-manager immediately retries issuance | Only touches terminal-failure CRs; skips Pending/in-progress; cert-manager recreates deleted CR |
| **StuckRSPods** | Pod in `CreateContainerConfigError` owned by a stale ReplicaSet (Deployment has rolled forward) — excluding "couldn't find key" events | Triggers `kubectl rollout restart` equivalent via strategic-merge patch on Deployment template annotation | Skips if event matches missing-key pattern (wrong fix for that class); skips protected namespaces |

### Protected Namespaces (fixers never act here)
`kube-system`, `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, `cnpg-system`

---

## What Srenix Reports But Cannot Auto-Fix

These require human intervention (Vault/git/infra change):

| Issue Class | Why No Auto-Fix | Human Action |
|---|---|---|
| **Missing Secret key** | Requires seeding Vault and syncing ExternalSecret — a config change | Add key to Vault path → ESO auto-syncs → pod restarts |
| **Failing ExternalSecret** | Root cause is in Vault (wrong path, missing property, expired token) | Fix Vault data, delete ESO to force re-sync |
| **Unprovisioned Secret** | Secret doesn't exist — cannot create from thin air safely | Create Secret or ESO pointing to correct Vault path |
| **ImagePull auth failure** | Requires creating/rotating imagePullSecret with valid registry creds | `kubectl create secret docker-registry` + patch ServiceAccount |
| **Certificate not renewing** | Issuer misconfiguration, DNS challenge failure, rate limit | Fix Issuer/ClusterIssuer, check DNS propagation, verify ACME account |
| **Node pressure / taint** | Requires node-level intervention (drain, resize, disk cleanup) | `kubectl drain node`, resize EBS/node, clean disk |
| **PVC stuck Pending** | Storage class / provisioner issue | Check StorageClass, CSI plugin, IAM permissions |
| **Vault path missing** | Vault data absent — cannot create secrets without source | Seed Vault at the referenced path |

---

## Sample Diagnostic Output

### SecretKeyMissing
```
Subject:     stale-pod/demo-app/api-server
Severity:    warning
Message:     Secret `demo-app/database-credentials` missing key `DB_PASSWORD`.
             Consuming Deployment: demo-app/api-server.
             ExternalSecret: demo-app/database-credentials.
             Add the key to the Vault path feeding this ExternalSecret,
             then delete the ESO to force re-sync.
```

### FailingExternalSecrets
```
Subject:     failing-externalsecret/demo-app/database-credentials
Severity:    warning
Message:     ExternalSecret demo-app/database-credentials is not Ready.
             Latest controller event: could not get secret data from provider:
             property "DB_PASSWORD" does not exist in secret path
             secret/t6-apps/demo-app/config
```

### CertExpiry (expiring soon)
```
Subject:     cert-expiry/livekit/livekit-api-cert
Severity:    warning
Message:     Certificate livekit/livekit-api-cert expires in 6d 14h (2026-05-15T18:00:00Z).
             Renewal may be stalled — check: kubectl describe certificate
             livekit/livekit-api-cert
```

### ImagePullAuth
```
Subject:     image-pull-auth/demo-app/worker
Severity:    warning
Message:     Pod demo-app/worker-7d9f8b-xpk4q cannot pull image
             docker4zerocool/worker-service:latest — auth failure:
             "pull access denied: unauthorized".
             Verify imagePullSecret on pod/ServiceAccount.
```

### Ceph Probe
```
Component:   Ceph/rook-ceph
Severity:    warning
Message:     CephCluster rook-ceph/rook-ceph: HEALTH_WARN — 1 nearfull OSD (osd.2 at 85%).
             Active alerts: OSD_NEARFULL
```

### Nodes Probe
```
Component:   Nodes/ip-192-168-85-233
Severity:    critical
Message:     Node ip-192-168-85-233 reports MemoryPressure=True.
             Cordoned: false. Ready: True.
```

### Postgres Probe
```
Component:   Postgres/cnpg/pg-srenix-test
Severity:    critical
Message:     CloudNativePG cluster pg-srenix-test: phase=Failing,
             reason: primary is not reachable, standby lag 47s
```

### StaleErrorPods Auto-Fix (Slack output)
```
🔴 Active Issues (1)
• stale-pod/demo-app/debug-probe — pod in Failed state

🔧 Fixes Applied (1)
• StaleErrorPods: Deleted stale Failed pod — Pod/demo-app/debug-probe
```

---

## Slack Alert Format

**New Issue:**
```
🔴 Active Issues (2)
• stale-pod/demo-app/api-server — Secret missing key DB_PASSWORD
• failing-externalsecret/demo-app/database-credentials — Vault property missing

🔧 Fixes Applied (0)
```

**Resolved:**
```
✅ Resolved (1)
• stale-pod/demo-app/debug-probe — pod in Failed state
```

**Auto-remediation cycle (watcher --remedy):**
```
🔴 Active Issues (1)
• stale-pod/demo-app/debug-probe — pod in Failed state

🔧 Fixes Applied (1) [Autopilot mode]
• StaleErrorPods: Deleted stale Failed pod — Pod/demo-app/debug-probe

✅ Resolved (1)
• stale-pod/demo-app/debug-probe — pod in Failed state
```

---

## Deployment Modes

| Mode | Command | Use Case |
|---|---|---|
| **Snapshot (zero-trust)** | `srenix diagnose --snapshot ./cluster-snapshot/` | Offline analysis, no RBAC needed, share with vendor |
| **Live diagnose** | `srenix diagnose --live` | One-shot live check |
| **Live watch** | `srenix watch --live --slack-webhook $URL` | Continuous alerting |
| **Autopilot** | `srenix watch --live --remedy --slack-webhook $URL` | Full detect→fix→report loop |
| **Dry-run** | `srenix watch --live --remedy --dry-run` | Preview fix decisions without acting |

---

## DriftReport CRDs

Srenix writes `driftreports.srenix.bionic-ai-solutions.io` objects to the cluster — one per active diagnostic. These are queryable like any Kubernetes resource:

```bash
kubectl get driftreports -A
kubectl describe driftreport <name> -n <ns>
```

On pod restart, Srenix reads existing DriftReports to pre-populate its dedup state — no Slack flood after rollouts.
