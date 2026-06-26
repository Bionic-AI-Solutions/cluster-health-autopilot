#!/usr/bin/env bash
# Manual fix: Missing Secret key
# Srenix reports this but cannot fix it — requires Vault + ESO update
#
# Usage: ./fix-missing-secret-key.sh <namespace> <secret-name> <key> <value>
# Example: ./fix-missing-secret-key.sh demo-app database-credentials DB_PASSWORD 'mysecretpw'
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash fix-scripts/fix-missing-secret-key.sh ...

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:?Usage: $0 <namespace> <secret-name> <key> <value>}"
SECRET_NAME="${2:?provide secret name}"
KEY="${3:?provide key name}"
VALUE="${4:?provide value}"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Fixing missing Secret key"
echo "    Namespace:   $NAMESPACE"
echo "    Secret:      $SECRET_NAME"
echo "    Missing key: $KEY"
echo ""

echo "Step 1: Patching Secret ${SECRET_NAME} to add missing key ${KEY}..."
ENCODED=$(echo -n "${VALUE}" | base64)
$KUBECTL patch secret "${SECRET_NAME}" -n "${NAMESPACE}" \
  --type='json' \
  -p="[{\"op\": \"add\", \"path\": \"/data/${KEY}\", \"value\": \"${ENCODED}\"}]"

echo ""
echo "Step 2: Restarting pods consuming this Secret..."
DEPLOYMENTS=$($KUBECTL get deployments -n "${NAMESPACE}" -o json | \
  python3 -c "
import sys, json
data = json.load(sys.stdin)
secret = '${SECRET_NAME}'
for item in data.get('items', []):
  containers = item.get('spec',{}).get('template',{}).get('spec',{}).get('containers',[])
  for c in containers:
    for env in c.get('env', []):
      ref = env.get('valueFrom', {}).get('secretKeyRef', {})
      if ref.get('name') == secret:
        print(item['metadata']['name'])
        break
" 2>/dev/null || true)

if [[ -n "$DEPLOYMENTS" ]]; then
  for dep in $DEPLOYMENTS; do
    echo "    Restarting Deployment: $dep"
    $KUBECTL rollout restart deployment/"${dep}" -n "${NAMESPACE}"
  done
else
  echo "    No Deployments found referencing ${SECRET_NAME} — pods may restart on their own."
fi

echo ""
echo "Step 3: Verifying Secret keys..."
$KUBECTL get secret "${SECRET_NAME}" -n "${NAMESPACE}" -o jsonpath='{.data}' 2>/dev/null | \
  python3 -c "import sys,json; d=json.load(sys.stdin); [print(f'  {k}: [set]') for k in sorted(d)]" 2>/dev/null || \
  $KUBECTL get secret "${SECRET_NAME}" -n "${NAMESPACE}" --no-headers

echo ""
echo "==> Done. Srenix watcher will detect the recovery and post ✅ Resolved to Slack."
echo ""
echo "NOTE: In production, the correct fix is:"
echo "  1. Add the key to Vault:  vault kv patch secret/t6-apps/<app>/config ${KEY}=<value>"
echo "  2. ESO auto-syncs the Secret within its refreshInterval"
echo "  3. Pod restarts automatically when Secret is updated"
