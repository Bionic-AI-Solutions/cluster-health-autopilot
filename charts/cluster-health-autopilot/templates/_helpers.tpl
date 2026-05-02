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
Slack-webhook env block. Empty when slack disabled.
Returns a list of env entries (NOT wrapped in `env:` block) so callers
can compose with their own env definitions.
*/}}
{{- define "cha.slackEnv" -}}
{{- if and .Values.slack.enabled .Values.slack.webhookSecretName -}}
- name: SLACK_WEBHOOK_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.slack.webhookSecretName }}
      key: {{ .Values.slack.webhookSecretKey | default "WEBHOOK_URL" }}
{{- end -}}
{{- end -}}
