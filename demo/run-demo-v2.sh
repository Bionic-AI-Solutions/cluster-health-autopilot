#!/usr/bin/env bash
# CHA Comprehensive Demo — v2
#
# A staged walkthrough that builds from zero to full autopilot on a live AWS EKS cluster.
# Each section pauses for narration. Press ENTER to advance.
#
# Prerequisites:
#   export KUBE_CONTEXT="test-cluster1"   (or the exact EKS context name)
#   bash demo/run-demo-v2.sh
#
# What this script does (and undoes at cleanup):
#   - Progressively upgrades the CHA Helm release through 4 modes
#   - Injects synthetic failures into the demo-app namespace
#   - Removes all injected resources and restores original Helm values on exit

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
HELM="helm${KUBE_CONTEXT:+ --kube-context ${KUBE_CONTEXT}}"
NAMESPACE="demo-app"
RELEASE="cha-remote"
CHA_NS="cha"
DIAGNOSE_CJ="${RELEASE}-cluster-health-autopilot-diagnose"
REMEDIATE_CJ="${RELEASE}-cluster-health-autopilot-remediate"
WATCHER_DEPLOY="${RELEASE}-cluster-health-autopilot-watcher"
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "${DEMO_DIR}")"
SNAPSHOT_DIR="/tmp/cha-demo-snapshot"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

header()  { echo ""; echo -e "${BLUE}${BOLD}══════════════════════════════════════════════════════${NC}"; echo -e "${BLUE}${BOLD}  $1${NC}"; echo -e "${BLUE}${BOLD}══════════════════════════════════════════════════════${NC}"; }
section() { echo ""; echo -e "${CYAN}${BOLD}── $1 ──${NC}"; }
info()    { echo -e "  ${GREEN}▶${NC} $*"; }
warn()    { echo -e "  ${YELLOW}⚠${NC}  $*"; }
pause()   { echo ""; echo -e "${YELLOW}[PAUSE]${NC} $1"; echo -e "${YELLOW}Press ENTER to continue...${NC}"; read -r; }

# ─── Cleanup trap — restores original Helm values on exit ─────────────────────
ORIG_VALUES_FILE="/tmp/cha-demo-orig-values.yaml"
cleanup() {
  echo ""
  echo -e "${BOLD}Running cleanup...${NC}"
  KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/06-cleanup-all.sh" "${NAMESPACE}" 2>/dev/null || true
  $KUBECTL delete driftreports --all 2>/dev/null || true
  $KUBECTL delete namespace "${NAMESPACE}" --ignore-not-found 2>/dev/null || true
  if [[ -f "${ORIG_VALUES_FILE}" ]]; then
    echo "Restoring original Helm values..."
    $HELM upgrade "${RELEASE}" "${REPO_DIR}/charts/cluster-health-autopilot" \
      -n "${CHA_NS}" -f "${ORIG_VALUES_FILE}" --wait --timeout=120s 2>/dev/null || true
    rm -f "${ORIG_VALUES_FILE}"
  fi
  rm -rf "${SNAPSHOT_DIR}"
  echo -e "${GREEN}Cleanup complete.${NC}"
}
trap cleanup EXIT

# ─── Helper: helm upgrade one step, wait for watcher rollout ──────────────────
helm_upgrade() {
  local extra_args=("$@")
  $HELM upgrade "${RELEASE}" "${REPO_DIR}/charts/cluster-health-autopilot" \
    -n "${CHA_NS}" --reuse-values "${extra_args[@]}" --wait --timeout=120s
}

# ─── Helper: trigger an ad-hoc diagnose job and tail it ───────────────────────
run_diagnose_now() {
  local job="cha-demo-diag-$(date +%s | tail -c 5)"
  $KUBECTL create job --from=cronjob/"${DIAGNOSE_CJ}" "${job}" -n "${CHA_NS}" 2>/dev/null || {
    warn "diagnose CronJob '${DIAGNOSE_CJ}' not found — skipping"
    return 0
  }
  echo "  Waiting for job/${job} to complete..."
  $KUBECTL wait --for=condition=complete job/"${job}" -n "${CHA_NS}" --timeout=120s 2>/dev/null || true
  $KUBECTL logs -n "${CHA_NS}" "job/${job}" 2>/dev/null || true
}

