# CHA Hosted Playground (P6.8)

A self-contained bundle that powers the website's **"Try it now" / playground**
CTA: a visitor watches Cluster Health Autopilot detect synthetic drift **live**.

It is honest about what it shows — real CHA, real DriftReports, real synthetic
K8s drift — and it is **isolated** so it can never disturb a production workload.

## What it demonstrates

A `drift-injector` CronJob continuously creates and rotates four synthetic
drift scenarios in the isolated `cha-playground` namespace. The OSS CHA watcher
(diagnose-only, **never** remediates) detects them and writes `DriftReport` CRs.
A tiny read-only viewer renders those reports as an auto-refreshing page.

| # | Scenario (object) | Detected by (shipped OSS analyzer/probe) | Source file |
|---|---|---|---|
| 1 | Deployment with bad imagePullSecret pulling a private Docker Hub repo → `ImagePullBackOff` with an **auth** failure | `ImagePullAuth` analyzer | `internal/diagnose/image_pull_auth.go` |
| 2 | Job whose pod references a key absent from an existing Secret → `CreateContainerConfigError` ("StuckJobsWithBadSecretRef") | `SecretKeyMissing` analyzer | `internal/diagnose/secret_key_missing.go` |
| 3 | Ingress serving an **expired** TLS secret while a healthy cert-manager `Certificate` renews the same host into a **different** secret | `TLSSecretMismatch` analyzer | `internal/diagnose/tls_secret_mismatch.go` |
| 4 | Deployment whose container exits 1 in a loop → `CrashLoopBackOff` | `CrashLoopBackOff` probe | `internal/probe/crashloop.go` |

All four were verified firing on a kind cluster (see "kind verification" below).

## Isolation

- **Dedicated namespace** `cha-playground`, Pod-Security `baseline`.
- **ResourceQuota** caps the whole demo at 30 pods / 1 CPU / 1Gi requests.
- **LimitRange** gives every container a tiny default + a hard max.
- **NetworkPolicy** default-deny ingress+egress; only DNS, in-namespace, and
  HTTPS-out (image pulls / apiserver) are re-opened.
- **RBAC**: the injector SA holds a **namespaced Role** (acts only inside
  `cha-playground`); the viewer SA holds a ClusterRole granting **only**
  `get/list/watch` on `driftreports` (nothing else).
- **GPU nodes excluded**: every workload (injector, viewer, scenario pods, the
  CHA watcher) carries a `nodeAffinity` requiring `nvidia.com/gpu.present`
  `DoesNotExist`.

> **Scope note (honest):** `DriftReport` is a **cluster-scoped** CRD and the CHA
> watcher lists cluster-wide (there is no namespace-scope flag on `cha watch`),
> so its reader ClusterRole is cluster-wide **read-only**. That is safe — with
> `watcher.remedy.enabled=false` it never mutates — and the injector only ever
> creates drift in `cha-playground`, so every DriftReport you see originates
> here. For hard read-isolation, run the playground on its own cluster (the kind
> quick-try does exactly that).

## Files

| File | Purpose |
|---|---|
| `namespace.yaml` | Namespace + ResourceQuota + LimitRange + NetworkPolicies + injector/viewer RBAC |
| `drift-injector.yaml` | CronJob (alpine/k8s + inline bash) that creates/rotates the 4 scenarios |
| `viewer.yaml` | Read-only viewer Deployment + Service + Ingress (`playground.asre.baisoln.com`) |
| `viewer/` | The ~180-line Go viewer (lists DriftReports, html/template, XSS-safe) + Dockerfile + test |
| `cha-values.yaml` | Helm values: CHA watcher, diagnose-only, remedy/AI/approval off |
| `kustomization.yaml` | Bundles namespace + injector + viewer for `kubectl apply -k` |

---

## Quick-try on kind (local, touches no real cluster)

Prereqs: `docker`, `kind`, `kubectl`, `helm`, and this repo checked out.

