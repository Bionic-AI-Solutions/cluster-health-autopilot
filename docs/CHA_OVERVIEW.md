# Cluster Health Autopilot — Two-Page Overview

*A read-it-once explainer of what CHA is, where the line between OSS and Paid
sits, what it does and refuses to do, and what AI does and doesn't do — and
why.*

---

## What is Cluster Health Autopilot?

Cluster Health Autopilot (`cha`) is a **self-healing operational layer** for
Kubernetes clusters. It runs as a single Helm chart with a small distroless
binary, and on every cycle it:

1. **Detects** — runs a fixed catalog of probes and analyzers against the
   cluster's API state (no metrics scraping, no log shipping). Read-only.
2. **Suppresses flakes** (v1.4) — a single failed probe doesn't page; only a
   second consecutive failure does. Transient blips get an in-cycle retry.
3. **Investigates** (v1.5+) — on every CRITICAL finding, a read-only
   Investigator runs a deep-dive (DNS / HTTP probe / TLS inspect / kubectl
   describe / recent events) and attaches a one-line root-cause Summary.
   In CHA-com with `ai.enabled=true`, the deep-RCA investigator additionally
   synthesizes a **Firecrawl-grounded web query** (all cluster identifiers
   redacted by the LLM before any external call) and persists the root-cause
   analysis to the `cha_investigations` Qdrant collection; that analysis is
   then forwarded into every AI tier (T0–T3) via a `<root_cause>` prompt block.
4. **Remediates** — if enabled, runs a whitelist of safe, idempotent,
   reversible fixers (delete a Failed pod, kill a frozen Job, rollout-restart
   a wedged ReplicaSet, delete a terminal CertificateRequest, optionally
   repoint an Ingress to the correct TLS Secret).
5. **Re-verifies** — re-runs the analyzers; the diagnostic for the fixed
   subject must clear or the action is recorded as failed. When a
   Jira/ServiceNow ticket is resolved (finding cleared), ticket closure is
   recorded to the RAG/audit as an Outcome with
   `verdict=cleared, delivery=ticket-closed` (best-effort).
6. **Reports** — posts a Slack message, fires Alertmanager alerts, and
   upserts a `DriftReport` custom resource that any other tooling (ArgoCD
   events, OPA admission, your own scripts) can read with kubectl.

It runs in two modes:

- **Zero-trust offline** — `cha diagnose --snapshot ./your-export/` against a
  captured `kubectl get … -o json`. No install, no RBAC, no SaaS, 30 seconds.
  This is the "evaluate without trust" entry point.
- **In-cluster live** — Helm chart installs a watcher Deployment +
  diagnose/remediate CronJobs. Watches Kubernetes events with a ~10 s
  debounce; reacts within seconds of state changes.

The container image is 13 MB distroless, runs nonroot, requires <100 m CPU
and <100 MB RAM, and has **no inbound traffic** — only outbound to the
Kubernetes API, optional Vault, optional Alertmanager, optional Slack.

---

## OSS vs Paid — what's the difference?

CHA is **open-core**. The engine and the foundational catalog are Apache 2.0
in the public repo. The paid tier ships as a separate signed binary
(`cha-com`) that imports the OSS module, registers additional patterns
through the same `pkg/registry` interfaces, and adds the LLM/approval
infrastructure.