# ─── Helper: trigger an ad-hoc remediate job and tail it ──────────────────────
run_remediate_now() {
  local job="cha-demo-rem-$(date +%s | tail -c 5)"
  $KUBECTL create job --from=cronjob/"${REMEDIATE_CJ}" "${job}" -n "${CHA_NS}" 2>/dev/null || {
    echo "  (remediate CronJob not enabled yet — skipping)"
    return 0
  }
  echo "  Waiting for job/${job} to complete..."
  $KUBECTL wait --for=condition=complete job/"${job}" -n "${CHA_NS}" --timeout=120s 2>/dev/null || true
  $KUBECTL logs -n "${CHA_NS}" "job/${job}" 2>/dev/null || true
}

# ═══════════════════════════════════════════════════════════════════════════════
header "CHA Demo v2 — Pre-flight"
# ═══════════════════════════════════════════════════════════════════════════════

section "1/3 — Cluster"
echo -e "  Context: ${BOLD}${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}${NC}"
$KUBECTL cluster-info 2>/dev/null | head -2
echo ""
$KUBECTL get nodes --no-headers 2>/dev/null | awk '{printf "  %-40s %s\n", $1, $2}'

section "2/3 — CHA Helm release"
$HELM status "${RELEASE}" -n "${CHA_NS}" 2>/dev/null | grep -E "NAME|CHART|APP VERSION|STATUS|REVISION"
echo ""
info "Saving current values for restore-on-exit..."
$HELM get values "${RELEASE}" -n "${CHA_NS}" -o yaml > "${ORIG_VALUES_FILE}" 2>/dev/null

section "3/3 — Slack webhook"
WEBHOOK=$($KUBECTL get secret cha-slack -n "${CHA_NS}" \
  -o jsonpath='{.data.WEBHOOK_URL}' 2>/dev/null | base64 -d 2>/dev/null)
if [[ -z "${WEBHOOK}" ]]; then
  warn "cha-slack secret not found — Slack posts will not appear"
else
  RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${WEBHOOK}" \
    -H 'Content-Type: application/json' \
    -d '{"text":"🚀 CHA Demo v2 starting — watch this channel for alerts"}' 2>/dev/null)
  [[ "${RESP}" == "200" ]] && info "Slack webhook verified (200 OK) — check #aws-alerts" \
                            || warn "Slack returned HTTP ${RESP}"
fi

pause "Pre-flight done. Open Slack #aws-alerts. Ready to start?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 1 — Cluster Setup & CHA State"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  CHA is installed via Helm on this AWS EKS cluster. Current state:

  ┌─────────────────────────────────────────────────────┐
  │  diagnose CronJob      → read-only probes daily      │
  │  watcher Deployment    → event-driven, always-on     │
  │  StuckCertReqs fixer   → auto-deletes failed CRs     │
  │  StaleErrorPods fixer  → auto-deletes orphan pods    │
  │  StuckJobs fixer       → auto-unblocks CronJobs      │
  │  StuckRSPods fixer     → rollout restart RS pods     │
  └─────────────────────────────────────────────────────┘

EOF

section "Current DriftReports (cluster health state)"
$KUBECTL get driftreports -A --no-headers 2>/dev/null | \
  awk '{printf "  %-12s %-10s %-60s\n", $2, $3, $4}' || \
  echo "  (none — cluster is clean)"

section "Active watcher pod"
$KUBECTL get pods -n "${CHA_NS}" --no-headers 2>/dev/null | grep watcher || echo "  (not running)"

pause "Section 1 done. Ready for zero-trust offline snapshot mode?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 2 — Zero-Trust: Offline Snapshot Mode"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  KEY POINT: CHA can diagnose any cluster with NO live access.
  Capture YAML once, run anywhere — air-gapped, laptop, CI pipeline.

  Captures only: pods, nodes, PVCs, events, deployments, replicasets,
  jobs, cronjobs, externalsecrets, secrets (names only), certificates,
  certificaterequests, orders, cnpg clusters, cephclusters.
  No secret values are ever read.

