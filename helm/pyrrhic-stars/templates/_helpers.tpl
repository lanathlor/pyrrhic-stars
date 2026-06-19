{{/*
Expand the name of the chart.
*/}}
{{- define "pyrrhic-stars.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name. Truncated to 63 chars for DNS-1035.
*/}}
{{- define "pyrrhic-stars.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart name and version, for the helm.sh/chart label.
*/}}
{{- define "pyrrhic-stars.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "pyrrhic-stars.labels" -}}
helm.sh/chart: {{ include "pyrrhic-stars.chart" . }}
{{ include "pyrrhic-stars.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "pyrrhic-stars.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pyrrhic-stars.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name.
*/}}
{{- define "pyrrhic-stars.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "pyrrhic-stars.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
The Secret name holding sensitive backend credentials.
*/}}
{{- define "pyrrhic-stars.secretName" -}}
{{- if .Values.existingSecret }}
{{- .Values.existingSecret }}
{{- else }}
{{- printf "%s-credentials" (include "pyrrhic-stars.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Resolve an image reference. Pass a dict: {root, repository, tag}.
Tag precedence: per-image tag -> .Values.image.tag -> "latest".
*/}}
{{- define "pyrrhic-stars.image" -}}
{{- $root := .root -}}
{{- $tag := default (default "latest" $root.Values.image.tag) .tag -}}
{{- printf "%s/%s:%s" $root.Values.image.registry .repository $tag -}}
{{- end }}

{{/*
Gateway environment variables, shared by gateway/zone/chat workloads.
Pass the root context.
*/}}
{{- define "pyrrhic-stars.serverEnv" -}}
{{- $c := .Values.config -}}
{{- if $c.devMode }}
- name: CODEX_DEV
  value: "1"
{{- end }}
- name: REDIS_ADDR
  value: {{ $c.redis.addr | quote }}
- name: DB_DRIVER
  value: {{ $c.postgres.driver | quote }}
- name: POSTGRES_DSN
  valueFrom:
    secretKeyRef:
      name: {{ include "pyrrhic-stars.secretName" . }}
      key: POSTGRES_DSN
- name: KRATOS_PUBLIC_URL
  value: {{ $c.kratos.publicUrl | quote }}
{{- if $c.clickhouse.addr }}
- name: CLICKHOUSE_ADDR
  value: {{ $c.clickhouse.addr | quote }}
- name: CLICKHOUSE_DB
  value: {{ $c.clickhouse.database | quote }}
- name: CLICKHOUSE_USER
  value: {{ $c.clickhouse.user | quote }}
- name: CLICKHOUSE_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "pyrrhic-stars.secretName" . }}
      key: CLICKHOUSE_PASSWORD
{{- end }}
{{- if $c.gameData.levelsDir }}
- name: CODEX_LEVELS_DIR
  value: {{ $c.gameData.levelsDir | quote }}
{{- end }}
{{- if $c.gameData.mobsDir }}
- name: CODEX_MOBS_DIR
  value: {{ $c.gameData.mobsDir | quote }}
{{- end }}
{{- end }}
