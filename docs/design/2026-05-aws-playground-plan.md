# AWS Playground — Implementation Plan (2026-05-27)

**Status:** Active — implementation gated on operator-provided AWS credentials.
**Driver:** The website's `/playground` route currently ships a `StubBody`
placeholder. The stub copy commits to a specific demo shape; this plan
turns that copy into a deployable artifact at `playground.asre.baisoln.com`
inside a $20/mo budget.

## Anchor spec (from the on-site stub copy)

  - Pre-provisioned `kind` cluster
  - LocalStack AWS sub-account (RDS, EBS, IAM, ALB, ACM)
  - Synthetic drift injected on a 30-second cycle
  - Live DriftReport stream visible in an iframe on the website
  - CTA: "Run this in your own cluster" → `helm install`

This plan honours that copy rather than re-litigating it.

## Locked decisions (2026-05-27)

  | Decision | Choice | Rationale |
  |---|---|---|
  | Cost ceiling | **$20/mo** — single `t3.small` on-demand | Predictable bill; no spot interruptions; comfortably fits kind + LocalStack + drift injector + viewer |
  | Access gate | **Anonymous read-only** | Matches the existing stub copy; lowest friction; lowest abuse surface (no mutation path) |
  | DNS | **`playground.asre.baisoln.com`** subdomain | Reuses existing Cloudflare zone + ACM cert flow; zero new domain registration |

## Architecture (one box, four containers)

```
playground.asre.baisoln.com (Cloudflare DNS → AWS ALB → EC2)
                    │
                    ▼
   ┌──────────────────────────────────────────────────────┐
   │   1× EC2 t3.small  (us-east-1, public subnet)        │
   │   Amazon Linux 2023 + Docker                         │
   │                                                      │
   │   ┌───────────┐  ┌────────────┐                      │
   │   │  kind     │  │ LocalStack │                      │
   │   │  cluster  │  │ community  │                      │
   │   │           │  │ (RDS, EBS, │                      │
   │   │  Srenix      │──│  IAM, ALB, │                      │
   │   │  watcher  │  │  ACM)      │                      │
   │   │           │  └────────────┘                      │
   │   │ drift-inj │  ┌────────────┐                      │
   │   │ CronJob   │  │ viewer-svc │                      │
   │   │ (30s)     │──│ SSE stream │── ALB :443           │
   │   └───────────┘  └────────────┘                      │
   └──────────────────────────────────────────────────────┘
```

### Drift-injector CronJob (4-scenario rotation, every 30s)

  1. Create a Pod with a deliberately bad image-pull secret → Srenix emits an
     `ImagePullAuth` DriftReport → Srenix fixer re-syncs the ExternalSecret →
     resolved.
  2. Trigger LocalStack RDS `StorageFull` event → Srenix cloud-probe emits a
     DriftReport → static "ticketed to OpenProject" message (no cloud
     mutation fixer in OSS).
  3. Patch a Service annotation to an invalid TLS secret name → Srenix
     `TLSSecretMismatch` fixer kicks in → resolved.
  4. Delete a Job pod mid-run → Srenix `StuckJob` fixer cleans it up.

Four scenarios on a rotation gives every visitor session at least one full
open → fix → verify cycle in under a minute.

### Viewer service

  - A 100-line nginx + Go SSE shim.
  - Scrapes `kubectl get driftreports -A -o json` every 2 seconds.
  - Emits Server-Sent Events to subscribers.
  - Tiny HTML page renders a live timeline (open / fix / resolved labels);
    no JS framework.
  - Read-only RBAC — even if the SSE endpoint is exfiltrated, the visitor
    cannot mutate.

## Cost breakdown (target $20/mo)

  | Line | Monthly |
  |---|---|
  | EC2 `t3.small` on-demand (us-east-1) 24/7 | $15.18 |
  | EBS gp3 20 GiB | $1.60 |
  | ALB (1 LCU avg for a public read-mostly endpoint) | $2.50 |
  | Route53 hosted zone | $0.00 (Cloudflare DNS) |
  | ACM cert | $0.00 |
  | CloudWatch logs (1 GiB/mo) | $0.50 |
  | Data egress (~10 GiB/mo) | $0.90 |
  | **Total** | **~$20.68/mo** |

Reserved-instance or savings-plan after 90 days of stable usage drops the
EC2 line ~30%.

