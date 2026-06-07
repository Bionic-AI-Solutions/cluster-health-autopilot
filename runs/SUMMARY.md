# Cluster Health Autopilot — Run Summary

_Auto-generated 2026-06-07 06:26 UTC · 34 run(s) · 2026-05-04 → 2026-06-06_

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
| 2026-05-18 | run-2026-05-18 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-19 | run-2026-05-19 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-20 | run-2026-05-20 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-21 | run-2026-05-21 | 6 | 6 | 0 | 0 | 0 | 1 |
| 2026-05-22 | run-2026-05-22 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-23 | run-2026-05-23 | 6 | 6 | 0 | 0 | 0 | 1 |
| 2026-05-24 | run-2026-05-24 | 6 | 6 | 0 | 0 | 0 | 2 |
| 2026-05-25 | run-2026-05-25 | 6 | 6 | 0 | 0 | 0 | 0 |
| 2026-05-26 | run-2026-05-26 | 12 | 11 | 0 | 0 | 3 | 0 |
| 2026-05-27 | run-2026-05-27 | 16 | 12 | 0 | 0 | 5 | 286 |
| 2026-05-28 | run-2026-05-28 | 16 | 14 | 1 | 0 | 5 | 286 |
| 2026-05-29 | run-2026-05-29 | 16 | 14 | 1 | 0 | 6 | 288 |
| 2026-05-30 | run-2026-05-30 | 16 | 14 | 1 | 0 | 5 | 291 |
| 2026-05-31 | run-2026-05-31 | 19 | 16 | 1 | 0 | 17 | 242 |
| 2026-06-01 | run-2026-06-01 | 19 | 16 | 1 | 0 | 5 | 243 |
| 2026-06-02 | run-2026-06-02 | 19 | 15 | 1 | 1 | 6 | 419 |
| 2026-06-03 | run-2026-06-03 | 19 | 16 | 0 | 1 | 5 | 398 |
| 2026-06-04 | run-2026-06-04 | 19 | 16 | 0 | 1 | 5 | 396 |
| 2026-06-05 | run-2026-06-05 | 19 | 16 | 0 | 1 | 5 | 402 |
| 2026-06-06 | run-2026-06-06 | 19 | 16 | 0 | 1 | 5 | 404 |

## Diagnostic patterns (top categories, anonymized)

| Category | Occurrences |
|---|---|
| `Pod` | 1830 |
| `Namespace` | 1198 |
| `ClusterRole` | 246 |
| `DNSChainDrift` | 239 |
| `HorizontalPodAutoscaler` | 63 |
| `ServiceAccount` | 48 |
| `PersistentVolumeClaim` | 16 |
| `missing-secret` | 14 |
| `unprovisioned` | 14 |
| `Role` | 11 |

## Component findings (top, anonymized)

| Severity/Component | Occurrences |
|---|---|
| `warning/component-68fc25e4` | 25 |
| `warning/component-a733dc9e` | 22 |
| `warning/component-09858a0e` | 12 |
| `info/component-80741754` | 7 |
| `critical/component-68fc25e4` | 5 |
| `warning/Ceph Storage` | 1 |
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

<details>
<summary><strong>2026-05-18</strong> — 6 component(s) · 0 diagnostic(s)</summary>

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
<summary><strong>2026-05-19</strong> — 6 component(s) · 0 diagnostic(s)</summary>

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
<summary><strong>2026-05-20</strong> — 6 component(s) · 0 diagnostic(s)</summary>

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
<summary><strong>2026-05-21</strong> — 6 component(s) · 1 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 28 endpoints reachable (20 auto-discovered) |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `image-pull-auth` | Pod `ad3c600e/bd9424fe` container "seed-model-cache" cannot pull image "img-482cf9d7:tag": auth failure. Check imagePullSecret in pod spec or ServiceAccount. Event: Failed to pull image "img-482cf9d7:tag": failed to pull and unpack image "img-5a01fadf:tag": failed to resolve r |

</details>

<details>
<summary><strong>2026-05-22</strong> — 6 component(s) · 0 diagnostic(s)</summary>

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
<summary><strong>2026-05-23</strong> — 6 component(s) · 1 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 29 endpoints reachable (21 auto-discovered) |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `cert-expiry` | Certificate `649e263a/8532da75` is not Ready: Secret was issued for "asre-baisoln-com". If this message is not transient, you might have two conflicting Certificates pointing to the same secret.. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-24</strong> — 6 component(s) · 2 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 29 endpoints reachable (21 auto-discovered) |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `image-pull-auth` | Pod `37a8eec1/08071df7` container "cha-soak-pull-auth" cannot pull image "img-2207b6af:tag": auth failure. Check imagePullSecret in pod spec or ServiceAccount. Event: Failed to pull image "img-2207b6af:tag": failed to pull and unpack image "img-2207b6af:tag": failed to res |
| 2 | `cert-expiry` | Certificate `649e263a/8532da75` is not Ready: Secret was issued for "asre-baisoln-com". If this message is not transient, you might have two conflicting Certificates pointing to the same secret.. Check Issuer/ClusterIssuer status and cert-manager controller logs. |

</details>

<details>
<summary><strong>2026-05-25</strong> — 6 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.1% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 75 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 29 endpoints reachable (21 auto-discovered) |

</details>

<details>
<summary><strong>2026-05-26</strong> — 12 component(s) · 0 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 77 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 30 endpoints reachable (24 auto-discovered, 2 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-142efee8 | warning | [transient, 1/2] https://host-802794af: connection failed — dial tcp: lookup host-802794af on img-2122b00c:tag: no such host |
| component-ba77a0cc | warning | [transient, 1/2] https://host-2c2e63d3: connection failed — dial tcp: lookup host-2c2e63d3 on img-2122b00c:tag: no such host |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |

</details>

<details>
<summary><strong>2026-05-27</strong> — 16 component(s) · 286 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 77 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 27 endpoints reachable (23 auto-discovered, 4 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | SKIPPED | Kong CRDs not installed (list kongplugins failed) |
| component-68fc25e4 | PROBE_FAILED | list horizontalpodautoscalers: horizontalpodautoscalers.autoscaling is forbidden: User "img-bbc5e661:tag:img-d10f5d3d:tag" cannot list resource "horizontalpodautoscalers" in API group "autoscaling" at the cluster scope |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | SKIPPED | Velero CRDs not installed (list backups failed) |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-41c64e8e | warning | [transient, 1/2] https://host-3891b54e: connection failed — dial tcp: lookup host-3891b54e on img-2122b00c:tag: no such host |
| component-e3985f6b | warning | [transient, 1/2] https://host-07340f5b: connection failed — dial tcp: lookup host-07340f5b on img-2122b00c:tag: no such host |
| component-d88c2311 | warning | [transient, 1/2] https://host-ac1bff25: connection failed — dial tcp: lookup host-ac1bff25 on img-2122b00c:tag: no such host |
| component-ba77a0cc | warning | [transient, 1/2] https://host-2c2e63d3: connection failed — dial tcp: lookup host-2c2e63d3 on img-2122b00c:tag: no such host |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `image-pull-auth` | Pod `ad3c600e/bd9424fe` container "seed-model-cache" cannot pull image "img-482cf9d7:tag": auth failure. Check imagePullSecret in pod spec or ServiceAccount. Event: Failed to pull image "img-482cf9d7:tag": failed to pull and unpack image "img-5a01fadf:tag": failed to resolve r |
| 2 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 22 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 23 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 24 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 25 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 26 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 27 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 28 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 29 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 30 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 31 | `ServiceAccount` | ServiceAccount langfuse/langfuse-s3 is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 32 | `ServiceAccount` | ServiceAccount langfuse/langfuse-zookeeper is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 33 | `ServiceAccount` | ServiceAccount langfuse/langfuse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 34 | `ServiceAccount` | ServiceAccount meilisearch/meilisearch is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 35 | `ServiceAccount` | ServiceAccount langfuse/langfuse-clickhouse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 36 | `ServiceAccount` | ServiceAccount olm/operatorhubio-catalog is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 37 | `ServiceAccount` | ServiceAccount openproject/openproject-memcached is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 38 | `ServiceAccount` | ServiceAccount openproject/openproject is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 39 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 41 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 43 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace comfyui has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace gharkaam has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace gpu-monitor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace kasten-io has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 58 | `Namespace` | Namespace langfuse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace live-avatar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 69 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace neo4j has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace openproject has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 76 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 77 | `Namespace` | Namespace playground has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 78 | `Namespace` | Namespace pulse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 79 | `Namespace` | Namespace qdrant has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 80 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 81 | `Namespace` | Namespace rag has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 82 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 83 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 84 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 85 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 86 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 87 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 88 | `Namespace` | Namespace vc-diligence has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 89 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 90 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 91 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 92 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 93 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 94 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 95 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 96 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 97 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 98 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 99 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 100 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 101 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 102 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 103 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 104 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 105 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 106 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 107 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 108 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 109 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 110 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 111 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 112 | `Pod` | Pod cha-website/cha-website-658b9644c6-9mfj4 mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 113 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29663100-zwqb8 mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 114 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29664540-gncgk mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 115 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29661660-5pdnp mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 116 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29663100-fn6zn mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 117 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29664540-kk72n mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 118 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-kwx8j mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-watcher-854d799575-4t7cc mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-diagnose-test-1779843454-c9nhh mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 121 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 122 | `Pod` | Pod default/cha-soak-pull-auth mounts 1 container image(s) without digest pin: cha-soak-pull-auth=img-2207b6af:tag |
| 123 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 124 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 125 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 126 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 127 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-6hv9g mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 128 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ffj8d mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 129 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-h57t6 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 130 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ht9sz mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 131 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-pxrsk mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 132 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-xwkrb mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 133 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 134 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 135 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 136 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 137 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 138 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 139 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 140 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 141 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 142 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 143 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 144 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 145 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 146 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 147 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 148 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 149 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 150 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 151 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 152 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 153 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 154 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 155 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 156 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 157 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 158 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 159 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 160 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 161 | `Pod` | Pod kube-flannel/kube-flannel-ds-9ldj8 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 162 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 163 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 164 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 165 | `Pod` | Pod kube-flannel/kube-flannel-ds-xzv56 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 166 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 167 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 168 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 169 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 170 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 171 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 172 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 173 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 174 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 175 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 176 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 177 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 178 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 179 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 180 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 181 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 182 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 183 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 184 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 185 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 186 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 187 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 188 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 189 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 190 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 191 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 192 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 193 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 194 | `Pod` | Pod metallb-system/speaker-z5n4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 195 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 196 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 197 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 198 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 199 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 200 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 201 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 202 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 203 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 204 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 205 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 206 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 207 | `Pod` | Pod operators/redis-operator-98f484cf8-dgzfj mounts 1 container image(s) without digest pin: manager=img-e3b32edf:tag |
| 208 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 209 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 210 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 211 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 212 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 213 | `Pod` | Pod pg/postgres-minio-backup-29662740-5g7cs mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 214 | `Pod` | Pod pg/postgres-minio-backup-29664180-bpdzc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 215 | `Pod` | Pod pg/postgres-minio-backup-29665620-t89vk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 216 | `Pod` | Pod pg/postgres-nfs-backup-29662680-c6kjm mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 217 | `Pod` | Pod pg/postgres-nfs-backup-29664120-wnl76 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 218 | `Pod` | Pod pg/postgres-nfs-backup-29665560-n2g6f mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 219 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 220 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 221 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 222 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 223 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 224 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 225 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 226 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 227 | `Pod` | Pod storethesoup/wordpress-7fb7855898-gtbvc mounts 1 container image(s) without digest pin: wordpress=img-576473d6:tag |
| 228 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 229 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 230 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 231 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 232 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 233 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 234 | `Pod` | Pod vc-livekit/livekit-agent-58857f9f4c-5txtw mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 235 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 236 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 237 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 238 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 239 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 240 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 241 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 242 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 243 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 244 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 245 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 246 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 247 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 248 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 249 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 250 | `Namespace` | Namespace kasten-io runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 251 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 252 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 253 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 254 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 255 | `Namespace` | Namespace kube-flannel runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 256 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 257 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 258 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 259 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 260 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 261 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 262 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 263 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 264 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 265 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace olm runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace operators runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |

</details>

<details>
<summary><strong>2026-05-28</strong> — 16 component(s) · 286 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 77 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 30 endpoints reachable (23 auto-discovered, 1 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 81 KongPlugin resource(s) inspected |
| component-68fc25e4 | DEGRADED | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-41c64e8e | warning | [transient, 1/2] https://host-3891b54e: connection failed — dial tcp: lookup host-3891b54e on img-2122b00c:tag: no such host |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-3e7d4aa2 | warning | HPA comfyui/keda-hpa-comfyui autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 3 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 6 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 7 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 8 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 9 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 10 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 11 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 12 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 13 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 15 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 16 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 17 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 19 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 20 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 21 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 22 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 23 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 24 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 25 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 26 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 27 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 28 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 29 | `ServiceAccount` | ServiceAccount meilisearch/meilisearch is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 30 | `ServiceAccount` | ServiceAccount langfuse/langfuse-clickhouse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 31 | `ServiceAccount` | ServiceAccount langfuse/langfuse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 32 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 33 | `ServiceAccount` | ServiceAccount langfuse/langfuse-zookeeper is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 34 | `ServiceAccount` | ServiceAccount openproject/openproject is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 35 | `ServiceAccount` | ServiceAccount openproject/openproject-memcached is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 36 | `ServiceAccount` | ServiceAccount langfuse/langfuse-s3 is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 37 | `ServiceAccount` | ServiceAccount olm/operatorhubio-catalog is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 38 | `HorizontalPodAutoscaler` | HPA gharkaam/gharkaam-web pinned at maxReplicas=6 for >24h0m0s; workload is chronically under-provisioned |
| 39 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 40 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 41 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 42 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 43 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 44 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 45 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace comfyui has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace gharkaam has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace gpu-monitor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace kasten-io has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 64 | `Namespace` | Namespace langfuse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace live-avatar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 75 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 76 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 77 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 78 | `Namespace` | Namespace neo4j has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 79 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 80 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 81 | `Namespace` | Namespace openproject has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 82 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 83 | `Namespace` | Namespace playground has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 84 | `Namespace` | Namespace pulse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 85 | `Namespace` | Namespace qdrant has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 86 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 87 | `Namespace` | Namespace rag has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 88 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 89 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 90 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 91 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 92 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 93 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 94 | `Namespace` | Namespace vc-diligence has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 95 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 96 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 97 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 98 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 99 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 100 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 101 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 102 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 103 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 104 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 105 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 106 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 107 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 108 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 109 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 110 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 111 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 112 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 113 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 114 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 115 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 116 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 117 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 118 | `Pod` | Pod cha-website/cha-website-6bb75cf879-mc5xg mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-aiwatch-748447f69c-zpfm6 mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-approval-server-6bb485c8bc-9qhgg mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 121 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29664540-gncgk mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 122 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29665980-wph8j mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 123 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29663100-fn6zn mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 124 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29664540-kk72n mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 125 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29665980-mj2cq mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 126 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-kwx8j mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 127 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-watcher-d85fd7946-mzmm5 mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 128 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 129 | `Pod` | Pod default/cha-soak-pull-auth mounts 1 container image(s) without digest pin: cha-soak-pull-auth=img-2207b6af:tag |
| 130 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 131 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 132 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 133 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 134 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-6hv9g mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 135 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ffj8d mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 136 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-h57t6 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 137 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ht9sz mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 138 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-pxrsk mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 139 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-xwkrb mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 140 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 141 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 142 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 143 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 144 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 145 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 146 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 147 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 148 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 149 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 150 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 151 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 152 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 153 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 154 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 155 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 156 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 157 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 158 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 159 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 160 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 161 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 162 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 163 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 164 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 165 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 166 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 167 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 168 | `Pod` | Pod kube-flannel/kube-flannel-ds-9ldj8 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 169 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 170 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 171 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 172 | `Pod` | Pod kube-flannel/kube-flannel-ds-xzv56 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 173 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 174 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 175 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 176 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 177 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 178 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 179 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 180 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 181 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 182 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 183 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 184 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 185 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 186 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 187 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 188 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 189 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 190 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 191 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 192 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 193 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 194 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 195 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 196 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 197 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 198 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 199 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 200 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 201 | `Pod` | Pod metallb-system/speaker-z5n4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 202 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 203 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 204 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 205 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 206 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 207 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 208 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 209 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 210 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 211 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 212 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 213 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 214 | `Pod` | Pod operators/redis-operator-98f484cf8-dgzfj mounts 1 container image(s) without digest pin: manager=img-e3b32edf:tag |
| 215 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 216 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 217 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 218 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 219 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 220 | `Pod` | Pod pg/postgres-minio-backup-29664180-bpdzc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 221 | `Pod` | Pod pg/postgres-minio-backup-29665620-t89vk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 222 | `Pod` | Pod pg/postgres-minio-backup-29667060-k859l mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 223 | `Pod` | Pod pg/postgres-nfs-backup-29664120-wnl76 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 224 | `Pod` | Pod pg/postgres-nfs-backup-29665560-n2g6f mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 225 | `Pod` | Pod pg/postgres-nfs-backup-29667000-qscj5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 226 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 227 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 228 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 229 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 230 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 231 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 232 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 233 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 234 | `Pod` | Pod storethesoup/wordpress-7fb7855898-gtbvc mounts 1 container image(s) without digest pin: wordpress=img-576473d6:tag |
| 235 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 236 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 237 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 238 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 239 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 240 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 241 | `Pod` | Pod vc-livekit/livekit-agent-764fcd7449-hsv72 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 242 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 243 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 244 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 245 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 246 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 247 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 248 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 249 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 250 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 251 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 252 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 253 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 254 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 255 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 256 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 257 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 258 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 259 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 260 | `Namespace` | Namespace kube-flannel runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 261 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 262 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 263 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 264 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 265 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |

</details>

<details>
<summary><strong>2026-05-29</strong> — 16 component(s) · 288 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 77 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 29 endpoints reachable (23 auto-discovered, 2 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 81 KongPlugin resource(s) inspected |
| component-68fc25e4 | DEGRADED | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-41c64e8e | warning | [transient, 1/2] https://host-3891b54e: connection failed — dial tcp: lookup host-3891b54e on img-2122b00c:tag: no such host |
| component-3d203015 | warning | [transient, 1/2] https://host-271e2cd1: connection failed — context deadline exceeded (Client.Timeout exceeded while awaiting headers) |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-3e7d4aa2 | warning | HPA comfyui/keda-hpa-comfyui autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `image-pull-auth` | Pod `ad3c600e/bd9424fe` container "seed-model-cache" cannot pull image "img-482cf9d7:tag": auth failure. Check imagePullSecret in pod spec or ServiceAccount. Event: Failed to pull image "img-482cf9d7:tag": failed to pull and unpack image "img-5a01fadf:tag": failed to resolve r |
| 2 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 22 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 23 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 24 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 25 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 26 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 27 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 28 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 29 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 30 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 31 | `ServiceAccount` | ServiceAccount langfuse/langfuse-s3 is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 32 | `ServiceAccount` | ServiceAccount openproject/openproject is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 33 | `ServiceAccount` | ServiceAccount langfuse/langfuse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 34 | `ServiceAccount` | ServiceAccount langfuse/langfuse-zookeeper is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 35 | `ServiceAccount` | ServiceAccount meilisearch/meilisearch is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 36 | `ServiceAccount` | ServiceAccount olm/operatorhubio-catalog is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 37 | `ServiceAccount` | ServiceAccount langfuse/langfuse-clickhouse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 38 | `ServiceAccount` | ServiceAccount openproject/openproject-memcached is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 39 | `HorizontalPodAutoscaler` | HPA gharkaam/gharkaam-web pinned at maxReplicas=6 for >24h0m0s; workload is chronically under-provisioned |
| 40 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 41 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 42 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 43 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 44 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 45 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 46 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace comfyui has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace gharkaam has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace gpu-monitor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace kasten-io has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 65 | `Namespace` | Namespace langfuse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace live-avatar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 76 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 77 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 78 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 79 | `Namespace` | Namespace neo4j has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 80 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 81 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 82 | `Namespace` | Namespace openproject has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 83 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 84 | `Namespace` | Namespace playground has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 85 | `Namespace` | Namespace pulse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 86 | `Namespace` | Namespace qdrant has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 87 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 88 | `Namespace` | Namespace rag has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 89 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 90 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 91 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 92 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 93 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 94 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 95 | `Namespace` | Namespace vc-diligence has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 96 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 97 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 98 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 99 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 100 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 101 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 102 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 103 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 104 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 105 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 106 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 107 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 108 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 109 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 110 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 111 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 112 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 113 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 114 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 115 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 116 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 117 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 118 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 119 | `Pod` | Pod cha-website/cha-website-6bb75cf879-mc5xg mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-aiwatch-77b6f7687c-zm68c mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 121 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-approval-server-8669f5d6f9-b8mmg mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 122 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29664540-gncgk mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 123 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29665980-wph8j mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 124 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29667420-wc5g5 mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 125 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29664540-kk72n mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 126 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29665980-mj2cq mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 127 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29667420-b6z9s mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 128 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-kwx8j mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 129 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-watcher-d94895dbb-dcwkj mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 130 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 131 | `Pod` | Pod default/cha-soak-pull-auth mounts 1 container image(s) without digest pin: cha-soak-pull-auth=img-2207b6af:tag |
| 132 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 133 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 134 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 135 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 136 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-6hv9g mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 137 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ffj8d mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 138 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-h57t6 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 139 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ht9sz mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 140 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-pxrsk mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 141 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-xwkrb mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 142 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 143 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 144 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 145 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 146 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 147 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 148 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 149 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 150 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 151 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 152 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 153 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 154 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 155 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 156 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 157 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 158 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 159 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 160 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 161 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 162 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 163 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 164 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 165 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 166 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 167 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 168 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 169 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 170 | `Pod` | Pod kube-flannel/kube-flannel-ds-9ldj8 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 171 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 172 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 173 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 174 | `Pod` | Pod kube-flannel/kube-flannel-ds-xzv56 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 175 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 176 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 177 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 178 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 179 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 180 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 181 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 182 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 183 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 184 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 185 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 186 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 187 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 188 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 189 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 190 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 191 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 192 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 193 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 194 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 195 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 196 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 197 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 198 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 199 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 200 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 201 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 202 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 203 | `Pod` | Pod metallb-system/speaker-z5n4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 204 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 205 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 206 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 207 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 208 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 209 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 210 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 211 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 212 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 213 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 214 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 215 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 216 | `Pod` | Pod operators/redis-operator-98f484cf8-dgzfj mounts 1 container image(s) without digest pin: manager=img-e3b32edf:tag |
| 217 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 218 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 219 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 220 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 221 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 222 | `Pod` | Pod pg/postgres-minio-backup-29665620-t89vk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 223 | `Pod` | Pod pg/postgres-minio-backup-29667060-k859l mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 224 | `Pod` | Pod pg/postgres-minio-backup-29668500-kkts5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 225 | `Pod` | Pod pg/postgres-nfs-backup-29665560-n2g6f mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 226 | `Pod` | Pod pg/postgres-nfs-backup-29667000-qscj5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 227 | `Pod` | Pod pg/postgres-nfs-backup-29668440-xg6t8 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 228 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 229 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 230 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 231 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 232 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 233 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 234 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 235 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 236 | `Pod` | Pod storethesoup/wordpress-7fb7855898-gtbvc mounts 1 container image(s) without digest pin: wordpress=img-576473d6:tag |
| 237 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 238 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 239 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 240 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 241 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 242 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 243 | `Pod` | Pod vc-livekit/livekit-agent-764fcd7449-hsv72 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 244 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 245 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 246 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 247 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 248 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 249 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 250 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 251 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 252 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 253 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 254 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 255 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 256 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 257 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 258 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 259 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 260 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 261 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 262 | `Namespace` | Namespace kube-flannel runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 263 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 264 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 265 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |

</details>

<details>
<summary><strong>2026-05-30</strong> — 16 component(s) · 291 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 78 PVCs bound |
| Critical Services | HEALTHY | All 32 critical services operational |
| component-a733dc9e | HEALTHY | All 30 endpoints reachable (23 auto-discovered, 1 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 81 KongPlugin resource(s) inspected |
| component-68fc25e4 | DEGRADED | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-41c64e8e | warning | [transient, 1/2] https://host-3891b54e: connection failed — dial tcp: lookup host-3891b54e on img-2122b00c:tag: no such host |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-3e7d4aa2 | warning | HPA comfyui/keda-hpa-comfyui autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `image-pull-auth` | Pod `37a8eec1/08071df7` container "cha-soak-pull-auth" cannot pull image "img-2207b6af:tag": auth failure. Check imagePullSecret in pod spec or ServiceAccount. Event: Failed to pull image "img-2207b6af:tag": failed to pull and unpack image "img-2207b6af:tag": failed to res |
| 2 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 22 | `ClusterRole` | ClusterRole redis.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redis]) |
| 23 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 24 | `ClusterRole` | ClusterRole redisclusters.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisclusters]) |
| 25 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 26 | `ClusterRole` | ClusterRole redisreplications.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 27 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta1-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 28 | `ClusterRole` | ClusterRole redissentinels.redis.redis.opstreelabs.in-v1beta2-admin grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redissentinels]) |
| 29 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 30 | `ServiceAccount` | ServiceAccount langfuse/langfuse-clickhouse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 31 | `ServiceAccount` | ServiceAccount langfuse/langfuse is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 32 | `ServiceAccount` | ServiceAccount openproject/openproject-memcached is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 33 | `ServiceAccount` | ServiceAccount openproject/openproject is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 34 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 35 | `ServiceAccount` | ServiceAccount langfuse/langfuse-zookeeper is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 36 | `ServiceAccount` | ServiceAccount langfuse/langfuse-s3 is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 37 | `ServiceAccount` | ServiceAccount meilisearch/meilisearch is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 38 | `ServiceAccount` | ServiceAccount olm/operatorhubio-catalog is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 39 | `HorizontalPodAutoscaler` | HPA gharkaam/gharkaam-web pinned at maxReplicas=6 for >24h0m0s; workload is chronically under-provisioned |
| 40 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 41 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 42 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 43 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 44 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 45 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 46 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace comfyui has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace gharkaam has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace gpu-monitor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace kasten-io has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 65 | `Namespace` | Namespace langfuse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace live-avatar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 76 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 77 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 78 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 79 | `Namespace` | Namespace neo4j has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 80 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 81 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 82 | `Namespace` | Namespace openproject has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 83 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 84 | `Namespace` | Namespace playground has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 85 | `Namespace` | Namespace pulse has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 86 | `Namespace` | Namespace qdrant has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 87 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 88 | `Namespace` | Namespace rag has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 89 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 90 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 91 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 92 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 93 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 94 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 95 | `Namespace` | Namespace vc-diligence has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 96 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 97 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 98 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 99 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 100 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 101 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 102 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 103 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 104 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 105 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 106 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 107 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 108 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 109 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 110 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 111 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 112 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 113 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 114 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 115 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 116 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 117 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 118 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 119 | `Pod` | Pod cha-website/cha-website-6bb75cf879-mc5xg mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-aiwatch-596d94b89c-4b7bl mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 121 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-approval-server-5897b9c848-7r9vp mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 122 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-approval-server-5897b9c848-rtczn mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 123 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29665980-wph8j mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 124 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29667420-wc5g5 mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 125 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-diagnose-29668860-jpqpv mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 126 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 127 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29665980-mj2cq mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 128 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29667420-b6z9s mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 129 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-remediate-29668860-g7mlr mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 130 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-kwx8j mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 131 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-watcher-d94895dbb-dcwkj mounts 1 container image(s) without digest pin: cha=img-94908202:tag |
| 132 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 133 | `Pod` | Pod default/cha-soak-pull-auth mounts 1 container image(s) without digest pin: cha-soak-pull-auth=img-2207b6af:tag |
| 134 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 135 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 136 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 137 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 138 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-6hv9g mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 139 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ffj8d mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 140 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-h57t6 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 141 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-ht9sz mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 142 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-pxrsk mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 143 | `Pod` | Pod gharkaam/gharkaam-web-89b7d8957-xwkrb mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 144 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 145 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 146 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 147 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 148 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 149 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 150 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 151 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 152 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 153 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 154 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 155 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 156 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 157 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 158 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 159 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 160 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 161 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 162 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 163 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 164 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 165 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 166 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 167 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 168 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 169 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 170 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 171 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 172 | `Pod` | Pod kube-flannel/kube-flannel-ds-9ldj8 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 173 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 174 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 175 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 176 | `Pod` | Pod kube-flannel/kube-flannel-ds-xzv56 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 177 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 178 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 179 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 180 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 181 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 182 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 183 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 184 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 185 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 186 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 187 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 188 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 189 | `Pod` | Pod livekit/cm-acme-http-solver-l57th mounts 1 container image(s) without digest pin: acmesolver=img-e94e2e74:tag |
| 190 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 191 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 192 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 193 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 194 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 195 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 196 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 197 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 198 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 199 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 200 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 201 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 202 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 203 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 204 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 205 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 206 | `Pod` | Pod metallb-system/speaker-z5n4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 207 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 208 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 209 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 210 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 211 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 212 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 213 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 214 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 215 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 216 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 217 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 218 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 219 | `Pod` | Pod operators/redis-operator-98f484cf8-dgzfj mounts 1 container image(s) without digest pin: manager=img-e3b32edf:tag |
| 220 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 221 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 222 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 223 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 224 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 225 | `Pod` | Pod pg/postgres-minio-backup-29667060-k859l mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 226 | `Pod` | Pod pg/postgres-minio-backup-29668500-kkts5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 227 | `Pod` | Pod pg/postgres-minio-backup-29669940-87hks mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 228 | `Pod` | Pod pg/postgres-nfs-backup-29667000-qscj5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 229 | `Pod` | Pod pg/postgres-nfs-backup-29668440-xg6t8 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 230 | `Pod` | Pod pg/postgres-nfs-backup-29669880-kh794 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 231 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 232 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 233 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 234 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 235 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 236 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 237 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 238 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 239 | `Pod` | Pod storethesoup/wordpress-7fb7855898-gtbvc mounts 1 container image(s) without digest pin: wordpress=img-576473d6:tag |
| 240 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 241 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 242 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 243 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 244 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 245 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 246 | `Pod` | Pod vc-livekit/livekit-agent-764fcd7449-hsv72 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 247 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 248 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 249 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 250 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 251 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 252 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 253 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 254 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 255 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 256 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 257 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 258 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 259 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 260 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 261 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 262 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 263 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 264 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 265 | `Namespace` | Namespace kube-flannel runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |

