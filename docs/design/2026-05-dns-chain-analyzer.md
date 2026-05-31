# DNS-chain drift analyzer — `DNSChainDrift` (2026-05-30)

**Status:** Design — implementation slated for v1.10.x (post v1.9 operator Phase 1c).
**Parent context:**
  - `internal/probe/endpoints.go` — current static probe target list + auto-discovery.
  - `internal/investigator/rules.go:131` — current DNS failure handling (lookup + "go check Cloudflare" hint).
  - `feedback_dns_new_subdomains` memory — every new host needs a Cloudflare record AND an entry in `deploy/lib/dns.sh`; today CHA detects neither.
  - `cha-ai-remediation-direction` memory — judgment-class fixes must be AI-gated + memory-grounded; this design lands the **detection** half, so a later AI-gated fixer has structured findings to act on.
**Scope:** A new OSS analyzer that walks the DNS-→-Ingress-→-Service-→-Endpoints chain for every host the cluster claims to serve, and emits a structured diagnostic naming the **specific broken link** when any layer fails. Optional Cloudflare integration extends the chain to include the external-DNS hop.

This closes a real gap. Today, when `https://livekit.bionicaisolutions.com` fails to probe, CHA reports "DNS resolution for livekit.bionicaisolutions.com failed: NXDOMAIN. Check Cloudflare / upstream DNS records." That's a hint, not a diagnosis. It does not tell the operator:

  - Whether a Cloudflare record exists at all, and if so where it points.
  - Whether the cluster Ingress for that host exists, and what Service it points to.
  - Whether that Service has any ready Endpoints.

Each of those is a different fix. The current design forces the operator (or oncall) to walk the chain by hand every time. This analyzer walks it once per cycle and emits a fingerprinted finding per broken host.

## 1. The chain

For each candidate host `H`, the analyzer verifies:

```
Cloudflare DNS record for H        ──→ value matches → cluster ingress LB IP (or expected target)
                                              │
                                              ▼
   Ingress resource with H in spec.rules[].host  ──→ backend → Service `S` (name/port) in namespace `N`
                                              │
                                              ▼
                              Service N/S exists ──→ selector matches → Endpoints has ready addresses
```

A finding is emitted for the **highest layer that's broken** (operator only needs to fix one thing at a time). Lower layers are reported as observations in the diagnostic body but don't generate independent findings — a missing Cloudflare record is the root cause; the resulting Ingress-orphan doesn't need its own ticket.

## 2. Why this is OSS, not a paid (CHA-com) analyzer

| Question | Answer |
|---|---|
| Is the K8s side (Ingress → Service → Endpoints) universal hygiene or org-specific? | Universal. Every K8s cluster has this chain. The existing OSS analyzers `TLSSecretMismatch` and `FailingExternalSecrets` already do equivalent cross-resource correlation. |
| Where does the Endpoints probe live today? | OSS (`internal/probe/endpoints.go`). Moving the chain check elsewhere splits the surface — DNS failures get reported in OSS but the diagnosis lives in paid. |
| Does Cloudflare integration require paid-tier infra? | No. The Cloudflare API is a single HTTP call with a read-only token; the credential surface is the same shape as the existing `secret/shared/cloudflare` Vault path. It's optional config, not a new infra dependency. |
| What's the natural paid-tier extension? | Multi-zone / multi-cloud DNS reasoning — Route53 + Cloudflare + GCP Cloud DNS across multi-cluster setups, plus drift detection vs a canonical DNS-as-code source (`deploy/lib/dns.sh`). That sits cleanly on top of this OSS analyzer; it does not replace it. |
| Does the existing GTM model support this? | Yes. Per `cha_gtm` memory, the open-core split is "OSS = base patterns / paid = pro patterns (vault drift pro, multi-cluster drift)". This analyzer is a base pattern. |

**Recommendation: OSS analyzer with Cloudflare config optional.** When Cloudflare creds are present, the full chain runs; when absent, the analyzer runs the K8s hops and emits an info-tier "external DNS hop not verified" annotation on each host so the operator knows the chain is partial.

## 3. Host discovery

