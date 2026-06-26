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
kubectl -n agentic-sre get events --field-selector reason=AIEnrichmentFailed --sort-by=lastTimestamp | tail -5
```

**Remediation**: No action required from Srenix's side. Deterministic
diagnostics continue to flow. Once the LLM endpoint returns, the next
watcher cycle picks up enrichment without restart.

If the LLM is permanently down, downshift to OSS behavior:
```sh
helm upgrade srenix srenix/agentic-sre --reuse-values --set ai.enabled=false
```

---

## Scenario 2 — Circuit breaker tripped

**Symptom**: `ai.circuit_breaker.tripped` Warning Event; Alertmanager
pages oncall; no new Apply Fix buttons emitted.

**Check**:
```sh
# Find the cause — look for 3+ recent action.failed events
kubectl -n agentic-sre get events --field-selector reason=AIActionFailed --sort-by=lastTimestamp | tail -5
```

**Remediation**: Investigate the failures. Common causes:
- A specific fixer is encountering an admission policy update
- The LLM is hallucinating targets that don't exist
- Network issue between approval-server and kube-apiserver

After diagnosing, manually reset:
```sh
kubectl -n agentic-sre port-forward svc/srenix-agentic-sre-approval-server 8443:8443
curl -X POST -H "X-Forwarded-User: $YOU" http://localhost:8443/admin/reset
```

---

## Scenario 3 — Approver out of office (T3)

**Symptom**: A T3 runbook posted; first approver clicked; second
approver not available.

**Behavior**: The runbook expires 90 minutes after creation (not after the first approval click). The next watcher cycle will generate a fresh
runbook if the underlying diagnostic remains.

**Remediation**: Update the `srenix.io/approver` group to include
additional members:
```sh
kubectl get rolebinding -n agentic-sre srenix-approvers -o yaml | \
  yq '.subjects += [{kind: "User", name: "bob@example.com"}]' | \
  kubectl apply -f -
```

If a specific runbook must be approved urgently (cannot wait for re-generation), the first approver can either:
1. Wait for a different second approver to come online, then click their slot
2. Mark the diagnostic resolved manually via `kubectl annotate driftreport`

---

## Scenario 4 — Rate limit hit

**Symptom**: `ai.rate_limited` Events appearing; some diagnostics
get `🤖` enrichment, others don't.

**Check**:
```sh
kubectl -n agentic-sre get events --field-selector reason=AIRateLimited \
  -o json | jq '.items[].metadata.annotations."srenix.ai/audit-details"' | tail -5
```

**Remediation**: If sustained, raise the budget:
```sh
helm upgrade srenix srenix/agentic-sre --reuse-values \
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
kubectl -n agentic-sre get events --field-selector reason=AIProposalInvalid \
  -o json | jq '.items[].metadata.annotations'
```

**Remediation**: The validator already rejected the proposal — no
cluster impact. But this is a signal to:
1. Audit the LLM endpoint logs for the matching `prompt_hash`
2. Verify the LLM provider's data-retention policy if SaaS
3. Consider tightening the system prompt (file a Srenix Enterprise support
   ticket with the prompt_hash so we can iterate)

If the same source diagnostic keeps producing invalid proposals,
add the analyzer's name to a temporary FixProposer exclusion list:
```sh
helm upgrade srenix srenix/agentic-sre --reuse-values \
  --set 'ai.fixProposer.excludeSources={TLSSecretMismatch}'
```

---

## Scenario 6 — Approval-server pod fails to start

**Symptom**: `kubectl get pod -l srenix.ai/role=approval-server`
shows ImagePullBackOff or CrashLoopBackOff.

**Common causes**:
- Image pull failure → check `imagePullSecrets`
- Signing key Secret missing → re-run the pre-install hook:
  ```sh
  kubectl -n agentic-sre delete job srenix-agentic-sre-approval-keygen 2>/dev/null
  helm upgrade srenix srenix/agentic-sre --reuse-values  # re-runs the hook
  ```
- Listening port already bound → check container args; default is `:8443`

---

## Scenario 7 — Layer-2 Investigator misbehaves or is noisy

**Symptom**: `🔬` blocks appear under findings in Slack with summaries
that look wrong, stale, or repeatedly say `insufficient_data`; or you
need to compare investigator output against the bare deterministic
diagnostic for a postmortem.

**Check** (OSS rule-based, no audit events by design — read the
DriftReport CR or Slack thread directly):
```sh
# What the investigator concluded on a specific report
kubectl -n agentic-sre get driftreport <name> \
  -o jsonpath='{.spec.investigation}' && echo

# Cycle-wide view — how many findings carry investigation summaries
kubectl -n agentic-sre get driftreports.srenix.ai \
  -o json | jq '[.items[] | select(.spec.investigation != "")] | length'
```

**Check** (paid LLM-backed investigator, which emits audit events):
```sh
kubectl -n agentic-sre get events --sort-by=lastTimestamp \
  | grep -E "AIInvestigator(Started|ToolCall|Completed|BudgetExceeded)"
