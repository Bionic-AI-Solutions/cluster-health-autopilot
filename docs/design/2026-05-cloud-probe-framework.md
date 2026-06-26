# Cloud Probe Framework — Design

> **STATUS: 🚧 PARTIAL — M1+M2 SHIPPED (30 cloud probes, v1.8.0); M3 NOT shipped.**
> _(P4.1 honest-header pass, 2026-06-11)_
>
> The full multi-cloud surface — **30 probes (10 each AWS / GCP / Azure)** with all three Live SDK wrappers wired to execute against real clouds — shipped bundled in **OSS v1.8.0** (CHANGELOG [1.8.0] 2026-05-28: "a complete 30-probe multi-cloud surface (10 each AWS/GCP/Azure) with all three Live SDK wrappers"). GCP/Azure slices landed incrementally (Sprint 1–3 PRs) then collapsed into the v1.8.0 release. Live SDK wrappers (`internal/cloud/{aws,gcp,azure}/live.go`) compile against the real SDKs but are NOT integration-tested against live cloud accounts (credential-gated) — documented as a verification boundary in the CHANGELOG.
>
> **DELTA — M3 (Cloud-aware AI tier, Srenix Enterprise) was DROPPED SILENTLY and is NOT shipped.** §6 M3 below (T0 cloud-context enricher, T1 cloud-aware fix proposer, T2 cloud+K8s planner) has **no CHANGELOG entry and no code** as of 2026-06-11 (verified: no `cloud-aware`/`Cloud M3` references outside this design doc). The shipped AI tiers (T0–T3, Srenix Enterprise v1.1.0–v1.4.0) carry K8s context only, not cloud-resource context. → **Follow-up: Cloud M3 (cloud-aware AI tier)** is genuinely open; tracked in the consolidated roadmap Q3 2026 forward plan. M4 (cloud mutations) remains a separate future design as stated.
>
> **DELTA — as-shipped chart surface (O6/O7, 2026-06-12):** the §3.7/§3.8 sketches below over-promise. `--cloud-rate-limit-per-min` / `cloud.rateLimitPerMin` were never implemented (rate protection is the `cloud.cadence` interval) and the dead chart value was removed. Auth modes `assumeRole` / `staticCredentials` were never implemented — workload identity (IRSA / GCP WI / AAD WI) is the only shipped auth path; the dead `auth.mode` / `assumeRoleArn` / `credentialsSecret` keys were removed (assume-role support would be a net-new feature, to be reintroduced deliberately). The per-probe `cloud.<provider>.probes.*` toggles ARE wired as of O6 (`SRENIX_CLOUD_PROBE_<PROVIDER>_<NAME>=off` env gates). GCP subnet IP utilization shipped capacity-only — GCP exposes no cheap used-IP count (the allocation-ratio insight is behind Network Analyzer / the Recommender API); see `internal/cloud/gcp/live.go`.
>
> Body below is the original design, preserved for context.

---

**Status:** Draft
**Author:** skadam
**Date:** 2026-05-23
**Target releases:** OSS v1.8 (M1 framework + AWS), OSS v1.9 (M2 GCP + Azure), Srenix Enterprise (M3 cloud-aware AI)

## 1. Summary

Extend Srenix from "K8s-resource health autopilot" to "K8s + cloud-resource health autopilot" — without giving up its in-cluster-operator posture, its air-gap deployability for the K8s portion, or its detect → fix → re-verify default loop.

Real customer outages routinely cross the K8s API boundary: RDS storage filling, an IAM role drift breaking IRSA, an ALB target group going empty, a managed certificate failing renewal, a Cloud SQL instance flipping to FAILED. Today Srenix's value story ends at the K8s API server. This design adds cloud-resource visibility as a peer to the existing K8s probe set, with the same probe → analyzer → fix → ticket loop the rest of Srenix already provides.

