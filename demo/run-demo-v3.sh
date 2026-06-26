#!/usr/bin/env bash
# Srenix Demo — v3 (AI SRE)
#
# Builds on the OSS engine demo (run-demo-v2.sh) to show Srenix Enterprise's
# AI SRE layer end-to-end on a live cluster:
#
#   OSS engine (already installed) → drift detection → T0 narration →
#   T1 fix proposal (signed-JWT click-to-fix) → T3 vault break-glass
#   (dual-approval timeline) → audit trail.
#
# Designed for sales/stakeholder calls. Each section pauses for
# narration; press ENTER to advance.
#
# Prerequisites:
#   - Srenix OSS engine deployed (Helm release "srenix" in namespace
#     "agentic-sre"). v2 demo can bring you here from
#     scratch; v3 assumes you're already there.
#   - Srenix Enterprise binary buildable from $REPO_DIR_COM/cmd/srenix-enterprise (or pre-built
#     and on $PATH as `srenix-enterprise`).
#   - AI_API_KEY env var available (or AI_API_KEY_SECRET=<ns>/<name>:<key>
#     pointing at a K8s secret the user can read).
#   - LLM endpoint reachable: AI_ENDPOINT (default
#     https://mcp.baisoln.com/gpu-ai/v1) + AI_MODEL (default
#     qwen3.6-35b-a3b-fp8).
#
# Usage:
#   export KUBE_CONTEXT="<your context>"
#   bash demo/run-demo-v3.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR_OSS="$(dirname "${DEMO_DIR}")"
REPO_DIR_COM="${REPO_DIR_COM:-$(cd "${REPO_DIR_OSS}/.." && pwd)/Srenix Enterprise}"
SRENIX_NS="${SRENIX_NS:-agentic-sre}"
TEST_NS="${TEST_NS:-srenix-demo-v3}"

AI_ENDPOINT="${AI_ENDPOINT:-https://mcp.baisoln.com/gpu-ai/v1}"
AI_MODEL="${AI_MODEL:-qwen3.6-35b-a3b-fp8}"
AI_HEADER="${AI_HEADER:-X-API-Key}"
AI_KEY_SECRET="${AI_KEY_SECRET:-mcp/mcp-admin-apikey:key}"

# ─── colours + helpers ────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; NC='\033[0m'

header()  { echo ""; echo -e "${BLUE}${BOLD}══════════════════════════════════════════════════════════════════${NC}"; echo -e "${BLUE}${BOLD}  $1${NC}"; echo -e "${BLUE}${BOLD}══════════════════════════════════════════════════════════════════${NC}"; }
section() { echo ""; echo -e "${CYAN}${BOLD}── $1 ──${NC}"; }
info()    { echo -e "  ${GREEN}▶${NC} $*"; }
warn()    { echo -e "  ${YELLOW}⚠${NC}  $*"; }
fail()    { echo -e "  ${RED}✗${NC}  $*"; }
pause()   { echo ""; echo -e "${YELLOW}[PAUSE]${NC} $1"; echo -e "${YELLOW}Press ENTER to continue...${NC}"; read -r; }

# ─── cleanup trap ─────────────────────────────────────────────────────────────
cleanup() {
  echo ""
  echo -e "${BOLD}Demo v3 cleanup...${NC}"
  $KUBECTL delete namespace "${TEST_NS}" --ignore-not-found 2>/dev/null || true
  # Wait briefly for the watcher to reconcile the DriftReport away so
  # the cluster ends in the same state as the demo started.
  echo "  Letting the watcher reconcile the synthetic DriftReports away..."
  sleep 10
  echo -e "${GREEN}Cleanup complete.${NC}"
}
trap cleanup EXIT

# ─── resolve API key into env (won't echo to terminal) ────────────────────────
resolve_api_key() {
  if [[ -n "${AI_API_KEY:-}" ]]; then
    info "AI_API_KEY already set in env (length: ${#AI_API_KEY} chars)"
    return 0
  fi
  local ns="${AI_KEY_SECRET%/*}"
  local rest="${AI_KEY_SECRET#*/}"
  local secret="${rest%%:*}"
  local key="${rest#*:}"
  info "Fetching API key from secret ${ns}/${secret}.${key} ..."
  AI_API_KEY=$($KUBECTL -n "${ns}" get secret "${secret}" -o jsonpath="{.data.${key}}" 2>/dev/null | base64 -d 2>/dev/null) || true
  if [[ -z "${AI_API_KEY:-}" ]]; then
    fail "Could not resolve API key. Set AI_API_KEY directly or AI_KEY_SECRET=<ns>/<name>:<key>."
    exit 1
  fi
  export AI_API_KEY
  info "API key resolved (length: ${#AI_API_KEY} chars) — never echoed."
}

