---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "ingress-anubis.fullname" . }}
  labels:
    {{- include "ingress-anubis.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ include "ingress-anubis.fullname" . }}
subjects:
  - kind: ServiceAccount
    name: {{ include "ingress-anubis.serviceAccountName" . }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "ingress-anubis.fullname" . }}
  labels:
    {{- include "ingress-anubis.labels" . | nindent 4 }}
rules:
  - apiGroups: [""]
    resources: ["services", "events"]
    verbs: ["get", "update", "list", "create", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "update", "list", "create", "delete"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "update", "list", "create", "delete"]
  - apiGroups: ["extensions", "networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get", "update",  "list", "create", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "ingress-anubis.fullname" . }}
  labels:
    {{- include "ingress-anubis.labels" . | nindent 4 }}
rules:
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["list", "watch"]
  - apiGroups: ["extensions", "networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get", "list", "watch", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "ingress-anubis.fullname" . }}
  labels:
    {{- include "ingress-anubis.labels" . | nindent 4 }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{ include "ingress-anubis.fullname" . }}
subjects:
  - kind: ServiceAccount
    name: {{ include "ingress-anubis.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
