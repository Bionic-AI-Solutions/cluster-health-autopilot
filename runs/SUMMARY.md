# Cluster Health Autopilot — Run Summary

_Auto-generated 2026-05-18 06:12 UTC · 14 run(s) · 2026-05-04 → 2026-05-17_

## Health trend

| Date | Run | Components | Healthy | Degraded | Critical | Findings | Diagnostics |
|---|---|---|---|---|---|---|---|
| 2026-05-04 | run-2026-05-04 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-05 | run-2026-05-05 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-06 | run-2026-05-06 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-07 | run-2026-05-07 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-08 | run-2026-05-08 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-09 | run-2026-05-09 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-10 | run-2026-05-10 | 5 | 5 | 0 | 0 | 0 | 7 |
| 2026-05-11 | run-2026-05-11 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-12 | run-2026-05-12 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-13 | run-2026-05-13 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-14 | run-2026-05-14 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-15 | run-2026-05-15 | 6 | 5 | 1 | 0 | 1 | 0 |
| 2026-05-16 | run-2026-05-16 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-17 | run-2026-05-17 | 6 | 6 | 0 | 0 | 0 | 0 |

## Diagnostic patterns (top categories, anonymized)

| Category | Occurrences |
|---|---|
| `missing-secret` | 14 |
| `unprovisioned` | 14 |
| `ExternalSecret` | 7 |
| `cert-expiry` | 7 |
| `missing-key` | 7 |

## Component findings (top, anonymized)

| Severity/Component | Occurrences |
|---|---|
| `warning/Critical Services` | 1 |

## Day-by-day details

<details>
<summary><strong>2026-05-04</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-05</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-06</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-07</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-08</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-09</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-10</strong> — 5 component(s) · 7 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 30 critical services operational |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ExternalSecret` | ExternalSecret `b143e54c/289879e3` not Ready: error processing spec.data[0] (key: counsellor/config), err: Secret does not exist. Check Vault path / property names. Vault path `15d8e727/b79606fb` does not follow t6 hierarchy; expected: `2bb80d53/6374e63e/b143e54c/b79606fb`. |
| 2 | `missing-secret` | Secret `7f8e2ea7/97356135` does NOT exist (referenced by Deployment/openproject-cron in ns openproject, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 3 | `missing-secret` | Secret `f3cc87f7/83e0fc4a` does NOT exist (referenced by Deployment/playground-agent in ns playground, envFrom whole-secret import). Pod will fail to start on next restart. Create the Secret or remove the envFrom entry. |
| 4 | `missing-key` | Secret `47c88e9e/73442ae4` exists but is missing key `github-token`b2a694ad/1124edcd`GITHUB_TOKEN` is a case/format variant — possible naming mismatch. |
| 5 | `unprovisioned` | Secret `7f8e2ea7/97356135` referenced by Deployment/openproject-cron has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=openproject-cron-environment pointing to Vault path `2bb80d53/6374e63e/7f8e2ea7/b79606fb`. |
| 6 | `unprovisioned` | Secret `f3cc87f7/83e0fc4a` referenced by Deployment/playground-agent has no ExternalSecret provisioning it. Create an ExternalSecret with spec.target.name=playground-agent-secrets pointing to Vault path `2bb80d53/6374e63e/f3cc87f7/b79606fb`. |
| 7 | `cert-expiry` | Certificate `25bf6a1d/9b6c2336` is not Ready: Issuing certificate as Secret does not exist. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-11</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-12</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-13</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-14</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-15</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | DEGRADED | 1 service(s) degraded, 31 healthy |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

### Findings

| Component | Severity | Message |
|---|---|---|
| Service: svc-b9730754 | warning | Degraded (3/4 pods ready) |

</details>

<details>
<summary><strong>2026-05-16</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-17</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

</details>

---
_All namespace, workload, and secret names are anonymized using deterministic SHA-256 hashing._
_cha version(s) in this dataset: cluster-health-autopilot-0.9.1-4-g66c47e8, cluster-health-autopilot-0.9.1-5-g665a915, cluster-health-autopilot-1.4.0, v1.5.2-1-g1e93148_