</details>

<details>
<summary><strong>2026-05-31</strong> — 19 component(s) · 242 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.2% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 79 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 19 endpoints reachable (31 auto-discovered, 12 transient under threshold) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 81 KongPlugin resource(s) inspected |
| component-68fc25e4 | DEGRADED | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-593c6663 | warning | [transient, 1/2] https://host-647db09d: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-7e9626dc | warning | [transient, 1/2] https://host-f1ba8d59: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-19e34bfb | warning | [transient, 1/2] https://host-3b05cb67: connection failed — dial tcp img-5ab4227a:tag: connect: connection refused |
| component-1645c7ed | warning | [transient, 1/2] https://host-e5673458: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-709c9a19 | warning | [transient, 1/2] https://host-32225d86: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-059f1171 | warning | [transient, 1/2] https://host-81ab186c: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-8b7952c7 | warning | [transient, 1/2] https://host-d63bb08e: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-5dc4fc30 | warning | [transient, 1/2] https://host-0ccdb59e: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-66c36fdf | warning | [transient, 1/2] https://host-29bd8929: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-85beb2bc | warning | [transient, 1/2] https://host-2249606b: connection failed — dial tcp img-5ab4227a:tag: connect: connection refused |
| component-894c9bb3 | warning | [transient, 1/2] https://host-d947e194: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-9082efc5 | warning | [transient, 1/2] https://host-bda455e8: connection failed — dial tcp img-41b8351d:tag: connect: connection refused |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-3e7d4aa2 | warning | HPA comfyui/keda-hpa-comfyui autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 3 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 6 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 7 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 8 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 9 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 10 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 11 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 12 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 13 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 15 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 16 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 17 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 19 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 20 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 21 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 22 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 23 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 24 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 28 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 29 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 30 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 34 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 41 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 43 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 52 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 70 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 71 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 72 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 73 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 74 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 75 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 76 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 81 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 82 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 83 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 84 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 85 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 86 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 87 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 88 | `Pod` | Pod cha-website/cha-website-6bb75cf879-mc5xg mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 89 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-b9db864c7-6qzsq mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 90 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-c5485557f-6plfd mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 91 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-c5485557f-tqdlj mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 92 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 93 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-5bd4c4d6f7-g4ktb mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 94 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-6c9f7887bd-wvj4l mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 95 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 96 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 97 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 98 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 99 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 100 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 101 | `Pod` | Pod gharkaam/gharkaam-redis-7984bf78cb-9m8vt mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 102 | `Pod` | Pod gharkaam/gharkaam-web-777dddd5b5-jvjqs mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 103 | `Pod` | Pod gharkaam/gharkaam-web-777dddd5b5-zrsm7 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 104 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 105 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 106 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 107 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 108 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 109 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 110 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 111 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 112 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 113 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 114 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 115 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 116 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 117 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 118 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 119 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 120 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 121 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 122 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 123 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 124 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 125 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 126 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 127 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 128 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 129 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 130 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 131 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 132 | `Pod` | Pod kube-flannel/kube-flannel-ds-9ldj8 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 133 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 134 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 135 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 136 | `Pod` | Pod kube-flannel/kube-flannel-ds-xzv56 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 137 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 138 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 139 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 140 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 141 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 142 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 143 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 144 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 145 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 146 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 147 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 148 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 149 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 150 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 151 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 152 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 153 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 154 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 155 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 156 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 157 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 158 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 159 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 160 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 161 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 162 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 163 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 164 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 165 | `Pod` | Pod metallb-system/speaker-z5n4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 166 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 167 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 168 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 169 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 170 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 171 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 172 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 173 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 174 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 175 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 176 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 177 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 178 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 179 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 180 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 181 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 182 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 183 | `Pod` | Pod pg/postgres-minio-backup-29668500-kkts5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 184 | `Pod` | Pod pg/postgres-minio-backup-29669940-87hks mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 185 | `Pod` | Pod pg/postgres-minio-backup-29671380-4gftn mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 186 | `Pod` | Pod pg/postgres-nfs-backup-29668440-xg6t8 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 187 | `Pod` | Pod pg/postgres-nfs-backup-29669880-kh794 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 188 | `Pod` | Pod pg/postgres-nfs-backup-29671320-gqkzc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 189 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 190 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 191 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 192 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 193 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 194 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 195 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 196 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 197 | `Pod` | Pod storethesoup/redis-6b45d66dc6-c65d2 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 198 | `Pod` | Pod storethesoup/wordpress-f87f66675-lbdnj mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 199 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 200 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 201 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 202 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 203 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 204 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 205 | `Pod` | Pod vc-livekit/livekit-agent-764fcd7449-hsv72 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 206 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 207 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 208 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 209 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 210 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 211 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 212 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 213 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 214 | `DNSChainDrift` | Ingress `d10f5d3d/0d96ec3b` routes host *host-f1ba8d59* to Service `d10f5d3d/0d96ec3b` (port http) but that Service does not exist in the cluster. |
| 215 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 216 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 217 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 218 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 219 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 220 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 221 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 222 | `DNSChainDrift` | Ingress `10182ab8/77b28987` routes host *host-32225d86* to Service `10182ab8/8bd1790c` (port 8000) but that Service does not exist in the cluster. |
| 223 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 224 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 225 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 226 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 227 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 228 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 229 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 230 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 231 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 232 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 233 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 234 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 235 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 236 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 237 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 238 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 239 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 240 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 241 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 242 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 32 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-01</strong> — 19 component(s) · 243 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.4% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 79 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 31 endpoints reachable (31 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 5 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 81 KongPlugin resource(s) inspected |
| component-68fc25e4 | DEGRADED | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-3e7d4aa2 | warning | HPA comfyui/keda-hpa-comfyui autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 3 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 6 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 7 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 8 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 9 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 10 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 11 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 12 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 13 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 15 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 16 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 17 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 19 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 20 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 21 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 22 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 23 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 24 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 28 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 29 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 30 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 34 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 41 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 43 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 52 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Pod` | Pod agents/token-server-7f6d869fc6-5vkr6 mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 70 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-8695d8997d-thjl6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 71 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-79b9d59f45-r6zhw mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 72 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-84b57d6465-9g5h7 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 73 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-75b6b6b9b5-6hnfp mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 74 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-ccc778977-2rnxs mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 75 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-999dbf868-4pmbv mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 76 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-cff59b44d-dvn9z mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-78f6964c69-qpt45 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-74fcbd997b-mgd65 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-5cb988b975-8f4v5 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod bionic-platform/dify-api-5db8c684d-gq5jj mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 81 | `Pod` | Pod bionic-platform/dify-plugin-daemon-865d5b74dd-x45vd mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 82 | `Pod` | Pod bionic-platform/dify-sandbox-854d555b75-4r29f mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 83 | `Pod` | Pod bionic-platform/dify-web-ccf9b7f48-flh7d mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 84 | `Pod` | Pod bionic-platform/dify-worker-5c467cd47b-77lhj mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 85 | `Pod` | Pod cert-manager/cert-manager-858fbcc458-g7v97 mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 86 | `Pod` | Pod cert-manager/cert-manager-cainjector-67644489c4-lc75p mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 87 | `Pod` | Pod cert-manager/cert-manager-webhook-6687664ccb-vpdkj mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 88 | `Pod` | Pod cha-website/cha-website-6bb75cf879-mc5xg mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 89 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-7f8947bb8d-w66wt mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 90 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-7fbc4dc8d6-972ts mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 91 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-7fbc4dc8d6-snkhc mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 92 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 93 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-85b49f8965-ww8kr mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 94 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-5f676cd55b-9fqr9 mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 95 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 96 | `Pod` | Pod code/devcontainer-58758d55c6-s879x mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 97 | `Pod` | Pod default/prometheus-operator-54866c5c7-qtwv8 mounts 1 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag |
| 98 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 99 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 100 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 101 | `Pod` | Pod gharkaam/gharkaam-redis-7984bf78cb-9m8vt mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 102 | `Pod` | Pod gharkaam/gharkaam-web-777dddd5b5-jvjqs mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 103 | `Pod` | Pod gharkaam/gharkaam-web-777dddd5b5-zrsm7 mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 104 | `Pod` | Pod kasten-io/aggregatedapis-svc-86558f785-dd47n mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 105 | `Pod` | Pod kasten-io/auth-svc-65b496c468-2l65q mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 106 | `Pod` | Pod kasten-io/catalog-svc-7d85c8d4b6-rwvzx mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 107 | `Pod` | Pod kasten-io/controllermanager-svc-7f67bbc55c-bhnxj mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 108 | `Pod` | Pod kasten-io/crypto-svc-698f54fd98-wv7gd mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 109 | `Pod` | Pod kasten-io/dashboardbff-svc-7bc499679-kkq6h mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 110 | `Pod` | Pod kasten-io/executor-svc-678b877f86-c9brc mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 111 | `Pod` | Pod kasten-io/executor-svc-678b877f86-pvhqp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 112 | `Pod` | Pod kasten-io/executor-svc-678b877f86-vgkkm mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 113 | `Pod` | Pod kasten-io/frontend-svc-685ff944b-r696k mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 114 | `Pod` | Pod kasten-io/gateway-75bd44fd8d-sg99g mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 115 | `Pod` | Pod kasten-io/jobs-svc-5cbcc5598d-dj246 mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 116 | `Pod` | Pod kasten-io/kanister-svc-79ffb6bc95-hppk2 mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 117 | `Pod` | Pod kasten-io/logging-svc-79c7b479dc-chs5r mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 118 | `Pod` | Pod kasten-io/metering-svc-7b8c678f77-gxzpj mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 119 | `Pod` | Pod kasten-io/prometheus-server-569cd85c55-zsdls mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 120 | `Pod` | Pod kasten-io/state-svc-9ddfcd765-jf2km mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 121 | `Pod` | Pod kb-system/snapshot-controller-59d94b5486-nwqbq mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 122 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-85b67466-fb85r mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 123 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-f96xd mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 124 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-h57w8 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 125 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-67c8b74657-wzqvm mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 126 | `Pod` | Pod keda/keda-add-ons-http-interceptor-64d648cd97-kzbwz mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 127 | `Pod` | Pod keda/keda-admission-webhooks-5d67c9bcfb-qs2rq mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 128 | `Pod` | Pod keda/keda-operator-85ff5bb446-87f8g mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 129 | `Pod` | Pod keda/keda-operator-metrics-apiserver-7ff5758fd7-rv8cd mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 130 | `Pod` | Pod keycloak/keycloak-0 mounts 1 container image(s) without digest pin: keycloak=img-a351cffb:tag |
| 131 | `Pod` | Pod kong/kong-kong-6d4b57d8bb-84zp6 mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 132 | `Pod` | Pod kube-flannel/kube-flannel-ds-5fn6d mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 133 | `Pod` | Pod kube-flannel/kube-flannel-ds-b5c7n mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 134 | `Pod` | Pod kube-flannel/kube-flannel-ds-bb2p4 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 135 | `Pod` | Pod kube-flannel/kube-flannel-ds-cfdk2 mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 136 | `Pod` | Pod kube-flannel/kube-flannel-ds-pvx5q mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 137 | `Pod` | Pod kube-flannel/kube-flannel-ds-z8vxr mounts 1 container image(s) without digest pin: kube-flannel=img-808fdb6a:tag |
| 138 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 139 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 140 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 1 container image(s) without digest pin: clickhouse=img-f72637ad:tag |
| 141 | `Pod` | Pod langfuse/langfuse-s3-699b5ddc85-kt5h9 mounts 1 container image(s) without digest pin: minio=img-14773e69:tag |
| 142 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 143 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 144 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 1 container image(s) without digest pin: zookeeper=img-eab8cce1:tag |
| 145 | `Pod` | Pod letta/letta-server-85d4f7b9c6-9g6jd mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 146 | `Pod` | Pod letta/letta-server-85d4f7b9c6-dh7zb mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 147 | `Pod` | Pod letta/letta-server-85d4f7b9c6-twf4k mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 148 | `Pod` | Pod livekit-agents/flash-agent-7bf6d47694-nmznh mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 149 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-2s266 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 150 | `Pod` | Pod livekit/livekit-egress-648bd8f6d8-xwlgw mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 151 | `Pod` | Pod livekit/livekit-server-64c47fff6c-z7j26 mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 152 | `Pod` | Pod livekit/livekit-sip-server-856f5c69d6-95bzc mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 153 | `Pod` | Pod livekit/livekit-token-server-64468cc96b-dnsft mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 154 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 155 | `Pod` | Pod mail/mail-service-7776dd9584-knhlr mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 156 | `Pod` | Pod mail/mail-service-7776dd9584-n4jrf mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 157 | `Pod` | Pod mcp/redis-7564b66579-t2ccm mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 158 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 159 | `Pod` | Pod metallb-system/controller-5ccfff46f4-v8qhh mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 160 | `Pod` | Pod metallb-system/speaker-54mx4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 161 | `Pod` | Pod metallb-system/speaker-5pmhl mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 162 | `Pod` | Pod metallb-system/speaker-ll6bv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 163 | `Pod` | Pod metallb-system/speaker-r8b5z mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 164 | `Pod` | Pod metallb-system/speaker-vggvs mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 165 | `Pod` | Pod metallb-system/speaker-z5lt6 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 166 | `Pod` | Pod minio-operator/console-558dc87767-wv86t mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 167 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-5sqzs mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 168 | `Pod` | Pod minio-operator/minio-operator-85bc587c54-tk2x9 mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 169 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 170 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 171 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 172 | `Pod` | Pod neo4j/neo4j-5d5c8669f6-s227d mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 173 | `Pod` | Pod nextcloud/nextcloud-78545bf8f8-snndw mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 174 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-667b7699fb-tv22t mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 175 | `Pod` | Pod openproject/openproject-memcached-6ff56bf694-rx4tl mounts 1 container image(s) without digest pin: memcached=img-6e51047e:tag |
| 176 | `Pod` | Pod openproject/openproject-web-dd6ddf7c7-mzvf4 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 177 | `Pod` | Pod openproject/openproject-worker-default-785bb4d78d-bnlv8 mounts 1 container image(s) without digest pin: openproject=img-328d2632:tag |
| 178 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 179 | `Pod` | Pod pg/haproxy-78c65848c-24lvz mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 180 | `Pod` | Pod pg/haproxy-78c65848c-kbjm7 mounts 1 container image(s) without digest pin: haproxy=img-cb2a3980:tag |
| 181 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 182 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 183 | `Pod` | Pod pg/postgres-minio-backup-29669940-87hks mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 184 | `Pod` | Pod pg/postgres-minio-backup-29671380-4gftn mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 185 | `Pod` | Pod pg/postgres-minio-backup-29672820-vh59q mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 186 | `Pod` | Pod pg/postgres-nfs-backup-29669880-kh794 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 187 | `Pod` | Pod pg/postgres-nfs-backup-29671320-gqkzc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 188 | `Pod` | Pod pg/postgres-nfs-backup-29672760-pjwk7 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 189 | `Pod` | Pod radar/radar-b8dcfd5df-bpbw7 mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 190 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 191 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 192 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 1 container image(s) without digest pin: redis=redis:7.2-alpine |
| 193 | `Pod` | Pod redis/redis-livekit-54c4997bfb-xtvd8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 194 | `Pod` | Pod redis/redis-proxy-56c5884f7-4gkd5 mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 195 | `Pod` | Pod redis/redis-proxy-56c5884f7-vxs9s mounts 1 container image(s) without digest pin: envoy=img-b8f88d7b:tag |
| 196 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 197 | `Pod` | Pod storethesoup/redis-6b45d66dc6-c65d2 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 198 | `Pod` | Pod storethesoup/wordpress-f87f66675-lbdnj mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 199 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 200 | `Pod` | Pod tutor/player-ui-6c677f9fd6-5d4jx mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 201 | `Pod` | Pod vc-livekit/backend-68864cd948-5nph8 mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 202 | `Pod` | Pod vc-livekit/backend-68864cd948-xnlvx mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 203 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-b5kzv mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 204 | `Pod` | Pod vc-livekit/frontend-7575ccfd65-p4d9v mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 205 | `Pod` | Pod vc-livekit/livekit-agent-764fcd7449-hsv72 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 206 | `Pod` | Pod vc-livekit/registry-846d97b78b-pkp8j mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 207 | `Pod` | Pod web/baisoln-web-5bc8b766cb-2gmpm mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 208 | `Pod` | Pod web/baisoln-web-5bc8b766cb-fr47v mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 209 | `Pod` | Pod web/contact-api-7ccbb4cfd4-knznv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 210 | `Cluster` | 41 namespace(s) have no NetworkPolicy, but CNI "flannel-only" does NOT enforce them. DaemonSet kube-flannel/kube-flannel-ds. Flannel-only (no Calico/Cilium/AWS-VPC-CNI/Azure-NPM/kube-router signal). Flannel does not enforce NetworkPolicy.. Adding NetworkPolicies here would be decorative-only. |
| 211 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 212 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 213 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 214 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 215 | `DNSChainDrift` | Ingress `d10f5d3d/0d96ec3b` routes host *host-f1ba8d59* to Service `d10f5d3d/0d96ec3b` (port http) but that Service does not exist in the cluster. |
| 216 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 217 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 218 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 219 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 220 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 221 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 222 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 223 | `DNSChainDrift` | Ingress `10182ab8/77b28987` routes host *host-32225d86* to Service `10182ab8/8bd1790c` (port 8000) but that Service does not exist in the cluster. |
| 224 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 225 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 226 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 227 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 228 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 229 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 230 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 231 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 232 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 233 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 234 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 235 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 236 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 237 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 238 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 239 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 240 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 241 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 242 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 243 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 32 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-02</strong> — 19 component(s) · 419 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | DEGRADED | 1 cluster(s): rook-ceph@rook-ceph WARN (12.7% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 95 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 32 endpoints reachable (32 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 6 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 80 KongPlugin resource(s) inspected |
| component-68fc25e4 | CRITICAL | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| Ceph Storage (ade8f28f/ade8f28f) | warning | Cluster reports HEALTH_WARN |
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-d52b37e4 | critical | HPA pg/haproxy-hpa ScalingActive=False (reason=FailedGetResourceMetric) |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole calico-tiered-policy-passthrough grants wildcard verb (verbs=[*], apiGroups=[host-092514ba], resources=[networkpolicies globalnetworkpolicies]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 22 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 23 | `ServiceAccount` | ServiceAccount calico-system/csi-node-driver is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 24 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 28 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 29 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 30 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-4zkvc requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 31 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-5vsmk requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 32 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-88qml requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 33 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-9wnzp requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 34 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-bmqf2 requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 35 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-bpb8n requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 36 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-bsffp requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 37 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-bxc4n requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 38 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-grrsk requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 39 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-jq5nq requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 40 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-ln94m requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 41 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-mqm95 requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 42 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-mvx4q requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 43 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-q7vnf requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 44 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-s52l6 requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 45 | `PersistentVolumeClaim` | PVC kasten-io/kanister-pvc-xdjbz requests storage=10737418240 but status.capacity=10Gi past 15m0s; volume expansion is stuck |
| 46 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace calico-system explicitly enforces PSS=privileged — the most-permissive profile |
| 50 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 62 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace media-services has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 72 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 76 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 77 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 78 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 79 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 80 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 81 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 82 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 83 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 84 | `Namespace` | Namespace tigera-operator explicitly enforces PSS=privileged — the most-permissive profile |
| 85 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 86 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 87 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 88 | `Namespace` | Namespace voice-studio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 89 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 90 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 91 | `Pod` | Pod agents/token-server-c5fdd6cfd-fld9q mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 92 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-845774f499-v4wzh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 93 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-6dd59c49bb-pvcvh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 94 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-5f8b86d976-85lr6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 95 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-7497bbcc6-b745d mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 96 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-7656694988-hw24t mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 97 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-f5f9bf597-wrq7l mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 98 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-94ccdd76c-bjnch mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 99 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-655c8d6b8-vj66p mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 100 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-65ff5d687-pfqxz mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 101 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-59d855755d-bpbw9 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 102 | `Pod` | Pod bionic-platform/dify-api-84c478c69d-qf6zd mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 103 | `Pod` | Pod bionic-platform/dify-plugin-daemon-5f49c6c6b-w4lq8 mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 104 | `Pod` | Pod bionic-platform/dify-sandbox-59b45f4494-gdcl4 mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 105 | `Pod` | Pod bionic-platform/dify-web-fc549b9f5-64jmv mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 106 | `Pod` | Pod bionic-platform/dify-worker-66d8546898-49w8t mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 107 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-qdd58 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 108 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-vcd49 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 109 | `Pod` | Pod calico-system/calico-kube-controllers-567df65779-9tw4k mounts 1 container image(s) without digest pin: calico-kube-controllers=img-a71fdf02:tag |
| 110 | `Pod` | Pod calico-system/calico-node-2kz79 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 111 | `Pod` | Pod calico-system/calico-node-8zhgc mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 112 | `Pod` | Pod calico-system/calico-node-lcdxv mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 113 | `Pod` | Pod calico-system/calico-node-pbt4l mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 114 | `Pod` | Pod calico-system/calico-node-pz288 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 115 | `Pod` | Pod calico-system/calico-node-qhhp8 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 116 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-dl9d2 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 117 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-kv6q4 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 118 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-pzw6z mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 119 | `Pod` | Pod calico-system/csi-node-driver-78wsq mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 120 | `Pod` | Pod calico-system/csi-node-driver-8fvss mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 121 | `Pod` | Pod calico-system/csi-node-driver-cxl2q mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 122 | `Pod` | Pod calico-system/csi-node-driver-dqlxs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 123 | `Pod` | Pod calico-system/csi-node-driver-n5tjs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 124 | `Pod` | Pod calico-system/csi-node-driver-vfk98 mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 125 | `Pod` | Pod cert-manager/cert-manager-78dccc55b5-7f8np mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 126 | `Pod` | Pod cert-manager/cert-manager-cainjector-ddc4bfbdd-p7tfj mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 127 | `Pod` | Pod cert-manager/cert-manager-webhook-6c549d4946-lr94s mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 128 | `Pod` | Pod cha-website/cha-website-55f445c785-98mrl mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 129 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-6b6b979cf8-2rnjs mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 130 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-ssrfx mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 131 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-wqbgr mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 132 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 133 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-5c5fcd78cd-hdfbj mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 134 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-79f48b654f-d4l6p mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 135 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 136 | `Pod` | Pod code/devcontainer-56dd7bcf6f-hdxxh mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 137 | `Pod` | Pod default/coder-8728a118-34a7-4b17-b807-04dfac7af2bd-5c5696867f-k828k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 138 | `Pod` | Pod default/prometheus-operator-5cd8886b4d-d67g5 mounts 2 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag, kanister-sidecar=img-973cc84e:tag |
| 139 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 140 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 141 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 142 | `Pod` | Pod gharkaam/gharkaam-redis-567db6f5d4-5bc84 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 143 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-6vwjc mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 144 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-kmslg mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 145 | `Pod` | Pod kasten-io/aggregatedapis-svc-575bd999bb-zzdw5 mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 146 | `Pod` | Pod kasten-io/auth-svc-7c566bc8d5-gs9hv mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 147 | `Pod` | Pod kasten-io/catalog-svc-596fdcbcd9-n6ts5 mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 148 | `Pod` | Pod kasten-io/controllermanager-svc-75f8bb657f-spmpf mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 149 | `Pod` | Pod kasten-io/copy-vol-data-677c2 mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 150 | `Pod` | Pod kasten-io/copy-vol-data-kxhb4 mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 151 | `Pod` | Pod kasten-io/copy-vol-data-mtg69 mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 152 | `Pod` | Pod kasten-io/copy-vol-data-nbfpj mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 153 | `Pod` | Pod kasten-io/crypto-svc-5f544c9ff5-x9jtg mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 154 | `Pod` | Pod kasten-io/dashboardbff-svc-758dbf4b58-z4bqv mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 155 | `Pod` | Pod kasten-io/data-mover-svc-flt8g mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 156 | `Pod` | Pod kasten-io/data-mover-svc-kzl95 mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 157 | `Pod` | Pod kasten-io/data-mover-svc-nh5x4 mounts 1 container image(s) without digest pin: container=img-973cc84e:tag |
| 158 | `Pod` | Pod kasten-io/executor-svc-8695797855-8hqmb mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 159 | `Pod` | Pod kasten-io/executor-svc-8695797855-bfz84 mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 160 | `Pod` | Pod kasten-io/executor-svc-8695797855-zzbvp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 161 | `Pod` | Pod kasten-io/frontend-svc-5587d84db9-xfzvb mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 162 | `Pod` | Pod kasten-io/gateway-5797fd9ddf-hggtl mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 163 | `Pod` | Pod kasten-io/jobs-svc-69656b9bbf-nw4cx mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 164 | `Pod` | Pod kasten-io/kanister-svc-65b7dff6c4-nchtt mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 165 | `Pod` | Pod kasten-io/logging-svc-57549b6b94-22wqs mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 166 | `Pod` | Pod kasten-io/metering-svc-54fbfb454d-jb454 mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 167 | `Pod` | Pod kasten-io/prometheus-server-5f8b6d7cf5-9vtlv mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 168 | `Pod` | Pod kasten-io/state-svc-fcb5d75f4-dswz9 mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 169 | `Pod` | Pod kb-system/snapshot-controller-6f58df9c-sqj4z mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 170 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-5fcbf79c9c-zjdbg mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 171 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-pndsf mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 172 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-v8rcn mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 173 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-zw5t4 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 174 | `Pod` | Pod keda/keda-add-ons-http-interceptor-6846778b66-7pvk5 mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 175 | `Pod` | Pod keda/keda-admission-webhooks-7b4b4b9657-76j6d mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 176 | `Pod` | Pod keda/keda-operator-d7cf58dbf-vtk72 mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 177 | `Pod` | Pod keda/keda-operator-metrics-apiserver-75f7bbc7f8-52n88 mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 178 | `Pod` | Pod keycloak/keycloak-0 mounts 2 container image(s) without digest pin: keycloak=img-a351cffb:tag, kanister-sidecar=img-973cc84e:tag |
| 179 | `Pod` | Pod kong/kong-kong-78587c8f46-x647j mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 180 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 181 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 182 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 183 | `Pod` | Pod langfuse/langfuse-s3-5d6c644945-hbd5m mounts 2 container image(s) without digest pin: minio=img-14773e69:tag, kanister-sidecar=img-973cc84e:tag |
| 184 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-2glcm mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 185 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-qkphl mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 186 | `Pod` | Pod langfuse/langfuse-worker-589cdbfd89-tzkkv mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 187 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 188 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 189 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 190 | `Pod` | Pod letta/letta-server-99f7fd9df-rzcln mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 191 | `Pod` | Pod letta/letta-server-99f7fd9df-xx4sw mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 192 | `Pod` | Pod letta/letta-server-99f7fd9df-zkkp6 mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 193 | `Pod` | Pod livekit-agents/flash-agent-5d94594594-rxcw7 mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 194 | `Pod` | Pod livekit/livekit-egress-74d8647f76-fc94q mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 195 | `Pod` | Pod livekit/livekit-egress-74d8647f76-zlhr7 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 196 | `Pod` | Pod livekit/livekit-server-cc77c59f6-tv7wr mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 197 | `Pod` | Pod livekit/livekit-sip-server-84b4d95b69-6vjtb mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 198 | `Pod` | Pod livekit/livekit-token-server-6fccdfffd6-fl5jz mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 199 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 200 | `Pod` | Pod mail/mail-service-78756555f8-qp24f mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 201 | `Pod` | Pod mail/mail-service-78756555f8-rw44r mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 202 | `Pod` | Pod mcp/mcp-ai-mcp-server-978566885-4pcdf mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 203 | `Pod` | Pod mcp/mcp-calculator-server-56747d68b8-6pb85 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 204 | `Pod` | Pod mcp/mcp-comfy-server-77f6d5587b-mxc8x mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 205 | `Pod` | Pod mcp/mcp-ffmpeg-server-867ff5bf96-v6d67 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 206 | `Pod` | Pod mcp/mcp-genimage-server-6777d4fc78-lv92t mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 207 | `Pod` | Pod mcp/mcp-inspector-64ddc4446d-ptpxk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 208 | `Pod` | Pod mcp/mcp-langfuse-server-7c8f954fdc-m7c6m mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 209 | `Pod` | Pod mcp/mcp-letta-server-695d7f696c-k84kd mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 210 | `Pod` | Pod mcp/mcp-mail-server-74d6bc8485-qgt4k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 211 | `Pod` | Pod mcp/mcp-meilisearch-server-657dd5c6f7-d9sdt mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 212 | `Pod` | Pod mcp/mcp-minio-server-55c55688f8-pcpc7 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 213 | `Pod` | Pod mcp/mcp-openproject-server-78756dc954-pljtx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 214 | `Pod` | Pod mcp/mcp-pdf-generator-server-ff6b9f79f-fp7j5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 215 | `Pod` | Pod mcp/mcp-postgres-server-784448d895-vbd9q mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 216 | `Pod` | Pod mcp/mcp-redis-server-ccbf9fc87-wlhd4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 217 | `Pod` | Pod mcp/redis-7b5cc855d6-2v898 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 218 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6qt42 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 219 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6thlx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 220 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-8zsxs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 221 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-9xtcs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 222 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-b9prk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 223 | `Pod` | Pod mcp/yt-helper-66c9576f55-64tt4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 224 | `Pod` | Pod media-services/scenes-worker-7749b5f855-575p2 mounts 1 container image(s) without digest pin: scenes-worker=img-ae1e1a06:tag |
| 225 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 226 | `Pod` | Pod metallb-system/controller-6fbb9f9499-nmgxq mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 227 | `Pod` | Pod metallb-system/speaker-484q9 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 228 | `Pod` | Pod metallb-system/speaker-566mf mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 229 | `Pod` | Pod metallb-system/speaker-75r4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 230 | `Pod` | Pod metallb-system/speaker-8bht4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 231 | `Pod` | Pod metallb-system/speaker-lqnvv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 232 | `Pod` | Pod metallb-system/speaker-zp66v mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 233 | `Pod` | Pod minio-operator/console-6bb586bb94-bv7zq mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 234 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-nnnbg mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 235 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-snm5c mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 236 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 237 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 238 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 239 | `Pod` | Pod neo4j/neo4j-6c75c665f9-nkhsc mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 240 | `Pod` | Pod nextcloud/nextcloud-6846455664-8tcdj mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 241 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-676b4b9644-ktkd7 mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 242 | `Pod` | Pod openproject/openproject-memcached-766d8d8d88-qph8c mounts 2 container image(s) without digest pin: memcached=img-6e51047e:tag, kanister-sidecar=img-973cc84e:tag |
| 243 | `Pod` | Pod openproject/openproject-web-846489fbf4-mwnd7 mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 244 | `Pod` | Pod openproject/openproject-worker-default-5fd9d9d68c-wtmww mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 245 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 246 | `Pod` | Pod pg/grafana-55bdbbf846-gd4fj mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 247 | `Pod` | Pod pg/grafana-59d84f686f-9cfl5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 248 | `Pod` | Pod pg/haproxy-589cbf7fb7-bwtbf mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 249 | `Pod` | Pod pg/haproxy-589cbf7fb7-gzq4x mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 250 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 251 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 252 | `Pod` | Pod pg/pgadmin-75b78585db-7jmkp mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 253 | `Pod` | Pod pg/pgadmin-cbc677d59-kq27g mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 254 | `Pod` | Pod pg/postgres-minio-backup-29671380-4gftn mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 255 | `Pod` | Pod pg/postgres-minio-backup-29672820-vh59q mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 256 | `Pod` | Pod pg/postgres-minio-backup-29674260-8b4dk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 257 | `Pod` | Pod pg/postgres-nfs-backup-29671320-gqkzc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 258 | `Pod` | Pod pg/postgres-nfs-backup-29672760-pjwk7 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 259 | `Pod` | Pod pg/postgres-nfs-backup-29674200-2mlms mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 260 | `Pod` | Pod pg/prometheus-686b96748b-7zxl9 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 261 | `Pod` | Pod pg/prometheus-746c6475d5-26mlg mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 262 | `Pod` | Pod radar/radar-58c4596675-5jjsq mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 263 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 264 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 265 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 266 | `Pod` | Pod redis/redis-livekit-79bdfcf7cd-8wcn6 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 267 | `Pod` | Pod redis/redis-proxy-746f8f8c59-bj9hl mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 268 | `Pod` | Pod redis/redis-proxy-746f8f8c59-pq6hk mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 269 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 270 | `Pod` | Pod storethesoup/redis-55cdd6df98-hk8f8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 271 | `Pod` | Pod storethesoup/wordpress-7bcc6c4d5-78sqs mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 272 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 273 | `Pod` | Pod tigera-operator/tigera-operator-6ffc76f5d-rnxlq mounts 1 container image(s) without digest pin: tigera-operator=img-b2621568:tag |
| 274 | `Pod` | Pod tutor/player-ui-5787697985-dk7tt mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 275 | `Pod` | Pod vc-livekit/backend-675cd66b9d-nhr5w mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 276 | `Pod` | Pod vc-livekit/backend-675cd66b9d-w974t mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 277 | `Pod` | Pod vc-livekit/frontend-58458fc46b-cvvpj mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 278 | `Pod` | Pod vc-livekit/frontend-58458fc46b-d5h8z mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 279 | `Pod` | Pod vc-livekit/livekit-agent-64bd77c58f-clcc6 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 280 | `Pod` | Pod vc-livekit/registry-64d9dc9bcb-mggtp mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 281 | `Pod` | Pod voice-studio/voice-studio-backend-674fb448f9-dv846 mounts 1 container image(s) without digest pin: backend=img-5107d098:tag |
| 282 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-jg8kl mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 283 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-r6q78 mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 284 | `Pod` | Pod voice-studio/voice-studio-worker-59d664986b-zjprd mounts 1 container image(s) without digest pin: worker=img-5107d098:tag |
| 285 | `Pod` | Pod web/baisoln-web-58d467899-bhb25 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 286 | `Pod` | Pod web/baisoln-web-58d467899-z9x85 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 287 | `Pod` | Pod web/contact-api-5795f9dd9c-8dtnv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 288 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace calico-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 292 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 293 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 294 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 295 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 296 | `Namespace` | Namespace comfyui runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 297 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 298 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 299 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 300 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 301 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 302 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 303 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 304 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 305 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 306 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 307 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 308 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 309 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 310 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 311 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 312 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 313 | `Namespace` | Namespace media-services runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 314 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 315 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 316 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 317 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 318 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 319 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 320 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 321 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 322 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 323 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 324 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 325 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 326 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 327 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 328 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 329 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 330 | `Namespace` | Namespace tigera-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 331 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 332 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 333 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 334 | `Namespace` | Namespace voice-studio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 335 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 336 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 337 | `Namespace` | Namespace agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 338 | `Namespace` | Namespace auth-proxy runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 339 | `Namespace` | Namespace bionic-platform runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 340 | `Namespace` | Namespace calico-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 341 | `Namespace` | Namespace cert-manager runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 342 | `Namespace` | Namespace cha-website runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 343 | `Namespace` | Namespace cluster-health-autopilot runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 344 | `Namespace` | Namespace code runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 345 | `Namespace` | Namespace comfyui runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 346 | `Namespace` | Namespace default runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 347 | `Namespace` | Namespace etcd runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 348 | `Namespace` | Namespace gharkaam runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 349 | `Namespace` | Namespace guruji runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 350 | `Namespace` | Namespace kb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 351 | `Namespace` | Namespace keda runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 352 | `Namespace` | Namespace keycloak runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 353 | `Namespace` | Namespace kong runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 354 | `Namespace` | Namespace langfuse runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 355 | `Namespace` | Namespace letta runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 356 | `Namespace` | Namespace livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 357 | `Namespace` | Namespace livekit-agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 358 | `Namespace` | Namespace local-path-storage runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 359 | `Namespace` | Namespace mail runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 360 | `Namespace` | Namespace mcp runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 361 | `Namespace` | Namespace mcp-gateway runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 362 | `Namespace` | Namespace media-services runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 363 | `Namespace` | Namespace meilisearch runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 364 | `Namespace` | Namespace metallb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 365 | `Namespace` | Namespace minio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 366 | `Namespace` | Namespace minio-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 367 | `Namespace` | Namespace miroshark runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 368 | `Namespace` | Namespace neo4j runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 369 | `Namespace` | Namespace nextcloud runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 370 | `Namespace` | Namespace nfs-provisioner runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 371 | `Namespace` | Namespace openproject runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 372 | `Namespace` | Namespace pg runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 373 | `Namespace` | Namespace radar runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 374 | `Namespace` | Namespace redis runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 375 | `Namespace` | Namespace repomind runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 376 | `Namespace` | Namespace search-infrastructure runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 377 | `Namespace` | Namespace socialx runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 378 | `Namespace` | Namespace storethesoup runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 379 | `Namespace` | Namespace tigera-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 380 | `Namespace` | Namespace tutor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 381 | `Namespace` | Namespace vc-livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 382 | `Namespace` | Namespace vc-tools runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 383 | `Namespace` | Namespace voice-studio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 384 | `Namespace` | Namespace wabuilder runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 385 | `Namespace` | Namespace web runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 386 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 387 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 388 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 389 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 390 | `DNSChainDrift` | Ingress `d10f5d3d/0d96ec3b` routes host *host-f1ba8d59* to Service `d10f5d3d/0d96ec3b` (port http) but that Service does not exist in the cluster. |
| 391 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 392 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 393 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 394 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 395 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 396 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 397 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 398 | `DNSChainDrift` | Ingress `10182ab8/20ddc429` routes host *host-32225d86* to Service `10182ab8/2f973c02` (port 8009) but that Service does not exist in the cluster. |
| 399 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 400 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 401 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 402 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 403 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 404 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 405 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 406 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 407 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 408 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 409 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 410 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 411 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 412 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 413 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 414 | `DNSChainDrift` | Ingress `899af357/fd28d191` routes host *host-1c73ef39* to Service `899af357/2a6923e2` (port 80) but that Service does not exist in the cluster. |
| 415 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 416 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 417 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 418 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 419 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 33 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-03</strong> — 19 component(s) · 398 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.8% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 79 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 32 endpoints reachable (32 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 6 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 80 KongPlugin resource(s) inspected |
| component-68fc25e4 | CRITICAL | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-d52b37e4 | critical | HPA pg/haproxy-hpa ScalingActive=False (reason=FailedGetResourceMetric) |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole calico-tiered-policy-passthrough grants wildcard verb (verbs=[*], apiGroups=[host-092514ba], resources=[networkpolicies globalnetworkpolicies]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 22 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 23 | `ServiceAccount` | ServiceAccount calico-system/csi-node-driver is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 24 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 28 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 29 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 30 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace calico-system explicitly enforces PSS=privileged — the most-permissive profile |
| 34 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 41 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 43 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 46 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace media-services has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 56 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace tigera-operator explicitly enforces PSS=privileged — the most-permissive profile |
| 69 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace voice-studio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Pod` | Pod agents/token-server-c5fdd6cfd-fld9q mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 76 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-845774f499-v4wzh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-6dd59c49bb-pvcvh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-5f8b86d976-85lr6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-7497bbcc6-b745d mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-7656694988-hw24t mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 81 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-f5f9bf597-wrq7l mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 82 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-94ccdd76c-bjnch mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 83 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-655c8d6b8-vj66p mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 84 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-65ff5d687-pfqxz mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 85 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-59d855755d-bpbw9 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 86 | `Pod` | Pod bionic-platform/dify-api-84c478c69d-qf6zd mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 87 | `Pod` | Pod bionic-platform/dify-plugin-daemon-5f49c6c6b-w4lq8 mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 88 | `Pod` | Pod bionic-platform/dify-sandbox-59b45f4494-gdcl4 mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 89 | `Pod` | Pod bionic-platform/dify-web-fc549b9f5-64jmv mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 90 | `Pod` | Pod bionic-platform/dify-worker-66d8546898-49w8t mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 91 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-qdd58 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 92 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-vcd49 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 93 | `Pod` | Pod calico-system/calico-kube-controllers-567df65779-9tw4k mounts 1 container image(s) without digest pin: calico-kube-controllers=img-a71fdf02:tag |
| 94 | `Pod` | Pod calico-system/calico-node-2kz79 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 95 | `Pod` | Pod calico-system/calico-node-8zhgc mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 96 | `Pod` | Pod calico-system/calico-node-lcdxv mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 97 | `Pod` | Pod calico-system/calico-node-pbt4l mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 98 | `Pod` | Pod calico-system/calico-node-pz288 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 99 | `Pod` | Pod calico-system/calico-node-qhhp8 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 100 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-dl9d2 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 101 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-kv6q4 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 102 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-pzw6z mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 103 | `Pod` | Pod calico-system/csi-node-driver-78wsq mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 104 | `Pod` | Pod calico-system/csi-node-driver-8fvss mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 105 | `Pod` | Pod calico-system/csi-node-driver-cxl2q mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 106 | `Pod` | Pod calico-system/csi-node-driver-dqlxs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 107 | `Pod` | Pod calico-system/csi-node-driver-n5tjs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 108 | `Pod` | Pod calico-system/csi-node-driver-vfk98 mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 109 | `Pod` | Pod cert-manager/cert-manager-78dccc55b5-7f8np mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 110 | `Pod` | Pod cert-manager/cert-manager-cainjector-ddc4bfbdd-p7tfj mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 111 | `Pod` | Pod cert-manager/cert-manager-webhook-6c549d4946-lr94s mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 112 | `Pod` | Pod cha-website/cha-website-55f445c785-98mrl mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 113 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-599cd96d55-nq8d6 mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 114 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-ssrfx mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 115 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-wqbgr mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 116 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 117 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-84c995d4bb-ftkhd mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 118 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-655c96dd5-knzdj mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 120 | `Pod` | Pod code/devcontainer-56dd7bcf6f-hdxxh mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 121 | `Pod` | Pod default/coder-8728a118-34a7-4b17-b807-04dfac7af2bd-5c5696867f-k828k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 122 | `Pod` | Pod default/prometheus-operator-5cd8886b4d-d67g5 mounts 2 container image(s) without digest pin: prometheus-operator=img-e4c18ee9:tag, kanister-sidecar=img-973cc84e:tag |
| 123 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 124 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 125 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 126 | `Pod` | Pod gharkaam/gharkaam-redis-567db6f5d4-5bc84 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 127 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-6vwjc mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 128 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-kmslg mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 129 | `Pod` | Pod kasten-io/aggregatedapis-svc-575bd999bb-zzdw5 mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 130 | `Pod` | Pod kasten-io/auth-svc-7c566bc8d5-gs9hv mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 131 | `Pod` | Pod kasten-io/catalog-svc-596fdcbcd9-n6ts5 mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 132 | `Pod` | Pod kasten-io/controllermanager-svc-75f8bb657f-spmpf mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 133 | `Pod` | Pod kasten-io/crypto-svc-5f544c9ff5-x9jtg mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 134 | `Pod` | Pod kasten-io/dashboardbff-svc-758dbf4b58-z4bqv mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 135 | `Pod` | Pod kasten-io/executor-svc-8695797855-8hqmb mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 136 | `Pod` | Pod kasten-io/executor-svc-8695797855-bfz84 mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 137 | `Pod` | Pod kasten-io/executor-svc-8695797855-zzbvp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 138 | `Pod` | Pod kasten-io/frontend-svc-5587d84db9-xfzvb mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 139 | `Pod` | Pod kasten-io/gateway-5797fd9ddf-hggtl mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 140 | `Pod` | Pod kasten-io/jobs-svc-69656b9bbf-nw4cx mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 141 | `Pod` | Pod kasten-io/kanister-svc-65b7dff6c4-nchtt mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 142 | `Pod` | Pod kasten-io/logging-svc-57549b6b94-22wqs mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 143 | `Pod` | Pod kasten-io/metering-svc-54fbfb454d-jb454 mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 144 | `Pod` | Pod kasten-io/prometheus-server-5f8b6d7cf5-9vtlv mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 145 | `Pod` | Pod kasten-io/state-svc-fcb5d75f4-dswz9 mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 146 | `Pod` | Pod kb-system/snapshot-controller-6f58df9c-sqj4z mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 147 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-5fcbf79c9c-zjdbg mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 148 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-pndsf mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 149 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-v8rcn mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 150 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-zw5t4 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 151 | `Pod` | Pod keda/keda-add-ons-http-interceptor-6846778b66-7pvk5 mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 152 | `Pod` | Pod keda/keda-admission-webhooks-7b4b4b9657-76j6d mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 153 | `Pod` | Pod keda/keda-operator-d7cf58dbf-vtk72 mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 154 | `Pod` | Pod keda/keda-operator-metrics-apiserver-75f7bbc7f8-52n88 mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 155 | `Pod` | Pod keycloak/keycloak-0 mounts 2 container image(s) without digest pin: keycloak=img-a351cffb:tag, kanister-sidecar=img-973cc84e:tag |
| 156 | `Pod` | Pod kong/kong-kong-78587c8f46-x647j mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 157 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 158 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 159 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 160 | `Pod` | Pod langfuse/langfuse-s3-5d6c644945-hbd5m mounts 2 container image(s) without digest pin: minio=img-14773e69:tag, kanister-sidecar=img-973cc84e:tag |
| 161 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-2glcm mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 162 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-qkphl mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 163 | `Pod` | Pod langfuse/langfuse-worker-589cdbfd89-tzkkv mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 164 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 165 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 166 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 167 | `Pod` | Pod letta/letta-server-99f7fd9df-rzcln mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 168 | `Pod` | Pod letta/letta-server-99f7fd9df-xx4sw mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 169 | `Pod` | Pod letta/letta-server-99f7fd9df-zkkp6 mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 170 | `Pod` | Pod livekit-agents/flash-agent-5d94594594-rxcw7 mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 171 | `Pod` | Pod livekit/livekit-egress-74d8647f76-fc94q mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 172 | `Pod` | Pod livekit/livekit-egress-74d8647f76-zlhr7 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 173 | `Pod` | Pod livekit/livekit-server-cc77c59f6-tv7wr mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 174 | `Pod` | Pod livekit/livekit-sip-server-84b4d95b69-6vjtb mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 175 | `Pod` | Pod livekit/livekit-token-server-6fccdfffd6-fl5jz mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 176 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 177 | `Pod` | Pod mail/mail-service-78756555f8-qp24f mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 178 | `Pod` | Pod mail/mail-service-78756555f8-rw44r mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 179 | `Pod` | Pod mcp/mcp-ai-mcp-server-978566885-4pcdf mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 180 | `Pod` | Pod mcp/mcp-calculator-server-56747d68b8-6pb85 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 181 | `Pod` | Pod mcp/mcp-comfy-server-77f6d5587b-mxc8x mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 182 | `Pod` | Pod mcp/mcp-ffmpeg-server-867ff5bf96-v6d67 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 183 | `Pod` | Pod mcp/mcp-genimage-server-6777d4fc78-lv92t mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 184 | `Pod` | Pod mcp/mcp-inspector-64ddc4446d-ptpxk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 185 | `Pod` | Pod mcp/mcp-langfuse-server-7c8f954fdc-m7c6m mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 186 | `Pod` | Pod mcp/mcp-letta-server-695d7f696c-k84kd mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 187 | `Pod` | Pod mcp/mcp-mail-server-74d6bc8485-qgt4k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 188 | `Pod` | Pod mcp/mcp-meilisearch-server-657dd5c6f7-d9sdt mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 189 | `Pod` | Pod mcp/mcp-minio-server-55c55688f8-pcpc7 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 190 | `Pod` | Pod mcp/mcp-openproject-server-78756dc954-pljtx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 191 | `Pod` | Pod mcp/mcp-pdf-generator-server-ff6b9f79f-fp7j5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 192 | `Pod` | Pod mcp/mcp-postgres-server-784448d895-vbd9q mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 193 | `Pod` | Pod mcp/mcp-redis-server-ccbf9fc87-wlhd4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 194 | `Pod` | Pod mcp/redis-7b5cc855d6-2v898 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 195 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6qt42 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 196 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6thlx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 197 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-8zsxs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 198 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-9xtcs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 199 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-b9prk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 200 | `Pod` | Pod mcp/yt-helper-66c9576f55-64tt4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 201 | `Pod` | Pod media-services/scenes-worker-7749b5f855-575p2 mounts 1 container image(s) without digest pin: scenes-worker=img-ae1e1a06:tag |
| 202 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 203 | `Pod` | Pod metallb-system/controller-6fbb9f9499-nmgxq mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 204 | `Pod` | Pod metallb-system/speaker-484q9 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 205 | `Pod` | Pod metallb-system/speaker-566mf mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 206 | `Pod` | Pod metallb-system/speaker-75r4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 207 | `Pod` | Pod metallb-system/speaker-8bht4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 208 | `Pod` | Pod metallb-system/speaker-lqnvv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 209 | `Pod` | Pod metallb-system/speaker-zp66v mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 210 | `Pod` | Pod minio-operator/console-6bb586bb94-bv7zq mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 211 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-nnnbg mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 212 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-snm5c mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 213 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 214 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 215 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 216 | `Pod` | Pod neo4j/neo4j-6c75c665f9-nkhsc mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 217 | `Pod` | Pod nextcloud/nextcloud-6846455664-8tcdj mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 218 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-676b4b9644-ktkd7 mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 219 | `Pod` | Pod openproject/openproject-memcached-766d8d8d88-qph8c mounts 2 container image(s) without digest pin: memcached=img-6e51047e:tag, kanister-sidecar=img-973cc84e:tag |
| 220 | `Pod` | Pod openproject/openproject-web-846489fbf4-mwnd7 mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 221 | `Pod` | Pod openproject/openproject-worker-default-5fd9d9d68c-wtmww mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 222 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 223 | `Pod` | Pod pg/grafana-55bdbbf846-gd4fj mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 224 | `Pod` | Pod pg/grafana-59d84f686f-9cfl5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 225 | `Pod` | Pod pg/haproxy-589cbf7fb7-bwtbf mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 226 | `Pod` | Pod pg/haproxy-589cbf7fb7-gzq4x mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 227 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 228 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 229 | `Pod` | Pod pg/pgadmin-75b78585db-7jmkp mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 230 | `Pod` | Pod pg/pgadmin-cbc677d59-kq27g mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 231 | `Pod` | Pod pg/postgres-minio-backup-29672820-vh59q mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 232 | `Pod` | Pod pg/postgres-minio-backup-29674260-8b4dk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 233 | `Pod` | Pod pg/postgres-minio-backup-29675700-2dxc5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 234 | `Pod` | Pod pg/postgres-nfs-backup-29672760-pjwk7 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 235 | `Pod` | Pod pg/postgres-nfs-backup-29674200-2mlms mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 236 | `Pod` | Pod pg/postgres-nfs-backup-29675640-5pc26 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 237 | `Pod` | Pod pg/prometheus-686b96748b-7zxl9 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 238 | `Pod` | Pod pg/prometheus-746c6475d5-26mlg mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 239 | `Pod` | Pod radar/radar-58c4596675-5jjsq mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 240 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 241 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 242 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 243 | `Pod` | Pod redis/redis-livekit-79bdfcf7cd-8wcn6 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 244 | `Pod` | Pod redis/redis-proxy-746f8f8c59-bj9hl mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 245 | `Pod` | Pod redis/redis-proxy-746f8f8c59-pq6hk mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 246 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 247 | `Pod` | Pod storethesoup/redis-55cdd6df98-hk8f8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 248 | `Pod` | Pod storethesoup/wordpress-7bcc6c4d5-78sqs mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 249 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 250 | `Pod` | Pod tigera-operator/tigera-operator-6ffc76f5d-rnxlq mounts 1 container image(s) without digest pin: tigera-operator=img-b2621568:tag |
| 251 | `Pod` | Pod tutor/player-ui-5787697985-dk7tt mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 252 | `Pod` | Pod vc-livekit/backend-675cd66b9d-nhr5w mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 253 | `Pod` | Pod vc-livekit/backend-675cd66b9d-w974t mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 254 | `Pod` | Pod vc-livekit/frontend-58458fc46b-cvvpj mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 255 | `Pod` | Pod vc-livekit/frontend-58458fc46b-d5h8z mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 256 | `Pod` | Pod vc-livekit/livekit-agent-64bd77c58f-clcc6 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 257 | `Pod` | Pod vc-livekit/registry-64d9dc9bcb-mggtp mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 258 | `Pod` | Pod voice-studio/voice-studio-backend-5f746fcbc9-r7qcn mounts 1 container image(s) without digest pin: backend=img-5107d098:tag |
| 259 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-jg8kl mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 260 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-r6q78 mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 261 | `Pod` | Pod voice-studio/voice-studio-worker-686d8f78dd-bzsfv mounts 1 container image(s) without digest pin: worker=img-5107d098:tag |
| 262 | `Pod` | Pod web/baisoln-web-58d467899-bhb25 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 263 | `Pod` | Pod web/baisoln-web-58d467899-z9x85 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 264 | `Pod` | Pod web/contact-api-5795f9dd9c-8dtnv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 265 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace calico-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace comfyui runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace kasten-io runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace media-services runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 292 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 293 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 294 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 295 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 296 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 297 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 298 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 299 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 300 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 301 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 302 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 303 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 304 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 305 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 306 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 307 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 308 | `Namespace` | Namespace tigera-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 309 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 310 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 311 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 312 | `Namespace` | Namespace voice-studio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 313 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 314 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 315 | `Namespace` | Namespace agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 316 | `Namespace` | Namespace auth-proxy runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 317 | `Namespace` | Namespace bionic-platform runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 318 | `Namespace` | Namespace calico-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 319 | `Namespace` | Namespace cert-manager runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 320 | `Namespace` | Namespace cha-website runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 321 | `Namespace` | Namespace cluster-health-autopilot runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 322 | `Namespace` | Namespace code runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 323 | `Namespace` | Namespace comfyui runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 324 | `Namespace` | Namespace default runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 325 | `Namespace` | Namespace etcd runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 326 | `Namespace` | Namespace gharkaam runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 327 | `Namespace` | Namespace guruji runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 328 | `Namespace` | Namespace kasten-io runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 329 | `Namespace` | Namespace kb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 330 | `Namespace` | Namespace keda runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 331 | `Namespace` | Namespace keycloak runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 332 | `Namespace` | Namespace kong runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 333 | `Namespace` | Namespace langfuse runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 334 | `Namespace` | Namespace letta runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 335 | `Namespace` | Namespace livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 336 | `Namespace` | Namespace livekit-agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 337 | `Namespace` | Namespace local-path-storage runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 338 | `Namespace` | Namespace mail runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 339 | `Namespace` | Namespace mcp runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 340 | `Namespace` | Namespace mcp-gateway runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 341 | `Namespace` | Namespace media-services runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 342 | `Namespace` | Namespace meilisearch runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 343 | `Namespace` | Namespace metallb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 344 | `Namespace` | Namespace minio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 345 | `Namespace` | Namespace minio-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 346 | `Namespace` | Namespace miroshark runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 347 | `Namespace` | Namespace neo4j runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 348 | `Namespace` | Namespace nextcloud runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 349 | `Namespace` | Namespace nfs-provisioner runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 350 | `Namespace` | Namespace openproject runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 351 | `Namespace` | Namespace pg runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 352 | `Namespace` | Namespace radar runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 353 | `Namespace` | Namespace redis runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 354 | `Namespace` | Namespace repomind runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 355 | `Namespace` | Namespace search-infrastructure runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 356 | `Namespace` | Namespace socialx runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 357 | `Namespace` | Namespace storethesoup runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 358 | `Namespace` | Namespace tigera-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 359 | `Namespace` | Namespace tutor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 360 | `Namespace` | Namespace vc-livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 361 | `Namespace` | Namespace vc-tools runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 362 | `Namespace` | Namespace voice-studio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 363 | `Namespace` | Namespace wabuilder runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 364 | `Namespace` | Namespace web runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 365 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 366 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 367 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 368 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 369 | `DNSChainDrift` | Ingress `d10f5d3d/0d96ec3b` routes host *host-f1ba8d59* to Service `d10f5d3d/0d96ec3b` (port http) but that Service does not exist in the cluster. |
| 370 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 371 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 372 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 373 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 374 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 375 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 376 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 377 | `DNSChainDrift` | Ingress `10182ab8/20ddc429` routes host *host-32225d86* to Service `10182ab8/2f973c02` (port 8009) but that Service does not exist in the cluster. |
| 378 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 379 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 380 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 381 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 382 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 383 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 384 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 385 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 386 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 387 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 388 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 389 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 390 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 391 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 392 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 393 | `DNSChainDrift` | Ingress `899af357/fd28d191` routes host *host-1c73ef39* to Service `899af357/2a6923e2` (port 80) but that Service does not exist in the cluster. |
| 394 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 395 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 396 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 397 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 398 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 33 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-04</strong> — 19 component(s) · 396 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.8% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 79 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 33 endpoints reachable (33 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 6 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 80 KongPlugin resource(s) inspected |
| component-68fc25e4 | CRITICAL | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-d52b37e4 | critical | HPA pg/haproxy-hpa ScalingActive=False (reason=FailedGetResourceMetric) |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole calico-tiered-policy-passthrough grants wildcard verb (verbs=[*], apiGroups=[host-092514ba], resources=[networkpolicies globalnetworkpolicies]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 22 | `ServiceAccount` | ServiceAccount calico-system/csi-node-driver is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 23 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 24 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 28 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 29 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 30 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace calico-system explicitly enforces PSS=privileged — the most-permissive profile |
| 34 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 41 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 43 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 46 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 47 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace media-services has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 56 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 57 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace tigera-operator explicitly enforces PSS=privileged — the most-permissive profile |
| 69 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 70 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace voice-studio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Pod` | Pod agents/token-server-c5fdd6cfd-fld9q mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 76 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-845774f499-v4wzh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-6dd59c49bb-pvcvh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-5f8b86d976-85lr6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-7497bbcc6-b745d mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-7656694988-hw24t mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 81 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-f5f9bf597-wrq7l mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 82 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-94ccdd76c-bjnch mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 83 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-655c8d6b8-vj66p mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 84 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-65ff5d687-pfqxz mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 85 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-59d855755d-bpbw9 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 86 | `Pod` | Pod bionic-platform/dify-api-84c478c69d-qf6zd mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 87 | `Pod` | Pod bionic-platform/dify-plugin-daemon-5f49c6c6b-w4lq8 mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 88 | `Pod` | Pod bionic-platform/dify-sandbox-59b45f4494-gdcl4 mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 89 | `Pod` | Pod bionic-platform/dify-web-fc549b9f5-64jmv mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 90 | `Pod` | Pod bionic-platform/dify-worker-66d8546898-49w8t mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 91 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-qdd58 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 92 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-vcd49 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 93 | `Pod` | Pod calico-system/calico-kube-controllers-567df65779-9tw4k mounts 1 container image(s) without digest pin: calico-kube-controllers=img-a71fdf02:tag |
| 94 | `Pod` | Pod calico-system/calico-node-2kz79 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 95 | `Pod` | Pod calico-system/calico-node-8zhgc mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 96 | `Pod` | Pod calico-system/calico-node-lcdxv mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 97 | `Pod` | Pod calico-system/calico-node-pbt4l mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 98 | `Pod` | Pod calico-system/calico-node-pz288 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 99 | `Pod` | Pod calico-system/calico-node-qhhp8 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 100 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-dl9d2 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 101 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-kv6q4 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 102 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-pzw6z mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 103 | `Pod` | Pod calico-system/csi-node-driver-78wsq mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 104 | `Pod` | Pod calico-system/csi-node-driver-8fvss mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 105 | `Pod` | Pod calico-system/csi-node-driver-cxl2q mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 106 | `Pod` | Pod calico-system/csi-node-driver-dqlxs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 107 | `Pod` | Pod calico-system/csi-node-driver-n5tjs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 108 | `Pod` | Pod calico-system/csi-node-driver-vfk98 mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 109 | `Pod` | Pod cert-manager/cert-manager-78dccc55b5-7f8np mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 110 | `Pod` | Pod cert-manager/cert-manager-cainjector-ddc4bfbdd-p7tfj mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 111 | `Pod` | Pod cert-manager/cert-manager-webhook-6c549d4946-lr94s mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 112 | `Pod` | Pod cha-website/cha-website-55f445c785-98mrl mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 113 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-7b87c6464b-g6n8r mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 114 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-ssrfx mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 115 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-wqbgr mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 116 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 117 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-7fbbbb6cb5-cflmw mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 118 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-5bfbff9768-4ljx2 mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 120 | `Pod` | Pod code/devcontainer-56dd7bcf6f-hdxxh mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 121 | `Pod` | Pod default/coder-8728a118-34a7-4b17-b807-04dfac7af2bd-5c5696867f-k828k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 122 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 123 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 124 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 125 | `Pod` | Pod gharkaam/gharkaam-redis-567db6f5d4-5bc84 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 126 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-6vwjc mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 127 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-kmslg mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 128 | `Pod` | Pod kasten-io/aggregatedapis-svc-575bd999bb-zzdw5 mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 129 | `Pod` | Pod kasten-io/auth-svc-7c566bc8d5-gs9hv mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 130 | `Pod` | Pod kasten-io/catalog-svc-596fdcbcd9-n6ts5 mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 131 | `Pod` | Pod kasten-io/controllermanager-svc-75f8bb657f-spmpf mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 132 | `Pod` | Pod kasten-io/crypto-svc-5f544c9ff5-x9jtg mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 133 | `Pod` | Pod kasten-io/dashboardbff-svc-758dbf4b58-z4bqv mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 134 | `Pod` | Pod kasten-io/executor-svc-8695797855-8hqmb mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 135 | `Pod` | Pod kasten-io/executor-svc-8695797855-bfz84 mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 136 | `Pod` | Pod kasten-io/executor-svc-8695797855-zzbvp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 137 | `Pod` | Pod kasten-io/frontend-svc-5587d84db9-xfzvb mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 138 | `Pod` | Pod kasten-io/gateway-5797fd9ddf-hggtl mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 139 | `Pod` | Pod kasten-io/jobs-svc-69656b9bbf-nw4cx mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 140 | `Pod` | Pod kasten-io/kanister-svc-65b7dff6c4-nchtt mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 141 | `Pod` | Pod kasten-io/logging-svc-57549b6b94-22wqs mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 142 | `Pod` | Pod kasten-io/metering-svc-54fbfb454d-jb454 mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 143 | `Pod` | Pod kasten-io/prometheus-server-5f8b6d7cf5-9vtlv mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 144 | `Pod` | Pod kasten-io/state-svc-fcb5d75f4-dswz9 mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 145 | `Pod` | Pod kb-system/snapshot-controller-6f58df9c-sqj4z mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 146 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-5fcbf79c9c-zjdbg mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 147 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-pndsf mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 148 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-v8rcn mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 149 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-zw5t4 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 150 | `Pod` | Pod keda/keda-add-ons-http-interceptor-6846778b66-7pvk5 mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 151 | `Pod` | Pod keda/keda-admission-webhooks-7b4b4b9657-76j6d mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 152 | `Pod` | Pod keda/keda-operator-d7cf58dbf-vtk72 mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 153 | `Pod` | Pod keda/keda-operator-metrics-apiserver-75f7bbc7f8-52n88 mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 154 | `Pod` | Pod keycloak/keycloak-0 mounts 2 container image(s) without digest pin: keycloak=img-a351cffb:tag, kanister-sidecar=img-973cc84e:tag |
| 155 | `Pod` | Pod kong/kong-kong-78587c8f46-x647j mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 156 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 157 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 158 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 159 | `Pod` | Pod langfuse/langfuse-s3-5d6c644945-hbd5m mounts 2 container image(s) without digest pin: minio=img-14773e69:tag, kanister-sidecar=img-973cc84e:tag |
| 160 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-2glcm mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 161 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-qkphl mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 162 | `Pod` | Pod langfuse/langfuse-worker-589cdbfd89-tzkkv mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 163 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 164 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 165 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 166 | `Pod` | Pod letta/letta-server-99f7fd9df-rzcln mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 167 | `Pod` | Pod letta/letta-server-99f7fd9df-xx4sw mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 168 | `Pod` | Pod letta/letta-server-99f7fd9df-zkkp6 mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 169 | `Pod` | Pod livekit-agents/flash-agent-5d94594594-rxcw7 mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 170 | `Pod` | Pod livekit/livekit-egress-74d8647f76-fc94q mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 171 | `Pod` | Pod livekit/livekit-egress-74d8647f76-zlhr7 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 172 | `Pod` | Pod livekit/livekit-server-cc77c59f6-tv7wr mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 173 | `Pod` | Pod livekit/livekit-sip-server-84b4d95b69-6vjtb mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 174 | `Pod` | Pod livekit/livekit-token-server-6fccdfffd6-fl5jz mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 175 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 176 | `Pod` | Pod mail/mail-service-78756555f8-qp24f mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 177 | `Pod` | Pod mail/mail-service-78756555f8-rw44r mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 178 | `Pod` | Pod mcp/mcp-ai-mcp-server-978566885-4pcdf mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 179 | `Pod` | Pod mcp/mcp-calculator-server-56747d68b8-6pb85 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 180 | `Pod` | Pod mcp/mcp-comfy-server-77f6d5587b-mxc8x mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 181 | `Pod` | Pod mcp/mcp-ffmpeg-server-867ff5bf96-v6d67 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 182 | `Pod` | Pod mcp/mcp-genimage-server-6777d4fc78-lv92t mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 183 | `Pod` | Pod mcp/mcp-inspector-64ddc4446d-ptpxk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 184 | `Pod` | Pod mcp/mcp-langfuse-server-7c8f954fdc-m7c6m mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 185 | `Pod` | Pod mcp/mcp-letta-server-695d7f696c-k84kd mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 186 | `Pod` | Pod mcp/mcp-mail-server-74d6bc8485-qgt4k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 187 | `Pod` | Pod mcp/mcp-meilisearch-server-657dd5c6f7-d9sdt mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 188 | `Pod` | Pod mcp/mcp-minio-server-55c55688f8-pcpc7 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 189 | `Pod` | Pod mcp/mcp-openproject-server-78756dc954-pljtx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 190 | `Pod` | Pod mcp/mcp-pdf-generator-server-ff6b9f79f-fp7j5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 191 | `Pod` | Pod mcp/mcp-postgres-server-784448d895-vbd9q mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 192 | `Pod` | Pod mcp/mcp-redis-server-ccbf9fc87-wlhd4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 193 | `Pod` | Pod mcp/redis-7b5cc855d6-2v898 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 194 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6qt42 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 195 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6thlx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 196 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-8zsxs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 197 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-9xtcs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 198 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-b9prk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 199 | `Pod` | Pod mcp/yt-helper-66c9576f55-64tt4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 200 | `Pod` | Pod media-services/scenes-worker-7749b5f855-575p2 mounts 1 container image(s) without digest pin: scenes-worker=img-ae1e1a06:tag |
| 201 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 202 | `Pod` | Pod metallb-system/controller-6fbb9f9499-nmgxq mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 203 | `Pod` | Pod metallb-system/speaker-484q9 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 204 | `Pod` | Pod metallb-system/speaker-566mf mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 205 | `Pod` | Pod metallb-system/speaker-75r4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 206 | `Pod` | Pod metallb-system/speaker-8bht4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 207 | `Pod` | Pod metallb-system/speaker-lqnvv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 208 | `Pod` | Pod metallb-system/speaker-zp66v mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 209 | `Pod` | Pod minio-operator/console-6bb586bb94-bv7zq mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 210 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-nnnbg mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 211 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-snm5c mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 212 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 213 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 214 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 215 | `Pod` | Pod neo4j/neo4j-6c75c665f9-nkhsc mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 216 | `Pod` | Pod nextcloud/nextcloud-6846455664-8tcdj mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 217 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-676b4b9644-ktkd7 mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 218 | `Pod` | Pod openproject/openproject-memcached-766d8d8d88-qph8c mounts 2 container image(s) without digest pin: memcached=img-6e51047e:tag, kanister-sidecar=img-973cc84e:tag |
| 219 | `Pod` | Pod openproject/openproject-web-846489fbf4-mwnd7 mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 220 | `Pod` | Pod openproject/openproject-worker-default-5fd9d9d68c-wtmww mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 221 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 222 | `Pod` | Pod pg/grafana-55bdbbf846-gd4fj mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 223 | `Pod` | Pod pg/haproxy-589cbf7fb7-bwtbf mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 224 | `Pod` | Pod pg/haproxy-589cbf7fb7-gzq4x mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 225 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 226 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 227 | `Pod` | Pod pg/pgadmin-75b78585db-7jmkp mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 228 | `Pod` | Pod pg/pgadmin-cbc677d59-kq27g mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 229 | `Pod` | Pod pg/postgres-minio-backup-29674260-8b4dk mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 230 | `Pod` | Pod pg/postgres-minio-backup-29675700-2dxc5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 231 | `Pod` | Pod pg/postgres-minio-backup-29677140-fkbpt mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 232 | `Pod` | Pod pg/postgres-nfs-backup-29674200-2mlms mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 233 | `Pod` | Pod pg/postgres-nfs-backup-29675640-5pc26 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 234 | `Pod` | Pod pg/postgres-nfs-backup-29677080-d2hqw mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 235 | `Pod` | Pod pg/prometheus-686b96748b-7zxl9 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 236 | `Pod` | Pod radar/radar-58c4596675-5jjsq mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 237 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 238 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 239 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 240 | `Pod` | Pod redis/redis-livekit-79bdfcf7cd-8wcn6 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 241 | `Pod` | Pod redis/redis-proxy-746f8f8c59-bj9hl mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 242 | `Pod` | Pod redis/redis-proxy-746f8f8c59-pq6hk mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 243 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 244 | `Pod` | Pod storethesoup/redis-55cdd6df98-hk8f8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 245 | `Pod` | Pod storethesoup/wordpress-7bcc6c4d5-78sqs mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 246 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 247 | `Pod` | Pod tigera-operator/tigera-operator-6ffc76f5d-rnxlq mounts 1 container image(s) without digest pin: tigera-operator=img-b2621568:tag |
| 248 | `Pod` | Pod tutor/player-ui-5787697985-dk7tt mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 249 | `Pod` | Pod vc-livekit/backend-675cd66b9d-nhr5w mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 250 | `Pod` | Pod vc-livekit/backend-675cd66b9d-w974t mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 251 | `Pod` | Pod vc-livekit/frontend-58458fc46b-cvvpj mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 252 | `Pod` | Pod vc-livekit/frontend-58458fc46b-d5h8z mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 253 | `Pod` | Pod vc-livekit/livekit-agent-64bd77c58f-clcc6 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 254 | `Pod` | Pod vc-livekit/registry-64d9dc9bcb-mggtp mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 255 | `Pod` | Pod voice-studio/voice-studio-backend-65589b4d78-4jl2q mounts 1 container image(s) without digest pin: backend=img-5107d098:tag |
| 256 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-jg8kl mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 257 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-r6q78 mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 258 | `Pod` | Pod voice-studio/voice-studio-worker-759fbc9545-c2g6c mounts 1 container image(s) without digest pin: worker=img-5107d098:tag |
| 259 | `Pod` | Pod web/baisoln-web-58d467899-bhb25 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 260 | `Pod` | Pod web/baisoln-web-58d467899-z9x85 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 261 | `Pod` | Pod web/contact-api-5795f9dd9c-8dtnv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 262 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 263 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 264 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 265 | `Namespace` | Namespace calico-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 266 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace comfyui runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace kasten-io runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace media-services runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 292 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 293 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 294 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 295 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 296 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 297 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 298 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 299 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 300 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 301 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 302 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 303 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 304 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 305 | `Namespace` | Namespace tigera-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 306 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 307 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 308 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 309 | `Namespace` | Namespace voice-studio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 310 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 311 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 312 | `Namespace` | Namespace agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 313 | `Namespace` | Namespace auth-proxy runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 314 | `Namespace` | Namespace bionic-platform runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 315 | `Namespace` | Namespace calico-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 316 | `Namespace` | Namespace cert-manager runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 317 | `Namespace` | Namespace cha-website runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 318 | `Namespace` | Namespace cluster-health-autopilot runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 319 | `Namespace` | Namespace code runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 320 | `Namespace` | Namespace comfyui runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 321 | `Namespace` | Namespace default runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 322 | `Namespace` | Namespace etcd runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 323 | `Namespace` | Namespace gharkaam runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 324 | `Namespace` | Namespace guruji runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 325 | `Namespace` | Namespace kasten-io runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 326 | `Namespace` | Namespace kb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 327 | `Namespace` | Namespace keda runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 328 | `Namespace` | Namespace keycloak runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 329 | `Namespace` | Namespace kong runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 330 | `Namespace` | Namespace langfuse runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 331 | `Namespace` | Namespace letta runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 332 | `Namespace` | Namespace livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 333 | `Namespace` | Namespace livekit-agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 334 | `Namespace` | Namespace local-path-storage runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 335 | `Namespace` | Namespace mail runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 336 | `Namespace` | Namespace mcp runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 337 | `Namespace` | Namespace mcp-gateway runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 338 | `Namespace` | Namespace media-services runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 339 | `Namespace` | Namespace meilisearch runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 340 | `Namespace` | Namespace metallb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 341 | `Namespace` | Namespace minio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 342 | `Namespace` | Namespace minio-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 343 | `Namespace` | Namespace miroshark runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 344 | `Namespace` | Namespace neo4j runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 345 | `Namespace` | Namespace nextcloud runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 346 | `Namespace` | Namespace nfs-provisioner runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 347 | `Namespace` | Namespace openproject runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 348 | `Namespace` | Namespace pg runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 349 | `Namespace` | Namespace radar runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 350 | `Namespace` | Namespace redis runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 351 | `Namespace` | Namespace repomind runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 352 | `Namespace` | Namespace search-infrastructure runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 353 | `Namespace` | Namespace socialx runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 354 | `Namespace` | Namespace storethesoup runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 355 | `Namespace` | Namespace tigera-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 356 | `Namespace` | Namespace tutor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 357 | `Namespace` | Namespace vc-livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 358 | `Namespace` | Namespace vc-tools runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 359 | `Namespace` | Namespace voice-studio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 360 | `Namespace` | Namespace wabuilder runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 361 | `Namespace` | Namespace web runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 362 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 363 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 364 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 365 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 366 | `DNSChainDrift` | Ingress `d10f5d3d/0d96ec3b` routes host *host-f1ba8d59* to Service `d10f5d3d/0d96ec3b` (port http) but that Service does not exist in the cluster. |
| 367 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 368 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 369 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 370 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 371 | `DNSChainDrift` | Ingress `d80dc0a2/f52b64bb` routes host *host-56420025* to Service `d80dc0a2/cace491b` (port 3000) but that Service does not exist in the cluster. |
| 372 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 373 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 374 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 375 | `DNSChainDrift` | Ingress `10182ab8/20ddc429` routes host *host-32225d86* to Service `10182ab8/2f973c02` (port 8009) but that Service does not exist in the cluster. |
| 376 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 377 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 378 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 379 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 380 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 381 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 382 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 383 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 384 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 385 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 386 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 387 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 388 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 389 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 390 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 391 | `DNSChainDrift` | Ingress `899af357/fd28d191` routes host *host-1c73ef39* to Service `899af357/2a6923e2` (port 80) but that Service does not exist in the cluster. |
| 392 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 393 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 394 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 395 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 396 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 34 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-05</strong> — 19 component(s) · 402 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.7% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 79 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 33 endpoints reachable (33 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 6 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 82 KongPlugin resource(s) inspected |
| component-68fc25e4 | CRITICAL | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-d52b37e4 | critical | HPA pg/haproxy-hpa ScalingActive=False (reason=FailedGetResourceMetric) |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole calico-tiered-policy-passthrough grants wildcard verb (verbs=[*], apiGroups=[host-092514ba], resources=[networkpolicies globalnetworkpolicies]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 22 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 23 | `ServiceAccount` | ServiceAccount calico-system/csi-node-driver is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 24 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 28 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 29 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 30 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace calico-system explicitly enforces PSS=privileged — the most-permissive profile |
| 34 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace gpu-monitor explicitly enforces PSS=privileged — the most-permissive profile |
| 41 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 43 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 47 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace media-services has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 57 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace tigera-operator explicitly enforces PSS=privileged — the most-permissive profile |
| 70 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace voice-studio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 76 | `Pod` | Pod agents/token-server-c5fdd6cfd-fld9q mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-845774f499-v4wzh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-cha-approve-59f4f77db5-pn5rc mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-6dd59c49bb-pvcvh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-5f8b86d976-85lr6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 81 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-7497bbcc6-b745d mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 82 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-7656694988-hw24t mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 83 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-f5f9bf597-wrq7l mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 84 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-94ccdd76c-bjnch mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 85 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-655c8d6b8-vj66p mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 86 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-65ff5d687-pfqxz mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 87 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-59d855755d-bpbw9 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 88 | `Pod` | Pod bionic-platform/dify-api-84c478c69d-qf6zd mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 89 | `Pod` | Pod bionic-platform/dify-plugin-daemon-5f49c6c6b-w4lq8 mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 90 | `Pod` | Pod bionic-platform/dify-sandbox-59b45f4494-gdcl4 mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 91 | `Pod` | Pod bionic-platform/dify-web-fc549b9f5-64jmv mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 92 | `Pod` | Pod bionic-platform/dify-worker-66d8546898-49w8t mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 93 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-qdd58 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 94 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-vcd49 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 95 | `Pod` | Pod calico-system/calico-kube-controllers-567df65779-9tw4k mounts 1 container image(s) without digest pin: calico-kube-controllers=img-a71fdf02:tag |
| 96 | `Pod` | Pod calico-system/calico-node-2kz79 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 97 | `Pod` | Pod calico-system/calico-node-8zhgc mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 98 | `Pod` | Pod calico-system/calico-node-lcdxv mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 99 | `Pod` | Pod calico-system/calico-node-pbt4l mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 100 | `Pod` | Pod calico-system/calico-node-pz288 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 101 | `Pod` | Pod calico-system/calico-node-qhhp8 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 102 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-dl9d2 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 103 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-kv6q4 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 104 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-pzw6z mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 105 | `Pod` | Pod calico-system/csi-node-driver-78wsq mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 106 | `Pod` | Pod calico-system/csi-node-driver-8fvss mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 107 | `Pod` | Pod calico-system/csi-node-driver-cxl2q mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 108 | `Pod` | Pod calico-system/csi-node-driver-dqlxs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 109 | `Pod` | Pod calico-system/csi-node-driver-n5tjs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 110 | `Pod` | Pod calico-system/csi-node-driver-vfk98 mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 111 | `Pod` | Pod cert-manager/cert-manager-78dccc55b5-7f8np mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 112 | `Pod` | Pod cert-manager/cert-manager-cainjector-ddc4bfbdd-p7tfj mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 113 | `Pod` | Pod cert-manager/cert-manager-webhook-6c549d4946-lr94s mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 114 | `Pod` | Pod cha-website/cha-website-55f445c785-98mrl mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 115 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-88ccfc549-znn8s mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 116 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-ssrfx mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 117 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-wqbgr mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 118 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-6856bb5947-vmn6v mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-7f797fbc8-kbtps mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 121 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 122 | `Pod` | Pod code/devcontainer-56dd7bcf6f-hdxxh mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 123 | `Pod` | Pod default/coder-8728a118-34a7-4b17-b807-04dfac7af2bd-5c5696867f-k828k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 124 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 125 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 126 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 127 | `Pod` | Pod gharkaam/gharkaam-redis-567db6f5d4-5bc84 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 128 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-6vwjc mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 129 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-kmslg mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 130 | `Pod` | Pod gpu-monitor/dcgm-exporter-4llg8 mounts 1 container image(s) without digest pin: dcgm-exporter=img-b6449f77:tag |
| 131 | `Pod` | Pod gpu-monitor/gpu-process-exporter-9xk8m mounts 1 container image(s) without digest pin: exporter=python:3.12-slim |
| 132 | `Pod` | Pod kasten-io/aggregatedapis-svc-575bd999bb-zzdw5 mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 133 | `Pod` | Pod kasten-io/auth-svc-7c566bc8d5-gs9hv mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 134 | `Pod` | Pod kasten-io/catalog-svc-596fdcbcd9-n6ts5 mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 135 | `Pod` | Pod kasten-io/controllermanager-svc-75f8bb657f-spmpf mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 136 | `Pod` | Pod kasten-io/crypto-svc-5f544c9ff5-x9jtg mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 137 | `Pod` | Pod kasten-io/dashboardbff-svc-758dbf4b58-z4bqv mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 138 | `Pod` | Pod kasten-io/executor-svc-8695797855-8hqmb mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 139 | `Pod` | Pod kasten-io/executor-svc-8695797855-bfz84 mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 140 | `Pod` | Pod kasten-io/executor-svc-8695797855-zzbvp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 141 | `Pod` | Pod kasten-io/frontend-svc-5587d84db9-xfzvb mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 142 | `Pod` | Pod kasten-io/gateway-5797fd9ddf-hggtl mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 143 | `Pod` | Pod kasten-io/jobs-svc-69656b9bbf-nw4cx mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 144 | `Pod` | Pod kasten-io/kanister-svc-65b7dff6c4-nchtt mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 145 | `Pod` | Pod kasten-io/logging-svc-57549b6b94-22wqs mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 146 | `Pod` | Pod kasten-io/metering-svc-54fbfb454d-jb454 mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 147 | `Pod` | Pod kasten-io/prometheus-server-5f8b6d7cf5-9vtlv mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 148 | `Pod` | Pod kasten-io/state-svc-fcb5d75f4-dswz9 mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 149 | `Pod` | Pod kb-system/snapshot-controller-6f58df9c-sqj4z mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 150 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-5fcbf79c9c-zjdbg mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 151 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-pndsf mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 152 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-v8rcn mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 153 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-zw5t4 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 154 | `Pod` | Pod keda/keda-add-ons-http-interceptor-6846778b66-7pvk5 mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 155 | `Pod` | Pod keda/keda-admission-webhooks-7b4b4b9657-76j6d mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 156 | `Pod` | Pod keda/keda-operator-d7cf58dbf-vtk72 mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 157 | `Pod` | Pod keda/keda-operator-metrics-apiserver-75f7bbc7f8-52n88 mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 158 | `Pod` | Pod keycloak/keycloak-0 mounts 2 container image(s) without digest pin: keycloak=img-a351cffb:tag, kanister-sidecar=img-973cc84e:tag |
| 159 | `Pod` | Pod kong/kong-kong-78587c8f46-x647j mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 160 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 161 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 162 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 163 | `Pod` | Pod langfuse/langfuse-s3-5d6c644945-hbd5m mounts 2 container image(s) without digest pin: minio=img-14773e69:tag, kanister-sidecar=img-973cc84e:tag |
| 164 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-2glcm mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 165 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-qkphl mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 166 | `Pod` | Pod langfuse/langfuse-worker-589cdbfd89-tzkkv mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 167 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 168 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 169 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 170 | `Pod` | Pod letta/letta-server-99f7fd9df-rzcln mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 171 | `Pod` | Pod letta/letta-server-99f7fd9df-xx4sw mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 172 | `Pod` | Pod letta/letta-server-99f7fd9df-zkkp6 mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 173 | `Pod` | Pod livekit-agents/flash-agent-5d94594594-rxcw7 mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 174 | `Pod` | Pod livekit/livekit-egress-74d8647f76-fc94q mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 175 | `Pod` | Pod livekit/livekit-egress-74d8647f76-zlhr7 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 176 | `Pod` | Pod livekit/livekit-server-cc77c59f6-tv7wr mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 177 | `Pod` | Pod livekit/livekit-sip-server-84b4d95b69-6vjtb mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 178 | `Pod` | Pod livekit/livekit-token-server-6fccdfffd6-fl5jz mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 179 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 180 | `Pod` | Pod mail/mail-service-78756555f8-qp24f mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 181 | `Pod` | Pod mail/mail-service-78756555f8-rw44r mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 182 | `Pod` | Pod mcp/mcp-ai-mcp-server-978566885-4pcdf mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 183 | `Pod` | Pod mcp/mcp-calculator-server-56747d68b8-6pb85 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 184 | `Pod` | Pod mcp/mcp-comfy-server-77f6d5587b-mxc8x mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 185 | `Pod` | Pod mcp/mcp-ffmpeg-server-867ff5bf96-v6d67 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 186 | `Pod` | Pod mcp/mcp-genimage-server-6777d4fc78-lv92t mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 187 | `Pod` | Pod mcp/mcp-inspector-64ddc4446d-ptpxk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 188 | `Pod` | Pod mcp/mcp-langfuse-server-7c8f954fdc-m7c6m mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 189 | `Pod` | Pod mcp/mcp-letta-server-695d7f696c-k84kd mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 190 | `Pod` | Pod mcp/mcp-mail-server-74d6bc8485-qgt4k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 191 | `Pod` | Pod mcp/mcp-meilisearch-server-657dd5c6f7-d9sdt mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 192 | `Pod` | Pod mcp/mcp-minio-server-55c55688f8-pcpc7 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 193 | `Pod` | Pod mcp/mcp-openproject-server-78756dc954-pljtx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 194 | `Pod` | Pod mcp/mcp-pdf-generator-server-ff6b9f79f-fp7j5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 195 | `Pod` | Pod mcp/mcp-postgres-server-784448d895-vbd9q mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 196 | `Pod` | Pod mcp/mcp-redis-server-ccbf9fc87-wlhd4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 197 | `Pod` | Pod mcp/redis-7b5cc855d6-2v898 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 198 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6qt42 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 199 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6thlx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 200 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-8zsxs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 201 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-9xtcs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 202 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-b9prk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 203 | `Pod` | Pod mcp/yt-helper-66c9576f55-64tt4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 204 | `Pod` | Pod media-services/scenes-worker-7749b5f855-575p2 mounts 1 container image(s) without digest pin: scenes-worker=img-ae1e1a06:tag |
| 205 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 206 | `Pod` | Pod metallb-system/controller-6fbb9f9499-nmgxq mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 207 | `Pod` | Pod metallb-system/speaker-484q9 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 208 | `Pod` | Pod metallb-system/speaker-566mf mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 209 | `Pod` | Pod metallb-system/speaker-75r4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 210 | `Pod` | Pod metallb-system/speaker-8bht4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 211 | `Pod` | Pod metallb-system/speaker-lqnvv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 212 | `Pod` | Pod metallb-system/speaker-zp66v mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 213 | `Pod` | Pod minio-operator/console-6bb586bb94-bv7zq mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 214 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-nnnbg mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 215 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-snm5c mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 216 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 217 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 218 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 219 | `Pod` | Pod neo4j/neo4j-6c75c665f9-nkhsc mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 220 | `Pod` | Pod nextcloud/nextcloud-6846455664-8tcdj mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 221 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-676b4b9644-ktkd7 mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 222 | `Pod` | Pod openproject/openproject-memcached-766d8d8d88-qph8c mounts 2 container image(s) without digest pin: memcached=img-6e51047e:tag, kanister-sidecar=img-973cc84e:tag |
| 223 | `Pod` | Pod openproject/openproject-web-846489fbf4-mwnd7 mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 224 | `Pod` | Pod openproject/openproject-worker-default-5fd9d9d68c-wtmww mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 225 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 226 | `Pod` | Pod pg/grafana-6b4d66c574-krmx2 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 227 | `Pod` | Pod pg/haproxy-589cbf7fb7-bwtbf mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 228 | `Pod` | Pod pg/haproxy-589cbf7fb7-gzq4x mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 229 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 230 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 231 | `Pod` | Pod pg/pgadmin-75b78585db-7jmkp mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 232 | `Pod` | Pod pg/pgadmin-cbc677d59-kq27g mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 233 | `Pod` | Pod pg/postgres-minio-backup-29675700-2dxc5 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 234 | `Pod` | Pod pg/postgres-minio-backup-29677140-fkbpt mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 235 | `Pod` | Pod pg/postgres-minio-backup-29678580-f97dc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 236 | `Pod` | Pod pg/postgres-nfs-backup-29675640-5pc26 mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 237 | `Pod` | Pod pg/postgres-nfs-backup-29677080-d2hqw mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 238 | `Pod` | Pod pg/postgres-nfs-backup-29678520-gr24c mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 239 | `Pod` | Pod pg/prometheus-686b96748b-7zxl9 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 240 | `Pod` | Pod radar/radar-58c4596675-5jjsq mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 241 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 242 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 243 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 244 | `Pod` | Pod redis/redis-livekit-79bdfcf7cd-8wcn6 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 245 | `Pod` | Pod redis/redis-proxy-746f8f8c59-bj9hl mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 246 | `Pod` | Pod redis/redis-proxy-746f8f8c59-pq6hk mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 247 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 248 | `Pod` | Pod storethesoup/redis-55cdd6df98-hk8f8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 249 | `Pod` | Pod storethesoup/wordpress-7bcc6c4d5-78sqs mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 250 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 251 | `Pod` | Pod tigera-operator/tigera-operator-6ffc76f5d-rnxlq mounts 1 container image(s) without digest pin: tigera-operator=img-b2621568:tag |
| 252 | `Pod` | Pod tutor/player-ui-5787697985-dk7tt mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 253 | `Pod` | Pod vc-livekit/backend-675cd66b9d-nhr5w mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 254 | `Pod` | Pod vc-livekit/backend-675cd66b9d-w974t mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 255 | `Pod` | Pod vc-livekit/frontend-58458fc46b-cvvpj mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 256 | `Pod` | Pod vc-livekit/frontend-58458fc46b-d5h8z mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 257 | `Pod` | Pod vc-livekit/livekit-agent-64bd77c58f-clcc6 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 258 | `Pod` | Pod vc-livekit/registry-64d9dc9bcb-mggtp mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 259 | `Pod` | Pod voice-studio/voice-studio-backend-65589b4d78-4jl2q mounts 1 container image(s) without digest pin: backend=img-5107d098:tag |
| 260 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-jg8kl mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 261 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-r6q78 mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 262 | `Pod` | Pod voice-studio/voice-studio-worker-759fbc9545-c2g6c mounts 1 container image(s) without digest pin: worker=img-5107d098:tag |
| 263 | `Pod` | Pod web/baisoln-web-58d467899-bhb25 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 264 | `Pod` | Pod web/baisoln-web-58d467899-z9x85 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 265 | `Pod` | Pod web/contact-api-5795f9dd9c-8dtnv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 266 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 267 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 268 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace calico-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace comfyui runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace gpu-monitor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace kasten-io runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 292 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 293 | `Namespace` | Namespace media-services runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 294 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 295 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 296 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 297 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 298 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 299 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 300 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 301 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 302 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 303 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 304 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 305 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 306 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 307 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 308 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 309 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 310 | `Namespace` | Namespace tigera-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 311 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 312 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 313 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 314 | `Namespace` | Namespace voice-studio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 315 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 316 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 317 | `Namespace` | Namespace agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 318 | `Namespace` | Namespace auth-proxy runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 319 | `Namespace` | Namespace bionic-platform runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 320 | `Namespace` | Namespace calico-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 321 | `Namespace` | Namespace cert-manager runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 322 | `Namespace` | Namespace cha-website runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 323 | `Namespace` | Namespace cluster-health-autopilot runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 324 | `Namespace` | Namespace code runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 325 | `Namespace` | Namespace comfyui runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 326 | `Namespace` | Namespace default runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 327 | `Namespace` | Namespace etcd runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 328 | `Namespace` | Namespace gharkaam runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 329 | `Namespace` | Namespace gpu-monitor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 330 | `Namespace` | Namespace guruji runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 331 | `Namespace` | Namespace kasten-io runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 332 | `Namespace` | Namespace kb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 333 | `Namespace` | Namespace keda runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 334 | `Namespace` | Namespace keycloak runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 335 | `Namespace` | Namespace kong runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 336 | `Namespace` | Namespace langfuse runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 337 | `Namespace` | Namespace letta runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 338 | `Namespace` | Namespace livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 339 | `Namespace` | Namespace livekit-agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 340 | `Namespace` | Namespace local-path-storage runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 341 | `Namespace` | Namespace mail runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 342 | `Namespace` | Namespace mcp runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 343 | `Namespace` | Namespace mcp-gateway runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 344 | `Namespace` | Namespace media-services runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 345 | `Namespace` | Namespace meilisearch runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 346 | `Namespace` | Namespace metallb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 347 | `Namespace` | Namespace minio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 348 | `Namespace` | Namespace minio-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 349 | `Namespace` | Namespace miroshark runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 350 | `Namespace` | Namespace neo4j runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 351 | `Namespace` | Namespace nextcloud runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 352 | `Namespace` | Namespace nfs-provisioner runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 353 | `Namespace` | Namespace openproject runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 354 | `Namespace` | Namespace pg runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 355 | `Namespace` | Namespace radar runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 356 | `Namespace` | Namespace redis runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 357 | `Namespace` | Namespace repomind runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 358 | `Namespace` | Namespace search-infrastructure runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 359 | `Namespace` | Namespace socialx runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 360 | `Namespace` | Namespace storethesoup runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 361 | `Namespace` | Namespace tigera-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 362 | `Namespace` | Namespace tutor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 363 | `Namespace` | Namespace vc-livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 364 | `Namespace` | Namespace vc-tools runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 365 | `Namespace` | Namespace voice-studio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 366 | `Namespace` | Namespace wabuilder runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 367 | `Namespace` | Namespace web runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 368 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 369 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 370 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 371 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 372 | `DNSChainDrift` | Ingress `d10f5d3d/562517e3` routes host *host-f1ba8d59* to Service `d10f5d3d/96a5c41b` (port 4180) but that Service does not exist in the cluster. |
| 373 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 374 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 375 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 376 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 377 | `DNSChainDrift` | Ingress `d80dc0a2/f52b64bb` routes host *host-56420025* to Service `d80dc0a2/cace491b` (port 3000) but that Service does not exist in the cluster. |
| 378 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 379 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 380 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 381 | `DNSChainDrift` | Ingress `10182ab8/aea3a068` routes host *host-32225d86* to Service `10182ab8/92f744a6` (port 8020) but that Service does not exist in the cluster. |
| 382 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 383 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 384 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 385 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 386 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 387 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 388 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 389 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 390 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 391 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 392 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 393 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 394 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 395 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 396 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 397 | `DNSChainDrift` | Ingress `899af357/fd28d191` routes host *host-1c73ef39* to Service `899af357/2a6923e2` (port 80) but that Service does not exist in the cluster. |
| 398 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 399 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 400 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 401 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 402 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 34 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

<details>
<summary><strong>2026-06-06</strong> — 19 component(s) · 404 diagnostic(s)</summary>

### Probes

| Component | Status | Detail |
|---|---|---|
| Ceph Storage | HEALTHY | 1 cluster(s): rook-ceph@rook-ceph OK (12.8% used) |
| Cluster Nodes | HEALTHY | All 6 nodes ready |
| PostgreSQL | HEALTHY | 1 CNPG cluster(s): pg-ceph@pg (2/2 ready, primary=pg-ceph-5) |
| Storage Claims | HEALTHY | All 81 PVCs bound |
| Critical Services | HEALTHY | All 0 critical services operational |
| component-a733dc9e | HEALTHY | All 33 endpoints reachable (33 auto-discovered) |
| component-6f130a4d | HEALTHY | All 6 nodes pressure-clear |
| component-35605956 | HEALTHY | All 6 system DaemonSets fully scheduled |
| component-e7e62774 | HEALTHY | No pods Pending past grace period |
| component-244066f0 | HEALTHY | No CrashLoopBackOff pods detected |
| component-09858a0e | WARNING | No in-cluster etcd pods found in kube-system (external etcd or non-kubeadm install) |
| component-514d9b4b | HEALTHY | No pods stuck on volume mount |
| component-aee58c5b | HEALTHY | 83 KongPlugin resource(s) inspected |
| component-68fc25e4 | CRITICAL | 9 HPA(s) inspected |
| component-2e83246f | HEALTHY | no Argo CD Applications |
| component-f929c3bb | HEALTHY | no Velero Backup resources |
| component-0cd84b69 | SKIPPED | Traefik CRDs not installed |
| component-b46467bf | HEALTHY | no local-path PVCs found |
| component-80741754 | HEALTHY | k3s SQLite datastore (single-node); no etcd pods expected |

### Findings

| Component | Severity | Message |
|---|---|---|
| component-09858a0e | warning | ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd. |
| component-7d31b4b6 | warning | HPA mcp-gateway/mcp-context-forge-hpa autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-d52b37e4 | critical | HPA pg/haproxy-hpa ScalingActive=False (reason=FailedGetResourceMetric) |
| component-2167a950 | warning | HPA vc-tools/agentchat autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage |
| component-80741754 | info | k3s cluster appears to use SQLite (single-node, no etcd static pods found); no HA for the datastore |

### Diagnostics

| # | Category | Message |
|---|---|---|
| 1 | `ClusterRole` | ClusterRole admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 2 | `ClusterRole` | ClusterRole calico-tiered-policy-passthrough grants wildcard verb (verbs=[*], apiGroups=[host-092514ba], resources=[networkpolicies globalnetworkpolicies]) |
| 3 | `ClusterRole` | ClusterRole cluster-owner grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 4 | `ClusterRole` | ClusterRole console-sa-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc], resources=[*]) |
| 5 | `ClusterRole` | ClusterRole k10-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-a997d3ec host-9bd66834 host-ccf5341b host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 6 | `ClusterRole` | ClusterRole k10-basic grants wildcard verb (verbs=[*], apiGroups=[host-2356746d], resources=[backupactions backupactions/details restoreactions restoreactions/details validateactions validateactions/details exportactions exportactions/details cancelactions runactions runactions/details]) |
| 7 | `ClusterRole` | ClusterRole k10-mc-admin grants wildcard verb (verbs=[*], apiGroups=[host-09e3f2f1 host-a997d3ec host-ca40aad1], resources=[*]) |
| 8 | `ClusterRole` | ClusterRole k3s-cloud-controller-manager grants wildcard verb (verbs=[*], apiGroups=[], resources=[nodes]) |
| 9 | `ClusterRole` | ClusterRole kasten-admin grants wildcard verb (verbs=[*], apiGroups=[host-2356746d host-4d6ecd8b host-09e3f2f1 host-a997d3ec host-dfd97b10 host-9bd66834 host-ca40aad1 host-ccf5341b host-fc5e354a host-fb02e51e host-4b45a737 host-95e197c2], resources=[*]) |
| 10 | `ClusterRole` | ClusterRole kasten-aggregatedapis-svc grants wildcard verb (verbs=[*], apiGroups=[], resources=[secrets]) |
| 11 | `ClusterRole` | ClusterRole local-clusterowner grants wildcard verb (verbs=[*], apiGroups=[host-fd783739], resources=[clusters]) |
| 12 | `ClusterRole` | ClusterRole local-path-provisioner-role grants wildcard verb (verbs=[*], apiGroups=[], resources=[endpoints persistentvolumes pods]) |
| 13 | `ClusterRole` | ClusterRole minio-operator grants wildcard verb (verbs=[*], apiGroups=[], resources=[*]) |
| 14 | `ClusterRole` | ClusterRole minio-operator-role grants wildcard verb (verbs=[*], apiGroups=[host-58bafcdc host-021e4405], resources=[*]) |
| 15 | `ClusterRole` | ClusterRole olm.og.global-operators.admin-5UD4U2IfBGbw51Qy2Jaefk1uawvkj2OJILlc3w grants wildcard verb (verbs=[*], apiGroups=[redis.redis.opstreelabs.in], resources=[redisreplications]) |
| 16 | `ClusterRole` | ClusterRole olm.og.olm-operators.admin-4ZLCGAP5QcGCG77n5nsv27O9w2VWNfAzuGGQ43 grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 17 | `ClusterRole` | ClusterRole p-k4z5l-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 18 | `ClusterRole` | ClusterRole p-nkvmw-namespaces-edit grants wildcard verb (verbs=[*], apiGroups=[], resources=[namespaces]) |
| 19 | `ClusterRole` | ClusterRole packagemanifests-v1-admin grants wildcard verb (verbs=[*], apiGroups=[host-2c241f60], resources=[packagemanifests]) |
| 20 | `ClusterRole` | ClusterRole prometheus-operator grants wildcard verb (verbs=[*], apiGroups=[host-3168fa50], resources=[alertmanagers alertmanagers/finalizers alertmanagers/status alertmanagerconfigs prometheuses prometheuses/finalizers prometheuses/status prometheusagents prometheusagents/finalizers prometheusagents/status thanosrulers thanosrulers/finalizers thanosrulers/status scrapeconfigs servicemonitors podmonitors probes prometheusrules]) |
| 21 | `Role` | Role kasten-admin grants wildcard verb (verbs=[*], apiGroups=[*], resources=[*]) |
| 22 | `ServiceAccount` | ServiceAccount external-secrets/external-secrets-webhook is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 23 | `ServiceAccount` | ServiceAccount calico-system/csi-node-driver is mounted by a Pod but has no RoleBinding or ClusterRoleBinding |
| 24 | `HorizontalPodAutoscaler` | HPA letta/letta-server pinned at minReplicas=3 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 25 | `HorizontalPodAutoscaler` | HPA livekit/livekit-dashboard-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 26 | `HorizontalPodAutoscaler` | HPA mcp-gateway/mcp-context-forge-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 27 | `HorizontalPodAutoscaler` | HPA pg/haproxy-hpa pinned at minReplicas=2 for >720h0m0s with maxReplicas=4 unused; HPA is not load-driven (effectively decorative) |
| 28 | `HorizontalPodAutoscaler` | HPA vc-tools/agentchat pinned at minReplicas=1 for >720h0m0s with maxReplicas=5 unused; HPA is not load-driven (effectively decorative) |
| 29 | `HorizontalPodAutoscaler` | HPA vc-tools/vc-tools pinned at minReplicas=2 for >720h0m0s with maxReplicas=10 unused; HPA is not load-driven (effectively decorative) |
| 30 | `Namespace` | Namespace agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 31 | `Namespace` | Namespace auth-proxy has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 32 | `Namespace` | Namespace bionic-platform has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 33 | `Namespace` | Namespace calico-system explicitly enforces PSS=privileged — the most-permissive profile |
| 34 | `Namespace` | Namespace cert-manager has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 35 | `Namespace` | Namespace cha-website has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 36 | `Namespace` | Namespace cluster-health-autopilot has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 37 | `Namespace` | Namespace code has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 38 | `Namespace` | Namespace default has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 39 | `Namespace` | Namespace etcd has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 40 | `Namespace` | Namespace gpu-monitor explicitly enforces PSS=privileged — the most-permissive profile |
| 41 | `Namespace` | Namespace guruji has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 42 | `Namespace` | Namespace kb-system has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 43 | `Namespace` | Namespace keda has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 44 | `Namespace` | Namespace keycloak has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 45 | `Namespace` | Namespace kong has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 46 | `Namespace` | Namespace kube-flannel explicitly enforces PSS=privileged — the most-permissive profile |
| 47 | `Namespace` | Namespace letta has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 48 | `Namespace` | Namespace livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 49 | `Namespace` | Namespace livekit-agents has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 50 | `Namespace` | Namespace local-path-storage has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 51 | `Namespace` | Namespace mail has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 52 | `Namespace` | Namespace mcp has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 53 | `Namespace` | Namespace mcp-gateway has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 54 | `Namespace` | Namespace media-services has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 55 | `Namespace` | Namespace meilisearch has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 56 | `Namespace` | Namespace metallb-system explicitly enforces PSS=privileged — the most-permissive profile |
| 57 | `Namespace` | Namespace minio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 58 | `Namespace` | Namespace minio-operator has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 59 | `Namespace` | Namespace miroshark has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 60 | `Namespace` | Namespace nextcloud has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 61 | `Namespace` | Namespace nfs-provisioner has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 62 | `Namespace` | Namespace pg has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 63 | `Namespace` | Namespace radar has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 64 | `Namespace` | Namespace redis has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 65 | `Namespace` | Namespace repomind has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 66 | `Namespace` | Namespace search-infrastructure has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 67 | `Namespace` | Namespace socialx has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 68 | `Namespace` | Namespace storethesoup has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 69 | `Namespace` | Namespace tigera-operator explicitly enforces PSS=privileged — the most-permissive profile |
| 70 | `Namespace` | Namespace tutor has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 71 | `Namespace` | Namespace vc-livekit has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 72 | `Namespace` | Namespace vc-tools has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 73 | `Namespace` | Namespace voice-studio has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 74 | `Namespace` | Namespace wabuilder has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 75 | `Namespace` | Namespace web has no host-42bc1117/enforce label; admission applies the cluster-wide default (typically privileged) |
| 76 | `Pod` | Pod agents/token-server-c5fdd6cfd-fld9q mounts 1 container image(s) without digest pin: token-server=node:18-alpine |
| 77 | `Pod` | Pod auth-proxy/oauth2-proxy-bionic-platform-845774f499-v4wzh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 78 | `Pod` | Pod auth-proxy/oauth2-proxy-cha-approve-59f4f77db5-pn5rc mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 79 | `Pod` | Pod auth-proxy/oauth2-proxy-comfyui-6dd59c49bb-pvcvh mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 80 | `Pod` | Pod auth-proxy/oauth2-proxy-dify-5f8b86d976-85lr6 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 81 | `Pod` | Pod auth-proxy/oauth2-proxy-livekit-dashboard-7497bbcc6-b745d mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 82 | `Pod` | Pod auth-proxy/oauth2-proxy-miroshark-7656694988-hw24t mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 83 | `Pod` | Pod auth-proxy/oauth2-proxy-repomind-f5f9bf597-wrq7l mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 84 | `Pod` | Pod auth-proxy/oauth2-proxy-socialx-94ccdd76c-bjnch mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 85 | `Pod` | Pod auth-proxy/oauth2-proxy-tutor-confidential-655c8d6b8-vj66p mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 86 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-livekit-65ff5d687-pfqxz mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 87 | `Pod` | Pod auth-proxy/oauth2-proxy-vc-tools-59d855755d-bpbw9 mounts 1 container image(s) without digest pin: oauth2-proxy=img-cb3f717e:tag |
| 88 | `Pod` | Pod bionic-platform/dify-api-6c4c5b5c7b-lwsrl mounts 1 container image(s) without digest pin: dify-api=img-ecb36086:tag |
| 89 | `Pod` | Pod bionic-platform/dify-plugin-daemon-5f49c6c6b-w4lq8 mounts 1 container image(s) without digest pin: plugin-daemon=img-e2e051d8:tag |
| 90 | `Pod` | Pod bionic-platform/dify-sandbox-59b45f4494-gdcl4 mounts 1 container image(s) without digest pin: dify-sandbox=img-dd019946:tag |
| 91 | `Pod` | Pod bionic-platform/dify-web-fc549b9f5-64jmv mounts 1 container image(s) without digest pin: dify-web=img-9852494f:tag |
| 92 | `Pod` | Pod bionic-platform/dify-worker-66d8546898-49w8t mounts 1 container image(s) without digest pin: dify-worker=img-ecb36086:tag |
| 93 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-qdd58 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 94 | `Pod` | Pod calico-apiserver/calico-apiserver-5ccb8577bd-vcd49 mounts 1 container image(s) without digest pin: calico-apiserver=img-5cec013e:tag |
| 95 | `Pod` | Pod calico-system/calico-kube-controllers-567df65779-9tw4k mounts 1 container image(s) without digest pin: calico-kube-controllers=img-a71fdf02:tag |
| 96 | `Pod` | Pod calico-system/calico-node-2kz79 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 97 | `Pod` | Pod calico-system/calico-node-8zhgc mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 98 | `Pod` | Pod calico-system/calico-node-lcdxv mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 99 | `Pod` | Pod calico-system/calico-node-pbt4l mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 100 | `Pod` | Pod calico-system/calico-node-pz288 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 101 | `Pod` | Pod calico-system/calico-node-qhhp8 mounts 1 container image(s) without digest pin: calico-node=img-e9bef616:tag |
| 102 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-dl9d2 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 103 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-kv6q4 mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 104 | `Pod` | Pod calico-system/calico-typha-6b7b66bf88-pzw6z mounts 1 container image(s) without digest pin: calico-typha=img-8f517df9:tag |
| 105 | `Pod` | Pod calico-system/csi-node-driver-78wsq mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 106 | `Pod` | Pod calico-system/csi-node-driver-8fvss mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 107 | `Pod` | Pod calico-system/csi-node-driver-cxl2q mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 108 | `Pod` | Pod calico-system/csi-node-driver-dqlxs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 109 | `Pod` | Pod calico-system/csi-node-driver-n5tjs mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 110 | `Pod` | Pod calico-system/csi-node-driver-vfk98 mounts 2 container image(s) without digest pin: calico-csi=img-483c6cef:tag, csi-node-driver-registrar=img-6ab0cf22:tag |
| 111 | `Pod` | Pod cert-manager/cert-manager-78dccc55b5-7f8np mounts 1 container image(s) without digest pin: cert-manager-controller=img-f8ff9f0e:tag |
| 112 | `Pod` | Pod cert-manager/cert-manager-cainjector-ddc4bfbdd-p7tfj mounts 1 container image(s) without digest pin: cert-manager-cainjector=img-d72005ed:tag |
| 113 | `Pod` | Pod cert-manager/cert-manager-webhook-6c549d4946-lr94s mounts 1 container image(s) without digest pin: cert-manager-webhook=img-f54054e7:tag |
| 114 | `Pod` | Pod cha-website/cha-website-55f445c785-98mrl mounts 1 container image(s) without digest pin: cha-website=img-22dab534:tag |
| 115 | `Pod` | Pod cluster-health-autopilot/bionic-aiwatch-6cbb975576-hg2kb mounts 1 container image(s) without digest pin: aiwatch=img-8cd780f7:tag |
| 116 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-ssrfx mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 117 | `Pod` | Pod cluster-health-autopilot/bionic-approval-server-54f4ff66bd-wqbgr mounts 1 container image(s) without digest pin: approval-server=img-8cd780f7:tag |
| 118 | `Pod` | Pod cluster-health-autopilot/bionic-rag-0 mounts 1 container image(s) without digest pin: qdrant=img-6d810a04:tag |
| 119 | `Pod` | Pod cluster-health-autopilot/bionic-watcher-6856bb5947-vmn6v mounts 1 container image(s) without digest pin: watcher=img-94908202:tag |
| 120 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-operator-7f797fbc8-kbtps mounts 1 container image(s) without digest pin: operator=img-94908202:tag |
| 121 | `Pod` | Pod cluster-health-autopilot/cha-cluster-health-autopilot-runner-9b8769976-5nc5b mounts 1 container image(s) without digest pin: runner=img-1d1d87c3:tag |
| 122 | `Pod` | Pod code/devcontainer-56dd7bcf6f-hdxxh mounts 2 container image(s) without digest pin: dev=ubuntu:24.04, dind=img-d548c5b8:tag |
| 123 | `Pod` | Pod default/coder-8728a118-34a7-4b17-b807-04dfac7af2bd-5c5696867f-k828k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 124 | `Pod` | Pod etcd/etcd-ceph-0 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 125 | `Pod` | Pod etcd/etcd-ceph-1 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 126 | `Pod` | Pod etcd/etcd-ceph-2 mounts 1 container image(s) without digest pin: etcd=img-aaa6a3c2:tag |
| 127 | `Pod` | Pod gharkaam/gharkaam-redis-567db6f5d4-5bc84 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 128 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-6vwjc mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 129 | `Pod` | Pod gharkaam/gharkaam-web-789944587f-kmslg mounts 1 container image(s) without digest pin: gharkaam=img-ce00959c:tag |
| 130 | `Pod` | Pod gpu-monitor/blackbox-external-6b47677649-nb7k7 mounts 1 container image(s) without digest pin: blackbox=img-bd3c227d:tag |
| 131 | `Pod` | Pod gpu-monitor/blackbox-origin-5c597dc5f-nqjg2 mounts 1 container image(s) without digest pin: blackbox=img-bd3c227d:tag |
| 132 | `Pod` | Pod gpu-monitor/dcgm-exporter-4llg8 mounts 1 container image(s) without digest pin: dcgm-exporter=img-b6449f77:tag |
| 133 | `Pod` | Pod gpu-monitor/gpu-process-exporter-9xk8m mounts 1 container image(s) without digest pin: exporter=python:3.12-slim |
| 134 | `Pod` | Pod gpu-monitor/grafana-78c79b59f5-4krhs mounts 1 container image(s) without digest pin: grafana=img-df4f9b9c:tag |
| 135 | `Pod` | Pod gpu-monitor/kube-state-metrics-5fb788945-g6qlf mounts 1 container image(s) without digest pin: kube-state-metrics=img-9e1cd8a9:tag |
| 136 | `Pod` | Pod kasten-io/aggregatedapis-svc-575bd999bb-zzdw5 mounts 1 container image(s) without digest pin: aggregatedapis-svc=img-b6bdc186:tag |
| 137 | `Pod` | Pod kasten-io/auth-svc-7c566bc8d5-gs9hv mounts 1 container image(s) without digest pin: auth-svc=img-fbbb51f0:tag |
| 138 | `Pod` | Pod kasten-io/catalog-svc-596fdcbcd9-n6ts5 mounts 2 container image(s) without digest pin: catalog-svc=img-a0a74c93:tag, kanister-sidecar=img-973cc84e:tag |
| 139 | `Pod` | Pod kasten-io/controllermanager-svc-75f8bb657f-spmpf mounts 1 container image(s) without digest pin: controllermanager-svc=img-24b333e4:tag |
| 140 | `Pod` | Pod kasten-io/crypto-svc-5f544c9ff5-x9jtg mounts 4 container image(s) without digest pin: crypto-svc=img-6fe0d4e6:tag, bloblifecyclemanager-svc=img-579f75ce:tag, garbagecollector-svc=img-43933de6:tag, repositories-svc=img-645ceb9a:tag |
| 141 | `Pod` | Pod kasten-io/dashboardbff-svc-758dbf4b58-z4bqv mounts 2 container image(s) without digest pin: dashboardbff-svc=img-add94ad0:tag, vbrintegrationapi-svc=img-1c7aa493:tag |
| 142 | `Pod` | Pod kasten-io/executor-svc-8695797855-8hqmb mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 143 | `Pod` | Pod kasten-io/executor-svc-8695797855-bfz84 mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 144 | `Pod` | Pod kasten-io/executor-svc-8695797855-zzbvp mounts 1 container image(s) without digest pin: executor-svc=img-3166c66d:tag |
| 145 | `Pod` | Pod kasten-io/frontend-svc-5587d84db9-xfzvb mounts 1 container image(s) without digest pin: frontend-svc=img-52c47c9e:tag |
| 146 | `Pod` | Pod kasten-io/gateway-5797fd9ddf-hggtl mounts 1 container image(s) without digest pin: gateway=img-100058ed:tag |
| 147 | `Pod` | Pod kasten-io/jobs-svc-69656b9bbf-nw4cx mounts 1 container image(s) without digest pin: jobs-svc=img-11f3880a:tag |
| 148 | `Pod` | Pod kasten-io/kanister-svc-65b7dff6c4-nchtt mounts 1 container image(s) without digest pin: kanister-svc=img-773f8d1c:tag |
| 149 | `Pod` | Pod kasten-io/logging-svc-57549b6b94-22wqs mounts 1 container image(s) without digest pin: logging-svc=img-96ac81d4:tag |
| 150 | `Pod` | Pod kasten-io/metering-svc-54fbfb454d-jb454 mounts 1 container image(s) without digest pin: metering-svc=img-6d1c011b:tag |
| 151 | `Pod` | Pod kasten-io/prometheus-server-5f8b6d7cf5-9vtlv mounts 2 container image(s) without digest pin: prometheus-server-configmap-reload=img-0bbcb73e:tag, prometheus-server=img-134afd0b:tag |
| 152 | `Pod` | Pod kasten-io/state-svc-fcb5d75f4-dswz9 mounts 2 container image(s) without digest pin: state-svc=img-eed87270:tag, events-svc=img-e78d28f8:tag |
| 153 | `Pod` | Pod kb-system/snapshot-controller-6f58df9c-sqj4z mounts 1 container image(s) without digest pin: snapshot-controller=img-e250bd1d:tag |
| 154 | `Pod` | Pod keda/keda-add-ons-http-controller-manager-5fcbf79c9c-zjdbg mounts 1 container image(s) without digest pin: keda-add-ons-http-operator=img-e7ebf4bd:tag |
| 155 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-pndsf mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 156 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-v8rcn mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 157 | `Pod` | Pod keda/keda-add-ons-http-external-scaler-f965d99b8-zw5t4 mounts 1 container image(s) without digest pin: keda-add-ons-http-external-scaler=img-d1d8f140:tag |
| 158 | `Pod` | Pod keda/keda-add-ons-http-interceptor-6846778b66-7pvk5 mounts 1 container image(s) without digest pin: keda-add-ons-http-interceptor=img-356ff8dd:tag |
| 159 | `Pod` | Pod keda/keda-admission-webhooks-7b4b4b9657-76j6d mounts 1 container image(s) without digest pin: keda-admission-webhooks=img-ea9f30f1:tag |
| 160 | `Pod` | Pod keda/keda-operator-d7cf58dbf-vtk72 mounts 1 container image(s) without digest pin: keda-operator=img-4c7ff1a2:tag |
| 161 | `Pod` | Pod keda/keda-operator-metrics-apiserver-75f7bbc7f8-52n88 mounts 1 container image(s) without digest pin: keda-operator-metrics-apiserver=img-f2a96f66:tag |
| 162 | `Pod` | Pod keycloak/keycloak-0 mounts 2 container image(s) without digest pin: keycloak=img-a351cffb:tag, kanister-sidecar=img-973cc84e:tag |
| 163 | `Pod` | Pod kong/kong-kong-78587c8f46-x647j mounts 2 container image(s) without digest pin: ingress-controller=img-b7101a2b:tag, proxy=img-28877ae8:tag |
| 164 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-0 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 165 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-1 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 166 | `Pod` | Pod langfuse/langfuse-clickhouse-shard0-2 mounts 2 container image(s) without digest pin: clickhouse=img-f72637ad:tag, kanister-sidecar=img-973cc84e:tag |
| 167 | `Pod` | Pod langfuse/langfuse-s3-5d6c644945-hbd5m mounts 2 container image(s) without digest pin: minio=img-14773e69:tag, kanister-sidecar=img-973cc84e:tag |
| 168 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-2glcm mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 169 | `Pod` | Pod langfuse/langfuse-web-5bfd5c67c-qkphl mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 170 | `Pod` | Pod langfuse/langfuse-worker-589cdbfd89-tzkkv mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 171 | `Pod` | Pod langfuse/langfuse-zookeeper-0 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 172 | `Pod` | Pod langfuse/langfuse-zookeeper-1 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 173 | `Pod` | Pod langfuse/langfuse-zookeeper-2 mounts 2 container image(s) without digest pin: zookeeper=img-eab8cce1:tag, kanister-sidecar=img-973cc84e:tag |
| 174 | `Pod` | Pod letta/letta-server-99f7fd9df-rzcln mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 175 | `Pod` | Pod letta/letta-server-99f7fd9df-xx4sw mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 176 | `Pod` | Pod letta/letta-server-99f7fd9df-zkkp6 mounts 1 container image(s) without digest pin: letta-server=img-d234e890:tag |
| 177 | `Pod` | Pod livekit-agents/flash-agent-5d94594594-rxcw7 mounts 1 container image(s) without digest pin: agent=img-f658050f:tag |
| 178 | `Pod` | Pod livekit/livekit-egress-74d8647f76-fc94q mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 179 | `Pod` | Pod livekit/livekit-egress-74d8647f76-zlhr7 mounts 1 container image(s) without digest pin: livekit-egress=img-48369a33:tag |
| 180 | `Pod` | Pod livekit/livekit-server-cc77c59f6-tv7wr mounts 1 container image(s) without digest pin: livekit-server=img-c20d64f7:tag |
| 181 | `Pod` | Pod livekit/livekit-sip-server-84b4d95b69-6vjtb mounts 1 container image(s) without digest pin: livekit-sip-server=img-4e2f040a:tag |
| 182 | `Pod` | Pod livekit/livekit-token-server-6fccdfffd6-fl5jz mounts 1 container image(s) without digest pin: token-server=img-f2eb9a07:tag |
| 183 | `Pod` | Pod local-path-storage/local-path-provisioner-57794bf4cd-f78nx mounts 1 container image(s) without digest pin: local-path-provisioner=img-48a86045:tag |
| 184 | `Pod` | Pod mail/mail-service-78756555f8-qp24f mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 185 | `Pod` | Pod mail/mail-service-78756555f8-rw44r mounts 1 container image(s) without digest pin: mail-service=img-7c154a40:tag |
| 186 | `Pod` | Pod mcp/mcp-ai-mcp-server-978566885-4pcdf mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 187 | `Pod` | Pod mcp/mcp-calculator-server-56747d68b8-6pb85 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 188 | `Pod` | Pod mcp/mcp-comfy-server-77f6d5587b-mxc8x mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 189 | `Pod` | Pod mcp/mcp-ffmpeg-server-867ff5bf96-v6d67 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 190 | `Pod` | Pod mcp/mcp-genimage-server-6777d4fc78-lv92t mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 191 | `Pod` | Pod mcp/mcp-inspector-64ddc4446d-ptpxk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 192 | `Pod` | Pod mcp/mcp-langfuse-server-7c8f954fdc-m7c6m mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 193 | `Pod` | Pod mcp/mcp-letta-server-695d7f696c-k84kd mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 194 | `Pod` | Pod mcp/mcp-mail-server-74d6bc8485-qgt4k mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 195 | `Pod` | Pod mcp/mcp-meilisearch-server-657dd5c6f7-d9sdt mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 196 | `Pod` | Pod mcp/mcp-minio-server-55c55688f8-pcpc7 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 197 | `Pod` | Pod mcp/mcp-openproject-server-78756dc954-pljtx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 198 | `Pod` | Pod mcp/mcp-pdf-generator-server-ff6b9f79f-fp7j5 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 199 | `Pod` | Pod mcp/mcp-postgres-server-784448d895-vbd9q mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 200 | `Pod` | Pod mcp/mcp-redis-server-ccbf9fc87-wlhd4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 201 | `Pod` | Pod mcp/redis-7b5cc855d6-2v898 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 202 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6qt42 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 203 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-6thlx mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 204 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-8zsxs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 205 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-9xtcs mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 206 | `Pod` | Pod mcp/search-mcp-server-747bdbf6db-b9prk mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 207 | `Pod` | Pod mcp/yt-helper-66c9576f55-64tt4 mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 208 | `Pod` | Pod media-services/scenes-worker-7749b5f855-575p2 mounts 1 container image(s) without digest pin: scenes-worker=img-ae1e1a06:tag |
| 209 | `Pod` | Pod meilisearch/meilisearch-0 mounts 1 container image(s) without digest pin: meilisearch=img-b196c46d:tag |
| 210 | `Pod` | Pod metallb-system/controller-6fbb9f9499-nmgxq mounts 1 container image(s) without digest pin: controller=img-71b010f2:tag |
| 211 | `Pod` | Pod metallb-system/speaker-484q9 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 212 | `Pod` | Pod metallb-system/speaker-566mf mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 213 | `Pod` | Pod metallb-system/speaker-75r4b mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 214 | `Pod` | Pod metallb-system/speaker-8bht4 mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 215 | `Pod` | Pod metallb-system/speaker-lqnvv mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 216 | `Pod` | Pod metallb-system/speaker-zp66v mounts 1 container image(s) without digest pin: speaker=img-5ed2c981:tag |
| 217 | `Pod` | Pod minio-operator/console-6bb586bb94-bv7zq mounts 1 container image(s) without digest pin: console=img-8285f064:tag |
| 218 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-nnnbg mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 219 | `Pod` | Pod minio-operator/minio-operator-5ccf8c86d7-snm5c mounts 1 container image(s) without digest pin: minio-operator=img-8285f064:tag |
| 220 | `Pod` | Pod minio/minio-tenant-pool-0-0 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 221 | `Pod` | Pod minio/minio-tenant-pool-0-1 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 222 | `Pod` | Pod minio/minio-tenant-pool-0-2 mounts 2 container image(s) without digest pin: minio=img-c811a0c7:tag, sidecar=img-8285f064:tag |
| 223 | `Pod` | Pod neo4j/neo4j-6c75c665f9-nkhsc mounts 1 container image(s) without digest pin: neo4j=img-13fd9e77:tag |
| 224 | `Pod` | Pod nextcloud/nextcloud-6846455664-8tcdj mounts 2 container image(s) without digest pin: nextcloud=img-a75a0c2a:tag, nextcloud-cron=img-a75a0c2a:tag |
| 225 | `Pod` | Pod nfs-provisioner/nfs-client-provisioner-676b4b9644-ktkd7 mounts 1 container image(s) without digest pin: nfs-client-provisioner=img-a483476c:tag |
| 226 | `Pod` | Pod openproject/openproject-memcached-766d8d8d88-qph8c mounts 2 container image(s) without digest pin: memcached=img-6e51047e:tag, kanister-sidecar=img-973cc84e:tag |
| 227 | `Pod` | Pod openproject/openproject-web-846489fbf4-mwnd7 mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 228 | `Pod` | Pod openproject/openproject-worker-default-5fd9d9d68c-wtmww mounts 2 container image(s) without digest pin: openproject=img-328d2632:tag, kanister-sidecar=img-973cc84e:tag |
| 229 | `Pod` | Pod pg/alertmanager-postgresql-alertmanager-0 mounts 2 container image(s) without digest pin: alertmanager=img-238e2809:tag, config-reloader=img-09aee518:tag |
| 230 | `Pod` | Pod pg/haproxy-589cbf7fb7-bwtbf mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 231 | `Pod` | Pod pg/haproxy-589cbf7fb7-gzq4x mounts 2 container image(s) without digest pin: haproxy=img-cb2a3980:tag, kanister-sidecar=img-973cc84e:tag |
| 232 | `Pod` | Pod pg/pg-ceph-5 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 233 | `Pod` | Pod pg/pg-ceph-7 mounts 1 container image(s) without digest pin: postgres=img-2fdbd549:tag |
| 234 | `Pod` | Pod pg/pgadmin-75b78585db-7jmkp mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 235 | `Pod` | Pod pg/pgadmin-cbc677d59-kq27g mounts 1 container image(s) without digest pin: kanister-sidecar=img-973cc84e:tag |
| 236 | `Pod` | Pod pg/postgres-minio-backup-29677140-fkbpt mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 237 | `Pod` | Pod pg/postgres-minio-backup-29678580-f97dc mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 238 | `Pod` | Pod pg/postgres-minio-backup-29680020-xk7hs mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 239 | `Pod` | Pod pg/postgres-nfs-backup-29677080-d2hqw mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 240 | `Pod` | Pod pg/postgres-nfs-backup-29678520-gr24c mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 241 | `Pod` | Pod pg/postgres-nfs-backup-29679960-n7vbq mounts 1 container image(s) without digest pin: postgres-backup=postgres:17 |
| 242 | `Pod` | Pod radar/radar-58c4596675-5jjsq mounts 1 container image(s) without digest pin: radar=img-7c18e752:tag |
| 243 | `Pod` | Pod redis/redis-cluster-ceph-0 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 244 | `Pod` | Pod redis/redis-cluster-ceph-1 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 245 | `Pod` | Pod redis/redis-cluster-ceph-2 mounts 2 container image(s) without digest pin: redis=redis:7.2-alpine, kanister-sidecar=img-973cc84e:tag |
| 246 | `Pod` | Pod redis/redis-livekit-79bdfcf7cd-8wcn6 mounts 2 container image(s) without digest pin: redis=redis:7-alpine, kanister-sidecar=img-973cc84e:tag |
| 247 | `Pod` | Pod redis/redis-proxy-746f8f8c59-bj9hl mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 248 | `Pod` | Pod redis/redis-proxy-746f8f8c59-pq6hk mounts 2 container image(s) without digest pin: envoy=img-b8f88d7b:tag, kanister-sidecar=img-973cc84e:tag |
| 249 | `Pod` | Pod storethesoup/mariadb-0 mounts 1 container image(s) without digest pin: mariadb=img-e08f4c9c:tag |
| 250 | `Pod` | Pod storethesoup/redis-55cdd6df98-hk8f8 mounts 1 container image(s) without digest pin: redis=redis:7-alpine |
| 251 | `Pod` | Pod storethesoup/wordpress-7bcc6c4d5-78sqs mounts 1 container image(s) without digest pin: wordpress=img-e9c0ca1e:tag |
| 252 | `Pod` | Pod storethesoup/wp-loader mounts 1 container image(s) without digest pin: loader=alpine:3.20 |
| 253 | `Pod` | Pod tigera-operator/tigera-operator-6ffc76f5d-rnxlq mounts 1 container image(s) without digest pin: tigera-operator=img-b2621568:tag |
| 254 | `Pod` | Pod tutor/player-ui-5787697985-dk7tt mounts 1 container image(s) without digest pin: player-ui=img-3cff2a31:tag |
| 255 | `Pod` | Pod vc-livekit/backend-675cd66b9d-nhr5w mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 256 | `Pod` | Pod vc-livekit/backend-675cd66b9d-w974t mounts 1 container image(s) without digest pin: backend=img-56bc67bf:tag |
| 257 | `Pod` | Pod vc-livekit/frontend-58458fc46b-cvvpj mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 258 | `Pod` | Pod vc-livekit/frontend-58458fc46b-d5h8z mounts 1 container image(s) without digest pin: frontend=img-5e9d5a78:tag |
| 259 | `Pod` | Pod vc-livekit/livekit-agent-64bd77c58f-clcc6 mounts 1 container image(s) without digest pin: livekit-agent=img-93275bff:tag |
| 260 | `Pod` | Pod vc-livekit/registry-64d9dc9bcb-mggtp mounts 1 container image(s) without digest pin: registry=img-872491a3:tag |
| 261 | `Pod` | Pod voice-studio/voice-studio-backend-65589b4d78-4jl2q mounts 1 container image(s) without digest pin: backend=img-5107d098:tag |
| 262 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-jg8kl mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 263 | `Pod` | Pod voice-studio/voice-studio-frontend-6d496969-r6q78 mounts 1 container image(s) without digest pin: frontend=img-31a07165:tag |
| 264 | `Pod` | Pod voice-studio/voice-studio-worker-759fbc9545-c2g6c mounts 1 container image(s) without digest pin: worker=img-5107d098:tag |
| 265 | `Pod` | Pod web/baisoln-web-58d467899-bhb25 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 266 | `Pod` | Pod web/baisoln-web-58d467899-z9x85 mounts 1 container image(s) without digest pin: web=img-fde54743:tag |
| 267 | `Pod` | Pod web/contact-api-5795f9dd9c-8dtnv mounts 1 container image(s) without digest pin: api=img-5192394b:tag |
| 268 | `Namespace` | Namespace agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 269 | `Namespace` | Namespace auth-proxy runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 270 | `Namespace` | Namespace bionic-platform runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 271 | `Namespace` | Namespace calico-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 272 | `Namespace` | Namespace cert-manager runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 273 | `Namespace` | Namespace cha-website runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 274 | `Namespace` | Namespace cluster-health-autopilot runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 275 | `Namespace` | Namespace code runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 276 | `Namespace` | Namespace comfyui runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 277 | `Namespace` | Namespace default runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 278 | `Namespace` | Namespace etcd runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 279 | `Namespace` | Namespace gharkaam runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 280 | `Namespace` | Namespace gpu-monitor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 281 | `Namespace` | Namespace guruji runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 282 | `Namespace` | Namespace kasten-io runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 283 | `Namespace` | Namespace kb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 284 | `Namespace` | Namespace keda runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 285 | `Namespace` | Namespace keycloak runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 286 | `Namespace` | Namespace kong runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 287 | `Namespace` | Namespace langfuse runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 288 | `Namespace` | Namespace letta runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 289 | `Namespace` | Namespace livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 290 | `Namespace` | Namespace livekit-agents runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 291 | `Namespace` | Namespace local-path-storage runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 292 | `Namespace` | Namespace mail runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 293 | `Namespace` | Namespace mcp runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 294 | `Namespace` | Namespace mcp-gateway runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 295 | `Namespace` | Namespace media-services runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 296 | `Namespace` | Namespace meilisearch runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 297 | `Namespace` | Namespace metallb-system runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 298 | `Namespace` | Namespace minio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 299 | `Namespace` | Namespace minio-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 300 | `Namespace` | Namespace miroshark runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 301 | `Namespace` | Namespace neo4j runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 302 | `Namespace` | Namespace nextcloud runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 303 | `Namespace` | Namespace nfs-provisioner runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 304 | `Namespace` | Namespace openproject runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 305 | `Namespace` | Namespace pg runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 306 | `Namespace` | Namespace radar runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 307 | `Namespace` | Namespace redis runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 308 | `Namespace` | Namespace repomind runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 309 | `Namespace` | Namespace search-infrastructure runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 310 | `Namespace` | Namespace socialx runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 311 | `Namespace` | Namespace storethesoup runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 312 | `Namespace` | Namespace tigera-operator runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 313 | `Namespace` | Namespace tutor runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 314 | `Namespace` | Namespace vc-livekit runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 315 | `Namespace` | Namespace vc-tools runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 316 | `Namespace` | Namespace voice-studio runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 317 | `Namespace` | Namespace wabuilder runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 318 | `Namespace` | Namespace web runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide |
| 319 | `Namespace` | Namespace agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 320 | `Namespace` | Namespace auth-proxy runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 321 | `Namespace` | Namespace bionic-platform runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 322 | `Namespace` | Namespace calico-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 323 | `Namespace` | Namespace cert-manager runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 324 | `Namespace` | Namespace cha-website runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 325 | `Namespace` | Namespace cluster-health-autopilot runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 326 | `Namespace` | Namespace code runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 327 | `Namespace` | Namespace comfyui runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 328 | `Namespace` | Namespace default runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 329 | `Namespace` | Namespace etcd runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 330 | `Namespace` | Namespace gharkaam runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 331 | `Namespace` | Namespace gpu-monitor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 332 | `Namespace` | Namespace guruji runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 333 | `Namespace` | Namespace kasten-io runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 334 | `Namespace` | Namespace kb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 335 | `Namespace` | Namespace keda runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 336 | `Namespace` | Namespace keycloak runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 337 | `Namespace` | Namespace kong runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 338 | `Namespace` | Namespace langfuse runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 339 | `Namespace` | Namespace letta runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 340 | `Namespace` | Namespace livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 341 | `Namespace` | Namespace livekit-agents runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 342 | `Namespace` | Namespace local-path-storage runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 343 | `Namespace` | Namespace mail runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 344 | `Namespace` | Namespace mcp runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 345 | `Namespace` | Namespace mcp-gateway runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 346 | `Namespace` | Namespace media-services runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 347 | `Namespace` | Namespace meilisearch runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 348 | `Namespace` | Namespace metallb-system runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 349 | `Namespace` | Namespace minio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 350 | `Namespace` | Namespace minio-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 351 | `Namespace` | Namespace miroshark runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 352 | `Namespace` | Namespace neo4j runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 353 | `Namespace` | Namespace nextcloud runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 354 | `Namespace` | Namespace nfs-provisioner runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 355 | `Namespace` | Namespace openproject runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 356 | `Namespace` | Namespace pg runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 357 | `Namespace` | Namespace radar runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 358 | `Namespace` | Namespace redis runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 359 | `Namespace` | Namespace repomind runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 360 | `Namespace` | Namespace search-infrastructure runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 361 | `Namespace` | Namespace socialx runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 362 | `Namespace` | Namespace storethesoup runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 363 | `Namespace` | Namespace tigera-operator runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace |
| 364 | `Namespace` | Namespace tutor runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 365 | `Namespace` | Namespace vc-livekit runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 366 | `Namespace` | Namespace vc-tools runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 367 | `Namespace` | Namespace voice-studio runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 368 | `Namespace` | Namespace wabuilder runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 369 | `Namespace` | Namespace web runs pods on a calico cluster (NetPol enforced) but has zero NetworkPolicies. Proposed: default-deny ingress + allow-from same namespace + allow-from kong (Ingress controllers routing into this ns) |
| 370 | `DNSChainDrift` | Ingress `649e263a/649e263a` routes host *host-647db09d* to Service `649e263a/649e263a` (port 80) but that Service does not exist in the cluster. |
| 371 | `DNSChainDrift` | Ingress `d63f4a0c/ef143c54` routes host *host-bacbe0e8* to Service `d63f4a0c/d63f4a0c` (port 80) but that Service does not exist in the cluster. |
| 372 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-6580714c* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 373 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-f039a048* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 374 | `DNSChainDrift` | Ingress `d10f5d3d/562517e3` routes host *host-f1ba8d59* to Service `d10f5d3d/96a5c41b` (port 4180) but that Service does not exist in the cluster. |
| 375 | `DNSChainDrift` | Ingress `42233297/40b33b89` routes host *host-df442be8* to Service `42233297/d98b1c8a` (port 4180) but that Service does not exist in the cluster. |
| 376 | `DNSChainDrift` | Ingress `25bf6a1d/6750a43a` routes host *host-3b05cb67* to Service `25bf6a1d/93bf22ed` (port 4180) but that Service does not exist in the cluster. |
| 377 | `DNSChainDrift` | Ingress `7b498b2d/235df681` routes host *host-b9f5e313* to Service `7b498b2d/950ecc2c` (port 5001) but that Service does not exist in the cluster. |
| 378 | `DNSChainDrift` | Ingress `83ac4576/70054f71` routes host *gharkaam.in* to Service `83ac4576/70054f71` (port http) but that Service does not exist in the cluster. |
| 379 | `DNSChainDrift` | Ingress `9c5b7a1d/f52b64bb` routes host *host-56420025* to Service `9c5b7a1d/cace491b` (port 3000) but that Service does not exist in the cluster. |
| 380 | `DNSChainDrift` | Ingress `d6bed788/ff95dd66` routes host *host-e5673458* to Service `d6bed788/41baa505` (port 3000) but that Service does not exist in the cluster. |
| 381 | `DNSChainDrift` | Ingress `6c8f4e88/3354c864` routes host *host-da567b3a* to Service `6c8f4e88/3354c864` (port http-api) but that Service does not exist in the cluster. |
| 382 | `DNSChainDrift` | Ingress `00d8d3f1/3aa97943` routes host *host-92b1cecb* to Service `00d8d3f1/03b41178` (port 80) but that Service does not exist in the cluster. |
| 383 | `DNSChainDrift` | Ingress `10182ab8/aea3a068` routes host *host-32225d86* to Service `10182ab8/92f744a6` (port 8020) but that Service does not exist in the cluster. |
| 384 | `DNSChainDrift` | Ingress `6d7f0086/5ff9b09b` routes host *host-81ab186c* to Service `6d7f0086/9113250d` (port 8080) but that Service does not exist in the cluster. |
| 385 | `DNSChainDrift` | Ingress `10f9fce6/26a1f8bb` routes host *host-d63bb08e* to Service `10f9fce6/2ec52f7a` (port 4180) but that Service does not exist in the cluster. |
| 386 | `DNSChainDrift` | Ingress `a2a1e69c/a2a1e69c` routes host *host-5a4ef2ea* to Service `a2a1e69c/a2a1e69c` (port 8080) but that Service does not exist in the cluster. |
| 387 | `DNSChainDrift` | Ingress `7f8e2ea7/7d3bb9cc` routes host *host-0ccdb59e* to Service `7f8e2ea7/7f8e2ea7` (port http) but that Service does not exist in the cluster. |
| 388 | `DNSChainDrift` | Ingress `d80dc0a2/a2b0bfbb` routes host *host-29bd8929* to Service `d80dc0a2/12af3905` (port 80) but that Service does not exist in the cluster. |
| 389 | `DNSChainDrift` | Ingress `7b498b2d/7b498b2d` routes host *host-49116b44* to Service `7b498b2d/7b498b2d` (port 80) but that Service does not exist in the cluster. |
| 390 | `DNSChainDrift` | Ingress `47c88e9e/975df461` routes host *host-4e3d9acc* to Service `47c88e9e/4d0fcefe` (port 4180) but that Service does not exist in the cluster. |
| 391 | `DNSChainDrift` | Ingress `5791b622/b2246b4d` routes host *host-2249606b* to Service `5791b622/5791b622` (port 80) but that Service does not exist in the cluster. |
| 392 | `DNSChainDrift` | Ingress `606299b2/7f3605e0` routes host *host-ca5821c0* to Service `606299b2/8a17086a` (port 4180) but that Service does not exist in the cluster. |
| 393 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-271e2cd1* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 394 | `DNSChainDrift` | Ingress `25bf6a1d/7f9e9e02` routes host *host-d947e194* to Service `25bf6a1d/7f9e9e02` (port 3000) but that Service does not exist in the cluster. |
| 395 | `DNSChainDrift` | Ingress `038740ef/18bb7265` routes host *host-bda455e8* to Service `038740ef/c8918531` (port 4180) but that Service does not exist in the cluster. |
| 396 | `DNSChainDrift` | Ingress `e6f0a1fb/83d21ef8` routes host *host-eb0db2a5* to Service `e6f0a1fb/e6f0a1fb` (port 8200) but that Service does not exist in the cluster. |
| 397 | `DNSChainDrift` | Ingress `0ec4366c/67b36f81` routes host *host-a214c828* to Service `0ec4366c/07241358` (port 4180) but that Service does not exist in the cluster. |
| 398 | `DNSChainDrift` | Ingress `3d69f4a0/3d69f4a0` routes host *host-238e6042* to Service `3d69f4a0/a2909668` (port 4180) but that Service does not exist in the cluster. |
| 399 | `DNSChainDrift` | Ingress `899af357/fd28d191` routes host *host-1c73ef39* to Service `899af357/2a6923e2` (port 80) but that Service does not exist in the cluster. |
| 400 | `DNSChainDrift` | Ingress `92b6ff2d/3caa6611` routes host *host-9b16de12* to Service `92b6ff2d/7faa7ec4` (port 8080) but that Service does not exist in the cluster. |
| 401 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-ec2da35b* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 402 | `DNSChainDrift` | Ingress `4b5e57f6/a95e8ed5` routes host *host-064f5f1e* to Service `4b5e57f6/e1b60c97` (port http) but that Service does not exist in the cluster. |
| 403 | `DNSChainDrift` | Ingress `06024ae9/06024ae9` routes host *host-42d68119* to Service `06024ae9/576473d6` (port 80) but that Service does not exist in the cluster. |
| 404 | `DNSChainDrift` | Cloudflare credentials not configured; external DNS hop not checked for 34 host(s). Set `CHA_CLOUDFLARE_API_TOKEN` (and optionally `CHA_CLOUDFLARE_ZONE_ID`) to enable the full DNS-chain analysis including the Cloudflare layer. |

</details>

---
_All namespace, workload, and secret names are anonymized using deterministic SHA-256 hashing._
_cha version(s) in this dataset: cluster-health-autopilot-0.9.1-4-g66c47e8, cluster-health-autopilot-0.9.1-5-g665a915, cluster-health-autopilot-1.18.3-1-gc0f1dc5, cluster-health-autopilot-1.4.0, cluster-health-autopilot-1.6.0, cluster-health-autopilot-1.8.0-1-g0dcdb96, cluster-health-autopilot-1.8.10, cluster-health-autopilot-1.8.12-16-g76748f8, cluster-health-autopilot-1.8.8, v1.11.1, v1.15.0, v1.17.0, v1.18.2, v1.19.0, v1.5.2-1-g1e93148, v1.5.2-3-g08ba6f9, v1.6.2-1-gf3bd85c_