Per the discussion that scoped this design: **all cluster Ingress hosts + `DefaultEndpointTargets` static list**.

  1. **Ingress discovery** — reuse `internal/probe/ingress_discovery.go` semantics: list every `Ingress` in every namespace, extract `spec.rules[].host`, honour the existing `cha.bionicaisolutions.com/probe-disable` annotation as an opt-out.
  2. **Static seed list** — pull from `probe.DefaultEndpointTargets()` (the same source the `Endpoints` probe uses), de-duped against the Ingress set. Hosts on this list that have NO matching Ingress in-cluster are flagged as "external-only" — they're expected to be served from outside the cluster (e.g. apex domains via Cloudflare Pages), so the analyzer skips the Ingress/Service/Endpoints layers and only verifies the Cloudflare layer.
  3. **Deduplication key:** lowercased host. Multiple Ingresses for the same host short-circuit to "the first Ingress with `H` in its rules; the others are co-routes."

This matches the existing OSS UX pattern (default-on, opt-out via annotation). It does mean dev/staging hosts get checked too; operators can silence with the existing annotation, and `feedback_dns_new_subdomains` makes clear that *every* host should be tracked in dns.sh anyway.

## 4. Layer-by-layer behaviour

### 4.1 Cloudflare layer (optional, fires when configured)

Config surface: a new top-level CRD field `spec.externalDNS.cloudflare` mirroring how the AI section is gated:

```yaml
spec:
  externalDNS:
    cloudflare:
      enabled: true
      apiTokenSecretRef:
        name: cloudflare-api-token
        key: token
      # Zone IDs CHA is allowed to query. Empty = all zones the token can see.
      zoneIDs: []
      # Optional: the expected target value(s) DNS records should point at.
      # Multiple values allowed (e.g. ingress LB IPv4 + IPv6).
      # When unset, the analyzer auto-derives from the ingress controller Service.
      expectedTargets: []
```

Per-host check:

| State | Diagnostic | Severity |
|---|---|---|
| No A/AAAA/CNAME record exists for `H` in any visible zone. | Subject: `missing-cloudflare-record/<H>`. Remediation: "Add `<H>` to `deploy/lib/dns.sh` and run the deploy step; expected target: `<expectedTarget>`." | error |
| Record exists, value ≠ `expectedTarget`. | Subject: `cloudflare-points-elsewhere/<H>`. Message includes both observed and expected values. Remediation: "Update the Cloudflare record for `<H>` to point at `<expectedTarget>`, or update `expectedTargets` if this is intentional." | error |
| Record exists, proxied=true, expected=unproxied (or vice-versa) | Subject: `cloudflare-proxy-drift/<H>`. | warn |
| Record exists and matches. | (no finding; chain continues) | — |

The Cloudflare client is a small HTTP wrapper (~200 LOC, no SDK dep) — list zones, list DNS records per zone, filter by name. Credential discovery: `apiTokenSecretRef` is a Kubernetes Secret resolved via the snapshot.Source (no direct API call), matching how `pkg/ticketing/openproject/client.go` resolves its token.

### 4.2 Ingress layer

For each `H` discovered via Ingress discovery:

| State | Diagnostic | Severity |
|---|---|---|
| `H` is on the static `DefaultEndpointTargets` list but no Ingress in any namespace has it in `spec.rules[].host`, AND it's not flagged external-only. | Subject: `missing-ingress/<H>`. Remediation: "Create an Ingress with `spec.rules[].host=<H>` pointing at the intended backend Service, or mark `<H>` external-only in the endpoint target list." | error |
| Ingress `<ns>/<name>` lists `H` but `spec.rules[].http.paths[].backend.service` references a Service `<ns>/<svc>` that does not exist. | Subject: `ingress-orphan-service/<ns>/<name>/<H>`. Remediation: "Service `<ns>/<svc>` referenced by Ingress `<name>` does not exist. Either deploy the Service or update the Ingress backend." | error |
| Multiple Ingresses claim `H`. | Subject: `duplicate-ingress-host/<H>`. Message lists all Ingresses with `H`. | warn |

### 4.3 Service / Endpoints layer

For each `(H, Service N/S)` pair resolved from the Ingress layer:

| State | Diagnostic | Severity |
|---|---|---|
| Service `N/S` exists but its `Endpoints` object has zero ready addresses. | Subject: `service-no-endpoints/<N>/<S>/<H>`. Remediation: "Service `<N>/<S>` selector matches no ready pods. Check backing Deployment/StatefulSet replica state and pod readiness probes." | error |
| Service `N/S` has `spec.type=ExternalName` but Ingress references it as a pod-backed service. | Subject: `service-external-name-mismatch/<N>/<S>/<H>`. | warn |

