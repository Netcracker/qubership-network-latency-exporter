{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "network-latency-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "network-latency-exporter.fullname" -}}
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
Create the name of the service account to use
*/}}
{{- define "network-latency-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "network-latency-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Find a network-latency-exporter image in various places.
Image can be found from:
* specified by user from .Values.image
* default value
*/}}
{{- define "network-latency-exporter.image" -}}
  {{- if .Values.image -}}
    {{- printf "%s" .Values.image -}}
  {{- else -}}
    {{- printf "ghcr.io/netcracker/qubership-network-latency-exporter:main" -}}
  {{- end -}}
{{- end -}}

{{/*
Return securityContext section for daemonset pods
*/}}
{{- define "daemonset.securityContext" -}}
  {{- if .Values.securityContext }}
    {{- toYaml .Values.securityContext | nindent 8 }}
    {{- if not (.Capabilities.APIVersions.Has "apps.openshift.io/v1") }}
      {{- if not .Values.securityContext.runAsUser }}
        runAsUser: 10001
      {{- end }}
      {{- if not .Values.elasticsearch.rollover.securityContext.fsGroup }}
        fsGroup: 2000
      {{- end }}
    {{- end }}
    {{- if (eq (.Values.securityContext.runAsNonRoot | toString) "false") }}
      runAsNonRoot: false
    {{- else }}
      runAsNonRoot: true
    {{- end }}
    {{- if and (ge .Capabilities.KubeVersion.Minor "25") (not .Values.securityContext.seccompProfile) }}
      seccompProfile:
        type: "RuntimeDefault"
    {{- end }}
  {{- else }}
       runAsUser: 10001
       fsGroup: 2000
       runAsNonRoot: true
    {{- if ge .Capabilities.KubeVersion.Minor "25" }}
       seccompProfile:
         type: "RuntimeDefault"
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Return container securityContext section for daemonset pods
*/}}
{{- define "daemonset.containerSecurityContext" -}}
  {{- if ge .Capabilities.KubeVersion.Minor "25" }}
    {{- if .Values.containerSecurityContext }}
      {{- toYaml .Values.containerSecurityContext | nindent 12 }}
    {{- else }}
           allowPrivilegeEscalation: false
           readOnlyRootFilesystem: true
           capabilities:
             drop:
               - ALL
    {{- end }}
  {{- else }}
    {{- if .Values.containerSecurityContext }}
      {{- toYaml .Values.containerSecurityContext | nindent 12 }}
    {{- else }}
      {}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Describes the readinessProbe behavior for daemonset
*/}}
{{- define "daemonset.readinessProbe" -}}
readinessProbe:
  tcpSocket:
    port: 9273
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 5
{{- end -}}

{{/*
Describes the livenessProbe behavior for daemonset
*/}}
{{- define "daemonset.livenessProbe" -}}
livenessProbe:
  tcpSocket:
    port: 9273
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 5
{{- end -}}
