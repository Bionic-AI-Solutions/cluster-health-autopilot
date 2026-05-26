# CHA-com Publishing Gap — v1.6.0 + Next Steps

**Status:** OPEN — closed-cluster pilots possible today; public paid sales blocked
**Date:** 2026-05-25
**Owner:** TBD (engineering + GTM)

The 2026-05-25 clean-machine deploy audit revealed three publishing gaps in
the paid (CHA-com) tier. This doc tracks them as a coherent piece of work
separate from the OSS engine.

## The three gaps

### G1 — No published CHA-com image

The CHA-com binary (`cmd/cha-com/main.go`) builds locally via the
repo's `Dockerfile`, but:

- There is **no GoReleaser config** at [CHA-com/.goreleaser.yaml](https://github.com/Bionic-AI-Solutions/CHA-com) (file doesn't exist)
- There is **no release-on-tag workflow** at `CHA-com/.github/workflows/`
- The OSS Helm chart's `approval.image.repository` defaults to
  `docker4zerocool/cha-com`, which **has never been pushed** to Docker Hub
- An operator who sets `approval.enabled=true` in `values.yaml` and runs
  `helm install` gets ImagePullBackOff with no diagnostic

**Fix:**
- Mirror the OSS engine's CI structure: `.github/workflows/release.yml` +
  `.goreleaser.yaml` in the CHA-com repo
- Publish to `docker4zerocool/cha-com` (canonical) + `ghcr.io/bionic-ai-solutions/cha-com` (mirror)
- Pin OSS dependency version in CHA-com's `go.mod` to the same tag as the
  CHA-com release tag (avoids OSS-vs-paid version drift)

**Estimate:** ~1 day. The OSS release.yml + .goreleaser.yaml already cover
everything; just port them over with the binary name changed.

### G2 — Paid catalog is empty ✅ CLOSED (2026-05-26)

All four planned paid analyzers shipped in CHA-com v1.0.1–v1.0.4:

- **`VaultPathDriftPro`** (v1.0.1, 8 tests) — OSS VaultPathMissing
  source is Apache-2.0 but requires manual Vault-client construction;
  the paid version auto-wires from env vars (VAULT_ADDR + token / K8s
  auth) AND adds an **unused-keys-at-path** detection: when a Vault
  payload has keys NO ExternalSecret references, surface them as a
  warning (attack surface + orphaned-config risk).
  Required promoting `internal/vault` → `pkg/vault` (OSS commit
  a9e78a4) so CHA-com could construct a client.
- **`CertificateChainAnomaly`** (v1.0.2, 9 tests) — static analysis
  of cert-manager-issued TLS Secrets. For every Certificate
  Ready=True, decodes the served `tls.crt` and surfaces weak keys
  (RSA <2048, ECDSA <P-256), deprecated signature algorithms
  (MD5/SHA1), SAN drift between spec.dnsNames and served cert's
  DNSNames, imminent-expiry-while-cert-manager-says-Ready races,
  and malformed Secret payloads. Test fixtures generate real x509
  certs at runtime (no hand-pasted base64).
- **`MultiClusterDrift`** (v1.0.3, 8 tests) — compares the current
  cluster's ExternalSecret references against N configured peer
  snapshots. Surfaces reference-differs (same ESO, different Vault
  paths) as warning; only-in-peer and only-in-current as info.
  Operator constructs peers in `cmd/cha-com` from snapshot directories.
- **`StatefulSetReplicaPressure`** (v1.0.4, 9 tests) — cluster-state-
  only (no external metrics) signals: replicas degraded (0/N critical,
  partial warning), PVC bind lag past 5min window (provisioner jammed),
  PVC resize stuck (spec.requests > status.capacity). Protected-
  namespace PVC-bind-lag is deferred to OSS DaemonSets probe to avoid
  double-flagging.

**Total**: 4 analyzers, 34 new tests, 4 paid-tier releases (v1.0.1
through v1.0.4) all pushed to docker4zerocool/cha-com and
ghcr.io/bionic-ai-solutions/cha-com. catalog.Config gained two new
fields (Vault, Peers) — operator wires either, both, or neither.

The marketing claim of "paid patterns 3 months earlier than OSS"
is now backed by concrete code; the G2 work also defined the
contribution pattern future analyzers follow (in-memory `memSource`
test fixture, single analyzer per file, RED-then-GREEN TDD, one
release tag per analyzer for rollback safety).

### G3 — AI tiers (T0–T3) are designed but not wired

`CHA-com/ai/` has `enricher.go`, `fix_proposer.go`, `planner.go`,
`vault_runbook.go` files with structs, validators, prompt templates,
and unit tests — but **none of them are wired into the cha-com binary's
runtime**. The main package at `cmd/cha-com/main.go` does not call
`enricher.Enrich()`, `fix_proposer.Propose()`, etc.

An operator deploying CHA-com with `ai.tier=t0` sees no 🤖 enrichment
blocks in Slack, even though all the code paths exist.

**What's needed:**

1. Wire `enricher.NewEnricher()` into the cha-com watcher's
   post-diagnose path (T0)
2. Wire `fix_proposer.NewFixProposer()` into the per-Critical-finding
   loop, alongside the OSS fixer registry (T1)
3. Wire `planner.NewPlanner()` for multi-step plans (T2)
4. Wire `vault_runbook.NewRunbookGenerator()` into the T3 dual-approval
   flow (T3)
5. Add a `cha-com` SETUP_GUIDE.md section bridging from OSS install to
   the AI-tier enablement
6. End-to-end test against an in-cluster vLLM endpoint (gpu-ai already
   serves `qwen3.6-35b-a3b-fp8` per the operator's environment)

**Estimate:** ~2 weeks (1 sprint) for T0+T1; T2+T3 are additional ~2 weeks each.

## Order of work

1. **G1 first (1 day).** Without a published image, nothing else can be
   demoed or sold. The CHA-com repo can ship an "empty" v1.0.0 cha-com
   image whose binary does only Layer-2 LLM-investigator (which IS
   already wired). That's a real, deployable v1.0.
2. **G3 in parallel with G2.** Wiring T0 + T1 unlocks the actual paid
   value prop (proposed-fix click flow). G2 (paid analyzers) unlocks
   the "patterns earlier than OSS" claim — important for marketing
   but not for the first paid pilot.

## What can be sold today

Closed-cluster pilots where Bionic builds and pushes the CHA-com image
to the customer's private registry manually. The OSS Helm chart's
`approval.image.repository` is operator-overridable; pointing it at a
customer's registry works. Not scalable, not a public-sale story.

## What can NOT be sold today

- Any deal where the customer expects to `helm install cha-com` from a
  public repo.
- Any deal where the marketing pitch of "AI tiers T0–T3" is taken at
  face value — the LLM-backed investigator (Layer-2) IS shipped, but
  T0/T1/T2/T3 enrichment + fix-proposal flows are not yet wired.

## Tracking

Companion to [2026-05-final-adversarial-review.md](2026-05-final-adversarial-review.md);
this doc supersedes the "AI tiers (T0–T3) are designed but not wired"
gap that the final review noted in passing.