| Capability | Free OSS (single cluster) | Paid (per cluster) | Enterprise |
|---|---|---|---|
| Zero-trust snapshot + live watcher | ✓ | ✓ | ✓ |
| 21 probes · 20 analyzers · 4 default fixers + 1 opt-in (TLSSecretMismatch) | ✓ | ✓ | ✓ |
| `VaultPathMissing` analyzer source (Apache-2.0; you supply the Vault client) | ✓ | ✓ | ✓ |
| Ingress host auto-discovery (v1.2) | ✓ | ✓ | ✓ |
| Layer-1 flake suppression — retry + 2-of-2 streak (v1.4) | ✓ | ✓ | ✓ |
| **Layer-2 Investigator — rule-based, deterministic, read-only (v1.5)** | ✓ | ✓ | ✓ |
| **Fixer safety nets — GitOps + paused + suspended + cert-mgr health (v1.6)** | ✓ | ✓ | ✓ |
| **6 new probes: node pressure, system DaemonSets, pending pods, generic CrashLoop, ETCD, failed mounts (v1.6)** | ✓ | ✓ | ✓ |
| **Configurable critical-workload list via env + annotation (v1.6)** | ✓ | ✓ | ✓ |
| **Lease-based leader election for HA watcher (v1.6)** | ✓ | ✓ | ✓ |
| DriftReport CRD · Slack · Alertmanager | ✓ | ✓ | ✓ |
| Helm chart with narrow RBAC + protected-namespace list | ✓ | ✓ | ✓ |
| Pattern registry — patterns 3 months EARLIER than OSS | — | ✓ | ✓ |
| Dry-run pattern simulation against your captured snapshot | — | ✓ | ✓ |
| 4-hour SLA on false-positive triage + state-change response | — | ✓ | ✓ |
| `VaultPathMissing` auto-wired from your Vault configuration (no glue code) | — | ✓ | ✓ |
| **Layer-2 Investigator — LLM-backed swap of the OSS interface** | — | ✓ | ✓ |
| AI T0 — diagnostic narrative (read-only LLM enrichment) | — | ✓ | ✓ |
| AI T1 — fix proposals with one-click human approval | — | ✓ | ✓ |
| AI T2 — multi-step planner (≤5 steps, prerequisite-linked) | — | ✓ | ✓ |
| AI T3 — Vault runbook generator (dual-approval, never auto-run) | — | ✓ | ✓ |
| Approval-server (Ed25519, OIDC, JTI replay-blocked) + circuit breaker + audit + rate limit | — | ✓ | ✓ |
| Custom pattern co-development from your incident history | — | — | ✓ |
| SLA indemnification · dedicated version lock · air-gap support | — | — | ✓ |

**Pricing**: $0 for OSS, **$99 per cluster per month** for Paid (expensable on
a corporate card; no PO needed for fleets up to ~15 clusters), custom for
Enterprise.

**Architectural invariant**: the paid binary cannot exceed the OSS RBAC
ceiling. Every mutation — AI-proposed or not — flows through the same
`snapshot.Mutator` interface defined in the OSS module, and that interface
sits inside the read+write ClusterRoles installed by the Helm chart. There
is no escape hatch. The paid tier expands *coverage* (more patterns, more AI
help, LLM-backed investigation) — never *autonomy*.

---

## What CHA does — and what it deliberately doesn't do

### CHA does

- Detect configuration drift across the **Vault → ExternalSecret →
  Deployment** chain — the failure class that dashboards miss because it
  doesn't move a metric.
- Detect TLS / cert-manager state mismatches (including the
  `TLSSecretMismatch` pattern: Ingress wired to wrong Secret while
  cert-manager renews the right one elsewhere).
- Detect stuck workloads (CCE pods, frozen CronJob Jobs, wedged ReplicaSet
  rollouts, terminal CertificateRequests) with kubectl-style precision.
- Probe every public Ingress host externally and surface real-world
  reachability problems (TLS, Kong route, DNS).
- Auto-fix a **whitelist** of patterns: stale Failed pods, frozen Jobs,
  wedged ReplicaSet rollouts, terminal CertificateRequests, and (opt-in)
  the Ingress→TLS Secret repoint. Every fixer is idempotent and reversible.
- Surface every action and finding as a `DriftReport` CR — `kubectl get
  driftreports -A` is the audit trail.
- Run completely offline against a captured snapshot — useful for air-gapped
  clusters, IR / forensics, pre-prod audits.
- Pass through Alertmanager (preferred) for fanout to whatever the team
  already uses — PagerDuty, Opsgenie, Slack, email.

### CHA does NOT

- **Write Secrets, ConfigMaps, or generic CRDs.** Not in OSS, not in Paid.
  RBAC excludes those verbs entirely. Resolution for any secret-class issue
  is surfaced as a diagnostic with the exact kubectl/vault command — the
  human runs it.
- **Touch protected namespaces.** `kube-system`, `kube-public`,
  `kube-node-lease`, `vault`, `external-secrets`, `rook-ceph`, `cnpg-system`
  are hardcoded skip-lists; enforced in code AND in RBAC.
