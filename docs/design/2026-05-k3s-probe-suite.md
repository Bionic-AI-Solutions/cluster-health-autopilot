# k3s Probe Suite — Design Document

> **STATUS: ✅ SHIPPED — v1.9.0** _(P4.1 honest-header pass, 2026-06-11)_
>
> The k3s probe suite (Traefik routes, LocalPath, Datastore/etcd-quorum, ingress discovery) shipped in **OSS v1.9.0** via PR #121 ("k3s probe suite — Traefik, LocalPath, Datastore, ingress discovery") + chart bump PR #122. Files: `internal/probe/traefik_routes.go`, `internal/probe/k3s_localpath.go`, `internal/probe/k3s_datastore.go`, `internal/probe/ingress_discovery.go`, `internal/probe/k3s_probes_test.go`. K3sDatastore clustered-etcd quorum checks extended in PR #126.
>
> No material as-shipped delta vs this design. Body below is the original design, preserved for context.

---

**Status:** Draft  
**Date:** 2026-05-31  
**Target version:** v1.10 (first sprint after v1.9.3 stabilisation)

---

## 1. Motivation

The running Bionic cluster is a bare-metal k3s cluster. k3s ships meaningful differences from a kubeadm-provisioned cluster, and several existing Srenix probes either produce false positives or are silently blind on it:

| Existing probe | Behaviour on k3s |
|---|---|
| `ETCD` | Looks for `component=etcd` pods in `kube-system`. k3s embedded etcd is not a static pod — it is a process inside the k3s binary itself. Probe always fires the "external etcd / blind" warning. |
| `DiscoverIngressTargets` | Discovers `networking.k8s.io/v1` Ingress objects. k3s users overwhelmingly use Traefik `IngressRoute` CRDs instead. All route-based endpoints are invisible. |
| `NodePressure` | Surfaces kubelet disk-pressure conditions. The `local-path` provisioner stores data in `/var/lib/rancher/k3s/storage/` with no capacity reporting — the kubelet `DiskPressure` condition fires only after the node root volume is full, giving no early-warning headroom. |

This design specifies four new probes plus one patch to fix the coverage gaps.

---

## 2. New GVR Constants (`internal/snapshot/source.go`)

Add the following to the `var (...)` block in `source.go`. All four are cluster-scoped (namespace list is cluster-wide via `src.List(ctx, gvr, "")`):

```go
// Traefik IngressRoute CRDs — present on any k3s cluster with the
// default Traefik ingress controller. Srenix probes auto-skip when the
// list call returns a "no such resource" error (same pattern as Kong).
GVRTraefikIngressRoute = schema.GroupVersionResource{
    Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes",
}
GVRTraefikIngressRouteTCP = schema.GroupVersionResource{
    Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutetcps",
}
GVRTraefikMiddleware = schema.GroupVersionResource{
    Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares",
}
GVRTraefikTLSStore = schema.GroupVersionResource{
    Group: "traefik.io", Version: "v1alpha1", Resource: "tlsstores",
}
```

These four GVRs are referenced by `TraefikRoutes` probe and `DiscoverTraefikRouteTargets`. No other probe currently needs them. They are intentionally separate from the `GVRIngress` entry so snapshot capture can list them independently.

---

## 3. Probe 1: `TraefikRoutes`

### File
`internal/probe/traefik_routes.go`

### Struct

```go
type TraefikRoutes struct{}
```

### `Name()` return value
`"Traefik Routes"`

### Auto-skip pattern
The probe opens with a `src.List(ctx, snapshot.GVRTraefikIngressRoute, "")` call. If the error is non-nil (CRD absent, RBAC missing, k3s without Traefik), it sets `Component.Status = "SKIPPED"` with `Detail = "Traefik CRDs not installed"` and returns immediately — identical to the Kong probe pattern. This makes the probe safe to register default-on on any cluster.

### Opt-out
`SRENIX_PROBE_TRAEFIK_ROUTES=off` — guarded in `catalog.go` exactly as the Kong/Velero/ArgoCD probes are today.

### Data collected per Run

1. **IngressRoutes** — `src.List(ctx, snapshot.GVRTraefikIngressRoute, "")` → all namespaces.
2. **IngressRouteTCPs** — `src.List(ctx, snapshot.GVRTraefikIngressRouteTCP, "")` → all namespaces.
3. **Services** — `src.List(ctx, snapshot.GVRService, "")` → used to build a lookup set `ns/name → exists`.
4. **Middlewares** — `src.List(ctx, snapshot.GVRTraefikMiddleware, "")` → build a lookup set `ns/name → exists`.
5. **TLSStores** — `src.List(ctx, snapshot.GVRTraefikTLSStore, "")` → build a set of `ns/name` entries. The special `default/default` TLSStore is the cluster-wide fallback; treat its presence as satisfying the TLS store requirement for any route in the cluster.

