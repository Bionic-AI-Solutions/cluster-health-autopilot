# Deployment Guide — OSS and Paid (Srenix Enterprise)

Agentic SRE installs from **one Helm chart**. The paid
(Srenix Enterprise AI) tier is **purely additive**: the exact same OSS install,
plus a single flag. There is no separate chart, no image-swap of the
core workloads, and no parallel install path to maintain.

> **TL;DR**
> - **OSS:** `helm install … agentic-sre`.
> - **Paid:** the same install, plus `--set ai.enabled=true` (and the
>   `ai.endpoint` / `ai.model` it requires). That stands up one extra
>   `aiwatch` Deployment running the commercial `srenix-enterprise` binary
>   alongside the untouched OSS workloads.

---

## 1. What runs in each mode

| Workload | OSS | Paid (`ai.enabled=true`) |
|---|---|---|
| `…-watcher` Deployment (event-driven probe → fix → Slack/ticketing/DriftReport) | ✅ OSS image | ✅ **OSS image, unchanged** |
| `…-diagnose` / `…-remediate` CronJobs | ✅ OSS image | ✅ **OSS image, unchanged** |
| `…-aiwatch` Deployment (`srenix-enterprise watch`, AI tiers t0→t3 on new diagnostics) | — | ✅ `docker4zerocool/srenix-enterprise` |
| `…-approval-server` (click-to-fix signing, **required for t1+**) | — | ✅ when `approval.enabled=true` |

**Key principle:** enabling the paid tier never modifies or replaces an
OSS workload. The commercial `srenix-enterprise watch` is the *AI-layered
counterpart* to the OSS watcher — it polls the same merged probe +
analyzer catalog on an interval and adds AI narration / fix proposals /
plans / Vault runbooks. The OSS watcher keeps owning the operational
loop. The two run side by side.

> Why not just swap the watcher image? Because `srenix-enterprise watch` and
> `srenix-enterprise diagnose` deliberately expose a *reduced* flag surface (only
> `--ai-*` / `--interval` / `--t3-vault-allowed-prefix`); they do **not**
> accept the OSS operational flags (`--live`, `--slack-*`, `--remedy`,
> `--ticketing-*`, `--cloud-*`). The additive model is the only correct
> deployment.

---

## 2. OSS install

```bash
helm repo add srenix https://srenix-ai.github.io/agentic-sre
helm repo update
helm install srenix srenix/agentic-sre \
  --namespace agentic-sre --create-namespace \
  --set watcher.enabled=true
```

Configure Slack, Alertmanager, ticketing, cloud probes, analyzers, and
M2 probe toggles via `values.yaml` as usual. Nothing AI-related is
required or rendered.

---

## 3. Paid (Srenix Enterprise AI) upgrade — additive

Prerequisites:
1. Access to the `docker4zerocool/srenix-enterprise` image (chart defaults to tag
   `v<chart appVersion>`, e.g. `v1.8.2`, which is the Srenix Enterprise release
   pinned to the same OSS engine — so the paid binary carries the full
   OSS detection surface plus the AI tiers).
2. An OpenAI-compatible LLM endpoint. In-cluster vLLM (BYOM) is the
   recommended posture; SaaS requires `ai.allowSaas=true`.
3. If the endpoint needs a key, provision it via **ESO** (never inline a
   secret — see §5).

### t0 (narration only) — simplest

```bash
helm upgrade srenix srenix/agentic-sre --reuse-values \
  --set ai.enabled=true \
  --set ai.tier=t0 \
  --set ai.endpoint=https://mcp.baisoln.com/gpu-ai/v1 \
  --set ai.model=qwen3.6-35b-a3b-fp8 \
  --set ai.apiKey.header=X-API-Key \
  --set ai.apiKey.secretName=srenix-ai-llm-key
```

### t1–t3 (proposals / plans / Vault runbooks) — add the approval server

t1 and above mint click-to-fix URLs, so the approval-server (which holds
the Ed25519 signing key and terminates the URLs) is required:

```bash
# 1. generate + persist the signing key (one-time)
#    the chart ships an approval-server-keygen-job that does this on install.

# 2. enable AI + approval together
helm upgrade srenix srenix/agentic-sre --reuse-values \
  --set ai.enabled=true \
  --set ai.tier=t3 \
  --set ai.endpoint=https://mcp.baisoln.com/gpu-ai/v1 \
  --set ai.model=qwen3.6-35b-a3b-fp8 \
  --set ai.apiKey.header=X-API-Key \
  --set ai.apiKey.secretName=srenix-ai-llm-key \
  --set ai.auditLog=- \
  --set 'ai.t3.vaultAllowedPrefixes={secret/data/srenix-recovery/}' \
  --set approval.enabled=true \
  --set approval.ingress.enabled=true \
  --set approval.ingress.host=srenix-approve.example.com
```

`ai.t3.vaultAllowedPrefixes` is the **blast-radius gate** for the t3
Vault-runbook proposer: the LLM may only *reference* paths under these
prefixes (and every proposal is still recommendation-only behind human
approval). Keep it as narrow as the recovery runbooks require.

---

## 4. Tier reference

| Tier | Adds | Approval server |
|---|---|---|
| t0 | LLM narration on new diagnostics | not needed |
| t1 | Single-step fix proposals (click-to-fix) | **required** |
| t2 | + multi-step remediation plans | **required** |
| t3 | + Vault break-glass recovery runbooks (`--t3-vault-allowed-prefix`) | **required** |

See [AI_TIERS.md](AI_TIERS.md) for the full tier specification and
[SETUP_GUIDE.md §14](SETUP_GUIDE.md#14-ai-tier-setup) for the AI-specific
walkthrough.

---

## 5. The LLM key — ESO only, never inlined

The bearer token for the LLM endpoint must flow Vault → External Secrets
Operator → K8s Secret. The chart references it by name; the value never
appears in `values.yaml`, a manifest, or a `--set`.

```yaml
# externalsecret-srenix-ai-llm-key.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: srenix-ai-llm-key
  namespace: agentic-sre
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: vault-backend
  target:
    name: srenix-ai-llm-key
    creationPolicy: Owner
  data:
    - secretKey: API_KEY          # matches ai.apiKey.secretKey
      remoteRef:
        key: <vault-kv-path>      # e.g. t6-apps/mcp/config
        property: <key-property>  # e.g. apikey_srenix_key
```

Then set `ai.apiKey.secretName=srenix-ai-llm-key`. For an unauthenticated
in-cluster vLLM (ClusterIP, no Kong), leave `ai.apiKey.secretName` empty.

---

## 6. Verify

```bash
kubectl -n agentic-sre get deploy
# expect: …-watcher (OSS, Running) AND …-aiwatch (srenix-enterprise, Running)
#         …-approval-server when approval.enabled=true

kubectl -n agentic-sre logs deploy/srenix-agentic-sre-aiwatch | head
# expect: poll-loop start, then AI-tier output on new diagnostics
```

Disable the paid tier just as additively: `--set ai.enabled=false`
(and `approval.enabled=false`) removes the aiwatch + approval workloads
and leaves OSS running untouched.