- **Patch GitOps-managed Ingresses.** If an Ingress carries an ArgoCD,
  Flux, or Helm management annotation/label, the (opt-in) TLSSecretMismatch
  fixer skips it — the right fix lives in the source repo, and CHA refuses
  to fight the reconcile loop.
- **Edit your manifests in git.** The catalog produces precise remediation
  hints ("patch this Ingress field to this value"); the operator applies
  them in the source repo. CHA does not have a git-write surface.
- **Run any AI / LLM call in OSS.** The Layer-2 Investigator in OSS is
  rule-based Go — every output is auditable, deterministic, and the same
  input produces the same Summary every time.
- **Send cluster state to any third-party SaaS by default.** Outbound traffic
  goes to the destinations the operator configures (Slack webhook, optional
  Alertmanager, optional Vault). No vendor analytics. No telemetry.
  **One deliberate exception (CHA-com paid, opt-in):** when
  `--firecrawl-enabled` is set and `FIRECRAWL_API_KEY` is present, the
  deep-RCA investigator sends a **client-redacted** web search query to
  Firecrawl. All namespace, pod, host, IP, and secret identifiers are
  stripped by the LLM and by `RedactDiagnostic`/`RedactEventMessage`
  before the query is constructed. Inert when the key is absent.
- **Page on a single transient failure.** First failure is tagged
  `[transient, 1/2]` and emits at warning; only a second consecutive failure
  promotes to CRITICAL. Deterministic failures (TLS error, status mismatch)
  bypass this and alert immediately.
- **Make a mutation an LLM proposed without a human signed-approval click.**
  Every paid AI mutation passes through Ed25519-signed JWT + OIDC + JTI
  single-use + admission re-check + post-apply verify. There is no
  auto-apply path from an LLM response.

---

## What AI does — and what it deliberately doesn't do

### Two AI surfaces

CHA has two distinct AI surfaces, in increasing order of trust required:

1. **Layer-2 Investigator** — runs on every critical finding, attaches a
   root-cause Summary to the alert.
2. **AI Tiers T0–T3** — paid only; enrichment, fix proposals, multi-step
   plans, Vault recovery runbooks.

Both share these invariants:

- **AI proposes — humans approve — deterministic Go code executes.** The
  `snapshot.Mutator` interface is never called directly from any LLM output.
- **AI cannot exceed the OSS RBAC ceiling.** The paid binary inherits the
  same ClusterRoles installed by the OSS Helm chart.
- **Tool surface is a closed enum.** The Layer-2 Investigator's `Environment`
  interface exposes exactly five read-only methods (`DNSLookup`, `HTTPProbe`,
  `TLSInspect`, `Describe`, `GetEvents`) — even an LLM cannot invent commands.
- **Soft-fail to deterministic.** Every AI failure mode reverts to the
  OSS-only behavior. AI being slow, broken, or absent is never a blocker for
  diagnose / fix / report.

### What AI does

- **Layer-2 Investigator (OSS, rule-based)** — pattern-matches the failure
  mode and runs the matching tools. Classifies TLS expiry, TLS SAN mismatch,
  DNS failure, slow-DNS root cause, transient-recovery (when a follow-up
  probe succeeds), HTTP status mismatches, and ExternalSecret / Certificate
  diagnostic states. Attaches a 🔬 Summary block. No LLM call.
- **Layer-2 Investigator (Paid, LLM-backed deep-RCA)** — same `Investigator`
  interface, same closed-enum `Environment` for cluster tools. Additionally
  synthesizes a client-redacted Firecrawl web query, scrapes relevant docs/
  runbooks, and persists the root-cause analysis to the `cha_investigations`
  Qdrant collection. The RCA is forwarded to every AI tier (T0–T3) via a
  `<root_cause>` prompt block, grounding proposals in the actual failure
  evidence gathered for that cycle.
- **T0 Enricher** — adds a 2–4 sentence root-cause narrative (🤖 block) to
  Diagnostics. Read-only.
