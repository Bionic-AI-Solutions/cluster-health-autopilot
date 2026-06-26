#!/usr/bin/env bash
# Cleanup all demo-injected resources
# Run this after the demo to reset the cluster to a clean state
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/06-cleanup-all.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Cleaning up demo resources in namespace: $NAMESPACE"
echo ""

echo "--- Deleting demo pods (stale-error, auth-fail)..."
$KUBECTL delete pods -n "${NAMESPACE}" -l demo=stale-error-pod --ignore-not-found
$KUBECTL delete pods -n "${NAMESPACE}" -l demo=image-pull-auth --ignore-not-found

echo "--- Deleting demo deployments..."
$KUBECTL delete deployment api-server worker -n "${NAMESPACE}" --ignore-not-found

echo "--- Deleting demo CronJob and related Jobs..."
$KUBECTL delete cronjob demo-data-sync -n "${NAMESPACE}" --ignore-not-found
$KUBECTL delete jobs -n "${NAMESPACE}" -l demo=stuck-job --ignore-not-found

echo "--- Deleting demo ExternalSecret (if ESO is installed)..."
$KUBECTL delete externalsecret demo-externalsecret -n "${NAMESPACE}" --ignore-not-found 2>/dev/null || true

echo "--- Deleting demo Secrets..."
$KUBECTL delete secret database-credentials sync-credentials demo-broken-secret -n "${NAMESPACE}" --ignore-not-found

echo ""
echo "==> Cleanup complete."
echo "    Srenix watcher will detect resolved issues on next cycle (~10s)"
echo "    and post ✅ Resolved messages to Slack #aws-alerts."
