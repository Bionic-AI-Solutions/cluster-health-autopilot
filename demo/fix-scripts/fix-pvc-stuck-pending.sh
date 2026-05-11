#!/usr/bin/env bash
# Manual fix: PVC stuck in Pending state
# CHA reports this (PVC probe) but cannot fix it — storage provisioner issue
#
# Usage: ./fix-pvc-stuck-pending.sh <namespace> <pvc-name>
# Example: ./fix-pvc-stuck-pending.sh demo-app data-volume
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash fix-scripts/fix-pvc-stuck-pending.sh ...

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:?Usage: $0 <namespace> <pvc-name>}"
PVC_NAME="${2:?provide PVC name}"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Diagnosing stuck PVC"
echo "    Namespace: $NAMESPACE"
echo "    PVC:       $PVC_NAME"
echo ""

echo "--- PVC status:"
$KUBECTL get pvc "${PVC_NAME}" -n "${NAMESPACE}" -o wide 2>/dev/null || {
  echo "  ⚠ PVC ${PVC_NAME} not found in namespace ${NAMESPACE}"
  echo "  Use: ${KUBECTL} get pvc -n ${NAMESPACE} to list available PVCs"
  exit 0
}
echo ""

echo "--- PVC events:"
$KUBECTL describe pvc "${PVC_NAME}" -n "${NAMESPACE}" | grep -A20 "Events:" || true
echo ""

STORAGE_CLASS=$($KUBECTL get pvc "${PVC_NAME}" -n "${NAMESPACE}" \
  -o jsonpath='{.spec.storageClassName}' 2>/dev/null || echo "unknown")
echo "--- StorageClass: ${STORAGE_CLASS}"
$KUBECTL get storageclass "${STORAGE_CLASS}" 2>/dev/null || \
  echo "    ⚠ StorageClass ${STORAGE_CLASS} not found!"
echo ""

echo "--- CSI provisioner pods:"
$KUBECTL get pods -n rook-ceph -l app=rook-ceph-provisioner --no-headers 2>/dev/null || \
$KUBECTL get pods -n kube-system -l app.kubernetes.io/name=aws-ebs-csi-driver --no-headers 2>/dev/null || \
  echo "    (check your CSI driver namespace)"
echo ""

echo "--- Common fixes:"
echo ""
echo "1. Wrong StorageClass for node zone (EKS with topology):"
echo "   Check: ${KUBECTL} get pvc ${PVC_NAME} -n ${NAMESPACE} -o jsonpath='{.spec.selector}'"
echo "   Fix:   Ensure StorageClass has volumeBindingMode: WaitForFirstConsumer"
echo ""
echo "2. Rook-Ceph not provisioning:"
echo "   Check: ${KUBECTL} logs -n rook-ceph -l app=rook-ceph-provisioner --tail=50 | grep ${PVC_NAME}"
echo "   Fix:   ${KUBECTL} rollout restart deployment/rook-ceph-operator -n rook-ceph"
echo ""
echo "3. EKS EBS CSI: IAM permissions missing:"
echo "   Check: ${KUBECTL} describe pvc ${PVC_NAME} -n ${NAMESPACE} | grep 'Error'"
echo "   Fix:   Attach AmazonEBSCSIDriverPolicy to node IAM role or IRSA role"
echo ""
echo "4. PVC stuck after node zone mismatch — recreate:"
echo "   ${KUBECTL} delete pvc ${PVC_NAME} -n ${NAMESPACE}"
echo "   # Then recreate with correct storageClassName or node affinity"
