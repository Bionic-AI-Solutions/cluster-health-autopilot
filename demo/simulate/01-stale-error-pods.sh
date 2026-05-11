#!/usr/bin/env bash
# Simulate: Stale Error Pods
# CHA auto-fix: StaleErrorPods fixer deletes these automatically
# Expected Slack: 🔴 Active Issues → 🔧 Fixes Applied → ✅ Resolved (~30s cycle)
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/01-stale-error-pods.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"
POD_NAME="debug-probe-$(date +%s | tail -c 5)"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Creating Failed pod in namespace: $NAMESPACE"
echo "    Pod name: $POD_NAME"

$KUBECTL apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    demo: stale-error-pod
    injected-by: cha-demo
spec:
  restartPolicy: Never
  containers:
  - name: probe
    image: busybox:1.36
    command: ["sh", "-c", "exit 1"]
EOF

echo ""
echo "==> Pod created. Waiting for it to reach Failed state..."
$KUBECTL wait --for=jsonpath='{.status.phase}'=Failed pod/"${POD_NAME}" \
  -n "${NAMESPACE}" --timeout=60s 2>/dev/null || true

$KUBECTL get pod "${POD_NAME}" -n "${NAMESPACE}" --no-headers
echo ""
echo "==> CHA watcher will detect this within ~10s debounce window."
echo "    In autopilot mode: StaleErrorPods fixer will delete it automatically."
echo "    Watch Slack #aws-alerts for the alert + fix confirmation."
echo ""
echo "    To watch manually:"
echo "    ${KUBECTL} get pod ${POD_NAME} -n ${NAMESPACE} -w"
