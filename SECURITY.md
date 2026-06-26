# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| `v0.1.x` (preview) | ✅ |
| `< v0.1.0` | ❌ (pre-release) |

Once `v1.0.0` ships, the latest minor and the previous minor will receive security fixes.

## Reporting a vulnerability

**Please do not open a public GitHub issue.**

Email **security@srenix.ai** with:

- A description of the issue and its impact.
- Reproduction steps or a proof-of-concept (if applicable).
- Affected versions.
- Your contact info (we'll credit you in the advisory unless you prefer otherwise).

You should receive an acknowledgement within **2 business days**. We aim to publish a fix and a public advisory within **30 days** of the initial report; longer-lived issues will be communicated to you.

## Scope

In scope:

- The `srenix` CLI binary and its source code.
- The Helm chart (`charts/agentic-sre/`).
- The container image published to `ghcr.io/<org>/srenix`.
- Documented attack surfaces in the README.

Out of scope:

- Third-party dependencies (please report upstream).
- Issues that require pre-existing cluster-admin access (the threat model assumes the operator already trusts the install).

## Security posture

- **No telemetry** is collected by the binary. The only network traffic is the Slack/webhook output you configure.
- **No SaaS dependency** for core operation.
- **RBAC is split into two narrow ClusterRoles** (read-only + bounded-write); the write role grants only `pods/delete`, `jobs/delete`, and `deployments/patch`. No Secret/ConfigMap/CRD writes.
- **Protected-namespace list** in-script: `kube-system`, `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, `cnpg-system`. Auto-fixers never touch these regardless of state.
- **Builds are reproducible**: GitHub Actions with SLSA-style provenance attached to each release artifact.