EOF

section "Step A — Capturing cluster snapshot"
mkdir -p "${SNAPSHOT_DIR}"
echo "  Snapshotting resource types into ${SNAPSHOT_DIR}..."
for resource in pods nodes persistentvolumeclaims events \
    deployments replicasets statefulsets jobs cronjobs \
    externalsecrets certificates certificaterequests; do
  $KUBECTL get "${resource}" -A -o yaml 2>/dev/null > "${SNAPSHOT_DIR}/${resource}.yaml" || true
  printf "    %-30s %s objects\n" "${resource}" \
    "$($KUBECTL get "${resource}" -A --no-headers 2>/dev/null | wc -l)"
done
echo ""
info "Snapshot written. No live cluster access used beyond this point."

section "Step B — Running cha diagnose --snapshot (zero trust)"
echo ""
cd "${REPO_DIR}"
go build -o /tmp/cha-demo-bin ./cmd/cha 2>/dev/null
/tmp/cha-demo-bin diagnose --snapshot "${SNAPSHOT_DIR}" 2>/dev/null || true
echo ""

section "Analyzer catalog — what CHA looks for"
cat <<'EOF'
  7 Analyzers (read-only, never mutate):
  ┌─────────────────────────────────┬────────────────────────────────────────────────────┐
  │ SecretKeyMissing                │ Deployment refs a key not in the Secret             │
  │ FailingExternalSecrets          │ ESO Ready=False — Vault path/property broken        │
  │ ProactiveSecretKeyCheck         │ ESO synced but referenced key not in output         │
  │ UnprovisionedSecret             │ envFrom whole-secret, no ESO to provision it        │
  │ ImagePullAuth                   │ ErrImagePull/auth failure event in pod events       │
  │ CertExpiry                      │ cert-manager Cert not Ready / expiring / expired    │
  │ VaultPathMissing (paid)         │ Vault path in ESO doesn't exist yet                 │
  └─────────────────────────────────┴────────────────────────────────────────────────────┘

  4 Fixers (whitelist-only, opt-in):
  ┌─────────────────────────────────┬────────────────────────────────────────────────────┐
  │ StaleErrorPods                  │ Deletes Failed pods with no restart controller      │
  │ StuckJobsWithBadSecretRef       │ Deletes Job stuck in ConfigError (CronJob parent)  │
  │ StuckCertificateRequests        │ Deletes terminal-fail CertReqs + Orders             │
  │ StuckRSPods                     │ rollout restart when RS pod stuck (non-secret root) │
  └─────────────────────────────────┴────────────────────────────────────────────────────┘
EOF

pause "Section 2 done. Ready to enable CronJob mode with dry-run remediation?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 3 — CronJob Mode + Dry-Run Remediation"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  We start with CronJobs only — no watcher yet.
  Remediation is enabled in DRY-RUN mode: fixers evaluate but do NOT mutate.
  Slack shows what WOULD be fixed on each cron tick.

EOF

section "Helm upgrade: watcher OFF, remediate ON (dryRun=true, every 2 min)"
helm_upgrade \
  --set watcher.enabled=false \
  --set remediation.enabled=true \
  --set remediation.dryRun=true \
  --set "remediation.schedule=*/2 * * * *"
info "Upgrade applied."

section "Running ad-hoc diagnose + remediate (dry-run) — cluster is clean"
run_diagnose_now
echo ""
run_remediate_now

pause "Section 3 done — clean cluster, no issues. Ready to inject first failure?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 4 — First Failure: Stale Pods (auto-fixable, dry-run)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  SCENARIO: Pods in Failed phase with restartPolicy=Never left behind after
  a failed init, batch job, or debug session. No controller will restart them.

  CHA fixer: StaleErrorPods
  Safety gate: pod phase=Failed AND no owner that would restart it.

  In DRY-RUN mode: reports "would delete" but does NOT delete.

