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
Image reference: <repo>:<tag>. Defaults tag to .Chart.AppVersion.
*/}}
{{- define "cha.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
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
