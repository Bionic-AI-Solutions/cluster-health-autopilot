# Phase 2.H — Cosign-Signed PRs from DigestPin

**Status:** active sub-plan; execution deferred to a focused supply-chain session.

**Parent:** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

---

## Goal

The DigestPinProposer opens PRs that pin container images to observed digests. Today those PRs are auto-merged on click but the only authentication of "did Srenix Enterprise really propose this?" is the GitHub PAT.

Phase 2.H attaches a cosign signature to the proposed change so reviewers can:
1. Verify the change came from Srenix Enterprise's signing identity (not someone with a stolen PAT)
2. Verify the proposed digest matches what Srenix Enterprise OBSERVED at proposal time (not an MITM-rewritten value)

## Anti-goals

- Don't sign the PR itself (not supported by cosign). Sign a sidecar artifact.
- Don't require a transparency log (no Rekor) for v1; offline signature is sufficient.
- Don't ship operator-facing key management. Key lives in Vault, mounted into the watcher.

## Sub-tasks

### 2.H.1 — Cosign signing identity provisioning

- Generate ECDSA-P256 keypair via `cosign generate-key-pair`
- Private half → Vault at `secret/srenix-enterprise/cosign/key` (encrypted, ESO-synced to a K8s Secret)
- Public half → checked into the Srenix repo's `docs/security/cosign-public-key.pem` so reviewers can fetch it from the canonical source
- Document in CHANGELOG + README how to verify

### 2.H.2 — Sign the proposed-digest payload at proposal time

- In `ai/proposer/digest_pin.go::Propose` BEFORE the `Forge.UpdateFile` call:
  - Build a canonical "proposal manifest" JSON: `{action_id, repo, ref, file_path, before_digest, after_digest, observed_at}`
  - Sign with `cosign sign-blob` (key from Vault) → base64 signature string
  - Include the signature + payload in the PR BODY as a `srenix-cosign-attestation:` block (machine-parseable)

### 2.H.3 — Optional verify step in PR template

- PR description includes a one-liner: `cosign verify-blob --key https://path/to/public.pem --signature <sig> --payload-file <file>`
- Reviewers can run it locally before approving

### 2.H.4 — Watcher binary embeds cosign-CLI calls

Two options:
- (A) Embed `sigstore/cosign/v2` Go library directly — no CLI dependency in the container
- (B) Bundle the cosign binary in the srenix-enterprise Docker image — simpler but adds ~30MB

Prefer (A) for image-size reasons.

### 2.H.5 — Backwards compatibility

When the signing key is unavailable (Vault outage, ESO not configured), DigestPin still opens the PR — just without the attestation block. Logs a warning. Reviewers can manually verify the digest matches the registry.

### 2.H.6 — Field-travels integration test

Mock a digest-pin proposal end-to-end. Verify the PR body contains a parseable `srenix-cosign-attestation:` block + the signature verifies against the embedded public key.

### 2.H.7 — Local build + cluster verify

- Build srenix-enterprise with the cosign sigstore library
- Trigger a real digest-pin proposal
- Verify the PR carries the attestation
- Manually run `cosign verify-blob` to confirm it validates

## Risk

- **Key rotation** — first version has no automated rotation. Document a manual process; revisit in 2.X.
- **Verify-blob CLI version skew** — pin a known-good cosign version in docs.
- **Vault dependency** — if Vault is down, digest-pin PRs degrade silently to unsigned. Acceptable: the existing GitHub PAT still authorizes the PR.
