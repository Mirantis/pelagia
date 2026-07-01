{{- define "rook.labels" -}}
operator: rook
storage-backend: ceph
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

{{- define "rook.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.operator.repository .Values.images.operator.tag }}
{{- end -}}

{{- define "csi.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.ceph.repository .Values.images.csi.ceph.tag }}
{{- end -}}

{{- define "csiregistrar.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.registrar.repository .Values.images.csi.registrar.tag }}
{{- end -}}

{{- define "csiprovisioner.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.provisioner.repository .Values.images.csi.provisioner.tag }}
{{- end -}}

{{- define "csisnapshotter.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.snapshotter.repository .Values.images.csi.snapshotter.tag }}
{{- end -}}

{{- define "csiattacher.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.attacher.repository .Values.images.csi.attacher.tag }}
{{- end -}}

{{- define "csiresizer.ceph.image" -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.csi.resizer.repository .Values.images.csi.resizer.tag }}
{{- end -}}
