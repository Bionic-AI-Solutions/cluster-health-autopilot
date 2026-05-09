#!/usr/bin/env bash
# Manual fix: ImagePullBackOff auth failure
# CHA reports this but cannot fix it — imagePullSecret must be created/patched
#
# Usage: ./fix-image-pull-secret.sh <namespace> <deployment-or-pod> [registry] [username] [password]
# Example: ./fix-image-pull-secret.sh demo-app worker docker4zerocool myuser mypassword
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash fix-scripts/fix-image-pull-secret.sh ...

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:?Usage: $0 <namespace> <deployment-or-pod> [registry] [username] [password]}"
TARGET="${2:?provide deployment or pod name}"
REGISTRY="${3:-docker.io}"
USERNAME="${4:-}"
PASSWORD="${5:-}"
SECRET_NAME="registry-pull-secret"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Fixing ImagePull auth failure"
echo "    Namespace:  $NAMESPACE"
echo "    Target:     $TARGET"
echo "    Registry:   $REGISTRY"
echo ""

if [[ -z "$USERNAME" || -z "$PASSWORD" ]]; then
  echo "--- Reading Docker Hub credentials from local ~/.docker/config.json..."
  if [[ -f "$HOME/.docker/config.json" ]]; then
    $KUBECTL create secret generic "${SECRET_NAME}" \
      --from-file=.dockerconfigjson="$HOME/.docker/config.json" \
      --type=kubernetes.io/dockerconfigjson \
      -n "${NAMESPACE}" \
      --dry-run=client -o yaml | $KUBECTL apply -f -
  else
    echo "⚠ No local Docker config and no credentials provided."
    echo "  Please provide: $0 ${NAMESPACE} ${TARGET} ${REGISTRY} <username> <password>"
    exit 1
  fi
else
  echo "Step 1: Creating imagePullSecret from provided credentials..."
  $KUBECTL create secret docker-registry "${SECRET_NAME}" \
    --docker-server="${REGISTRY}" \
    --docker-username="${USERNAME}" \
    --docker-password="${PASSWORD}" \
    -n "${NAMESPACE}" \
    --dry-run=client -o yaml | $KUBECTL apply -f -
fi

echo ""
echo "Step 2: Checking if ${TARGET} is a Deployment or Pod..."
if $KUBECTL get deployment "${TARGET}" -n "${NAMESPACE}" &>/dev/null; then
  echo "    Found Deployment ${TARGET}. Patching imagePullSecrets..."
  $KUBECTL patch deployment "${TARGET}" -n "${NAMESPACE}" \
    --type='json' \
    -p="[{\"op\":\"add\",\"path\":\"/spec/template/spec/imagePullSecrets\",\"value\":[{\"name\":\"${SECRET_NAME}\"}]}]"
  echo ""
  echo "Step 3: Rolling out..."
  $KUBECTL rollout restart deployment/"${TARGET}" -n "${NAMESPACE}"
  $KUBECTL rollout status deployment/"${TARGET}" -n "${NAMESPACE}" --timeout=120s
else
  echo "    Not a Deployment — patching default ServiceAccount and deleting pod..."
  $KUBECTL patch serviceaccount default -n "${NAMESPACE}" \
    -p "{\"imagePullSecrets\": [{\"name\": \"${SECRET_NAME}\"}]}" 2>/dev/null || true
  $KUBECTL delete pod "${TARGET}" -n "${NAMESPACE}" --ignore-not-found 2>/dev/null || true
  echo "    Pod deleted — kubelet will recreate it with the new imagePullSecret."
fi

echo ""
echo "==> Done. CHA watcher will detect the recovery and post ✅ Resolved to Slack."