Per the competitive-baseline roadmap (`Srenix Enterprise/docs/competitive/roadmap.md`), this is **H2 priority**: every commercial competitor has cloud-resource visibility; Srenix has none. AWS first (3 weeks), k3s polish in parallel (1 week), then GCP + Azure in parallel (4 weeks). All shipped in OSS — matches the precedent that core sinks (Slack/Alertmanager/DriftReport) and core probes ship in OSS; paid value lives in AI tiers + enterprise integrations.

## 2. Goals & non-goals

### Goals
- Detect drift in cloud resources (RDS, EBS, IAM, ALB, ACM, etc.) the same way Srenix detects drift in K8s resources
- Persist findings on the same DriftReport CRD operators already learned (`subject="aws-rds/prod/db-1/storage-full"` looks like K8s subjects)
- Route findings to the same sinks (Slack, Alertmanager, ticketing) with zero new sink code
- Snapshot/offline mode preserved: `srenix snapshot capture --include-cloud` so audit/compliance flows keep working
- Per-probe enable/disable so air-gapped or single-cloud deployments don't pay for unused clouds
- Auth via the cloud-native K8s-integrated patterns (IRSA, GCP Workload Identity, Azure AAD Workload Identity) — no long-lived secrets in the cluster

### Non-goals (this design)
- Cloud-resource **mutation** (delete RDS instance, resize EBS, rotate IAM key) — read-only probes only in M1/M2. Cloud mutations land in a separate, later design with its own safety envelope.
- Cross-cloud correlation ("RDS storage filled because Lambda kept retrying because…") — that's a knowledge-graph workstream
- Cost-optimization probes (kubecost overlap) — explicitly out of scope per the roadmap's deliberate non-goals
- Multi-account / multi-subscription / multi-project federation — single-account v1; federation in a follow-up

## 3. Architecture

### 3.1 Why a new interface (not a `snapshot.Source` extension)

The existing `pkg/probe.Probe` interface takes a `snapshot.Source` whose contract is "list/get K8s resources by GVR." Cloud APIs are not GVR-shaped; jamming them in would either lose type safety (everything becomes `unstructured`) or pollute `snapshot.Source` with N cloud-specific methods.

The clean shape: a parallel interface family.

```
pkg/probe/         existing K8s probes (unchanged)
pkg/cloud/         NEW — cloud.Source abstraction
  source.go        interface; sub-clients per provider
  aws/             aws-sdk-go-v2 wrapper
  gcp/             cloud.google.com/go wrapper
  azure/           azure-sdk-for-go wrapper
pkg/cloudprobe/    NEW — cloudprobe.Probe interface
internal/cloud/    NEW — per-resource probe impls
  aws/
  gcp/
  azure/
catalog/cloud.go   NEW — RegisterCloudOSS()
```

### 3.2 Interface — `pkg/cloud.Source`

```go
// pkg/cloud/source.go
package cloud

import "context"

// Source is the cloud-API counterpart to snapshot.Source. It exposes
// per-provider sub-clients without forcing every probe to know about
// every cloud — a probe that only uses AWS holds an aws.Client and
// never imports gcp or azure code paths.
type Source interface {
    AWS() AWSClient   // nil if AWS not configured
    GCP() GCPClient   // nil if GCP not configured
    Azure() AzureClient // nil if Azure not configured
    Mode() Mode       // Live | Snapshot
}

type Mode int
const (
    ModeLive Mode = iota
    ModeSnapshot
)
```

Sub-clients are minimal — narrow surfaces over the official SDKs, scoped to read operations Srenix actually needs. Example:

```go
// pkg/cloud/aws/client.go
package aws

type Client interface {
    Region() string

    // RDS
    DescribeDBInstances(ctx context.Context) ([]DBInstance, error)
    DescribeDBClusters(ctx context.Context) ([]DBCluster, error)

    // EBS
    DescribeVolumes(ctx context.Context) ([]Volume, error)

    // EKS
    DescribeCluster(ctx context.Context, name string) (*Cluster, error)
    ListNodeGroups(ctx context.Context, clusterName string) ([]NodeGroup, error)

    // IAM (IRSA verification)
    GetRole(ctx context.Context, name string) (*Role, error)

    // ALB/NLB
    DescribeTargetGroups(ctx context.Context) ([]TargetGroup, error)
    DescribeTargetHealth(ctx context.Context, tgArn string) ([]TargetHealth, error)

    // ACM
    DescribeCertificates(ctx context.Context) ([]Certificate, error)

    // (etc. — only what probes need)
}
```