# ─── helper: run srenix-enterprise locally pointed at the cluster ───────────────────────
srenix_com() {
  if command -v srenix-enterprise >/dev/null 2>&1; then
    srenix-enterprise "$@"
  else
    (cd "${REPO_DIR_COM}" && go run ./cmd/srenix-enterprise/ "$@")
  fi
}

# ═══════════════════════════════════════════════════════════════════════════════
header "Srenix Demo v3 — Pre-flight"
# ═══════════════════════════════════════════════════════════════════════════════

section "1/4 — Cluster"
echo -e "  Context: ${BOLD}${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}${NC}"
$KUBECTL cluster-info 2>/dev/null | head -2
echo ""
$KUBECTL get nodes --no-headers 2>/dev/null | awk '{printf "  %-40s %s\n", $1, $2}'

section "2/4 — Srenix OSS engine (must already be deployed)"
if ! $KUBECTL -n "${SRENIX_NS}" get deploy -l "app.kubernetes.io/name=agentic-sre" --no-headers 2>/dev/null | grep -q .; then
  fail "Srenix OSS engine not found in namespace ${SRENIX_NS}. Run demo/run-demo-v2.sh first to set up, or 'helm install'."
  exit 1
fi
$KUBECTL -n "${SRENIX_NS}" get deploy -l "app.kubernetes.io/name=agentic-sre" --no-headers 2>/dev/null | \
  awk '{printf "  %-50s ready=%s\n", $1, $2}'
$KUBECTL -n "${SRENIX_NS}" get lease srenix-watcher -o jsonpath='{"  Watcher lease holder: "}{.spec.holderIdentity}{"\n  Lease renewed:        "}{.spec.renewTime}{"\n"}' 2>/dev/null || true

section "3/4 — AI endpoint"
echo -e "  Endpoint: ${BOLD}${AI_ENDPOINT}${NC}"
echo -e "  Model:    ${BOLD}${AI_MODEL}${NC}"
echo -e "  Header:   ${BOLD}${AI_HEADER}${NC}"
resolve_api_key

section "4/4 — Srenix Enterprise binary"
if command -v srenix-enterprise >/dev/null 2>&1; then
  info "Using installed srenix-enterprise: $(command -v srenix-enterprise)"
  srenix-enterprise version 2>/dev/null || true
else
  info "Will compile srenix-enterprise from ${REPO_DIR_COM} on first use (go run)"
  [[ -d "${REPO_DIR_COM}/cmd/srenix-enterprise" ]] || { fail "REPO_DIR_COM not set or wrong: ${REPO_DIR_COM}"; exit 1; }
fi

pause "Pre-flight done. We'll show the AI SRE flow end-to-end. Ready?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 1 — The OSS engine is the trustable runtime"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  Before any LLM gets near the cluster, the OSS engine is already
  doing its job:

    • 12 K8s probes (Ceph, Postgres, Nodes, PVCs, Critical services,
      Endpoints, NodePressure, DaemonSets, PendingPods, CrashLoopBackOff,
      ETCD, FailedMounts)
    • 8 analyzers covering the secret/cert/ESO/image-pull/TLS boundary
    • 5 policy-bounded fixers (StaleErrorPods, StuckJobs, StuckRSPods,
      StuckCertificateRequests, opt-in TLSSecretMismatch)
    • lease-based leader election (active right now — see above)

  That's the trustable runtime. Srenix Enterprise sits on top and adds an
  LLM Investigator agent + four AI SRE tiers (T0 → T3). Same policy
  bounds, signed actions, hash-chained audit.

EOF

section "Current DriftReports"
$KUBECTL get driftreports -A --no-headers 2>/dev/null | \
  awk '{printf "  %-40s %-10s %-60s\n", $2, $3, $4}' || echo "  (cluster is clean)"

pause "On to the AI tiers."

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 2 — Inject a synthetic broken pod"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  We'll create an ImagePullBackOff pod in a fresh namespace. The OSS
  watcher will detect it within ~60s (event-triggered cycle) and emit
  a DriftReport CR. That diagnostic becomes the input to the AI tiers.

