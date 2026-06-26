#!/usr/bin/env bash
# Simulate: ImagePullBackOff due to auth failure
# Srenix reports: ImagePullAuth analyzer
# Srenix auto-fix: NONE — imagePullSecret must be created/fixed
# Manual fix:   demo/fix-scripts/fix-image-pull-secret.sh
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/05-image-pull-auth-failure.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"
POD_NAME="auth-fail-demo-$(date +%s | tail -c 4)"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Simulating ImagePullBackOff auth failure in namespace: $NAMESPACE"
echo ""

$KUBECTL apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    demo: image-pull-auth
spec:
  restartPolicy: Never
  containers:
  - name: app
    image: docker4zerocool/private-demo-image:latest
  imagePullSecrets:
  - name: invalid-registry-secret
EOF

echo ""
echo "==> Pod created referencing a non-existent imagePullSecret."
echo "    Pod will enter ImagePullBackOff with auth failure events."
echo ""
echo "    Watch pod status:"
echo "    ${KUBECTL} get pod ${POD_NAME} -n ${NAMESPACE} -w"
echo "    ${KUBECTL} describe pod ${POD_NAME} -n ${NAMESPACE} | grep -A5 Events"
echo ""
echo "==> Srenix ImagePullAuth analyzer will detect auth-failure keywords in events."
echo "    Check Slack #aws-alerts for: 'cannot pull image — auth failure'"
echo ""
echo "    To fix: run demo/fix-scripts/fix-image-pull-secret.sh ${NAMESPACE} ${POD_NAME}"
