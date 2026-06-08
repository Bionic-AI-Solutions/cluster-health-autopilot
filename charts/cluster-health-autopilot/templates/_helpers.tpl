{{/*
Expand the name of the chart.
*/}}
{{- define "cha.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a fully-qualified app name. Truncated at 63 chars (k8s label limit).
*/}}
{{- define "cha.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Common labels.
*/}}
{{- define "cha.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "cha.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: cluster-health-autopilot
{{- end -}}

{{/*
Selector labels.
*/}}
{{- define "cha.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cha.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "cha.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (printf "%s-sa" (include "cha.fullname" .)) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Reader ClusterRole name.
*/}}
{{- define "cha.readerName" -}}
{{- default (printf "%s-reader" (include "cha.fullname" .)) .Values.rbac.reader.name -}}
{{- end -}}

{{/*
Remediator ClusterRole name.
*/}}
{{- define "cha.remediatorName" -}}
{{- default (printf "%s-remediator" (include "cha.fullname" .)) .Values.rbac.remediator.name -}}
{{- end -}}

{{/*
Operator (cha-operator) workload name. Phase 1b — the controller-runtime
manager that reconciles ClusterHealthAutopilot CRs.
*/}}
{{- define "cha.operatorName" -}}
{{- printf "%s-operator" (include "cha.fullname" .) -}}
{{- end -}}

{{/*
Image reference: <repo>:<tag>. Defaults tag to .Chart.AppVersion.
*/}}
{{- define "cha.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{- /*
cha.aiImage — the commercial CHA-com image for the AI-companion
Deployment. Repository defaults to docker4zerocool/cha-com; tag
defaults to "v<AppVersion>" (cha-com images are tagged with a leading
"v", unlike the OSS image), so a chart at appVersion 1.8.2 pulls
docker4zerocool/cha-com:v1.8.2 — the CHA-com release pinned to the same
OSS engine. Override ai.image.tag to decouple.
*/ -}}
{{- define "cha.aiImage" -}}
{{- $repo := (.Values.ai.image).repository | default "docker4zerocool/cha-com" -}}
{{- $tag := (.Values.ai.image).tag | default (printf "v%s" .Chart.AppVersion) -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end -}}

{{/*
Effective pull policy: Always for mutable tags (latest / -latest suffix /
empty + AppVersion that happens to be a dev marker), IfNotPresent for
release-style semver tags. Overridable via .Values.image.pullPolicy when
set explicitly.

Closes the operational gotcha from v1.6.0 deployment: pushing the same
mutable tag twice with different content silently kept the kubelet-
cached digest. Always-pull on mutable tags makes the chart safe for
operators who don't enforce immutable-tag discipline upstream.
*/}}
{{- define "cha.pullPolicy" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if .Values.image.pullPolicy -}}
{{- .Values.image.pullPolicy -}}
{{- else if or (eq $tag "latest") (hasSuffix "-latest" $tag) (eq $tag "main") (eq $tag "dev") -}}
Always
{{- else -}}
IfNotPresent
{{- end -}}
{{- end -}}

{{/*
Slack env blocks — one helper per channel.
Each returns a list of env entries (NOT wrapped in `env:` block) so callers
can compose with their own env definitions.
*/}}
{{- define "cha.slackAlertsEnv" -}}
{{- if and .Values.slack.alerts.enabled .Values.slack.alerts.secretName -}}
- name: SLACK_ALERTS_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.slack.alerts.secretName }}
      key: {{ .Values.slack.alerts.secretKey | default "WEBHOOK_URL" }}
{{- end -}}
{{- end -}}

{{- define "cha.slackCriticalEnv" -}}
{{- if and .Values.slack.critical.enabled .Values.slack.critical.secretName -}}
- name: SLACK_CRITICAL_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.slack.critical.secretName }}
      key: {{ .Values.slack.critical.secretKey | default "WEBHOOK_URL" }}
{{- end -}}
{{- end -}}

{{- define "cha.slackHealthinfoEnv" -}}
{{- if and .Values.slack.healthinfo.enabled .Values.slack.healthinfo.secretName -}}
- name: SLACK_HEALTHINFO_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.slack.healthinfo.secretName }}
      key: {{ .Values.slack.healthinfo.secretKey | default "WEBHOOK_URL" }}
{{- end -}}
{{- end -}}