- **T1 Fix Proposer** — proposes one mutation matching an existing OSS fixer
  (`DeletePod`, `DeleteJob`, `PatchDeployment`, `DeleteCertRequest`,
  `DeleteACMEOrder`). Slack message gets an "Apply Fix" button.
- **T2 Multi-Step Planner** — emits up to 5 prerequisite-linked steps for
  compound failures. Step N+1's button appears only after step N's
  post-apply verification passes.
- **T3 Vault Runbook Proposer** — generates a `vault kv patch` command
  template with `${VALUE_*}` placeholders (never values) for human
  execution under dual approval. CHA-com never executes Vault writes.

### What AI does NOT do

- **AI does NOT mutate cluster state directly.** Every mutation is one of
  five whitelisted action kinds, parsed against a strict schema BEFORE
  reaching the executor. Anything outside that enum is dropped.
- **AI does NOT see Secret values.** The Investigator's `Describe` tool
  reads via the watcher's snapshot.Source, which respects the read-only
  RBAC ceiling (Secret keys only, never values). Prompts are pre-redacted
  for base64≥40 and hex≥32 patterns via `pkg/ai/redact.go`.
- **AI does NOT touch protected namespaces.** Enforced at the planner,
  proposer, validator, and approval-server layers — four independent gates.
- **AI does NOT bypass approval.** The Ed25519 signing key lives only in
  the approval-server pod; JWTs are single-use (JTI replay-blocked); the
  click flow is OIDC-authenticated; admission policy is re-checked at apply
  time.
- **AI does NOT run unbounded.** Hard wall-clock cap on every LLM call,
  rate-limit on mutations per hour, circuit breaker (3 consecutive
  post-apply verification failures → AI auto-disabled).
- **AI does NOT replace the deterministic spine.** OSS analyzers and fixers
  run identically with or without any AI tier. Turning off AI returns the
  cluster to OSS behavior bit-for-bit.

### Why AI is structured this way

- **Determinism is a feature.** Operations teams need same-input-same-output
  for postmortems and compliance review. The deterministic spine is the load-bearing element; AI is augmentation, not substitution.
- **Audit-grade by default.** Every AI proposal, approval, application, and
  verification emits an audit event (`ai.proposal.created`,
  `ai.approval.granted`, `ai.action.applied`, etc.) shipped to Loki / OTLP /
  stdout. The Layer-2 Investigator's OSS rule-based path is auditable by
  source-code inspection.
- **GitOps-aware.** A mutation that would be reverted by a reconcile loop
  isn't a fix — it's a fight. The TLSSecretMismatch fixer's GitOps detection
  is a small example of this principle applied to AI; the paid tier's fix
  proposals inherit the same checks.
- **Mistakes recover in seconds.** Every whitelisted fixer is reversible
  (`kubectl delete pod` reschedules in seconds; `kubectl rollout restart` is
  standard SRE vocabulary). The AI mutation surface inherits this property
  — anything the model could propose, a human could undo in <60 s.
- **No SaaS lock-in.** OSS runs no LLM. Paid lets the operator bring their
  own model endpoint. Customer state stays inside the cluster.

---

## In one paragraph

CHA detects the configuration-drift failure class that metric dashboards
miss, fixes the safe ones automatically (within a whitelist of reversible
patterns), investigates every critical alert with a read-only root-cause
classifier before paging anyone, and never touches Secrets, ConfigMaps,
CRDs, protected namespaces, or GitOps-managed resources. The OSS version
is fully self-contained — deterministic Go, rule-based investigator, no
LLM, Apache 2.0. The paid version adds a pattern registry with a 4-hour
SLA and an LLM-backed AI tier that proposes fixes for human approval,
inheriting every safety boundary the OSS engine enforces. AI is opt-in,
gated, audited, and architecturally incapable of exceeding the OSS RBAC
ceiling.

---

*For deeper dives: [docs/ONE_PAGER.md](ONE_PAGER.md) ·
[docs/AI_TIERS.md](AI_TIERS.md) ·
[docs/FAILURE_MODES.md](FAILURE_MODES.md) ·
[docs/AI_USAGE.md](AI_USAGE.md) ·
[docs/THREAT_MODEL_AI.md](THREAT_MODEL_AI.md).*