EOF

$KUBECTL create namespace "${TEST_NS}" --dry-run=client -o yaml | $KUBECTL apply -f -
info "Creating srenix-test-imagepull pod (will go ImagePullBackOff)"
cat <<YAML | $KUBECTL -n "${TEST_NS}" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: srenix-test-imagepull
  labels: { demo: srenix-v3 }
spec:
  restartPolicy: Never
  containers:
  - name: c
    image: docker.io/this-image-does-not-exist:nope
YAML

section "Waiting for the OSS watcher to emit a DriftReport (max 90s)..."
DRIFT_REPORT=""
for _ in $(seq 1 18); do
  DRIFT_REPORT=$($KUBECTL get driftreports -A --no-headers 2>/dev/null | grep "srenix-test-imagepull" | head -1 | awk '{print $2}')
  [[ -n "${DRIFT_REPORT}" ]] && break
  sleep 5
done
if [[ -z "${DRIFT_REPORT}" ]]; then
  warn "DriftReport didn't appear within 90s. Continuing anyway."
else
  info "DriftReport CR: ${DRIFT_REPORT}"
  $KUBECTL get driftreport "${DRIFT_REPORT}" -n "${SRENIX_NS}" -o jsonpath='{"  Severity:    "}{.spec.severity}{"\n  Source:      "}{.spec.source}{"\n  Subject:     "}{.spec.subject}{"\n  Message:     "}{.spec.message}{"\n"}' 2>/dev/null
fi

pause "OSS engine has seen the drift. Now Srenix Enterprise adds the AI."

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 3 — T0 narration (Qwen 3.6 35B)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  T0 = LLM-driven narration on every diagnostic. The OSS analyzer
  output is the structured fact ("auth failure pulling image X");
  T0 turns it into an actionable explanation for the operator's
  Slack channel or ticket.

  No mutation. Read-only. Useful immediately as a "what does this
  even mean" sidekick.

EOF

section "Running: srenix-enterprise diagnose --ai-tier=t0"
srenix_com diagnose \
  --ai-tier=t0 \
  --ai-endpoint="${AI_ENDPOINT}" \
  --ai-model="${AI_MODEL}" \
  --ai-api-key-header="${AI_HEADER}" 2>&1 | \
  grep -A2 -B1 -E "srenix-test-imagepull|🤖|🟢|⚠️|🔴" | head -40

pause "T0 narration above. Notice the 🤖 lines — that's the LLM."

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 4 — T1 fix proposer (the click-to-fix flow)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  T1 proposes a specific action_kind (DeletePod, PatchDeploymentAnnotation,
  etc.) inside the operator policy. Each proposal carries:

    • a JWT-signable ActionID
    • a rollback description (validator rejects proposals without one)
    • a target (Pod/Deployment/Job) with PII redacted before LLM input

  For a click-to-fix flow, the approval-server signs the ActionID with
  Ed25519 and the operator gets a Slack URL. One click executes; nothing
  mutates without the click.

  This part of the demo runs the proposer against a synthetic
  "stale Failed pod" diagnostic — the kind T1's matcher recognises.

EOF

# Synthetic Failed pod (the OSS ImagePullBackOff diag doesn't match
# T1's keyword matcher; this one does).
info "Creating a deliberately-failing pod to trigger the T1 matcher"
cat <<YAML | $KUBECTL -n "${TEST_NS}" apply -f - >/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: srenix-test-failed
  labels: { demo: srenix-v3 }
spec:
  restartPolicy: Never
  containers:
  - { name: c, image: busybox, command: ["sh","-c","echo fail && exit 1"] }
YAML

# Wait for Failed phase.
for _ in $(seq 1 10); do
  phase=$($KUBECTL -n "${TEST_NS}" get pod srenix-test-failed -o jsonpath='{.status.phase}' 2>/dev/null || true)
  [[ "${phase}" == "Failed" ]] && break
  sleep 2
done

section "Running T1 propose against a synthetic StaleErrorPods diagnostic"
cat <<'GO' > /tmp/srenix-demo-v3-t1.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	chacomai "github.com/srenix-ai/agentic-sre-enterprise/ai"
	"github.com/srenix-ai/agentic-sre-enterprise/ai/client"
	"github.com/srenix-ai/agentic-sre/pkg/ai"
	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
)