{{/*
Vault-probe env block. Empty when vaultProbe disabled.
Auth precedence: $VAULT_TOKEN (from Secret reference, dev/test) → kubernetes
auth via $VAULT_K8S_ROLE (production posture: SA JWT login). Both modes
inject $VAULT_ADDR and $VAULT_KV_MOUNT.
*/}}
{{- define "cha.vaultEnv" -}}
{{- if .Values.vaultProbe.enabled -}}
{{- if not .Values.vaultProbe.address -}}
{{- fail "vaultProbe.address is required when vaultProbe.enabled=true" -}}
{{- end -}}
- name: VAULT_ADDR
  value: {{ .Values.vaultProbe.address | quote }}
- name: VAULT_KV_MOUNT
  value: {{ .Values.vaultProbe.kvMount | default "secret" | quote }}
{{- if eq (.Values.vaultProbe.auth.method | default "kubernetes") "kubernetes" }}
{{- if not .Values.vaultProbe.auth.role -}}
{{- fail "vaultProbe.auth.role is required when vaultProbe.enabled=true and auth.method=kubernetes" -}}
{{- end -}}
- name: VAULT_K8S_ROLE
  value: {{ .Values.vaultProbe.auth.role | quote }}
{{- else if eq .Values.vaultProbe.auth.method "token" }}
- name: VAULT_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.vaultProbe.auth.tokenSecretRef.name | quote }}
      key: {{ .Values.vaultProbe.auth.tokenSecretRef.key | default "token" | quote }}
{{- end }}
{{- end -}}
{{- end -}}

{{- /*
cha.optionalFixerEnv — env-var toggles for opt-in fixers. Off by default.
Keep additive — never break existing installs by changing default semantics.
*/ -}}
{{- define "cha.optionalFixerEnv" -}}
{{- if (.Values.fixers).tlsSecretMismatch }}
{{- if .Values.fixers.tlsSecretMismatch.enabled }}
- name: CHA_FIXER_TLS_SECRET_MISMATCH
  value: "true"
{{- end }}
{{- end }}
{{- end -}}

{{- /*
cha.analyzerToggleEnv — env-var toggles for the v1.7+ drift-class
expansion analyzers (Workstreams B1+B2+B3+B4). Each defaults to ON;
flip analyzers.<name>.enabled=false in values.yaml to silence the
no-target list cycle on clusters that don't host the asset class.
*/ -}}
{{- define "cha.analyzerToggleEnv" -}}
{{- if (.Values.analyzers).gitopsDrift }}
{{- if not .Values.analyzers.gitopsDrift.enabled }}
- name: CHA_ANALYZER_GITOPS_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.analyzers).workloadStateDrift }}
{{- if not .Values.analyzers.workloadStateDrift.enabled }}
- name: CHA_ANALYZER_WORKLOAD_STATE_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.analyzers).rbacDrift }}
{{- if not .Values.analyzers.rbacDrift.enabled }}
- name: CHA_ANALYZER_RBAC_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.analyzers).configDrift }}
{{- if not .Values.analyzers.configDrift.enabled }}
- name: CHA_ANALYZER_CONFIG_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.analyzers).capacityDrift }}
{{- if not .Values.analyzers.capacityDrift.enabled }}
- name: CHA_ANALYZER_CAPACITY_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.analyzers).securityDrift }}
{{- if not .Values.analyzers.securityDrift.enabled }}
- name: CHA_ANALYZER_SECURITY_DRIFT
  value: "off"
{{- end }}
{{- end }}
{{- end -}}

