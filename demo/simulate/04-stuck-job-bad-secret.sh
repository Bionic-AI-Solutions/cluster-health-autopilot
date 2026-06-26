#!/usr/bin/env bash
# Simulate: Stuck Job with bad SecretRef (CronJob concurrencyPolicy=Forbid deadlock)
#
# SCENARIO (mirrors the real gpu-docker-monitor incident):
#   A Secret key was renamed from API_KEY → API_KEY_NEW.
#   The CronJob template still references the OLD key (API_KEY).
#   Every Job spawned has a pod stuck in CreateContainerConfigError.
#   concurrencyPolicy=Forbid means the CronJob can't run a fresh job.
#
#   THIS SCRIPT:
#   1. Creates the broken state (Secret has API_KEY_NEW, CronJob wants API_KEY → stuck pod)
#   2. Fixes the CronJob template to reference API_KEY_NEW (root cause resolved)
#   3. But the old stuck Job is still blocking (Forbid policy)
#   4. Srenix StuckJobsWithBadSecretRef fixer deletes the frozen Job
#   5. CronJob next tick spawns a new Job → succeeds ✓
#
# Set KUBE_CONTEXT to target a specific cluster:
#   KUBE_CONTEXT=arn:aws:eks:ap-south-1:123456789:cluster/test-cluster1 bash simulate/04-stuck-job-bad-secret.sh

set -euo pipefail

KUBECTL="kubectl${KUBE_CONTEXT:+ --context ${KUBE_CONTEXT}}"
NAMESPACE="${1:-demo-app}"
CRONJOB_NAME="demo-data-sync"
SECRET_NAME="sync-credentials"

echo "==> Cluster context: ${KUBE_CONTEXT:-$(kubectl config current-context 2>/dev/null)}"
echo "==> Simulating stuck CronJob with stale SecretRef in namespace: $NAMESPACE"
echo ""

# Step 1: Secret already has the NEW key name — old key is gone
echo "Step 1: Create Secret with NEW key name 'API_KEY_NEW' (old 'API_KEY' does not exist)..."
$KUBECTL create secret generic "${SECRET_NAME}" \
  --from-literal=API_KEY_NEW=working-secret-value \
  -n "${NAMESPACE}" \
  --dry-run=client -o yaml | $KUBECTL apply -f -

# Step 2: CronJob template still references the OLD key — this is the stale ref
echo "Step 2: Create CronJob still referencing stale key 'API_KEY'..."
$KUBECTL apply -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ${CRONJOB_NAME}
  namespace: ${NAMESPACE}
  labels:
    demo: stuck-job
spec:
  schedule: "*/2 * * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: sync
            image: busybox:1.36
            command: ["sh", "-c", "echo sync token=\$API_TOKEN && sleep 5"]
            env:
            - name: API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: ${SECRET_NAME}
                  key: API_KEY
EOF

# Step 3: Spawn a job from the stale template → pod immediately fails (key not found)
echo "Step 3: Trigger a Job from the stale template → pod enters CreateContainerConfigError..."
STUCK_JOB="${CRONJOB_NAME}-stuck-$(date +%s | tail -c 5)"
$KUBECTL create job "${STUCK_JOB}" --from=cronjob/"${CRONJOB_NAME}" -n "${NAMESPACE}"

echo ""
echo "    Waiting for pod to enter CreateContainerConfigError..."
for i in $(seq 1 15); do
  sleep 2
  STATUS=$($KUBECTL get pods -n "${NAMESPACE}" -l "job-name=${STUCK_JOB}" \
    -o jsonpath='{.items[0].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || echo "")
  echo "    [${i}] Pod status: ${STATUS:-Pending}"
  if [[ "$STATUS" == "CreateContainerConfigError" ]]; then
    break
  fi
done
echo ""

# Step 4: Fix the root cause — update CronJob template to the correct key
echo "Step 4: Root cause fixed — patching CronJob template to use correct key 'API_KEY_NEW'..."
$KUBECTL patch cronjob "${CRONJOB_NAME}" -n "${NAMESPACE}" \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/jobTemplate/spec/template/spec/containers/0/env/0/valueFrom/secretKeyRef/key","value":"API_KEY_NEW"}]'

echo "        CronJob template is now correct. Secret has 'API_KEY_NEW'. Root cause resolved."
echo ""

# Show the blocking state
echo "==> Current state:"
$KUBECTL get jobs -n "${NAMESPACE}" --no-headers 2>/dev/null
$KUBECTL get pods -n "${NAMESPACE}" -l "job-name=${STUCK_JOB}" --no-headers 2>/dev/null
echo ""
echo "    OLD Job '${STUCK_JOB}' is ACTIVE (pod in CreateContainerConfigError)"
echo "    CronJob is correct NOW but cannot spawn a new Job (concurrencyPolicy=Forbid)"
echo ""
echo "==> Srenix StuckJobsWithBadSecretRef fixer will:"
echo "    1. Find the pod with 'couldn't find key API_KEY' in waiting message"
echo "    2. Confirm its parent Job '${STUCK_JOB}' has a CronJob owner"
echo "    3. Delete Job '${STUCK_JOB}'"
echo "    4. CronJob's next tick (≤2 min) spawns a fresh Job using the correct key → ✓"
echo ""
echo "    Watch the Job lifecycle:"
echo "    ${KUBECTL} get jobs -n ${NAMESPACE} -w"
