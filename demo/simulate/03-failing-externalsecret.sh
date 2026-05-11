#!/usr/bin/env bash
# Simulate: Failing ExternalSecret (ESO Ready=False)
# CHA reports: FailingExternalSecrets analyzer
# CHA auto-fix: NONE — Vault data must be fixed first
# Manual fix:   demo/fix-scripts/fix-failing-externalsecret.sh
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/03-failing-externalsecret.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"
ESO_NAME="demo-externalsecret"
VAULT_PATH="secret/t6-apps/demo-app/broken-path"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Simulating a failing ExternalSecret in namespace: $NAMESPACE"
echo "    ESO: ${ESO_NAME}"
echo "    Vault path (intentionally wrong): ${VAULT_PATH}"
echo ""

$KUBECTL apply -f - <<EOF
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ${ESO_NAME}
  namespace: ${NAMESPACE}
  labels:
    demo: failing-eso
spec:
  refreshInterval: 30s
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: demo-broken-secret
    creationPolicy: Owner
  data:
  - secretKey: MISSING_KEY
    remoteRef:
      key: ${VAULT_PATH}
      property: nonexistent_property
EOF

echo ""
echo "==> ExternalSecret created with an invalid Vault path."
echo "    ESO controller will attempt sync and fail (Ready=False)."
echo ""
echo "    Watch ESO status:"
echo "    ${KUBECTL} get externalsecret ${ESO_NAME} -n ${NAMESPACE} -w"
echo ""
echo "==> CHA FailingExternalSecrets analyzer will detect this on next cycle."
echo "    Check Slack #aws-alerts for: 'ExternalSecret ${ESO_NAME} is not Ready'"
echo ""
echo "    To fix: run demo/fix-scripts/fix-failing-externalsecret.sh ${NAMESPACE} ${ESO_NAME}"