{{- /*
cha.probeToggleEnv — env-var opt-outs for the v1.8 M2 probe-class
additions (Kong / HPA-scaling / ArgoCD-app / Velero). Each defaults to
ON and auto-skips when its CRD is absent (or no-ops on an empty list
for HPA), so the toggle is only needed to silence a probe on a cluster
that DOES host the CRD but doesn't want CHA watching it. Flip
probes.<name>.enabled=false in values.yaml to emit CHA_PROBE_<NAME>=off.
*/ -}}
{{- define "cha.probeToggleEnv" -}}
{{- if (.Values.probes).kong }}
{{- if not .Values.probes.kong.enabled }}
- name: CHA_PROBE_KONG
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.probes).hpaScaling }}
{{- if not .Values.probes.hpaScaling.enabled }}
- name: CHA_PROBE_HPA_SCALING
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.probes).argocdApp }}
{{- if not .Values.probes.argocdApp.enabled }}
- name: CHA_PROBE_ARGOCD_APP
  value: "off"
{{- end }}
{{- end }}
{{- if (.Values.probes).velero }}
{{- if not .Values.probes.velero.enabled }}
- name: CHA_PROBE_VELERO
  value: "off"
{{- end }}
{{- end }}
{{- end -}}

{{- /*
cha.aiArgs — the `cha-com watch` flag surface for the AI-companion
Deployment (templates/aiwatch-deployment.yaml). These are the ONLY
flags the commercial `cha-com watch` binary accepts — it is the
"AI-layered counterpart" to the OSS watcher and deliberately does NOT
re-expose the OSS operational flags (--live / --slack-* / --remedy /
--ticketing-* / --cloud-* / --write-driftreports). The OSS watcher and
diagnose/remediate CronJobs are NEVER swapped to cha-com; they keep
running the OSS image + operational loop. See docs/DEPLOYMENT.md.
*/ -}}
{{- define "cha.aiArgs" -}}
- watch
- --ai-tier={{ .Values.ai.tier | default "t0" }}
- --ai-endpoint={{ required "ai.endpoint is required when ai.enabled=true" .Values.ai.endpoint }}
- --ai-model={{ required "ai.model is required when ai.enabled=true" .Values.ai.model }}
- --interval={{ .Values.ai.interval | default "60s" }}
{{- with (.Values.ai.apiKey).header }}
- --ai-api-key-header={{ . }}
{{- end }}
{{- with (.Values.ai.apiKey).envName }}
- --ai-api-key-env={{ . }}
{{- end }}
{{- if .Values.ai.allowSaas }}
- --ai-allow-saas
{{- end }}
{{- if .Values.ai.llmFixerMatcher }}
- --ai-llm-fixer-matcher
{{- end }}
{{- with .Values.ai.auditLog }}
- --ai-audit-log={{ . }}
{{- end }}
{{- with .Values.ai.approvalServerUrl }}
- --approval-server-url={{ . }}
{{- end }}
{{- range (.Values.ai.t3).vaultAllowedPrefixes }}
- --t3-vault-allowed-prefix={{ . }}
{{- end }}
{{- if (.Values.ai.memory).enabled }}
- --memory-store-url={{ (.Values.ai.memory).storeUrl | default (printf "http://%s-rag.%s.svc:6333" (include "cha.fullname" .) .Release.Namespace) }}
{{- with (.Values.ai.memory.embeddings).endpoint }}
- --memory-embeddings-endpoint={{ . }}
{{- end }}
- --memory-embeddings-model={{ required "ai.memory.embeddings.model is required when ai.memory.enabled" (.Values.ai.memory.embeddings).model }}
- --memory-topk={{ (.Values.ai.memory).topK | default 5 }}
{{- end }}
{{- /* Phase 2.F — leader election when replicas > 1. Single-replica
       deploys take the noop path (no Lease, no RBAC dependency). */ -}}
{{- if gt (int (.Values.ai.replicas | default 1)) 1 }}
- --leader-election=true
- --leader-election-namespace={{ .Release.Namespace }}
- --leader-election-name={{ include "cha.fullname" . }}-aiwatch-leader
{{- end }}
{{- end -}}

{{- /*
cha.aiEnv — injects the LLM bearer token into the env var the CHA-com
binary reads (--ai-api-key-env, default AI_API_KEY). Sourced from a
K8s Secret (ESO-managed); never inlined. Empty when ai.apiKey.secretName
is unset (in-cluster vLLM with no auth).
*/ -}}
{{- define "cha.aiEnv" -}}
{{- if and (.Values.ai).enabled (.Values.ai.apiKey).secretName }}
- name: {{ .Values.ai.apiKey.envName | default "AI_API_KEY" }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.ai.apiKey.secretName }}
      key: {{ .Values.ai.apiKey.secretKey | default "API_KEY" }}
{{- end }}
{{- end -}}
