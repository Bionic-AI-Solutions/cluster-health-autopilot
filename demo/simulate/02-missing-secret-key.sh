#!/usr/bin/env bash
# Simulate: Missing Secret Key (CreateContainerConfigError)
# CHA reports: SecretKeyMissing analyzer
# CHA auto-fix: NONE — requires Vault/ESO update
# Manual fix:   demo/fix-scripts/fix-missing-secret-key.sh
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/02-missing-secret-key.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"
DEPLOYMENT="${2:-api-server}"
SECRET_NAME="database-credentials"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Simulating missing secret key in namespace: $NAMESPACE"
echo ""

echo "Step 1: Stripping DB_PASSWORD from Secret ${SECRET_NAME}..."
$KUBECTL create secret generic "${SECRET_NAME}" \
  --from-literal=DB_HOST=postgres.demo-app.svc.cluster.local \
  --from-literal=DB_NAME=appdb \
  -n "${NAMESPACE}" \
  --dry-run=client -o yaml | $KUBECTL apply -f -

echo ""
echo "Step 2: Creating Deployment that references the missing key..."
$KUBECTL apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${DEPLOYMENT}
  namespace: ${NAMESPACE}
  labels:
    demo: missing-secret-key
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${DEPLOYMENT}
  template:
    metadata:
      labels:
        app: ${DEPLOYMENT}
    spec:
      containers:
      - name: app
        image: busybox:1.36
        command: ["sh", "-c", "echo \$DB_PASSWORD && sleep 3600"]
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: ${SECRET_NAME}
              key: DB_PASSWORD
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: ${SECRET_NAME}
              key: DB_HOST
EOF

echo ""
echo "==> Waiting for pod to reach CreateContainerConfigError..."
sleep 8
$KUBECTL get pods -n "${NAMESPACE}" -l app="${DEPLOYMENT}" --no-headers
echo ""
echo "==> CHA SecretKeyMissing analyzer will detect this within ~10-30s."
echo "    Check Slack #aws-alerts for: 'Secret missing key DB_PASSWORD'"
echo ""
echo "    To watch pod events:"
echo "    ${KUBECTL} describe pod -n ${NAMESPACE} -l app=${DEPLOYMENT} | grep -A5 Events"
echo ""
echo "    To fix: run demo/fix-scripts/fix-missing-secret-key.sh ${NAMESPACE} ${SECRET_NAME} DB_PASSWORD <value>"
