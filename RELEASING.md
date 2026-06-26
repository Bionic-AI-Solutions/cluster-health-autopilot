# Releasing Agentic SRE

Per-checkin verification runs **locally**. GitHub Actions spends CI minutes
only on release tags and on-demand dispatch — there is no per-PR CI gate.

## 1. Per change (every PR)

Run the full verification suite locally, then open and merge the PR:

```bash
make verify          # == bash scripts/verify-local.sh
# … open PR, get review, merge. No CI gate — verification is local.
```

`make verify` mirrors `.github/workflows/ci.yml` exactly: `go mod verify`,
`go vet ./...`, `go build`, `srenix version`, `go test -race -count=1`,
`golangci-lint run`, the CHANGELOG lint + tag-check (+ its selftest), and
`helm lint` / `helm unittest` / `helm template` smoke. It fails fast on the
first failing step and prints a PASS banner at the end.

## 2. Major release (tag-driven)

1. Update `CHANGELOG.md` — add a new `## [0.X.Y-alpha.N] — DATE` section at the
   top (directly under `## [Unreleased]`).
2. Bump `charts/agentic-sre/Chart.yaml` `version` **and**
   `appVersion` to `0.X.Y-alpha.N`.
3. Run `make verify` one last time.
4. Tag and push:

   ```bash
   git tag v0.X.Y-alpha.N
   git push origin v0.X.Y-alpha.N
   ```

The tag push triggers:

- **`release.yml`** — builds the signed + SBOM release (goreleaser; the
  version is injected from the tag, not hardcoded).
- **`bundle-smoke.yml`** — OLM-on-kind validation of the operator bundle.
- **`helm-publish.yml`** — packages the chart and publishes it to the Helm
  repo at <https://srenix-ai.github.io/agentic-sre/>.

## 3. Versioning policy

The project is **pre-launch**. Releases through `v1.26.3` were internal
pre-alpha iterations mis-numbered as `1.x`; versioning was reset at
`0.1.0-alpha.1` to honest SemVer.

- Pre-launch releases use SemVer **0.x** with `-alpha.N` pre-releases.
- Bump `alpha.N` for each pre-alpha release (`0.1.0-alpha.1` →
  `0.1.0-alpha.2` → …).
- Promote to a `0.x.0` minor / `1.0.0` GA only at the relevant product
  milestones — not per checkin.

## 4. On-demand CI (optional pre-tag gate)

The CI workflows are manual-only / tag-triggered. To run the GitHub gate
before tagging (e.g. to validate on a clean runner):

```bash
gh workflow run ci.yml
gh workflow run bundle-smoke.yml
```
