# Cluster Health Autopilot — Run Summary

_Auto-generated 2026-05-31 06:27 UTC · 27 run(s) · 2026-05-04 → 2026-05-30_

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

## Diagnostic patterns (top categories, anonymized)

| Category | Occurrences |
|---|---|
| `Pod` | 592 |
| `Namespace` | 387 |
| `ClusterRole` | 108 |
| `ServiceAccount` | 36 |
| `HorizontalPodAutoscaler` | 21 |
| `missing-secret` | 14 |
| `unprovisioned` | 14 |
| `cert-expiry` | 9 |
| `ExternalSecret` | 7 |
| `missing-key` | 7 |

## Component findings (top, anonymized)

| Severity/Component | Occurrences |
|---|---|
| `warning/component-a733dc9e` | 10 |
| `warning/component-68fc25e4` | 9 |
| `warning/component-09858a0e` | 5 |
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

---
_All namespace, workload, and secret names are anonymized using deterministic SHA-256 hashing._
_cha version(s) in this dataset: cluster-health-autopilot-0.9.1-4-g66c47e8, cluster-health-autopilot-0.9.1-5-g665a915, cluster-health-autopilot-1.4.0, cluster-health-autopilot-1.6.0, cluster-health-autopilot-1.8.0-1-g0dcdb96, cluster-health-autopilot-1.8.10, cluster-health-autopilot-1.8.12-16-g76748f8, cluster-health-autopilot-1.8.8, v1.5.2-1-g1e93148, v1.5.2-3-g08ba6f9, v1.6.2-1-gf3bd85c_