EOF

$KUBECTL get namespace "${NAMESPACE}" &>/dev/null || \
  $KUBECTL create namespace "${NAMESPACE}"

section "Injecting 2 stale Failed pods"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/01-stale-error-pods.sh" "${NAMESPACE}"

echo ""
info "Triggering diagnose + remediate (dry-run)..."
run_diagnose_now
echo ""
run_remediate_now

pause "Check Slack #aws-alerts — you should see the stale pod report with DRY-RUN label. Ready for Section 5?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 5 — Multiple Failures (dry-run remediation report)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  Now we inject the full set of failure types:
  AUTO-FIXABLE (CHA will act when live):
    • Stale failed pods          → StaleErrorPods fixer
    • Stuck CronJob/Job          → StuckJobsWithBadSecretRef fixer
  REPORTED ONLY (human action required):
    • Missing Secret key         → SecretKeyMissing analyzer
    • Failing ExternalSecret     → FailingExternalSecrets analyzer
    • ImagePullAuth failure      → ImagePullAuth analyzer

EOF

section "Injecting missing secret key"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/02-missing-secret-key.sh" "${NAMESPACE}"

section "Injecting failing ExternalSecret"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/03-failing-externalsecret.sh" "${NAMESPACE}" 2>/dev/null || \
  warn "ESO not installed on this cluster — skipping ExternalSecret simulation"

section "Injecting stuck CronJob with bad SecretRef"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/04-stuck-job-bad-secret.sh" "${NAMESPACE}"

section "Injecting ImagePull auth failure"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/05-image-pull-auth-failure.sh" "${NAMESPACE}"

echo ""
info "Waiting 30s for failures to settle, then running diagnose + remediate..."
sleep 30
run_diagnose_now
echo ""
run_remediate_now

pause "Check Slack — full issue list + which are auto-fixable (DRY-RUN). Ready for Section 6?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 6 — Enable Watcher (observe mode, no remedy yet)"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  The watcher is an event-driven Deployment — reacts to Kubernetes watch events
  within a 10s debounce window, vs. waiting for the next cron tick.

  In this step: watcher ON, auto-remedy OFF.
  Slack will receive ONE message with all currently-active issues.
  Subsequent cycles are silent unless something changes.

EOF

section "Helm upgrade: watcher ON, remedy OFF"
helm_upgrade \
  --set watcher.enabled=true \
  --set "watcher.remedy.enabled=false" \
  --set "watcher.remedy.dryRun=false" \
  --set remediation.enabled=false

section "Watcher rollout"
$KUBECTL rollout status deploy/"${WATCHER_DEPLOY}" -n "${CHA_NS}" --timeout=90s 2>/dev/null
echo ""
$KUBECTL get pods -n "${CHA_NS}" --no-headers 2>/dev/null | grep watcher

echo ""
info "Clearing any leftover DriftReports so the watcher starts with an empty seen-map..."
$KUBECTL delete driftreports --all 2>/dev/null || true
echo ""
echo -e "  ${BOLD}Restarting watcher to trigger first-post of all active issues...${NC}"
$KUBECTL rollout restart deploy/"${WATCHER_DEPLOY}" -n "${CHA_NS}" 2>/dev/null
$KUBECTL rollout status deploy/"${WATCHER_DEPLOY}" -n "${CHA_NS}" --timeout=90s 2>/dev/null

pause "Watch Slack — first Slack message with all active issues (🔔 Active Issues). Ready for live remediation?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 7 — Full Autopilot: Watcher + Live Remediation"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  AUTOPILOT MODE: watcher detects → fixers run → re-diagnose → Slack diff posted.
  The Slack message shows:
    🔴 Active Issues    — what was wrong before fix
    🔧 Fixes Applied    — what CHA deleted/patched
    ✅ Resolved (next cycle) — what is now clean

  Only whitelisted fixers run. Everything else stays reported-only.

EOF

