#!/usr/bin/env bash
# Manual fix: Certificate not renewing (cert-manager)
# Srenix auto-fixes: Deletes stuck CertificateRequests/Orders (allows retry)
# This script handles Issuer-level problems Srenix cannot fix automatically
#
# Usage: ./fix-certificate-renewal.sh <namespace> <certificate-name>
# Example: ./fix-certificate-renewal.sh livekit livekit-api-cert
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash fix-scripts/fix-certificate-renewal.sh ...

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:?Usage: $0 <namespace> <certificate-name>}"
CERT_NAME="${2:?provide certificate name}"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Diagnosing certificate renewal failure"
echo "    Namespace:   $NAMESPACE"
echo "    Certificate: $CERT_NAME"
echo ""

echo "--- Certificate status:"
$KUBECTL describe certificate "${CERT_NAME}" -n "${NAMESPACE}" | grep -A5 "Conditions:" || true
echo ""

echo "--- Related CertificateRequests:"
$KUBECTL get certificaterequest -n "${NAMESPACE}" --no-headers 2>/dev/null | head -10 || \
  echo "    (none found or CRD not installed)"
echo ""

echo "--- Fixing: Delete failed CertificateRequests to trigger cert-manager retry..."
FAILED_CRS=$($KUBECTL get certificaterequest -n "${NAMESPACE}" \
  -o jsonpath='{range .items[?(@.status.conditions[?(@.type=="Ready")].reason=="Failed")]}{.metadata.name}{" "}{end}' 2>/dev/null || true)

if [[ -n "$FAILED_CRS" ]]; then
  for cr in $FAILED_CRS; do
    echo "    Deleting failed CertificateRequest: $cr"
    $KUBECTL delete certificaterequest "${cr}" -n "${NAMESPACE}"
  done
  echo "    ✓ cert-manager will immediately recreate and retry issuance"
else
  echo "    No failed CertificateRequests found (Srenix may have already cleaned them up)"
fi

echo ""
echo "--- Fixing: Delete errored ACME Orders..."
FAILED_ORDERS=$($KUBECTL get order -n "${NAMESPACE}" \
  -o jsonpath='{range .items[?(@.status.state=="errored")]}{.metadata.name}{" "}{end}' 2>/dev/null || true)

if [[ -n "$FAILED_ORDERS" ]]; then
  for order in $FAILED_ORDERS; do
    echo "    Deleting errored Order: $order"
    $KUBECTL delete order "${order}" -n "${NAMESPACE}"
  done
else
  echo "    No errored Orders found"
fi

echo ""
echo "--- Verifying Issuer/ClusterIssuer is Ready..."
ISSUER_REF=$($KUBECTL get certificate "${CERT_NAME}" -n "${NAMESPACE}" \
  -o jsonpath='{.spec.issuerRef.name}' 2>/dev/null || echo "unknown")
ISSUER_KIND=$($KUBECTL get certificate "${CERT_NAME}" -n "${NAMESPACE}" \
  -o jsonpath='{.spec.issuerRef.kind}' 2>/dev/null || echo "ClusterIssuer")

if [[ "$ISSUER_KIND" == "ClusterIssuer" ]]; then
  echo "    ClusterIssuer: ${ISSUER_REF}"
  $KUBECTL get clusterissuer "${ISSUER_REF}" \
    -o jsonpath='    Status: {.status.conditions[?(@.type=="Ready")].status} — {.status.conditions[?(@.type=="Ready")].message}{"\n"}' 2>/dev/null || true
else
  echo "    Issuer: ${NAMESPACE}/${ISSUER_REF}"
  $KUBECTL get issuer "${ISSUER_REF}" -n "${NAMESPACE}" \
    -o jsonpath='    Status: {.status.conditions[?(@.type=="Ready")].status} — {.status.conditions[?(@.type=="Ready")].message}{"\n"}' 2>/dev/null || true
fi

echo ""
echo "==> Monitoring renewal progress:"
echo "    ${KUBECTL} describe certificate ${CERT_NAME} -n ${NAMESPACE}"
echo "    ${KUBECTL} get certificaterequest -n ${NAMESPACE} -w"