```

**Remediation — disable the investigator entirely**:
```sh
kubectl -n agentic-sre set env deployment/srenix-agentic-sre \
  SRENIX_INVESTIGATOR=off
```
Restart picks up the new env. DriftReports continue to be emitted with
`spec.investigation` empty; Slack/AM rendering drops the `🔬` block.

**Remediation — override with the paid LLM-backed implementation**:
This is automatic when Srenix Enterprise is installed and `ai.enabled=true`. The
paid binary's catalog runs after `catalog.RegisterOSS` and re-calls
`RegisterInvestigator` to replace the rule-based one. No env-var change
required; downshift via `ai.enabled=false` reverts to the rule-based
investigator.

**When the investigator is correct but the finding is the real problem**:
investigation is additive — the underlying Finding / Diagnostic is
unchanged. Fixers, T1 buttons, and alerting all behave identically with
or without the `🔬` block.

---

## Scenario 8a — Firecrawl deep-RCA not enriching investigations

**Context**: Srenix Enterprise v0.2.0-alpha.1 paid binary; `ai.enabled=true`; `--firecrawl-enabled`
defaults to `true` but is inert without the API key.

**Symptom**: Investigation `🔬` blocks appear but contain no web citations or
external root-cause context; `ai.investigator.tool_call` events show no
`details.tool=firecrawl` entries.

**Check**:
```sh
# Confirm the key env is set on the aiwatch pod
kubectl -n agentic-sre exec deploy/bionic-aiwatch -- \
  sh -c 'echo ${FIRECRAWL_API_KEY:0:4}...'

# Check ESO sync
kubectl -n agentic-sre get externalsecret srenix-firecrawl-key
```

**Remediation — provision the ESO ExternalSecret**:
```yaml
# Vault path: secret/data/shared/api-keys  key: firecrawl_api_key
# Produces: K8s Secret srenix-firecrawl-key  key: FIRECRAWL_API_KEY
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: srenix-firecrawl-key
  namespace: agentic-sre
spec:
  refreshInterval: 1h
  secretStoreRef: { name: vault-backend, kind: ClusterSecretStore }
  target: { name: srenix-firecrawl-key, creationPolicy: Owner }
  data:
    - secretKey: FIRECRAWL_API_KEY
      remoteRef: { key: shared/api-keys, property: firecrawl_api_key }
```
Then patch the aiwatch Deployment to mount the secret as an env var:
```sh
kubectl -n agentic-sre set env deploy/bionic-aiwatch \
  --from=secret/srenix-firecrawl-key
```

**Flags reference**:
| Flag | Default | Notes |
|---|---|---|
| `--firecrawl-endpoint` | `https://api.firecrawl.dev` | Override for self-hosted Firecrawl |
| `--firecrawl-enabled` | `true` | Set `false` to disable entirely |
| `--firecrawl-api-key-env` | `FIRECRAWL_API_KEY` | Name of the env var holding the key |
| `--investigator-web-timeout` | `8s` | Wall-clock cap per Firecrawl request |

---

## Scenario 8b — RAG short-circuit replaying stale fixes

**Symptom**: T1 proposals are being applied without a fresh LLM proposal;
proposals reference an older fix that no longer applies to the current state.

**Background**: `--rag-short-circuit` now defaults **on**. When a prior
cleared fix scores ≥ `--rag-short-circuit-threshold` (default `0.92`) cosine
similarity, the LLM proposal call is skipped and the prior fix is replayed.
Replayed fixes still pass the G6 precondition re-check.

**Check**:
```sh
# Look for replay events (rag_short_circuit=true in proposal details)
kubectl -n agentic-sre get events \
  --field-selector reason=AIProposalCreated -o json | \
  jq '.items[] | select(.metadata.annotations."srenix.ai/audit-details"
      | contains("rag_short_circuit"))'
```

**Remediation** — lower the threshold or disable via the operator CR `spec.ai.extraArgs`:

Srenix Enterprise flags are passed through the `AgenticSRE` CR, not Helm values.
Edit the CR directly:

```sh
kubectl edit agenticsre bionic -n agentic-sre
```

Then adjust `spec.ai.extraArgs`. Examples:

```yaml
spec:
  ai:
    extraArgs:
      # ... existing args such as --rag-store-url, --autonomy=true, etc. ...

      # Require a tighter similarity match (default is 0.92):
      - --rag-short-circuit-threshold=0.97

      # To disable the short-circuit entirely, add:
      - --rag-short-circuit=false
      # (omitting the flag leaves it ON, which is the default)
```

Save and exit; the operator will roll out a new aiwatch pod with the updated flags.
Confirm the new flag is active:
```sh
kubectl -n agentic-sre exec deploy/bionic-aiwatch -- \
  srenix-enterprise --help 2>&1 | grep rag-short-circuit
```

---

## Scenario 8 — Click-to-fix URL gives 401 Unauthorized

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
