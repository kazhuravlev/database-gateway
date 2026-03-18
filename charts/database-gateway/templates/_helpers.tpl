{{/*
Database Gateway provides access to servers with ACL for safe and restricted database interactions.
Copyright (C) 2024  Kirill Zhuravlev

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/}}

{{- define "database-gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "database-gateway.fullname" -}}
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

{{- define "database-gateway.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "database-gateway.labels" -}}
helm.sh/chart: {{ include "database-gateway.chart" . }}
app.kubernetes.io/name: {{ include "database-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "database-gateway.selectorLabels" -}}
app.kubernetes.io/name: {{ include "database-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "database-gateway.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "database-gateway.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "database-gateway.configSecretName" -}}
{{- if .Values.config.existingSecret -}}
{{- .Values.config.existingSecret -}}
{{- else -}}
{{- printf "%s-config" (include "database-gateway.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "database-gateway.policyConfigMapName" -}}
{{- if .Values.policy.existingConfigMap -}}
{{- .Values.policy.existingConfigMap -}}
{{- else -}}
{{- printf "%s-policy" (include "database-gateway.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "database-gateway.configPayload" -}}
{{- $cfg := dict
  "targets" .Values.config.targets
  "users" (dict
    "client_id" .Values.config.users.clientID
    "client_secret" .Values.config.users.clientSecret
    "issuer_url" .Values.config.users.issuerURL
    "redirect_url" .Values.config.users.redirectURL
    "access_token_audience" .Values.config.users.accessTokenAudience
    "scopes" .Values.config.users.scopes
    "role_claim" .Values.config.users.roleClaim
    "role_mapping" .Values.config.users.roleMapping
  )
  "policy" (dict
    "path" .Values.policy.mountPath
  )
  "facade" (dict
    "port" .Values.config.facade.port
    "cookie_secret" .Values.config.facade.cookieSecret
    "unsafe_cors_allow_all" .Values.config.facade.unsafeCORSAllowAll
  )
  "storage" (dict
    "host" .Values.config.storage.host
    "port" .Values.config.storage.port
    "database" .Values.config.storage.database
    "username" .Values.config.storage.username
    "password" .Values.config.storage.password
    "use_ssl" .Values.config.storage.useSSL
    "max_pool_size" .Values.config.storage.maxPoolSize
  )
-}}
{{- mustToPrettyJson $cfg -}}
{{- end -}}
