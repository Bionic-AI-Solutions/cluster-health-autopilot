# Changelog

All notable changes to this project will be documented in this file. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The Helm chart `cluster-health-autopilot` follows the same version line as the
`cha` binary (`appVersion == version`). Released chart artifacts are tagged
`cluster-health-autopilot-X.Y.Z`; the binary itself is tagged `vX.Y.Z`. The
published Helm repository at
`https://bionic-ai-solutions.github.io/cluster-health-autopilot/` always
serves the latest tagged chart cut.

## [Unreleased]

### Added
- `LICENSE-VERIFIED-LIBRARY.md` — formal terms for the paid Verified Signature Library subscription, replacing the placeholder reference in README.
- README section documenting the AWS cloud probes (RDS, EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC) that were already shipping but undocumented.
- README link to `docs/READINESS.md` so prospects find the pilot-vs-production limits doc before the install step.
- `docs/AI_COST_MODEL.md` — failure-mode amplification section covering flapping-workload cost blowup, pre-Sprint-3 investigation rate-limiter gaps, and the worst-case planning table.
- `docs/design/2026-05-hardening-plan.md` — TDD-driven Sprint 0–4 plan closing the 2026-05-22 adversarial review.

### Changed
- README architecture section now describes the actual Go-binary-on-distroless image and the three ClusterRoles (reader, remediator, driftreport) — the old description of a bash/jq/curl container and "two ClusterRoles" was inherited from a v0.x iteration.
- README and `docs/CHA_OVERVIEW.md` clarify that `VaultPathMissing` source code is Apache-2.0 OSS but ships unwired (you supply the Vault client); the paid CHA Enterprise binary auto-wires it.
- README roadmap section replaced the user-local path with links to `docs/design/`.
- `docs/FAILURE_MODES.md` analyzer count corrected from "seven" to "eight"; intro now distinguishes "source ships OSS" vs. "auto-wired in paid."

### Fixed
- (Sprint 0 docs truth-up — code unchanged.)

---

## [1.5.2] — 2026-05-11

### Fixed
- Watcher now re-runs the Layer-2 investigator after fixers apply; the resulting investigation is reflected in the DriftReport CR.
- DriftReport CR severity and message refresh on update, not just on create.

## [1.5.1] — 2026-05-11

### Fixed
- Investigation field now persists on the DriftReport CR.

## [1.5.0] — 2026-05-12

### Added
- Layer-2 Investigator: read-only deep-dive on CRITICAL findings.
- OSS ships a deterministic, rule-based investigator (TLS expiry, TLS SAN mismatch, DNS, slow-DNS, status mismatch, ExternalSecret, Certificate state).
- CHA-com paid binary swaps in an LLM-backed investigator via the same `Environment` interface.

## [1.4.0] — 2026-05-12

### Added
- Probe flake suppression: retry + N-of-M streak gating before escalation to CRITICAL. Deterministic failures (TLS error, status mismatch) bypass the streak counter.

## [1.3.0] — 2026-05-12

### Added
- `TLSSecretMismatch` analyzer + opt-in fixer that repoints `Ingress.spec.tls[].secretName` to the cert-manager-managed Secret. Skips GitOps-managed Ingresses.

## [1.2.0] — 2026-05-12

### Added
- Ingress host auto-discovery: every Ingress host in the cluster is probed externally by default.

### Removed
- `IngressCoverage` analyzer (replaced by auto-discovery).

## [1.1.0] — 2026-05-12

### Added
- Expanded default endpoint probe coverage.

## [1.0.0] — 2026-05-11

### Fixed
- AI-related Helm templates are now nil-safe for `--reuse-values` upgrades.

## [0.9.5] — 2026-05-11

### Added
- External endpoint probe.
- Ingress coverage analyzer (later superseded in 1.2.0).
- Rewritten `SETUP_GUIDE.md` for v0.9.5; corrected `NOTES.txt` template.

## [0.9.1] — 2026-05-08

### Added
- `StuckCertificateRequests` fixer: deletes terminal-failed cert-manager Certificate Requests so cert-manager re-issues.

## [0.9.0] — 2026-05-07

### Added
- `cha watch --live` event-driven watcher with Slack dedup (Phase 1).

---

For releases earlier than 0.9.0, see the git tag list and PR titles on GitHub.
