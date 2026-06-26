#!/usr/bin/env bash
# Srenix Demo Master Script
# Runs through the key capabilities in sequence with pause points for narration
#
# Set KUBE_CONTEXT to target the remote AWS cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash demo/run-demo.sh
#
# Or switch context first:
#   kubectl config use-context <aws-context>
#   bash demo/run-demo.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="demo-app"
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

pause() {
  echo ""
  echo -e "${YELLOW}[PAUSE] $1${NC}"
  echo -e "${YELLOW}Press ENTER to continue...${NC}"
  read -r
}

header() {
  echo ""
  echo -e "${BLUE}${BOLD}══════════════════════════════════════════${NC}"
  echo -e "${BLUE}${BOLD}  $1${NC}"
  echo -e "${BLUE}${BOLD}══════════════════════════════════════════${NC}"
}

# ─── Pre-flight ───────────────────────────────────────────────────────────────
header "Srenix Demo — Pre-flight Checks"

echo -e "${BOLD}Cluster context:${NC} ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo ""
echo -e "${BOLD}Cluster info:${NC}"
$KUBECTL cluster-info 2>/dev/null | head -2

echo ""
echo -e "${BOLD}Srenix watcher status:${NC}"
$KUBECTL get pods -n srenix --no-headers 2>/dev/null || echo "  ⚠ Srenix not running in 'srenix' namespace"

echo ""
echo -e "${BOLD}Demo namespace:${NC}"
$KUBECTL get ns "${NAMESPACE}" --no-headers 2>/dev/null || {
  echo "  Creating demo-app namespace..."
  $KUBECTL create namespace "${NAMESPACE}"
}
$KUBECTL get pods -n "${NAMESPACE}" --no-headers 2>/dev/null | head -10 || true

pause "Pre-flight done. Open Slack #aws-alerts in browser. Ready to start demo?"

# ─── Section 1: Architecture Overview ────────────────────────────────────────
header "Section 1 — What Srenix Is"

cat <<'EOF'
Agentic SRE (Srenix) is an event-driven Kubernetes operator that:

  DETECT  → Runs 5 infrastructure probes + 7 application analyzers
  FIX     → Applies 4 whitelisted auto-remediations (no human needed)
  VERIFY  → Re-probes post-fix to confirm resolution
  REPORT  → Posts to Slack with full context: what failed + what was fixed

Key differentiator: the watcher captures pre-fix diagnostic state before
running fixers, so Slack always shows WHAT was wrong even when Srenix fixed
it in the same cycle (< 10 seconds from detection to resolution).

DriftReport CRDs persist state in-cluster — no Slack flood on pod restart.
EOF

pause "Section 1 narrated. Ready for live demo — Section 2?"

# ─── Section 2: Current Cluster Health ───────────────────────────────────────
header "Section 2 — Live Cluster Health Snapshot"

echo -e "${BOLD}DriftReports in cluster (Srenix's persistent diagnostic state):${NC}"
$KUBECTL get driftreports -A --no-headers 2>/dev/null | head -20 || \
  echo "  (no active diagnostics — cluster is healthy)"

pause "Section 2 done. Ready to simulate a stale pod failure?"

# ─── Section 3: Auto-Fix Demo — Stale Error Pods ─────────────────────────────
header "Section 3 — Autopilot: Stale Pod Detection + Auto-Fix"

echo -e "${BOLD}Creating a Failed pod that Srenix will auto-delete...${NC}"
echo ""
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/01-stale-error-pods.sh" "${NAMESPACE}"

echo ""
echo -e "${BOLD}Expected sequence (watch Slack #aws-alerts):${NC}"
echo "  1. Pod enters Failed state"
echo "  2. Srenix watcher detects within ~10s debounce"
echo "  3. StaleErrorPods fixer deletes it"
echo "  4. Slack: 🔴 Active Issues + 🔧 Fixes Applied"
echo "  5. Next cycle: ✅ Resolved"