func main() {
	llm, err := client.NewOpenAIClient(client.OpenAIConfig{
		Endpoint:     os.Getenv("AI_ENDPOINT"),
		APIKey:       os.Getenv("AI_API_KEY"),
		APIKeyHeader: os.Getenv("AI_HEADER"),
		DefaultModel: os.Getenv("AI_MODEL"),
	})
	if err != nil { fmt.Println("ERR:", err); os.Exit(1) }
	d := diagnose.Diagnostic{
		Source: "StaleErrorPods", Subject: "Pod/" + os.Getenv("TEST_NS") + "/srenix-test-failed",
		Severity: "warning",
		Message:  "Pod " + os.Getenv("TEST_NS") + "/srenix-test-failed is in Failed phase (stale error pod, exited 5m ago).",
		Remediation: "Delete the pod; the controller will recreate if needed.",
	}
	fp, _ := chacomai.NewFixProposer(chacomai.FixProposerConfig{Client: llm, Audit: ai.NoOpAuditSink{}})
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); defer cancel()
	prop, err := fp.Propose(ctx, d)
	if err != nil { fmt.Println("ERR:", err); os.Exit(1) }
	out, _ := json.MarshalIndent(prop, "", "  ")
	fmt.Println(string(out))
}
GO

# We need to run this in a directory that can resolve both modules.
# Srenix Enterprise module already pulls agentic-sre via go.mod.
mkdir -p "${REPO_DIR_COM}/cmd/.demo-v3-t1"
cp /tmp/srenix-demo-v3-t1.go "${REPO_DIR_COM}/cmd/.demo-v3-t1/main.go"

export AI_HEADER AI_ENDPOINT AI_MODEL TEST_NS
(cd "${REPO_DIR_COM}" && go run ./cmd/.demo-v3-t1/) 2>&1

rm -rf "${REPO_DIR_COM}/cmd/.demo-v3-t1" /tmp/srenix-demo-v3-t1.go

cat <<EOF

  Notice:
    • action_kind is from the operator policy (DeletePod, not anything
      the LLM dreams up)
    • a rollback description is present — validator would reject
      proposals without one
    • the ActionID is a UUID — the approval-server signs THIS string
      into a JWT, the JWT is the click-to-fix URL
    • target/namespace names are HASHED in the LLM input (look at
      Target.Namespace) — pkg/ai/redact runs before any LLM call

  The click-to-fix URL would land in your Slack channel. Without the
  click, nothing mutates.

EOF
pause "On to T3 — the dual-approval vault break-glass."

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 5 — T3 vault break-glass runbook (dual-approval)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  T3 is for "we lost a vault key" scenarios. Srenix Enterprise never executes
  Vault writes itself — the agent proposes a runbook the operator
  runs manually after dual approval (two distinct approvers separated
  by ≥30 minutes).

  Constraints baked into the proposer:
    • Vault path must be inside the operator-defined allowlist
      (default: secret/t6-apps/)
    • KeyNames are listed by name only, never values
    • CommandTemplate uses \${VALUE_*} placeholders the validator
      enforces are never replaced with concrete values

  Synthetic diagnostic this time too — a missing-ExternalSecret-path
  scenario.

EOF

section "Running T3 ProposeRunbook"
cat <<'GO' > /tmp/srenix-demo-v3-t3.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	chacomai "github.com/srenix-ai/agentic-sre-enterprise/ai"
	"github.com/srenix-ai/agentic-sre-enterprise/ai/client"
	"github.com/srenix-ai/agentic-sre/pkg/ai"
	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
)

