# agentic-sre

Helm chart for [Agentic SRE](https://github.com/srenix-ai/agentic-sre) — the in-cluster install of `srenix`.

Deploys two CronJobs (one is opt-in):

1. **diagnose** (always on): `srenix diagnose --live` runs probes + analyzers on a schedule, optionally posts to Slack.
2. **remediate** (opt-in): `srenix remediate --live` runs the whitelisted auto-fixers. Off by default — turn on once your team trusts the catalog.

## Install

```sh
helm repo add srenix https://srenix-ai.github.io/agentic-sre
helm repo update
helm install srenix srenix/agentic-sre \
  --namespace agentic-sre --create-namespace
```

The chart's default image pulls from Docker Hub:
`docker4zerocool/agentic-sre`. Operators who prefer GHCR can
switch via `--set image.repository=ghcr.io/srenix-ai/agentic-sre`
— both registries receive every release from the same GoReleaser pipeline.

Pre-launch / from local source:

```sh
git clone https://github.com/srenix-ai/agentic-sre
cd agentic-sre
helm install srenix charts/agentic-sre \
  --namespace agentic-sre --create-namespace
```

## With Slack

Create a Secret holding your incoming-webhook URL (preferred: ESO from a Vault path):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: srenix-slack-webhook
  namespace: agentic-sre
type: Opaque
data:
  WEBHOOK_URL: <base64 of https://hooks.slack.com/services/...>
```

Then enable Slack:

```sh
helm upgrade srenix charts/agentic-sre \
  --reuse-values \
  --set slack.enabled=true \
  --set slack.webhookSecretName=srenix-slack-webhook
```

## Turn on auto-remediation

```sh
helm upgrade srenix charts/agentic-sre \
  --reuse-values \
  --set remediation.enabled=true
```

This adds a second CronJob (`*-remediate`) and grants the narrow **remediator** ClusterRole (pods/delete, jobs/delete, deployments/patch — nothing else). Start with `remediation.dryRun=true` to see what it would do without mutating anything.

## Run on demand

```sh
kubectl create job --from=cronjob/srenix-diagnose -n agentic-sre srenix-now
kubectl logs -f -n agentic-sre job/srenix-now
```

## Values

| Key | Default | Notes |
|---|---|---|
| `image.repository` | `docker4zerocool/agentic-sre` | GHCR mirror at `ghcr.io/srenix-ai/agentic-sre` is published by GoReleaser on every release. |
| `image.tag` | `""` (uses `Chart.appVersion`) | Override per-deployment |
| `diagnose.enabled` | `true` | |
| `diagnose.schedule` | `0 9 * * *` | Daily 09:00 UTC |
| `diagnose.format` | `text` | or `json` for fleet-console pipelines |
| `remediation.enabled` | `false` | Opt-in |
| `remediation.schedule` | `*/30 * * * *` | |
| `remediation.dryRun` | `false` | When true, fixers report Refused without acting |
| `slack.enabled` | `false` | |
| `slack.webhookSecretName` | `""` | Pre-existing Secret name |
| `slack.webhookSecretKey` | `WEBHOOK_URL` | Key inside the Secret |
| `rbac.create` | `true` | |
| `serviceAccount.create` | `true` | |
| `resources.{requests,limits}` | small | <100 MB / negligible CPU |
| `podSecurityContext` | runAsNonRoot, runAsUser=65532 | |
| `securityContext` | readOnlyRootFilesystem, no caps | |

See [`values.yaml`](values.yaml) for the full schema.

## RBAC scope

The chart creates two ClusterRoles, with intentionally tight scope:

**reader** (always created):
- `get,list` on pods, nodes, pvcs, events, namespaces, deployments, replicasets, statefulsets, daemonsets, jobs, cronjobs, externalsecrets, postgresql.cnpg.io/clusters, ceph.rook.io/cephclusters

**remediator** (only when `remediation.enabled=true`):
- `delete` on pods
- `delete` on jobs
- `patch` on deployments (for `kubectl rollout restart`)
- **never**: secrets, configmaps, or any CRD

The fixers also enforce a protected-namespace skip list in code: `kube-system`, `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, `cnpg-system`. Even if RBAC permitted a write, the binary would refuse.

## Upgrade / uninstall

```sh
helm upgrade srenix charts/agentic-sre --reuse-values
helm uninstall srenix -n agentic-sre
```

Uninstall leaves the namespace alone; remove it explicitly if desired.