### Finding classes

#### F1 — Missing backend Service (Severity: Critical)

Walk `IngressRoute.spec.routes[*].services[*]` (each entry has `name` string and optional `namespace` string; namespace defaults to the IngressRoute's own namespace). For each referenced service, check the lookup set. If the `ns/name` pair is absent, emit one finding.

```
Message:    "IngressRoute <ns>/<name>: route[<i>] references Service <svcNs>/<svcName> which does not exist"
Remediation: "Create the missing Service or update the IngressRoute's backend: kubectl get svc -n <svcNs>"
```

The same check applies to `IngressRouteTCP.spec.routes[*].services[*]`.

#### F2 — Missing Middleware reference (Severity: Warning)

Walk `IngressRoute.spec.routes[*].middlewares[*]` (each entry has `name` string and optional `namespace` string). Resolve the namespace using the same defaulting rule as F1. If `ns/name` pair is absent from the middleware lookup set, emit a warning per missing reference.

```
Message:    "IngressRoute <ns>/<name>: route[<i>] references Middleware <mwNs>/<mwName> which does not exist"
Remediation: "Create the missing Middleware CRD or remove the reference: kubectl get middlewares.traefik.io -n <mwNs>"
```

Do not emit F2 for TCP routes — `IngressRouteTCP` routes do not carry middleware refs in the v1alpha1 schema.

#### F3 — TLS enabled but no TLSStore or certResolver (Severity: Warning)

Check `IngressRoute.spec.tls` (the field exists when TLS is configured). A route is TLS-enabled when this field is non-nil. It is properly configured when at least one of the following is true:

- `spec.tls.certResolver` is non-empty (Traefik ACME resolver handles cert provisioning).
- `spec.tls.secretName` is non-empty (a pre-existing cert Secret is referenced).
- A TLSStore exists at `<namespace>/default` or `default/default` (the cluster-wide default TLSStore carries the fallback cert).

If none of the three conditions hold, emit a warning per affected IngressRoute.

```
Message:    "IngressRoute <ns>/<name>: TLS is enabled but no certResolver, secretName, or default TLSStore found"
Remediation: "Set spec.tls.certResolver, spec.tls.secretName, or create a TLSStore default in the route's namespace or the default namespace"
```

### Component status rollup

Uses the same `rollupComponentStatus(findings)` helper that Kong and other probes use. A healthy result sets `Detail = "<N> IngressRoute(s) and <M> IngressRouteTCP(s) inspected, all healthy"`.

### Snapshot capture RBAC note

The RBAC reader ClusterRole template in `charts/agentic-sre/templates/clusterrole-reader.yaml` must add `get` / `list` / `watch` for the four Traefik GVRs when `probes.traefikRoutes.enabled` is true. The Helm values key is described in Section 7.

---

## 4. Probe 2: `K3sLocalPathStorage`

### File
`internal/probe/k3s_localpath.go`

### Struct

```go
type K3sLocalPathStorage struct {
    // DiskFreeThreshold is the fraction below which a node is flagged.
    // Zero → use defaultLocalPathDiskThreshold (0.20 = 20%).
    DiskFreeThreshold float64
}
```

### `Name()` return value
`"K3s LocalPath Storage"`

### Opt-out
`SRENIX_PROBE_K3S_LOCALPATH=off`

### Background

The `local-path-provisioner` (Rancher) backs PVCs by creating subdirectories in `/var/lib/rancher/k3s/storage/<namespace>_<pvcname>_<pvname>/` on the node where the pod is scheduled. The StorageClass advertises no capacity — the kubelet `DiskPressure` condition fires only after the node root volume is full, giving no early-warning headroom. The best available signal is `status.allocatable["ephemeral-storage"]` vs `status.capacity["ephemeral-storage"]` on each node.

### Finding classes

#### F1 — Node ephemeral storage low (Severity: Warning → Critical)

Compute `freeFraction = allocatable / capacity`. When `freeFraction < DiskFreeThreshold` (default 0.20):

- `freeFraction >= 0.10` → Severity Warning.
- `freeFraction < 0.10` → Severity Critical (kubelet eviction imminent within the default 10% threshold).

#### F2 — local-path PVCs exist but node ephemeral-storage not reported (Severity: Info)

A visibility gap finding when nodes do not report `allocatable["ephemeral-storage"]`.

#### F3 — local-path PVC Pending (Severity: Warning)

PVCs in `Pending` phase with `storageClassName=local-path` indicate a scheduling failure.

### Component status rollup

Uses `rollupComponentStatus`. Healthy detail: `"<N> local-path PVC(s) on <M> node(s); ephemeral-storage headroom within threshold"`.

---

## 5. Probe 3: `K3sDatastore`

### File
`internal/probe/k3s_datastore.go`

### Struct

```go
type K3sDatastore struct{}
```

### `Name()` return value
`"K3s Datastore"`

### Opt-out
`SRENIX_PROBE_K3S_DATASTORE=off`

### Background and detection approach

k3s embeds etcd or SQLite as its datastore. Neither runs as a pod visible to the Kubernetes API. Detection:

1. **Check node providerID / annotations for k3s identity.** Any node whose `spec.providerID` starts with `k3s://` or has any `k3s.io/*` annotation identifies this as a k3s cluster. If none match, the probe auto-skips.

2. **Determine etcd vs SQLite.** Static etcd pods in kube-system (same `component=etcd` label the ETCD probe uses) confirm embedded-etcd HA mode. Their absence on a k3s node means SQLite single-node mode.

3. **For embedded etcd clusters:** evaluate pod readiness + restart counts (same signals as the ETCD probe) and check for recent `k3s-etcd-snapshot-*` ConfigMaps in `kube-system`.

4. **For SQLite mode:** emit a SeverityInfo advisory about the lack of HA. Suppressed when `SRENIX_K3S_SINGLE_NODE_OK=true`.

### Finding classes

#### F1 — Not a k3s cluster (auto-skip)
#### F2 — Embedded etcd member not Ready (Severity: Critical)
#### F3 — Embedded etcd member restarted (Severity: Warning)
#### F4 — No recent etcd snapshot ConfigMap (Severity: Warning, threshold 26 h)
#### F5 — SQLite mode advisory (Severity: Info, suppressible)

### Component status rollup
Uses `rollupComponentStatus`.

---

## 6. Patch: `DiscoverTraefikRouteTargets` (`internal/probe/ingress_discovery.go`)

### New function signature

```go
func DiscoverTraefikRouteTargets(
    ctx context.Context,
    src snapshot.Source,
    opts DiscoveryOptions,
    existing []string,
) []EndpointTarget
```

### Host extraction from IngressRoute match expressions

Uses a regexp scan:

```go
var traefikHostRe = regexp.MustCompile("Host\\(`([^`]+)`\\)")
```

Handles single-host, multi-host OR, and combined Host + PathPrefix/Headers expressions. Discards candidates that are empty, contain `*`, or have no `.`.

### Integration into the existing Endpoints probe

`NewEndpoints` in `probe/endpoints.go` currently calls `DiscoverIngressTargets`. Add a second call to `DiscoverTraefikRouteTargets` alongside it to cover k3s IngressRoute-based hosts.

### Opt-out annotation

The same `srenix.ai/probe-disable: "true"` annotation on the IngressRoute object suppresses its hosts from discovery.

---

## 7. Chart Values Overlay (`examples/k3s-edge/values.yaml`)

Ready-to-use Helm values overlay for k3s clusters. Apply alongside chart defaults:

```bash
helm upgrade --install srenix \
  bionic-ai-solutions/agentic-sre \
  -f examples/k3s-edge/values.yaml \
  --namespace srenix --create-namespace
```

Key overrides:
- `SRENIX_PROBE_ETCD=off` — prevents spurious "no etcd pods" warning.
- `SRENIX_PROBE_TRAEFIK_ROUTES=on`, `SRENIX_PROBE_K3S_LOCALPATH=on`, `SRENIX_PROBE_K3S_DATASTORE=on` — enable the k3s-specific probes.
- `probes.kong.enabled: false` — most k3s edge clusters use Traefik, not Kong.
- Tolerations for `node-role.kubernetes.io/master`, `node-role.kubernetes.io/control-plane`, and `CriticalAddonsOnly` — allows Srenix pods to run on combined control-plane+worker nodes.

---

## 8. Catalog wiring (`catalog/catalog.go`)

Three new opt-out blocks in `RegisterOSS`, after the Velero block:

```go
if os.Getenv("SRENIX_PROBE_TRAEFIK_ROUTES") != "off" {
    r.RegisterProbe(probe.TraefikRoutes{})
}
if os.Getenv("SRENIX_PROBE_K3S_LOCALPATH") != "off" {
    r.RegisterProbe(probe.K3sLocalPathStorage{})
}
if os.Getenv("SRENIX_PROBE_K3S_DATASTORE") != "off" {
    r.RegisterProbe(probe.K3sDatastore{})
}
```

---

## 9. RBAC additions (`charts/.../clusterrole-reader.yaml`)

```yaml
# Traefik IngressRoute CRDs (k3s default ingress controller)
- apiGroups: ["traefik.io"]
  resources:
    - ingressroutes
    - ingressroutetcps
    - middlewares
    - tlsstores
  verbs: ["get", "list", "watch"]

# StorageClass — needed by K3sLocalPathStorage to detect the default class
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]

# Endpoints — needed by K3sDatastore to check apiserver reachability
- apiGroups: [""]
  resources: ["endpoints"]
  verbs: ["get", "list", "watch"]
```

The `ConfigMap` read in `kube-system` (for etcd snapshot detection) is already covered by the existing `configmaps` rule in the reader role.

---

## 10. Helm chart values keys

Add under the `probes:` block in `charts/agentic-sre/values.yaml`:

```yaml
probes:
  traefikRoutes:
    enabled: true
  k3sLocalPath:
    enabled: true
  k3sDatastore:
    enabled: true
```

The chart templates translate `probes.traefikRoutes.enabled: false` into `SRENIX_PROBE_TRAEFIK_ROUTES=off`, following the same template helper pattern as the existing M2 probes.

---

## 11. Testing notes

Each new probe follows the established test pattern using the `snapshot.File` source backed by captured `kubectl get -o json` outputs. Test fixtures live at:

- `internal/probe/testdata/traefik_routes/` — IngressRoute, IngressRouteTCP, Middleware, TLSStore, and Service lists covering: all-healthy, missing-service, missing-middleware, tls-no-resolver.
- `internal/probe/testdata/k3s_localpath/` — PVC list with local-path class, node list with varying ephemeral-storage allocatable values.
- `internal/probe/testdata/k3s_datastore/` — node lists with and without k3s annotations, with and without etcd role labels; ConfigMap lists for etcd snapshot CMs; Endpoints lists with zero and non-zero ready addresses.

Unit test files: `traefik_routes_test.go`, `k3s_localpath_test.go`, `k3s_datastore_test.go` following the table-driven pattern used in `kong.go`, `velero.go`, etc.

`DiscoverTraefikRouteTargets` test lives in `ingress_discovery_test.go` alongside the existing `TestDiscoverIngressTargets` test.

---

## 12. Sequencing and implementation order

| Step | File(s) | Status |
|---|---|---|
| 1 | `internal/snapshot/source.go` | Done — 4 Traefik GVRs, `GVRService`, `GVRStorageClass`, `GVREndpoints` added |
| 2 | `internal/probe/traefik_routes.go` | Done |
| 3 | `internal/probe/k3s_localpath.go` | Done |
| 4 | `internal/probe/k3s_datastore.go` | Done |
| 5 | `internal/probe/ingress_discovery.go` | Done — `DiscoverTraefikRouteTargets` added |
| 6 | `catalog/catalog.go` | Done — 3 new probes registered with env opt-outs |
| 7 | `charts/.../clusterrole-reader.yaml` | Pending — RBAC rules for Traefik, StorageClass, Endpoints |
| 8 | `charts/.../values.yaml` | Pending — `probes.traefikRoutes`, `k3sLocalPath`, `k3sDatastore` keys |
| 9 | `examples/k3s-edge/values.yaml` | Done |
| 10 | Test fixtures + unit tests | Pending |

---

## 13. Open questions / follow-up

- **Traefik v2 vs v3 GVR:** Traefik v2 uses `traefik.containo.us/v1alpha1`; Traefik v3 (default in k3s 1.30+) uses `traefik.io/v1alpha1`. The `TraefikRoutes` probe and `DiscoverTraefikRouteTargets` implement a try-v3-then-fallback-to-v2 pattern. The canonical GVRs in `source.go` use the v3 group; the probe-local fallback vars cover v2.

- **K3sLocalPathStorage disk usage visibility:** The Kubernetes API only exposes ephemeral-storage at the node level, not per-PVC. A future enhancement would deploy a lightweight DaemonSet that runs `du -sb /var/lib/rancher/k3s/storage/` and publishes the result as a node annotation or custom metric.

- **K3sDatastore etcd member health via API:** k3s 1.27+ exposes an `/api/v1/namespaces/kube-system/configmaps/k3s-etcd-endpoint-management` object that lists known etcd member URLs and their status. If reliably present in production k3s HA clusters, it is a cleaner health signal than the snapshot-age heuristic.

- **Endpoints endpoint for K3sDatastore apiserver check:** The design doc originally specified checking `kubernetes` Endpoints in the `default` namespace for apiserver reachability. The implemented probe uses pod readiness (same as the ETCD probe) as the primary signal, which is more portable. The Endpoints check is a follow-up enhancement.
