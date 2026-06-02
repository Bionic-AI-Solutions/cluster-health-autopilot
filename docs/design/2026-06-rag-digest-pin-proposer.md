# Phase 2d-γ — RAG-Driven Image Digest-Pin Proposer

**Status:** Design draft
**Tier:** Paid (CHA-com / AI tier, gated on `spec.ai.enabled`)
**Author:** opened 2026-06-01
**Parent:** [2026-06-rag-cluster-knowledge.md](2026-06-rag-cluster-knowledge.md)

## Why

The CHA Pod analyzer flags every container running with a mutable `:tag` reference instead of an immutable `@sha256:` digest. Real-world hit: 143 pods on the dev cluster, spread across:

  - **28 in-house** (`docker4zerocool/*` — built by the team)
  - **115 upstream** (quay.io oauth2-proxy, gcr.io GCP images, ghcr.io kasten, docker.io postgres/redis/etc.)

The remediation isn't uniform:

  - **In-house images**: pinning needs to happen in the **release pipeline**, not retroactively on the cluster. `kubectl patch deploy` with `@sha256:` works once, but the next `helm upgrade --set image.tag=X` reverts it. The pin must live in the Helm chart, Kustomize overlay, or Argo CD app spec — whatever the team's source of truth is.
  - **Upstream images**: pinning is a paranoia call. `postgres:17` from docker.io is signed and immutable-by-convention; pinning to `@sha256:abc` adds operational toil (every upstream patch release requires digest update). Most orgs accept the upstream-tag contract.

The CHA analyzer **doesn't distinguish** between these two cases today, which is why the 143-finding number reads as "supply-chain noise" instead of actionable work.

## What this proposer does

Two intertwined improvements:

### 1. Analyzer enhancement (OSS, v1.11.2+)

Categorize each digest-pin finding by registry trust class:

```yaml
trustClasses:
  - name: in-house
    matches:
      - "docker4zerocool/*"        # operator-configurable
      - "<org>/.*"
    severity: warning              # original behaviour
    remediation: "Pin in the release pipeline that builds this image."

  - name: trusted-upstream
    matches:
      - "quay.io/*"
      - "gcr.io/*"
      - "ghcr.io/*"
      - "registry.k8s.io/*"
      - "docker.io/(postgres|redis|haproxy|mariadb|envoyproxy)/*"
    severity: info                 # downgrade; many orgs accept
    remediation: "Optional. Pin if your threat model includes registry tampering."

  - name: unknown
    severity: warning              # default for anything else
```

`spec.probe.imagePin.trustClasses` is operator-supplied. With sane defaults, the 115 upstream findings drop to severity=info (off the warning channel) while the 28 in-house findings stay actionable.

### 2. AI tier proposer (paid, 2d-γ)

For workloads in the `in-house` class, the proposer:

  1. Reads `kind=workload` from RAG to find the owning workload (Deployment / StatefulSet / Argo CD Application / Flux HelmRelease).
  2. Reads the release pipeline source — checks for `Chart.yaml` + `values.yaml` (Helm), `kustomization.yaml`, or Argo CD `Application.spec.source` — and identifies the file + line that sets the image tag.
  3. Queries the registry API for the current resolved digest of the running tag (`crane digest`-equivalent via OCI distribution API).
  4. Generates a PR/MR against the source repo replacing `:tag` with `@sha256:<digest>` (or in some shops, ADDING the digest alongside the tag).
  5. Renders in Slack with `✅ Approve · ❌ Deny · 📄 Details` — Details links to the proposed PR.
  6. On Approve: ship the PR. On Deny: record `kind=finding_outcome` with `action=denied; finding_class=digest-pin`; proposer doesn't re-propose for the same workload until the running digest changes.

The proposer **never mutates the cluster directly** — only the source repo. The cluster picks up the pin on the next normal deploy.

## Data model

Extends `kind=workload`:

```go
{
  Kind: "workload",
  Key:  "<namespace>/<deployment>",
  Features: {
    "image":          "docker4zerocool/cluster-health-autopilot:1.11.0",
    "image_digest":   "sha256:abc123...",       // resolved at observation time
    "release_source": {
      "type": "helm",                            // helm | kustomize | argocd | flux
      "repo": "github.com/Bionic-AI-Solutions/cluster-health-autopilot",
      "path": "charts/cluster-health-autopilot/values.yaml",
      "image_field": "image.tag",
    },
    "trust_class": "in-house",
  }
}
```

Discovery feeder populates `image_digest` by reading `pod.status.containerStatuses[].imageID` (kubelet writes the resolved digest there). Release-source detection is best-effort — many workloads will lack it, in which case the proposer falls back to "open a PR template" rather than auto-discovering the exact line.

## Operator surface

```yaml
spec:
  probe:
    imagePin:
      trustClasses:
        - name: in-house
          matches: ["docker4zerocool/*"]
          severity: warning
        - name: trusted-upstream
          matches: ["quay.io/*", "gcr.io/*", "ghcr.io/*", "registry.k8s.io/*"]
          severity: info
  ai:
    digestPinProposer:
      enabled: true
      requireReleaseSource: true       # if no source detected, skip (don't propose)
      gitForges:
        - type: github
          token:
            secretName: cha-github-token
            key: token
```

## Failure modes & tests

  - **Registry API outage** — proposer skips that finding for the cycle, retries next cycle. No findings escalated to red.
  - **Release source not detected** — finding stays at original severity; proposer emits an info Slack message saying "I'd propose a pin here but I can't find the release source. Configure `spec.probe.imagePin.releaseHints` for this workload."
  - **Digest drift while PR is open** — proposer notices on next cycle that the running digest differs from the proposed one; comments on the PR with the new digest.
  - **PR merged but cluster doesn't pick up** — proposer monitors; if 7d post-merge the cluster still runs the old tag, fires a `release-drift` finding so the SRE can rerun the deploy.

## Phasing

| Phase | Scope | Gate |
|---|---|---|
| 2d-γ-1 | Analyzer enhancement (OSS): `trustClasses` config + severity downgrade. Ships separately as v1.11.2; drops the 115 upstream findings from the warning channel. | OLM-on-kind smoke + live verification on the dev cluster. |
| 2d-γ-2 | RAG `kind=workload.features.image_digest` populated by discovery feeder. No proposing yet — just observing. | 7-day observation across the 28 in-house workloads. |
| 2d-γ-3 | Release-source detection (Helm Chart.yaml + Argo CD Application probing). | 80%+ accuracy at finding the right `image.tag` field. |
| 2d-γ-4 | PR-generation proposer + Slack Approve/Deny render. | Adversarial review of proposed PRs against a corpus of past in-house releases. |
| 2d-γ-5 | Drift detection (PR merged + 7d, cluster still on old digest → fire). | One real cycle of drift detection on the dev cluster. |

## Out of scope

  - Multi-arch digest pinning (manifest-list vs single digest) — defer to 2d-γ-future
  - Auto-merge of generated PRs — never. Always requires human Approve.
  - Cross-cluster cohort: each cluster's `image_digest` observations stay local (parent doc rule).

## Adjacent decisions

  - `analyzer enhancement` ships in OSS because it's pure analysis, not learning. Trust classes are operator config, not learned.
  - The proposer (PR generation) is paid because it needs git-forge tokens + workflow management — both operationally heavy and the moat.
  - The 115 upstream findings can be silenced TODAY via Silence CRs (v1.10.4 Approve/Deny renders the snippet). 2d-γ-1 just makes that the default, not an opt-in click.
