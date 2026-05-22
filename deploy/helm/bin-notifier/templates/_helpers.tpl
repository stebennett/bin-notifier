{{- define "bin-notifier.fullname" -}}
{{- printf "%s" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "bin-notifier.labels" -}}
app.kubernetes.io/name: bin-notifier
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "bin-notifier.apiImage" -}}
{{- $tag := .Values.image.api.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.api.repository $tag -}}
{{- end -}}

{{- define "bin-notifier.notifierImage" -}}
{{- $tag := .Values.image.notifier.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.notifier.repository $tag -}}
{{- end -}}

{{- define "bin-notifier.configSecretName" -}}
{{- if .Values.existingConfigSecret -}}
{{ .Values.existingConfigSecret }}
{{- else -}}
{{ include "bin-notifier.fullname" . }}-config
{{- end -}}
{{- end -}}

{{- define "bin-notifier.tokensSecretName" -}}
{{- if .Values.existingTokensSecret -}}
{{ .Values.existingTokensSecret }}
{{- else -}}
{{ include "bin-notifier.fullname" . }}-tokens
{{- end -}}
{{- end -}}
