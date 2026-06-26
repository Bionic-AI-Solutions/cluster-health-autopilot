# Phase 2.F — HA aiwatch (Leader-Election)

**Status:** active — starts 2026-06-08 in local-only mode.

**Parent:** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

**Branch:** `phase2b/approve-remember-class` (stacking; will split when pushed).

---

## Goal

The watcher currently runs as a single replica. If that pod dies, no cycles run until kubelet restarts it (10s–60s gap). Phase 2.F adds leader-election so 2+ replicas can run simultaneously; exactly one is the active "leader" (runs tick), the others stand by ready to take over instantly.

## Anti-goals

- Followers do NOT run analyzers in shadow mode. Idle until promoted. (No per-cycle LLM cost duplication; no per-cycle Slack/PR dedup work.)
- No external coordination service. Lease lives in the cluster the watcher already has access to (ConfigMap or coordination.k8s.io/v1.Lease).
- No split-brain mitigation beyond the standard client-go lease semantics — clusters with broken kube-api are already in worse trouble than Srenix can fix.

## Sub-tasks

### 2.F.1 — `LeaderElector` interface + production adapter
- [ ] Define `LeaderElector` interface (Start, OnStart cb, OnStop cb)
- [ ] Adapter wraps `k8s.io/client-go/tools/leaderelection`
- [ ] When replicas=1, use a `noopLeaderElector` that immediately calls OnStart — keeps single-replica deploys byte-identical

### 2.F.2 — Gate watchLoop.run() behind leader status
- [ ] watchLoop holds `LeaderElector`; run() blocks waiting for OnStart
- [ ] OnStop closes a `stopCh`; run() drains the in-flight tick and exits
- [ ] Failing test: two watchLoop instances + fake elector; assert only one runs tick at a time

### 2.F.3 — `--replicas` flag + Helm/CR support
- [ ] CR `spec.ai.replicas: int` (default 1)
- [ ] Helm template uses it for `bionic-aiwatch` Deployment replicas
- [ ] When > 1, the deployment also gets the `leader-election-role` ServiceAccount + RBAC

### 2.F.4 — Lease lock implementation
- [ ] Use `coordination.k8s.io/v1.Lease` (modern; replaces ConfigMap lock)
- [ ] Lock name: `srenix-aiwatch-leader` in install namespace
- [ ] Lease duration 15s, renew deadline 10s, retry 2s (matches client-go defaults)

### 2.F.5 — Audit lease transitions
- [ ] OnStart → audit "ai.aiwatch.became_leader" + log line
- [ ] OnStop → audit "ai.aiwatch.lost_leadership" + log line + clean shutdown

### 2.F.6 — Smoke test: two replicas
- [ ] OLM bundle smoke step starts 2 replicas
- [ ] Assert exactly 1 logs "became leader"; the other stays in candidate state
- [ ] Kill the leader pod; assert the other logs "became leader" within 30s

### 2.F.7 — Local build + cluster verify
- [ ] Bump `spec.ai.replicas: 2` on the bionic CR
- [ ] Verify lease appears in kube-api
- [ ] Verify only the leader logs "outcomes:" cycle lines

## Acceptance

- 2 replicas of bionic-aiwatch run simultaneously
- Exactly one runs tick at any moment
- Failover within 30s on leader loss
- Single-replica deploy is byte-identical (noop elector path)
