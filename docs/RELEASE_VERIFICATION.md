# Release Verification (SBOM + Cosign)

Every tagged `agentic-sre` release (`vX.Y.Z`) ships with
supply-chain provenance you can verify yourself:

- a **CycloneDX JSON SBOM** for each binary archive, attached to the GitHub
  Release;
- a **keyless cosign signature** over the `checksums.txt` file (covers every
  binary archive + SBOM transitively), attached as `checksums.txt.sigstore.json`;
- **keyless cosign signatures** over every container image and multi-arch
  manifest (Docker Hub `docker4zerocool/agentic-sre` and GHCR
  `ghcr.io/srenix-ai/agentic-sre`), stored as OCI
  signature objects in the registry and logged to the Rekor transparency log.

All signing is **keyless** (Sigstore): there is no long-lived private key.
The release workflow's GitHub OIDC token is exchanged for a short-lived Fulcio
certificate, so verification keys on the workflow identity rather than a key
file. You need [cosign](https://github.com/sigstore/cosign) v2.0+ installed.

The identity to verify against:

- **certificate-identity-regexp**: the release workflow file, on the tag ref —
  `https://github.com/srenix-ai/agentic-sre/.github/workflows/release.yml@refs/tags/v.*`
- **certificate-oidc-issuer**: `https://token.actions.githubusercontent.com`

> For convenience the snippets below export these as shell variables.

```bash
IMAGE=ghcr.io/srenix-ai/agentic-sre
VERSION=1.25.1   # the release you are verifying (image tag has no leading v)
IDENTITY='https://github.com/srenix-ai/agentic-sre/.github/workflows/release.yml@refs/tags/v.*'
ISSUER='https://token.actions.githubusercontent.com'
```

## 1. Verify a container image (or manifest) signature

```bash
cosign verify \
  --certificate-identity-regexp "$IDENTITY" \
  --certificate-oidc-issuer "$ISSUER" \
  "${IMAGE}:${VERSION}"
```

This works against the multi-arch manifest tag and the per-arch tags
(`:${VERSION}-amd64`, `:${VERSION}-arm64`). The same command works against the
Docker Hub copy — swap `IMAGE=docker4zerocool/agentic-sre`.

A successful run prints the verified certificate subject (the workflow
identity), the OIDC issuer, and the Rekor transparency-log entry. Any
tampering, a wrong identity, or an unsigned image fails the command non-zero.

## 2. Verify the checksums signature (covers binaries + SBOMs)

Download `checksums.txt` and `checksums.txt.sigstore.json` from the Release,
then:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp "$IDENTITY" \
  --certificate-oidc-issuer "$ISSUER" \
  checksums.txt
```

Once `checksums.txt` is trusted, verify any downloaded archive or SBOM against
it:

```bash
sha256sum --check --ignore-missing checksums.txt
```

## 3. Download and inspect the CycloneDX SBOM

Each binary archive has a sibling `*.cyclonedx.json` SBOM on the Release page,
e.g. `agentic-sre_linux_amd64.tar.gz.cyclonedx.json`.

```bash
# Confirm it is CycloneDX and list its components.
jq '{bomFormat, specVersion, components: (.components | length)}' \
  agentic-sre_linux_amd64.tar.gz.cyclonedx.json

# List component name@version pairs.
jq -r '.components[] | "\(.name)@\(.version)"' \
  agentic-sre_linux_amd64.tar.gz.cyclonedx.json
```

You can also extract the SBOM that cosign records as an attestation alongside
the image (when present) and scan any SBOM with Grype:

```bash
grype sbom:agentic-sre_linux_amd64.tar.gz.cyclonedx.json
```

## Notes

- The signing pipes (`signs`, `docker_signs`) only execute in CI, where a real
  GitHub OIDC token is available. Local `goreleaser release --snapshot` runs
  generate SBOMs but skip signing (use `--skip=sign`).
- `cosign verify` reaches out to the public Rekor + Fulcio roots by default; no
  extra trust configuration is required for releases built by this workflow.
