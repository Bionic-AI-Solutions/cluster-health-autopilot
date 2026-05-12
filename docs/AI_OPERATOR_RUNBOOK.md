# AI Operator Runbook

What to do when the AI tier misbehaves. Each scenario maps to a
diagnostic check + an explicit remediation step.

**Companion docs**: [AI_TIERS.md](AI_TIERS.md), [SETUP_GUIDE.md §14](SETUP_GUIDE.md#14-ai-tier-setup), [AI_AUDIT_TRAIL.md](AI_AUDIT_TRAIL.md)

---

## Scenario 1 — LLM endpoint is down

**Symptom**: `🤖` enrichment blocks stop appearing in Slack/AM; no
Apply Fix buttons emitted.

**Check**:
```sh
kubectl -n cluster-health-autopilot get events --field-selector reason=AIEnrichmentFailed --sort-by=lastTimestamp | tail -5
```

**Remediation**: No action required from CHA's side. Deterministic
diagnostics continue to flow. Once the LLM endpoint returns, the next
watcher cycle picks up enrichment without restart.

If the LLM is permanently down, downshift to OSS behavior:
```sh
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.enabled=false
```

---

## Scenario 2 — Circuit breaker tripped

**Symptom**: `ai.circuit_breaker.tripped` Warning Event; Alertmanager
pages oncall; no new Apply Fix buttons emitted.

**Check**:
```sh
# Find the cause — look for 3+ recent action.failed events
kubectl -n cluster-health-autopilot get events --field-selector reason=AIActionFailed --sort-by=lastTimestamp | tail -5
```

**Remediation**: Investigate the failures. Common causes:
- A specific fixer is encountering an admission policy update
- The LLM is hallucinating targets that don't exist
- Network issue between approval-server and kube-apiserver

After diagnosing, manually reset:
```sh
kubectl -n cluster-health-autopilot port-forward svc/cha-cluster-health-autopilot-approval-server 8443:8443
curl -X POST -H "X-Forwarded-User: $YOU" http://localhost:8443/admin/reset
```

---

## Scenario 3 — Approver out of office (T3)

**Symptom**: A T3 runbook posted; first approver clicked; second
approver not available.

**Behavior**: After 60 min from the first approval, the runbook
expires automatically. The next watcher cycle will generate a fresh
runbook if the underlying diagnostic remains.

**Remediation**: Update the `cha.io/approver` group to include
additional members:
```sh
kubectl get rolebinding -n cluster-health-autopilot cha-approvers -o yaml | \
  yq '.subjects += [{kind: "User", name: "bob@example.com"}]' | \
  kubectl apply -f -
```

If a specific runbook must be approved urgently (cannot wait 60 min
for re-generation), the first approver can either:
1. Wait for a different second approver to come online, then click their slot
2. Mark the diagnostic resolved manually via `kubectl annotate driftreport`

---

## Scenario 4 — Rate limit hit

**Symptom**: `ai.rate_limited` Events appearing; some diagnostics
get `🤖` enrichment, others don't.

**Check**:
```sh
kubectl -n cluster-health-autopilot get events --field-selector reason=AIRateLimited \
  -o json | jq '.items[].metadata.annotations."cha.bionicaisolutions.com/audit-details"' | tail -5
```

**Remediation**: If sustained, raise the budget:
```sh
helm upgrade cha cha/cluster-health-autopilot --reuse-values \
  --set ai.rateLimit.actionsPerHour=20
```

If the rate is spiky (incident-driven), the default is intentional —
the spike will subside as issues resolve.

---

## Scenario 5 — Suspicious LLM response

**Symptom**: An `ai.proposal.invalid` Event with `reason: ErrSecretValueInOutput`
or similar. The LLM tried to smuggle a value past the validator.

**Check**:
```sh
kubectl -n cluster-health-autopilot get events --field-selector reason=AIProposalInvalid \
  -o json | jq '.items[].metadata.annotations'
```

**Remediation**: The validator already rejected the proposal — no
cluster impact. But this is a signal to:
1. Audit the LLM endpoint logs for the matching `prompt_hash`
2. Verify the LLM provider's data-retention policy if SaaS
3. Consider tightening the system prompt (file a CHA-com support
   ticket with the prompt_hash so we can iterate)

If the same source diagnostic keeps producing invalid proposals,
add the analyzer's name to a temporary FixProposer exclusion list:
```sh
helm upgrade cha cha/cluster-health-autopilot --reuse-values \
  --set 'ai.fixProposer.excludeSources={IngressCoverage}'
```

---

## Scenario 6 — Approval-server pod fails to start

**Symptom**: `kubectl get pod -l cha.bionicaisolutions.com/role=approval-server`
shows ImagePullBackOff or CrashLoopBackOff.

**Common causes**:
- Image pull failure → check `imagePullSecrets`
- Signing key Secret missing → re-run the pre-install hook:
  ```sh
  kubectl -n cluster-health-autopilot delete job cha-cluster-health-autopilot-approval-keygen 2>/dev/null
  helm upgrade cha cha/cluster-health-autopilot --reuse-values  # re-runs the hook
  ```
- Listening port already bound → check container args; default is `:8443`

---

## Scenario 7 — Click-to-fix URL gives 401 Unauthorized

**Symptom**: SRE clicks the Apply Fix URL, sees:
> No approver identity. Configure OIDC at the Ingress layer.

**Cause**: The Ingress controller did not inject the
`X-Forwarded-User` header — typically OIDC is misconfigured or the
SRE isn't logged in.

**Remediation**: Ensure your oauth2-proxy / Kong key-auth setup adds
the `X-Forwarded-User` header to requests for `/approve*`. Example
for oauth2-proxy:
```yaml
oauth2_proxy_options:
  pass_user_headers: true
  set_xauthrequest: true
```
