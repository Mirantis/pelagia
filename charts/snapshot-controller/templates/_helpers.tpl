{{/*
Expand the name of the chart.
*/}}
{{- define "snapshot-controller.name" -}}
{{- default .Chart.Name .Values.snapshotControllerConfig.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "snapshot-controller.image" -}}
{{- if (.Values.images.fullName) -}}
{{- .Values.images.fullName -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.snapshotController.repository .Values.images.snapshotController.tag -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "snapshot-controller.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "snapshot-controller.labels" -}}
app.kubernetes.io/name: {{ include "snapshot-controller.name" . }}
helm.sh/chart: {{ include "snapshot-controller.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
