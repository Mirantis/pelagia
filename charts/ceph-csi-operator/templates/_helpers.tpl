{{- define "ceph-csi-operator.labels" -}}
operator: ceph-csi
{{- end -}}

{{- define "operator.image" -}}
{{- if (.Values.images.operator.fullName) -}}
{{- .Values.images.operator.fullName -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.operator.repository .Values.images.operator.tag -}}
{{- end -}}
{{- end -}}

{{- define "chart.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/instance: {{ .Release.Name }}
  {{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
  {{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