section "Helm upgrade: watcher + remedy LIVE (dryRun=false)"
helm_upgrade \
  --set watcher.enabled=true \
  --set "watcher.remedy.enabled=true" \
  --set "watcher.remedy.dryRun=false"

$KUBECTL rollout status deploy/"${WATCHER_DEPLOY}" -n "${CHA_NS}" --timeout=90s 2>/dev/null

echo ""
info "Watcher now running in autopilot mode."
info "Auto-fixable issues (stale pods, stuck CronJob) will resolve within ~20s."
echo ""
echo -e "  ${BOLD}Expected Slack sequence:${NC}"
echo "    🔴 Active Issues (5) + 🔧 Fixes Applied (2) → in one message"
echo "    ✅ Resolved (2) → next cycle after pods/jobs confirm deleted"

pause "Section 7 done — auto-fixable issues cleaned. Ready to inject a live issue?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 8 — Live Detection: New Issue → Immediate Fix"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  The watcher reacts to Kubernetes watch events, not polling.
  A new failure in the cluster triggers the full cycle within ~10s debounce.

  Sequence:
    1. Pod created → k8s watch event fires
    2. Watcher debounce expires (~10s)
    3. diagnose + fix cycle runs
    4. Slack posts: 🔴 Detected + 🔧 Fixed
    5. Next cycle: ✅ Resolved

EOF

section "Injecting a new stale pod right now"
POD_NAME="live-probe-$(date +%s | tail -c 5)"
$KUBECTL apply -f - <<PODEOF
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    demo: live-inject
spec:
  restartPolicy: Never
  containers:
  - name: probe
    image: busybox:1.36
    command: ["sh", "-c", "exit 1"]
PODEOF
info "Pod ${POD_NAME} created — exits immediately with code 1 → Failed"
echo ""
info "Watch the watcher log:"
echo "    $KUBECTL logs -n ${CHA_NS} deploy/${WATCHER_DEPLOY} -f --tail=5"
echo ""
info "Watch Slack — you should see the alert + fix within ~30 seconds."

pause "Did you see the live alert + immediate auto-fix in Slack? Ready for Section 9?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 9 — Non-Auto-Fixable Issues + Manual Fix Scripts"
# ═══════════════════════════════════════════════════════════════════════════════

cat <<'EOF'

  Three issues remain that CHA reports but cannot safely auto-fix:
    A. Missing Secret key (DB_PASSWORD)   → fix-missing-secret-key.sh
    B. Failing ExternalSecret             → fix-failing-externalsecret.sh
    C. ImagePull auth failure             → fix-image-pull-secret.sh

  Each fix script simulates the operator action. After each fix, CHA's watcher
  detects the recovery and posts ✅ Resolved to Slack within ~20s.

EOF

# ── Fix A: Missing Secret key ──────────────────────────────────────────────────
section "Fix A — Missing Secret key (DB_PASSWORD)"
echo ""
echo -e "  ${BOLD}Why CHA doesn't auto-fix this:${NC}"
echo "  In production: secrets come from Vault via ESO. CHA never writes secret values."
echo "  The correct fix is: add the key to Vault → ESO auto-syncs → pod restarts."
echo "  This script simulates the Vault+ESO outcome by patching the Secret directly."
echo ""
pause "Press ENTER to run fix-missing-secret-key.sh..."

KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/fix-scripts/fix-missing-secret-key.sh" \
  "${NAMESPACE}" "database-credentials" "DB_PASSWORD" "demo-password-$(date +%s | tail -c 6)"

echo ""
info "Secret patched + Deployment restarted. Watch Slack for ✅ Resolved (~20s)."

pause "Press ENTER to continue to Fix B..."

# ── Fix B: Failing ExternalSecret ─────────────────────────────────────────────
section "Fix B — Failing ExternalSecret (broken Vault path)"
echo ""
echo -e "  ${BOLD}Why CHA doesn't auto-fix this:${NC}"
echo "  The Vault path/property in the ESO spec is wrong. CHA can't rewrite the spec"
echo "  without knowing the correct Vault path — that requires human/operator knowledge."
echo "  Fix: correct the Vault data or fix the ESO spec, then force re-sync."
echo ""
pause "Press ENTER to run fix-failing-externalsecret.sh..."

KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/fix-scripts/fix-failing-externalsecret.sh" \
  "${NAMESPACE}" "demo-externalsecret" 2>/dev/null || \
  warn "ESO not installed — skipping (no ExternalSecret was injected in Section 5)"

echo ""
info "ExternalSecret forced re-sync. Watch Slack for ✅ Resolved."

pause "Press ENTER to continue to Fix C..."

# ── Fix C: ImagePull auth failure ─────────────────────────────────────────────
section "Fix C — ImagePull auth failure"
echo ""
echo -e "  ${BOLD}Why CHA doesn't auto-fix this:${NC}"
echo "  Creating a registry credential requires knowing the username + password."
echo "  CHA has read-only RBAC on secrets — it never writes auth material."
echo "  Fix: create the imagePullSecret and patch the pod or ServiceAccount."
echo ""
pause "Press ENTER to run fix-image-pull-secret.sh..."

POD=$($KUBECTL get pods -n "${NAMESPACE}" -l demo=image-pull-auth \
  -o name 2>/dev/null | head -1 | cut -d/ -f2 || true)
if [[ -n "${POD}" ]]; then
  KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/fix-scripts/fix-image-pull-secret.sh" \
    "${NAMESPACE}" "${POD}"
else
  warn "No image-pull-auth pod found — may have been cleaned up already"
fi

echo ""
info "imagePullSecret created + pod restarted. Watch Slack for ✅ Resolved."

section "Final state check"
echo ""
$KUBECTL get driftreports -A --no-headers 2>/dev/null | \
  awk '{printf "  %-10s %-60s\n", $3, $4}' || echo "  (all resolved)"

pause "Section 9 done. All issues either auto-fixed or manually resolved. Ready for cleanup?"

# ═══════════════════════════════════════════════════════════════════════════════
header "Section 10 — Cleanup"
# ═══════════════════════════════════════════════════════════════════════════════

section "Removing all demo-injected resources"
KUBE_CONTEXT="${KUBE_CONTEXT:-}" bash "${DEMO_DIR}/simulate/06-cleanup-all.sh" "${NAMESPACE}"
$KUBECTL delete pods -n "${NAMESPACE}" -l demo=live-inject --ignore-not-found 2>/dev/null || true
$KUBECTL delete driftreports --all 2>/dev/null || true
$KUBECTL delete namespace "${NAMESPACE}" --ignore-not-found 2>/dev/null || true

section "Restoring original Helm values"
$HELM upgrade "${RELEASE}" "${REPO_DIR}/charts/cluster-health-autopilot" \
  -n "${CHA_NS}" -f "${ORIG_VALUES_FILE}" --wait --timeout=120s
rm -f "${ORIG_VALUES_FILE}"

section "Final cluster state"
$KUBECTL get pods -n "${CHA_NS}" --no-headers 2>/dev/null
echo ""
$KUBECTL get driftreports -A --no-headers 2>/dev/null | wc -l | \
  xargs -I{} echo "  {} DriftReports active"

echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD}  Demo complete.${NC}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  What was demonstrated:"
echo "  ✓ Zero-trust offline snapshot mode (no kubeconfig needed)"
echo "  ✓ All 7 analyzers catalogued with real examples"
echo "  ✓ All 4 fixers explained with safety contracts"
echo "  ✓ CronJob mode with dry-run remediation report in Slack"
echo "  ✓ Watcher first-post: all active issues in one Slack message"
echo "  ✓ Full autopilot: detect → fix → Slack diff in <30s"
echo "  ✓ Live new-issue detection within debounce window"
echo "  ✓ Manual fix scripts close the loop on non-auto-fixable issues"
echo ""
echo "  Repo: https://github.com/Bionic-AI-Solutions/cluster-health-autopilot"
echo "  Branch: demo/v2-comprehensive"
echo ""

# trap cleanup will run on exit — values already restored above, it will no-op
