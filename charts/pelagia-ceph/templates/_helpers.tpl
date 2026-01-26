{{- define "chart.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/instance: {{ .Release.Name }}
  {{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
  {{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "rook.labels" -}}
operator: rook
storage-backend: ceph
{{- end -}}

{{- define "release.namespace" -}}
{{- if .Values.global.namespace -}}
{{- .Values.global.namespace -}}
{{- else -}}
{{- .Release.Namespace -}}
{{- end -}}
{{- end -}}

{{- define "controller.image" -}}
{{- if (.Values.images.pelagia.fullName) -}}
{{- .Values.images.pelagia.fullName -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.global.dockerBaseUrl .Values.images.pelagia.repository .Values.images.pelagia.tag -}}
{{- end -}}
{{- end -}}

{{- define "get.image" -}}
  {{- if or (eq .release "tentacle") (eq .release "squid") -}}
    {{- if (eq .release "tentacle") -}}
      {{- printf "%s:%s" .values.repository .values.tag.tentacle -}}
    {{- else if (eq .release "squid") -}}
      {{- printf "%s:%s" .values.repository .values.tag.squid -}}
    {{- end -}}
  {{- else -}}
    {{- printf "%s:%s" .values.repository .values.tag.latest -}}
  {{- end -}}
{{- end -}}

{{- define "rook.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.rook.operator)) }}
{{- end -}}

{{- define "ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.ceph)) }}
{{- end -}}

{{- define "csi.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.ceph)) }}
{{- end -}}

{{- define "csiregistrar.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.registrar)) }}
{{- end -}}

{{- define "csiprovisioner.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.provisioner)) }}
{{- end -}}

{{- define "csisnapshotter.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.snapshotter)) }}
{{- end -}}

{{- define "csiattacher.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.attacher)) }}
{{- end -}}

{{- define "csiresizer.ceph.image" -}}
{{- printf "%s/%s" .Values.global.dockerBaseUrl (include "get.image" (dict "release" .Values.cephRelease "values" .Values.images.csi.resizer)) }}
{{- end -}}
