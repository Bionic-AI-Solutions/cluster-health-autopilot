# Examples

## `sample-cluster/`

A hand-crafted JSON snapshot you can run `srenix diagnose --snapshot` against without a Kubernetes cluster. Used as the README's "30-second demo" — install `srenix`, point it at this directory, see the output.

The fixture is intentionally small (~6 KB) and shaped like a realistic small-team cluster:

| Resource | Items | Notes |
|---|---|---|
| Nodes | 4 | one control-plane + three workers, all Ready |
| PVCs | 3 | all Bound |
| Pods | 4 | three healthy + **one stuck in `CreateContainerConfigError`** (billing service expects a Secret key that doesn't exist) |
| ReplicaSets | 3 | owner chain wired so the consuming Deployment can be named in diagnostics |
| ExternalSecrets | 3 | one healthy + **two failing** (one with a missing-property error, one orphan from an old vault path) |
| Events | 2 | `UpdateFailed` events surfacing the precise Vault property names |
| CephCluster | 1 | `HEALTH_OK`, 11.5% capacity |
| CNPG Cluster | 1 | 3/3 instances ready |

### Running the demo

```sh
$ srenix diagnose --snapshot examples/sample-cluster
Agentic SRE — diagnose (snapshot mode)
============================================================

• Ceph Storage: 🟢 HEALTHY
  1 cluster(s): rook-ceph@rook-ceph OK (11.5% used)

• Cluster Nodes: 🟢 HEALTHY
  All 4 nodes ready

• PostgreSQL: 🟢 HEALTHY
  1 CNPG cluster(s): main@data (3/3 ready, primary=main-1)

• Storage Claims: 🟢 HEALTHY
  All 3 PVCs bound

• Critical Services: 🟢 HEALTHY
  All 0 critical services operational

Diagnostics (3):
  🔎 Secret `billing/billing-svc-secrets` missing key `STRIPE_API_KEY` (referenced by Deployment/billing-svc in ns billing).
     Owning ExternalSecret: `billing/billing-svc-secrets` — add data/template entry exposing `STRIPE_API_KEY`,
     or remove the env reference if unused.
  🔎 ExternalSecret `billing/billing-svc-secrets` not Ready:
     error processing spec.data[0] (key: shared/billing/config), err: cannot find secret data for key: "stripe_api_key".
     Check Vault path / property names.
  🔎 ExternalSecret `billing/old-payment-gateway` not Ready:
     error processing spec.data[0] (key: shared/legacy/payments), err: vault path not found.
     Check Vault path / property names.

============================================================
Total findings: 0, diagnostics: 3
```

The three diagnostics demonstrate the two analyzers working in concert:

1. The `SecretKeyMissing` analyzer correlates the kubelet's "couldn't find key X" event with the consuming Deployment (via `Pod → ReplicaSet → Deployment` owner chain) AND the owning ExternalSecret in the same namespace — naming all three in one line.
2. The `FailingExternalSecrets` analyzer walks every ExternalSecret cluster-wide, picks up the controller's most recent `UpdateFailed` event, and surfaces the precise missing Vault property name. Notice the second diagnostic ties to the same ExternalSecret as the first but from the controller's perspective: the user immediately sees both "kubelet says key X is missing" and "vault provider says property `stripe_api_key` is missing" — same root cause, two failure surfaces.

### JSON output

```sh
srenix diagnose --snapshot examples/sample-cluster --format json
```

Same data as a structured object suitable for piping into a fleet console or a custom alerting integration. See [`json-output.md`](./json-output.md) *(coming soon)* for the schema.

### What's NOT in this fixture (yet)

- Frozen-CronJob / stuck-RS scenarios — the corresponding fixers are live-only (Week 4 work).
- Multi-cluster Postgres / Ceph examples.
- A failing-Ceph fixture demonstrating the `HEALTH_WARN` / `HEALTH_ERR` paths.

These will land alongside their corresponding fixer ports.

## Notes on shape compatibility

Every JSON file is the exact output of `kubectl get <resource> -o json` (a `kind: <Resource>List` document with an `items` array). The same shape is produced by `srenix snapshot capture --out`, so a captured snapshot from a real cluster drops in here verbatim:

```sh
srenix snapshot capture --out my-cluster
srenix diagnose --snapshot my-cluster
```

This round-trip — capture → diagnose — is the headline product feature. No install, no RBAC, no write permissions, no external service. The only thing you give us is a snapshot file you already have permission to take.