(Pod-state diagnosis is left to the existing `workload_state_drift` analyzer; this analyzer stops at "Endpoints has zero ready addresses" and the operator follows the existing diagnostic chain down to the workload.)

## 5. Output shape — worked example

For `https://livekit.bionicaisolutions.com` when the Cloudflare record is missing:

```
Subject:     missing-cloudflare-record/livekit.bionicaisolutions.com
Severity:    error
Source:      DNSChainDrift
Message:     livekit.bionicaisolutions.com: no Cloudflare DNS record found in the
             configured zones (bionicaisolutions.com, baisoln.com). Ingress
             vc-livekit/livekit-ingress exists and points at Service
             vc-livekit/livekit (5 ready endpoints) — the cluster side is healthy;
             only the external DNS hop is missing.
Remediation: Add `livekit.bionicaisolutions.com` to deploy/lib/dns.sh with target
             <ingress-LB-IP>, then re-run the deploy step. Suggested record:
             A livekit.bionicaisolutions.com → <ingress-LB-IP> (proxied=false).
```

Two qualities to call out:
  1. The Message names what IS working (Ingress + Service + Endpoints) so the operator knows the K8s side is fine. This is the chain analyzer's distinguishing feature vs the existing probe.
  2. The Remediation names the *exact* file (`deploy/lib/dns.sh`) the change goes in, per the operational pattern in `feedback_dns_new_subdomains`.

## 6. Interface + file layout

Files to add:

| File | Purpose |
|---|---|
| `internal/diagnose/dns_chain_drift.go` | Analyzer impl. Implements `pkg/diagnose.Analyzer`. |
| `internal/diagnose/dns_chain_drift_test.go` | Table-driven tests across all layers + fixture snapshots. |
| `internal/dns/cloudflare/client.go` | Minimal Cloudflare API client (zones list + records list). ~200 LOC, no SDK. |
| `internal/dns/cloudflare/client_test.go` | Test against a fake HTTP server (httptest.Server). |
| `internal/dns/resolver.go` | Cluster-LB-target derivation (find the ingress controller Service, extract its `status.loadBalancer.ingress[].ip`). Used as the "expected target" when `expectedTargets` is empty. |

Files to modify:

| File | Change |
|---|---|
| `catalog/catalog.go` | Append `diagnose.DNSChainDrift{...}` to the `r.RegisterAnalyzer(...)` call. |
| `api/v1alpha1/clusterhealthautopilot_types.go` | Add `ExternalDNS` field to Spec; CRD codegen via `make generate manifests`. |
| `charts/cluster-health-autopilot/templates/configmap.yaml` (and the operator builder for the equivalent env vars) | Plumb `externalDNS.cloudflare.*` through to the analyzer. |
| `internal/probe/endpoints.go` | No code change, but the comment at L350–360 (`DefaultEndpointTargets returns the canonical set...`) gets a back-reference: "Also walked by the DNS-chain analyzer to verify Cloudflare wiring." |

Interface signature:

```go
type DNSChainDrift struct {
    Client       cloudflare.Client     // nil when Cloudflare hop is disabled
    SeedTargets  []probe.EndpointTarget // typically probe.DefaultEndpointTargets()
    OptOutAnno   string                 // "cha.bionicaisolutions.com/probe-disable"
    Resolver     dnsresolver.LBResolver // derives expected target from ingress-controller Svc
}

func (d DNSChainDrift) Name() string { return "DNSChainDrift" }
func (d DNSChainDrift) Run(ctx context.Context, src snapshot.Source) []diagnose.Diagnostic
```

The Cloudflare client is injected as an interface (not a concrete client) so tests use a fake, and so a future paid analyzer can inject a multi-provider `dns.Resolver` without touching the OSS analyzer.

## 7. Test plan

