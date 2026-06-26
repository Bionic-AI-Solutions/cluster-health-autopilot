#!/usr/bin/env bash
# Manual fix: Failing ExternalSecret (ESO Ready=False)
# Srenix reports this but cannot fix it — Vault data must be correct first
#
# Usage: ./fix-failing-externalsecret.sh <namespace> <eso-name> [vault-path] [property] [value]
# Example: ./fix-failing-externalsecret.sh demo-app database-credentials \
#            secret/t6-apps/demo-app/config DB_PASSWORD 'mysecretpw'
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash fix-scripts/fix-failing-externalsecret.sh ...

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:?Usage: $0 <namespace> <eso-name> [vault-path] [property] [value]}"
ESO_NAME="${2:?provide ESO name}"
VAULT_PATH="${3:-}"
PROPERTY="${4:-}"
VALUE="${5:-}"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Fixing Failing ExternalSecret"
echo "    Namespace: $NAMESPACE"
echo "    ESO:       $ESO_NAME"
echo ""

echo "--- Current ESO status:"
$KUBECTL get externalsecret "${ESO_NAME}" -n "${NAMESPACE}" \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}{" — "}{.status.conditions[?(@.type=="Ready")].message}{"\n"}' 2>/dev/null || \
  $KUBECTL get externalsecret "${ESO_NAME}" -n "${NAMESPACE}" --no-headers
echo ""

if [[ -n "$VAULT_PATH" && -n "$PROPERTY" && -n "$VALUE" ]]; then
  echo "Step 1: Writing missing property to Vault..."
  echo "    vault kv patch ${VAULT_PATH} ${PROPERTY}=${VALUE}"
  if command -v vault &>/dev/null && [[ -n "${VAULT_ADDR:-}" ]]; then
    vault kv patch "${VAULT_PATH}" "${PROPERTY}=${VALUE}"
    echo "    ✓ Vault updated"
  else
    echo "    ⚠ vault CLI not available or VAULT_ADDR not set."
    echo "    Manually run: vault kv patch ${VAULT_PATH} ${PROPERTY}=${VALUE}"
    echo "    Then re-run this script without the vault args to force ESO re-sync."
  fi
  echo ""
fi

echo "Step 2: Force ExternalSecret re-sync..."
$KUBECTL annotate externalsecret "${ESO_NAME}" -n "${NAMESPACE}" \
  force-sync="$(date +%s)" --overwrite

echo ""
echo "Step 3: Waiting for ESO to reconcile (up to 60s)..."
for i in $(seq 1 12); do
  sleep 5
  STATUS=$($KUBECTL get externalsecret "${ESO_NAME}" -n "${NAMESPACE}" \
    -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
  echo "    [${i}] Ready: ${STATUS}"
  if [[ "$STATUS" == "True" ]]; then
    echo ""
    echo "==> ExternalSecret is now Ready ✓"
    echo "    Srenix watcher will post ✅ Resolved to Slack on next cycle."
    exit 0
  fi
done

echo ""
echo "==> ESO not Ready after 60s. Check:"
echo "    ${KUBECTL} describe externalsecret ${ESO_NAME} -n ${NAMESPACE}"
echo "    ${KUBECTL} logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=50"