### 3.3 Probe interface — `pkg/cloudprobe.Probe`

```go
// pkg/cloudprobe/probe.go
package cloudprobe

import (
    "context"
    "github.com/srenix-ai/agentic-sre/pkg/cloud"
    "github.com/srenix-ai/agentic-sre/pkg/probe"
)

// Probe is the cloud-resource counterpart to pkg/probe.Probe. It returns
// the same probe.Result type so downstream rendering (Slack, Alertmanager,
// DriftReport, ticketing) needs zero changes.
type Probe interface {
    Component() probe.ComponentResult
    Run(ctx context.Context, src cloud.Source) probe.Result
}
```

Reusing `probe.Result` is deliberate. Findings flow through `report.AssembleEntries` unchanged → `DriftReport` reconcile unchanged → ticketing unchanged. Subject convention: `"aws-rds/<region>/<db-id>"`, `"gcp-cloudsql/<project>/<instance>"`, `"azure-sql/<rg>/<server>/<db>"`.

### 3.4 Auth model

Three first-class auth modes per provider, picked in this order:

| Mode | When chosen | How |
|---|---|---|
| **Workload identity (cloud-native)** | Default in-cluster | AWS: IRSA (projected SA token); GCP: Workload Identity (k8s SA → GCP SA); Azure: AAD Workload Identity (k8s SA → AAD app). No long-lived secrets in cluster. |
| **Explicit role assumption** | When `cloud.aws.assumeRoleArn` (etc.) is set | Helm-configured target role; uses workload identity to assume |
| **Static credentials (escape hatch)** | When `cloud.aws.credentialsSecret` is set | K8s Secret (ESO-managed); explicit opt-in; warned against in docs |

Auth is **lazy-evaluated** per provider: if `cloud.aws.enabled=false`, the AWS SDK never initializes and the AWS probes are not registered. Air-gapped deployments turn all three off and pay zero overhead.

### 3.5 Snapshot/offline mode