```bash
cd cluster-health-autopilot

# 1. cluster
kind create cluster --name cha-pg

# 2. build + load the viewer image (and the CHA watcher image from this repo)
docker build -f examples/playground/viewer/Dockerfile -t cha-playground-viewer:dev .
docker build -t cha-local:playground .                 # the cha watcher binary
kind load docker-image cha-playground-viewer:dev --name cha-pg
kind load docker-image cha-local:playground --name cha-pg

# 3. cert-manager CRDs ONLY (so the Certificate object for scenario 3 can exist;
#    no controller needed — the injector stamps Ready=True itself)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.3/cert-manager.crds.yaml

# 4. install the CHA watcher (CRDs + diagnose-only watcher) into cha-playground
helm install cha-playground charts/cluster-health-autopilot \
  -n cha-playground --create-namespace \
  -f examples/playground/cha-values.yaml \
  --set image.repository=cha-local --set image.tag=playground

# 5. apply the isolated namespace, injector, and viewer
#    (point the viewer Deployment at your loaded tag first)
# (kind quick-try uses a locally-loaded viewer image; edit viewer.yaml's image tag
# for prod, or override via kustomize — do not commit the local-tag edit)
kubectl apply -k examples/playground/

# 6. fire one injection immediately (the CronJob runs every 15 min otherwise)
kubectl -n cha-playground create job --from=cronjob/cha-playground-drift-injector inject-now

# 7. watch the live findings
kubectl get driftreports -w
kubectl -n cha-playground port-forward svc/cha-playground-viewer 8080:80
#   open http://localhost:8080
```

You should see DriftReports for all four scenarios (sources `analyzer`
[`image-pull-auth` + `ABSENT_KEY`], `TLSSecretMismatch`, and `CrashLoopBackOff`).

### Teardown

```bash
kind delete cluster --name cha-pg
```

---

## Production / hosted deploy

On a real cluster (Kong ingress + cert-manager + GPU nodes labelled
`nvidia.com/gpu.present`):

```bash
# 1. CHA watcher into cha-playground (use the published OSS image tag)
helm install cha-playground charts/cluster-health-autopilot \
  -n cha-playground --create-namespace \
  -f examples/playground/cha-values.yaml

# 2. build + push the viewer to your registry, set viewer.yaml's image, then
docker build -f examples/playground/viewer/Dockerfile -t docker4zerocool/cha-playground-viewer:<tag> .
docker push docker4zerocool/cha-playground-viewer:<tag>
# edit examples/playground/viewer.yaml -> spec...image: docker4zerocool/cha-playground-viewer:<tag>

# 3. apply the isolated namespace + injector + viewer + ingress
kubectl apply -k examples/playground/
```

The viewer Ingress is `playground.asre.baisoln.com` and mirrors the cluster's
website/grafana ingress pattern: `ingressClassName: kong` +
`cert-manager.io/cluster-issuer: letsencrypt-prod` + TLS secret
`playground-asre-baisoln-com-tls`.

### DNS step (operator's final manual step — documented, NOT executed here)

Per the `dns-new-subdomains` rule, a new subdomain needs **both** a Cloudflare A
record **and** an entry in the deploy repo's DNS map:

1. Add to `deploy/lib/dns.sh` `DNS_DOMAINS`:
   ```bash
   ["playground.asre.baisoln.com"]="$METALLB_INGRESS_IP"   # 192.168.0.210
   ```
2. Create / upsert the Cloudflare A record (creds in Vault
   `secret/shared/cloudflare`):
   ```bash
   # from the deploy repo:
   ./deploy/lib/dns.sh   # upserts all DNS_DOMAINS entries
   ```
3. Verify resolution before cert-manager issues the cert (HTTP-01 self-check
   needs it):
   ```bash
   dig +short playground.asre.baisoln.com @1.1.1.1   # -> 192.168.0.210
   ```
   If the record was just created and cert-manager challenges hang, restart
   CoreDNS (it caches negative results).

> The `asre.baisoln.com` zone must exist in Cloudflare. If only `baisoln.com` is
> hosted, either add the `asre` sub-zone or use a host under an existing zone.

### Teardown (prod)

```bash
helm uninstall cha-playground -n cha-playground
kubectl delete -k examples/playground/
kubectl delete ns cha-playground
# the DriftReport/Silence/* CRDs are cluster-scoped and shared — only remove
# them if no other CHA install uses them.
```

## Notes / honesty

- The viewer is the **OSS** tiny read-only page (not the CHA-com P6.6 dashboard),
  so the whole playground is self-contained OSS and `kind`-runnable by anyone.
  Swapping in the CHA-com dashboard is possible but needs its private image.
- The CHA watcher runs **diagnose-only** (`remedy.enabled=false`). The playground
  shows DETECTION; it never mutates the cluster.
- Scenario 3 stamps the `Certificate` Ready itself because no cert-manager
  controller runs in the playground; the `TLSSecretMismatch` analyzer only reads
  `status.conditions`, so this is faithful to what it detects in production.