## Implementation phases (~6 days end-to-end)

  | Phase | Deliverable | Effort | Operator-provided input |
  |---|---|---|---|
  | **0** | Dedicated AWS sub-account + budget alert ($25 hard cap) + IAM admin role | 0.5d | Account ID + admin IAM credentials (or an OIDC role to assume); budget-alert email |
  | **1** | Terraform: VPC + public subnet + IGW + SG + EC2 + EBS + ALB + ACM + Route53 record (in Cloudflare) | 1d | Confirm region (default us-east-1); confirm Cloudflare API token in Vault at `secret/shared/cloudflare` |
  | **2** | EC2 bootstrap: Docker + kind + LocalStack + Srenix helm install + bootstrap script (idempotent, reruns clean) | 1d | — |
  | **3** | Drift-injector: 4-scenario rotation CronJob + LocalStack fixtures + idle-state cleanup | 1d | — |
  | **4** | Viewer service: SSE shim + minimal HTML timeline + ALB ingress rule | 1d | — |
  | **5** | TLS + Cloudflare DNS + smoke tests + abuse-rate-limit at ALB + CloudWatch dashboard | 1d | — |
  | **6** | Replace `/playground/` `StubBody` in `srenix-website` with an iframe of the live viewer; cutover; verify live | 0.5d | — |

## Risks & mitigations

  - **R1 — `t3.small` memory pressure** (2 GiB w/ Docker + kind + LocalStack
    + Srenix). Mitigation: LocalStack community-edition lite mode (skip heavy
    services like S3 multipart); cap Srenix's watcher resync to 5 minutes;
    profile actual usage on day 1 — if memory utilisation spikes > 70%,
    upgrade to `t3.medium` (~$30/mo).
  - **R2 — LocalStack community coverage gaps**. The 5 services Srenix's AWS
    probes touch (RDS / EBS / IAM / ALB / ACM) are all in the community
    tier, so we sidestep the Pro tier ($30/mo). Documented limitation:
    GCP / Azure playground probes wait for the v1.8 cloud-probe expansion
    against real cloud accounts (LocalStack doesn't emulate them).
  - **R3 — Anonymous abuse**. Read-only iframe → no mutation surface. ALB
    rate-limit (100 req/min per IP) gates SSE reconnect storms. CloudWatch
    alarms on 4xx spike + EC2 CPU > 80%.
  - **R4 — Crawlers grabbing the SSE endpoint as a feed**. Don't care — it
    *is* the marketing artefact; the live DriftReport stream IS the demo.
  - **R5 — Credentials supplied to the implementer**. Use only for the
    EC2 / VPC / ALB / EBS / ACM / Route53 scope listed in Phase 1
    Terraform; the SCP on the sub-account should block anything else. No
    persistent storage of credentials in the website repo or chat memory.
  - **R6 — Cost overrun**. Budget alert at $25/mo is a hard guardrail;
    Phase 0 sets a CloudWatch billing alarm that pages on threshold.

## Out of scope (deliberately)

  - **Per-visitor ephemeral cluster.** Cost blow-up; visitors don't expect
    to mutate a demo.
  - **Srenix Enterprise paid features in the playground.** No LLM call out — that
    needs an upstream BYO-LLM endpoint, and playground visitors aren't
    paying for the agent flow. The CTA on the playground page makes the
    upgrade path explicit.
  - **Multi-region or HA.** Single AZ; if it goes down, the iframe shows a
    "playground is down" banner and the CTA still works.
  - **GCP / Azure cloud-probe demonstration.** Waits for v1.8 with real
    cloud accounts (LocalStack doesn't emulate them).
  - **Real RDS / EBS in the playground AWS account.** Everything stays
    inside LocalStack — that's the cost containment.

## Operator-provided inputs (collected at implementation start)

  1. AWS sub-account ID (recommend creating a new one specifically for the
     playground; isolate the blast radius).
  2. IAM admin role / credentials in that sub-account (or an OIDC trust the
     implementer can assume).
  3. Budget alert email address.
  4. Confirmation that the Cloudflare API token currently in Vault at
     `secret/shared/cloudflare` works for the `asre.baisoln.com` zone (per
     the `dns-new-subdomains` memory, it does — `deploy/lib/dns.sh` already
     manages records under this zone).

## Acceptance criteria

  - `playground.asre.baisoln.com` returns HTTP 200 over TLS.
  - The viewer renders at least one DriftReport state-change per minute
    when observed for 5 minutes (4-scenario rotation × 30-second cycle =
    ~8 state transitions / 4 minutes).
  - CloudWatch billing alarm exists and is wired to the operator-provided
    email.
  - Helm install CTA on the iframe redirects to the OSS repo `README.md`
    install section.
  - `/playground/` on the srenix-website iframes the live viewer (replaces
    the `StubBody`).