pause "Section 3 done. Did you see the Slack alert with 🔴 + 🔧?"

# ─── Section 4: Reported but Not Auto-Fixed — Missing Secret Key ──────────────
header "Section 4 — Reporting: Missing Secret Key (Manual Fix Required)"

echo -e "${BOLD}Creating a Deployment with a missing Secret key...${NC}"
echo ""
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/02-missing-secret-key.sh" "${NAMESPACE}"

echo ""
pause "Srenix is detecting this. Check Slack for SecretKeyMissing alert. Ready to fix it manually?"

echo -e "${BOLD}Fixing: Adding missing key to Secret...${NC}"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/fix-scripts/fix-missing-secret-key.sh" \
  "${NAMESPACE}" "database-credentials" "DB_PASSWORD" "demo-password-$(date +%s | tail -c 6)"

echo ""
pause "Section 4 done. Did you see ✅ Resolved in Slack after the fix?"

# ─── Section 5: Stuck Job Auto-Fix ───────────────────────────────────────────
header "Section 5 — Autopilot: Stuck CronJob Auto-Fix"

echo -e "${BOLD}Creating a CronJob with concurrencyPolicy=Forbid and a bad SecretRef...${NC}"
echo ""
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/04-stuck-job-bad-secret.sh" "${NAMESPACE}"

echo ""
echo -e "${BOLD}What just happened:${NC}"
echo "  • Secret has API_KEY_NEW but CronJob template referenced stale API_KEY"
echo "  • Job's pod entered CreateContainerConfigError immediately"
echo "  • Script patched CronJob template to the correct key (root cause fixed)"
echo "  • But the OLD Job is still Active — blocking CronJob (concurrencyPolicy=Forbid)"
echo ""
echo -e "${BOLD}Srenix will:${NC}"
echo "  1. Detect pod with 'couldn't find key API_KEY' in waiting.message"
echo "  2. Confirm parent CronJob exists"
echo "  3. Delete the frozen Job"
echo "  4. CronJob's next tick (≤2 min): fresh Job with correct key → succeeds ✓"

pause "Section 5 done. Ready for the snapshot / zero-trust demo?"

# ─── Section 6: Zero-Trust Snapshot Demo ─────────────────────────────────────
header "Section 6 — Zero-Trust: Offline Snapshot Analysis"

echo -e "${BOLD}Srenix can diagnose without any live cluster access:${NC}"
echo ""
echo "  # On any machine — no kubeconfig, no RBAC, no network to cluster:"
echo "  srenix diagnose --snapshot examples/sample-cluster/"
echo ""

REPO_DIR="$(dirname "${DEMO_DIR}")"
if [[ -d "${REPO_DIR}/examples/sample-cluster" ]]; then
  cd "${REPO_DIR}"
  go run ./cmd/srenix diagnose --snapshot examples/sample-cluster/ 2>/dev/null || true
else
  echo "  (examples/sample-cluster/ not found — skip or run: srenix diagnose --snapshot <path>)"
fi

pause "Section 6 done. Ready for cleanup?"

# ─── Cleanup ──────────────────────────────────────────────────────────────────
header "Cleanup"

echo -e "${BOLD}Cleaning up all demo-injected resources...${NC}"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/06-cleanup-all.sh" "${NAMESPACE}"

echo ""
echo -e "${GREEN}${BOLD}Demo complete.${NC}"
echo ""
echo "Summary of what was demonstrated:"
echo "  ✓ Live cluster health scan (5 probes, 7 analyzers)"
echo "  ✓ Stale pod auto-detected and auto-deleted in < 30s"
echo "  ✓ Missing Secret key surfaced with full Deployment/ESO context"
echo "  ✓ Manual fix → Slack ✅ Resolved confirmation"
echo "  ✓ Frozen CronJob Job auto-deleted to unblock concurrencyPolicy=Forbid"
echo "  ✓ Zero-trust offline snapshot mode (no cluster access needed)"
echo ""
echo "Repo: https://github.com/srenix-ai/agentic-sre"