Unit tests, all driven by snapshot fixtures (`pkg/snapshot/fakesrc.go` pattern, same as `tls_secret_mismatch_test.go`):

  1. **Happy path** — Cloudflare record matches, Ingress exists, Service has Endpoints. Expect: zero diagnostics.
  2. **Missing Cloudflare record** — no record in any zone. Expect: one `missing-cloudflare-record/<H>` finding, no lower-layer findings (chain short-circuits).
  3. **Cloudflare points elsewhere** — record exists with wrong IP. Expect: `cloudflare-points-elsewhere/<H>` finding with both values in Message.
  4. **Missing Ingress for seed host** — host on `DefaultEndpointTargets` but no Ingress claims it, and not flagged external-only. Expect: `missing-ingress/<H>`.
  5. **Ingress orphan Service** — Ingress references a Service that doesn't exist. Expect: `ingress-orphan-service/<ns>/<name>/<H>`.
  6. **Service no endpoints** — Service exists, selector matches nothing. Expect: `service-no-endpoints/<N>/<S>/<H>`.
  7. **Duplicate Ingress hosts** — two Ingresses claim the same host. Expect: `duplicate-ingress-host/<H>` (warn).
  8. **Opt-out annotation** — Ingress has `cha.bionicaisolutions.com/probe-disable=true`. Expect: no findings for any host on that Ingress.
  9. **Cloudflare disabled** — `Client` is nil. Expect: K8s-layer findings only, plus exactly one info-tier "external DNS hop not verified" diagnostic per host (collapsed into one summary finding, not N).
  10. **External-only seed host** — on seed list, no Ingress, Cloudflare record exists and matches. Expect: zero findings (Cloudflare-verified, K8s side intentionally absent).
  11. **CRD absent (Ingress GVR not installed)** — degenerate cluster. Expect: nil return, no panic.
  12. **Cloudflare API timeout** — fake client returns ctx.DeadlineExceeded. Expect: nil return, no panic, no findings (analyzer fails open per `pkg/diagnose` contract — better to miss a check than to flap).

Integration verification (post-merge, run by hand on the Bionic cluster):

  - Confirm finding on `livekit.bionicaisolutions.com` matches reality.
  - Toggle Cloudflare token off, re-run, confirm degraded mode.
  - Add a new Ingress for `dns-chain-test.baisoln.com` *without* a Cloudflare record, confirm `missing-cloudflare-record` finding fires within one watcher cycle.

## 8. Failure modes & guardrails

  - **Cloudflare API rate limit (1200 req / 5 min per token).** The client caches the zone list for the watcher's cycle interval (default 60s) and lists all records per zone in one paged call rather than one-per-host. For a cluster with ~100 Ingress hosts across 2 zones, that's 2 + 4 calls/cycle (assuming 200 records/page × 4 pages max), well under the limit.
  - **Reduced read RBAC if the Cloudflare token can't see a zone.** The analyzer treats "zone not visible to token" as "external DNS hop not verified for hosts in this zone" — same degraded mode as Cloudflare disabled.
  - **Analyzer fail-open contract.** Per `pkg/diagnose` Run contract, any error inside the analyzer returns nil/empty rather than propagating. The Cloudflare call timeout is bounded (5s default); the entire analyzer is bounded by the watcher's per-analyzer deadline.
  - **No mutation, ever.** This analyzer is read-only. The follow-up AI-gated fixer (deferred to a separate design) is the only thing allowed to modify Cloudflare records or Ingress hosts.

## 9. Cross-references to the trigger-expansion roadmap

The `project_cha_trigger_expansion_roadmap` memory already includes a milestone for expanding trigger classes A→A+B+C+D+E. This analyzer fits cleanly under **Class B (config drift)** — DNS configuration is config-as-data — and adds a new probe / analyzer pair that doesn't overlap with the existing Kong / HPA / Velero / Vault / ArgoCD probes scoped in v1.8.

If the implementation slips, the roadmap doc should be updated to call this out explicitly under the v1.10 milestone.

## 10. Out of scope (deferred follow-ups)

  - **AI-gated fixer.** The natural next step (per `cha_ai_remediation_direction` memory): an `AIProposedAction` linked to `missing-cloudflare-record/<H>` findings that proposes the exact `deploy/lib/dns.sh` patch + Cloudflare API call, gated through approval-server. This needs its own design doc.
  - **DNS-as-code drift detection** vs `deploy/lib/dns.sh`. Logical follow-on but requires CHA to read the deploy repo. Better as a multi-cluster / paid-tier analyzer that runs against a checked-in source-of-truth.
  - **Route53 / GCP Cloud DNS** providers. Deferred to the paid `MultiCloudDNSDrift` analyzer (parallel to existing `MultiClusterDrift` paid analyzer).
  - **Cert chain validation.** Already covered by `CertExpiry` + `TLSSecretMismatch` analyzers; this analyzer deliberately stops at DNS.
  - **Cloudflare WAF / page rule drift.** Not health-related; out of scope for CHA.

---

**Implementation estimate:** ~3 days. Cloudflare client (1d) + analyzer (1d) + tests + chart wiring + CHANGELOG (1d). Aim to land in v1.10.0.