func main() {
	llm, err := client.NewOpenAIClient(client.OpenAIConfig{
		Endpoint:     os.Getenv("AI_ENDPOINT"),
		APIKey:       os.Getenv("AI_API_KEY"),
		APIKeyHeader: os.Getenv("AI_HEADER"),
		DefaultModel: os.Getenv("AI_MODEL"),
	})
	if err != nil { fmt.Println("ERR:", err); os.Exit(1) }
	d := diagnose.Diagnostic{
		Source: "VaultPathMissing", Subject: "ExternalSecret/billing/billing-config",
		Severity: "critical",
		Message:  "ExternalSecret billing/billing-config references vault path secret/t6-apps/billing/config which does not exist in Vault.",
		Remediation: "Create the vault path and populate it with the required keys.",
	}
	t3, _ := chacomai.NewVaultRunbookProposer(chacomai.VaultRunbookProposerConfig{
		Client: llm, Audit: ai.NoOpAuditSink{},
		AllowedPathPrefixes: []string{"secret/t6-apps/"},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second); defer cancel()
	rb, err := t3.ProposeRunbook(ctx, d)
	if err != nil { fmt.Println("ERR:", err); os.Exit(1) }
	out, _ := json.MarshalIndent(rb, "", "  ")
	fmt.Println(string(out))
	fmt.Println()
	fmt.Println("Dual-approval timeline:")
	fmt.Println("  Approver A clicks at T+0  → first slot recorded")
	fmt.Println("  ≥30 minutes elapse        → MinT3Delay enforced")
	fmt.Println("  Approver B clicks at T+N  → second slot recorded (MUST be a different identity)")
	fmt.Println("  Runbook becomes executable → operator runs it MANUALLY")
}
GO

mkdir -p "${REPO_DIR_COM}/cmd/.demo-v3-t3"
cp /tmp/srenix-demo-v3-t3.go "${REPO_DIR_COM}/cmd/.demo-v3-t3/main.go"

export AI_HEADER AI_ENDPOINT AI_MODEL
(cd "${REPO_DIR_COM}" && go run ./cmd/.demo-v3-t3/) 2>&1

rm -rf "${REPO_DIR_COM}/cmd/.demo-v3-t3" /tmp/srenix-demo-v3-t3.go

cat <<EOF

  Notice:
    • VaultPath is inside secret/t6-apps/ — the allowlist gate
    • KeyNames lists DB_PASSWORD, API_KEY (or similar) — names only,
      never values
    • CommandTemplate uses \${VALUE_*} placeholders, not concrete values
    • RunbookID is a UUID — the approval-server signs THIS into TWO
      separate JWTs for the two approvers, with NotBefore set to
      enforce MinT3Delay

  Even with both approvals, Srenix Enterprise never writes Vault itself. The
  operator runs the command template by hand. That's deliberate — Vault
  writes are the last asset class we want LLM autonomy on.

EOF

pause "T1 + T3 demonstrated end-to-end. One more thing: the audit trail."

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 6 — Hash-chained audit (no SaaS, no vendor)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  Every AI action emits an AuditEvent: ai.llm.call, ai.proposal.created,
  ai.proposal.validated, ai.approval.granted, ai.action.applied,
  ai.action.failed, ai.runbook.dual_approval.

  Each event's prev_hash is sha256 of the canonical JSON of the prior
  event. Tamper-evident even if the downstream sink is compromised.

  Live demo with --ai-audit-log writing JSONL to stdout, 2 cycles
  3 seconds apart.

EOF

section "Running: srenix-enterprise watch --max-cycles=2 --ai-audit-log=- (JSONL on stdout)"
srenix_com watch \
  --interval=3s --max-cycles=2 \
  --ai-tier=t0 \
  --ai-endpoint="${AI_ENDPOINT}" \
  --ai-model="${AI_MODEL}" \
  --ai-api-key-header="${AI_HEADER}" \
  --ai-audit-log=- 2>&1 | grep -E '^\{|cycle=|RESOLVED|🤖' | head -25

cat <<EOF

  The JSON lines above are the audit trail. In production, point
  --ai-audit-log at a file (mode 0600) or pipe to Loki / Vector /
  fluent-bit. The chain integrity is verifiable post-hoc.

EOF

pause "Demo complete. Cleanup runs automatically on exit."

# ═══════════════════════════════════════════════════════════════════════════════
header "Demo v3 wrap-up"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<EOF

  What you saw, in one sentence per layer:

    [OSS] The trustable runtime — 12 K8s probes, 8 analyzers, 5 fixers,
          lease-based leader election. The agent's substrate.

    [T0]  The LLM reads each DriftReport and writes a one-paragraph
          actionable narration for the operator's Slack.

    [T1]  The LLM proposes a specific operator-policy-bounded action_kind
          (DeletePod / PatchAnnotation / etc.) with a rollback. The
          ActionID is signed into a Slack URL; one click executes.

    [T3]  For Vault break-glass, two distinct operators must approve,
          separated by ≥30 minutes. Srenix Enterprise never writes Vault — the
          operator runs the proposed runbook by hand.

    [Audit] Every step emits a hash-chained event. The whole loop is
            replayable from the JSONL log.

  This is the AI SRE story: policy-bounded autonomy with an in-cluster
  agent and a bring-your-own-LLM endpoint. Everyone else observes;
  Srenix actually mutates.

EOF

info "Cleanup runs on exit (trap)."