Cloud state is more time-sensitive than K8s state (RDS storage fills smoothly; a Pod either exists or it doesn't). To preserve `srenix diagnose --snapshot` semantics:

- `srenix snapshot capture` does NOT capture cloud state by default
- `srenix snapshot capture --include-cloud` captures cloud-resource state as additional JSON files (`cloud/aws/rds.json`, `cloud/gcp/cloudsql.json`, etc.)
- `srenix diagnose --snapshot <dir>` only runs cloud probes if cloud state is present
- All cloud probe outputs are stamped with `capturedAt` so operators see "this is point-in-time as of …" in DriftReport messages

This keeps the zero-trust offline diagnose flow intact.

### 3.6 Registry & wiring

```go
// catalog/cloud.go
package catalog

func RegisterCloudOSS(reg *registry.Registry) {
    // AWS probes — only registered if AWS sub-client is non-nil at startup
    if reg.CloudSource().AWS() != nil {
        reg.RegisterCloudProbe(awsprobes.RDS{})
        reg.RegisterCloudProbe(awsprobes.EBSVolumes{})
        reg.RegisterCloudProbe(awsprobes.EKSControlPlane{})
        reg.RegisterCloudProbe(awsprobes.IAMRoles{})
        reg.RegisterCloudProbe(awsprobes.ALBTargetHealth{})
        reg.RegisterCloudProbe(awsprobes.ACMCertExpiry{})
        // (etc.)
    }
    // GCP / Azure same pattern, in M2
}
```

The watcher iterates cloud probes the same way it iterates K8s probes. Per-cycle overhead is bounded by cloud-API rate limits (Srenix respects per-provider rate limit defaults; configurable via Helm).

### 3.7 CLI

| Subcommand | New behavior |
|---|---|
| `srenix diagnose --live` | Runs K8s + cloud probes if any cloud sub-client is configured |
| `srenix diagnose --include-cloud` | Explicit form (forces cloud probes on; errors if none configured) |
| `srenix diagnose --exclude-cloud` | Skip cloud probes this run (debugging / cost control) |
| `srenix snapshot capture --include-cloud` | Capture cloud-resource state alongside K8s snapshot |
| `srenix watch` | Cloud probes run on the existing `resyncPeriod` cadence (default 10m) — not on K8s event triggers (cloud-resource events don't flow through K8s watch) |
| New `--cloud-rate-limit-per-min` flag | Per-provider rate limit (default 60); guards against API quota burn |

### 3.8 Helm chart surface

```yaml
cloud:
  enabled: false                 # master switch; if false none of this matters
  rateLimitPerMin: 60            # per-provider; tunable per-provider below

  aws:
    enabled: false
    region: ""                   # required; e.g. us-east-1
    auth:
      mode: irsa                 # irsa | assumeRole | staticCredentials
      assumeRoleArn: ""          # required if mode=assumeRole
      credentialsSecret: ""      # required if mode=staticCredentials
    probes:
      rds: true
      ebs: true
      eks: true
      iam: true
      alb: true
      acm: true
      # (etc — each probe individually toggleable)
    rateLimitPerMin: 60          # overrides cloud.rateLimitPerMin

  gcp:
    enabled: false               # M2
    project: ""
    auth:
      mode: workloadIdentity     # workloadIdentity | staticCredentials
      credentialsSecret: ""
    probes:
      cloudsql: true
      persistentDisk: true
      gke: true
      iam: true
      loadBalancer: true
      managedCert: true

  azure:
    enabled: false               # M2
    subscriptionId: ""
    resourceGroup: ""            # optional scope
    auth:
      mode: workloadIdentity     # workloadIdentity | staticCredentials
      credentialsSecret: ""
    probes:
      sql: true
      disks: true
      aks: true
      managedIdentity: true
      appGateway: true
      certs: true
```

The schema mirrors the existing `ticketing` block pattern (master switch, per-provider sub-block, per-feature toggles).

### 3.9 RBAC additions

Srenix's K8s ServiceAccount needs a projected token volume for workload-identity auth on each cloud. The Helm chart auto-injects when `cloud.<provider>.enabled=true && cloud.<provider>.auth.mode in {irsa, workloadIdentity}`:

- AWS IRSA: ServiceAccount annotation `eks.amazonaws.com/role-arn`
- GCP Workload Identity: ServiceAccount annotation `iam.gke.io/gcp-service-account`
- Azure AAD WI: ServiceAccount annotation `azure.workload.identity/client-id`

No new K8s RBAC required (cloud probes don't read K8s resources).

### 3.10 DriftReport extensions

Add an optional `resourceRef.cloud` field to the existing CRD schema to disambiguate cloud-resource references from K8s resources:

```yaml
spec:
  resourceRef:
    cloud:           # NEW optional block
      provider: aws  # aws | gcp | azure
      region: us-east-1
      service: rds
      id: prod-db-1
```

Additive change — existing CRs are unaffected. Backward compatible.

## 4. AWS M1 probe set (10 probes)

The first 10 — picked for coverage density and operator demand, not completeness:

| Probe | What it detects | Severity model |
|---|---|---|
| `RDS` | Instance state ≠ `available`; storage utilization > 80%; failover events; replica lag | Critical / Warning |
| `EBSVolumes` | Detached volumes > 7 days; volumes referenced by terminated EC2 | Warning |
| `EKSControlPlane` | Cluster status, version drift vs cluster nodes, endpoint reachability | Critical |
| `EKSNodeGroups` | Node group `health.issues`, ASG capacity mismatch | Warning / Critical |
| `IAMRoles` | IRSA-pointed roles that don't exist or have no trust policy for the cluster OIDC issuer | Critical |
| `ALBTargetHealth` | Target groups with 0 healthy targets; rapid flapping (>3 transitions/min) | Critical |
| `ACMCertExpiry` | Certificates expiring within configurable window (default 14d); failed renewals | Warning / Critical |
| `KMSKeys` | Disabled keys; pending deletion keys still referenced; key policy drift | Critical |
| `S3BucketPublicAccess` | Buckets with PAB disabled when expected enabled (drift) | Critical |
| `VPCSubnetCapacity` | Subnets with < 10 IPs free | Warning |

Each is a single Go file under `internal/cloud/aws/`, ~150-300 lines, single SDK client call + finding emission.

## 5. k3s support (parallel work, separate from cloud)

k3s is conformant K8s — most Srenix code already works. The actual gaps:

| Check | Action |
|---|---|
| Embedded Traefik ingress discovery | Verify `internal/probe/ingress_discovery.go` recognizes Traefik annotations (`traefik.io/router.*`); add test fixture |
| Local-path-provisioner PVCs | Verify `internal/probe/pvcs.go` doesn't false-flag local-path PVCs; add k3s storage class to expected list |
| Single-node mode | Verify watcher tolerates `nodeCount=1`; test against k3d locally |
| Resource limits | Validate default Helm requests (50m / 64Mi) fit a 2-CPU edge node |
| Edge deployment Helm example | New `examples/k3s-edge/values.yaml` showing reduced footprint + recommended fixers (StaleErrorPods, StuckRSPods only — skip cert-manager flows that don't apply at edge) |

Effort: ~1 week. Parallelizable with the cloud framework work because zero code overlap.

## 6. Rollout phases

### M1 — Framework + AWS (OSS v1.8)

- `pkg/cloud/` interface + AWS sub-client
- `pkg/cloudprobe/` interface
- `internal/cloud/aws/` — 10 probes above
- `catalog/cloud.go` — `RegisterCloudOSS()`
- Watcher wiring — cloud probes run on `resyncPeriod` cadence
- Snapshot mode — `--include-cloud` flag
- Helm chart — cloud values block + IRSA SA annotation injection
- DriftReport CRD — additive `resourceRef.cloud` field
- E2E test against LocalStack (AWS API emulator) in CI
- Live test against the GPU cluster's actual AWS account (if any) — otherwise sandbox account

**Acceptance:** in a cluster with AWS configured, intentionally fill an RDS instance's storage to 85% → Srenix opens a DriftReport + an OpenProject ticket within one resync cycle.

### M1b — k3s polish (OSS v1.8, parallel)

- Traefik ingress discovery test fixture + fix if needed
- Local-path-provisioner PVC test fixture + fix if needed
- k3d-based CI smoke test
- `examples/k3s-edge/` with values.yaml + README
- Update Srenix README to list k3s as supported

**Acceptance:** `helm install` on a fresh k3d cluster passes the existing smoke test + k3s-specific edge cases.

### M2 — GCP + Azure (OSS v1.9)

Parallel work streams; each follows the M1 framework template:

- `pkg/cloud/gcp/` + `internal/cloud/gcp/` — 10 probes (CloudSQL, PersistentDisk, GKE control plane, GKE node pool, IAM service-account bindings, LB backend health, Google-managed certs, GCS public-access, KMS state, subnet capacity)
- `pkg/cloud/azure/` + `internal/cloud/azure/` — 10 probes (Azure SQL DB, Disks, AKS control plane, AKS node pool, Managed identity drift, App Gateway backend, certs, Storage public-access, Key Vault state, VNet/subnet capacity)
- Helm chart — GCP + Azure values blocks; Workload Identity / AAD WI SA annotation injection
- LocalStack-equivalent CI for GCP (`fake-gcs-server` for the bits we can; otherwise sandbox GCP project) and Azure (Azurite for storage; sandbox Azure subscription for the rest)

**Acceptance:** parity with M1 on AWS — each cloud has 10 probes, each shippable independently.

### M3 — Cloud-aware AI tier integration (Srenix Enterprise)

- T0 Enricher: cloud-resource context in enrichment prompts ("this DriftReport is about RDS db-1; the instance is `db.r5.large`, region us-east-1, in VPC vpc-abc; the last failover was 14 days ago")
- T1 FixProposer: propose K8s-side fixes that work around cloud drift (e.g., add a Pod toleration when an EKS node group is unhealthy)
- T2 Planner: multi-step plans that span K8s + cloud read-only context (cloud mutations remain out of scope)

**Acceptance:** AI-tier-enabled enrichments include cloud context in DriftReport messages.

### M4 — Cloud mutations (separate design, not this doc)

Adding the ability to mutate cloud resources (resize EBS, rotate IAM key, etc.) is a bigger safety conversation and belongs in its own design doc — modeled on this repo's existing approval-server / signed JWT pattern, plus per-action allowlists per-provider. Explicitly out of scope here.

## 7. Open decisions (need your call)

1. **Snapshot capture default.** Should `srenix snapshot capture` include cloud state by default once any cloud is configured? Pro: lower friction. Con: snapshot files get big and time-sensitive. *Recommendation:* opt-in via `--include-cloud`.
2. **Multi-account scope.** v1 = single account/project/subscription per cloud. Multi-account federation (e.g., one Srenix instance probing 5 AWS accounts via cross-account roles) is a real ask. Defer to v2 or design it in now? *Recommendation:* defer; v1 ships first, federation is a one-week add later.
3. **Rate-limit posture.** Default rate limit 60 req/min/provider — is that the right shape, or should we use AWS-style adaptive throttling? *Recommendation:* fixed cap for v1; adaptive later if customers hit it.
4. **OSS vs paid line for AWS probes.** All 10 in OSS, or hold IAM + ALB-target-health for paid (the two with the most direct compliance value)? *Recommendation:* all 10 in OSS — matches the precedent that core sinks are OSS; AI enrichment is what makes the paid tier worth paying for.

## 8. Risks

- **Scope creep per cloud.** "RDS state" is a one-day probe; "complete RDS observability" is a month. Hold the 10-probe line per cloud; resist completeness.
- **Auth complexity in air-gap.** Workload identity assumes cloud-API reachability. Air-gapped deployments must work with cloud disabled — covered by the `cloud.enabled=false` master switch.
- **Snapshot/cloud mismatch.** Cloud state is point-in-time; K8s state in a snapshot is a list-as-of-then. Need clear UI messaging in DriftReports for cloud findings sourced from a snapshot ("AWS RDS state captured at 2026-05-22T18:00Z").
- **Probe-count inflation.** Going from 6 → 36+ probes increases noise. Each cloud probe defaults to ON when the cloud is enabled, but each is individually toggleable.
- **SDK churn.** All three major-cloud Go SDKs see frequent minor releases. Pin minor versions; let dependabot batch-bump quarterly.

## 9. Testing strategy

- **Unit:** mock `cloud.Source` for probe logic tests (same pattern as the existing `snapshot.Source` mocks)
- **Integration (LocalStack):** for AWS; LocalStack covers RDS/EBS/IAM/EKS-Control-Plane-light/ALB/ACM/KMS/S3 at fidelity sufficient for probe correctness
- **Integration (Azurite / fake-gcs):** partial GCP/Azure; full integration needs sandbox accounts
- **E2E (live cluster):** AWS first against the GPU cluster's account; sandbox GCP/Azure for M2

## 10. Not in this scope

- Cloud-resource mutation (separate design)
- Multi-account / multi-subscription / multi-project federation
- Cost-optimization probes (per the roadmap's deliberate non-goal)
- Cross-cloud correlation knowledge graph (separate workstream)
- Datadog / Splunk / NewRelic / Dynatrace observability connectors (separate workstream — H2 in the roadmap)
