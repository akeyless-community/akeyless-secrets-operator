# Akeyless Secrets Operator — User Guide

Kubernetes operator that syncs secrets from [Akeyless](https://www.akeyless.io/) into native Kubernetes `Secret` objects.

This guide describes what the operator provides, how to install it, and how to configure each resource type.

---

## Overview

The Akeyless Secrets Operator is a focused fork of [External Secrets Operator](https://external-secrets.io/), built and maintained by the [Akeyless Community](https://github.com/akeyless-community). It uses **Akeyless-owned Custom Resource Definitions (CRDs)** under API group `secrets.akeyless.io/v1alpha1`.

### What it does

- Connects to Akeyless (SaaS API or self-hosted Gateway)
- Fetches static, rotated, and dynamic secrets
- Creates and updates Kubernetes `Secret` objects
- Detects remote changes and re-syncs on a schedule or via webhook
- Optionally triggers rolling restarts of Deployments, StatefulSets, or DaemonSets when synced data changes

### Custom resources

| Resource | Scope | Short name | Purpose |
|----------|-------|------------|---------|
| `AkeylessSecretStore` | Namespace | `ass` | Akeyless connection and auth for one namespace |
| `ClusterAkeylessSecretStore` | Cluster | `cass` | Shared Akeyless connection (with namespace restrictions) |
| `AkeylessSecret` | Namespace | `as` | Sync one or more static Akeyless items into a K8s Secret |
| `AkeylessDynamicSecret` | Namespace | `ads` | Sync a dynamic or rotated Akeyless item into a K8s Secret |

Legacy `external-secrets.io` resources (`ExternalSecret`, `SecretStore`, etc.) are **disabled by default**.

---

## Installation

### Prerequisites

- Kubernetes 1.25+
- Helm 3.x (recommended) or raw manifests
- Network access from the cluster to Akeyless (SaaS or Gateway)
- Akeyless credentials with read access to target items

### Install with Helm (local build)

The operator is installed from the Helm chart in this repository. **Build the container image yourself** and point Helm at your registry — there is no public OCI chart or image pull.

**1. Build and push the image**

```bash
git clone https://github.com/akeyless-community/akeyless-secrets-operator.git
cd akeyless-secrets-operator

ARCH=amd64 make build-amd64
docker build --platform linux/amd64 -f Dockerfile \
  -t docker.io/<your-user>/akeyless-secrets-operator:dev .

docker login
docker push docker.io/<your-user>/akeyless-secrets-operator:dev
```

For arm64 clusters, use `ARCH=arm64` and `--platform linux/arm64`.

**2. Install from the local chart**

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set installCRDs=true \
  --set image.repository=docker.io/<your-user>/akeyless-secrets-operator \
  --set image.tag=dev
```

If your registry is private, create an `imagePullSecret` in the operator namespace and set `imagePullSecrets` in Helm values.

### Co-existing with External Secrets Operator

If **External Secrets Operator (ESO)** is already installed in another namespace, keep the default chart values — only Akeyless CRDs (`secrets.akeyless.io`) are installed. Legacy ESO CRDs (`external-secrets.io`, `generators.external-secrets.io`) are **not** created by default, so Helm will not conflict with the existing ESO release.

If the Akeyless CRDs are already present (for example from a prior manual apply), set `installCRDs=false` and install only the operator Deployment:

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set installCRDs=false \
  --set image.repository=docker.io/<your-user>/akeyless-secrets-operator \
  --set image.tag=dev
```

See [image-publishing.md](image-publishing.md) for Make targets, scoped installs, and registry options.

### Create Akeyless credentials

Store operator credentials in a Kubernetes Secret. Supported `accessType` values include `api_key`, `aws_iam`, `azure_ad`, `gcp`, and `k8s`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: akeyless-creds
  namespace: my-app
type: Opaque
stringData:
  accessId: "<AKEYLESS_ACCESS_ID>"
  accessType: "api_key"
  accessTypeParam: "<AKEYLESS_API_KEY>"
```

---

## AkeylessSecretStore

Defines how the operator connects to Akeyless. Each namespace typically has one store (or references a cluster store).

### Spec fields

| Field | Required | Description |
|-------|----------|-------------|
| `akeylessGWApiURL` | Yes | Akeyless endpoint. SaaS: `https://api.akeyless.io`. Gateway: `https://<gateway-host>:8080/v2` |
| `auth` | Yes | Authentication configuration (see below) |
| `ignoreCache` | No | When `true`, bypass Gateway cache on reads. Relevant for Gateway deployments only |
| `caBundle` | No | PEM/base64 CA bundle to validate the Gateway TLS certificate |
| `caProvider` | No | Load CA from a `Secret` or `ConfigMap` |

### Authentication options

**Secret reference** (most common):

```yaml
auth:
  secretRef:
    accessID:
      name: akeyless-creds
      key: accessId
    accessType:
      name: akeyless-creds
      key: accessType
    accessTypeParam:
      name: akeyless-creds
      key: accessTypeParam
```

**Kubernetes auth** (Gateway auth method):

```yaml
auth:
  kubernetesAuth:
    accessID: "<access-id>"
    k8sConfName: "<k8s-auth-config-name>"
    serviceAccountRef:
      name: my-sa
      namespace: my-app
```

**Azure Workload Identity** — use `serviceAccountRef` on the store auth block with an Azure AD access type in the credentials Secret.

### Example — SaaS store

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecretStore
metadata:
  name: akeyless
  namespace: my-app
spec:
  akeylessGWApiURL: "https://api.akeyless.io"
  auth:
    secretRef:
      accessID:
        name: akeyless-creds
        key: accessId
      accessType:
        name: akeyless-creds
        key: accessType
      accessTypeParam:
        name: akeyless-creds
        key: accessTypeParam
```

---

## AkeylessSecret

Syncs one or more Akeyless static secrets (or key/value fields) into a single Kubernetes Secret.

### Spec fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `storeRef.name` | Yes | — | Name of `AkeylessSecretStore` or `ClusterAkeylessSecretStore` |
| `storeRef.kind` | No | `AkeylessSecretStore` | Store kind |
| `target.name` | No | Same as CR name | Name of the Kubernetes Secret to create |
| `target.creationPolicy` | No | `Owner` | `Owner` (delete Secret with CR) or `Orphan` (keep Secret) |
| `target.template` | No | — | Optional Secret `type`, labels, and annotations |
| `syncPolicy` | No | `OnRemoteChange` | When to sync (see [Sync policies](#sync-policies)) |
| `syncInterval` | No | `5m` | Polling interval for applicable policies |
| `data` | Yes | — | List of key mappings (see below) |
| `rolloutRestartTargets` | No | — | Workloads to restart when secret **data** changes |

### Data mapping

Each entry in `data` maps an Akeyless item path to a Kubernetes Secret key:

| Field | Description |
|-------|-------------|
| `secretKey` | Key written in the Kubernetes Secret |
| `remoteRef.key` | Full Akeyless item path (e.g. `/prod/db/password`) |
| `remoteRef.property` | Optional JSON field to extract from the item value |
| `remoteRef.version` | Optional specific item version |

### Example — static secret with rollout restart

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecret
metadata:
  name: app-secret
  namespace: my-app
spec:
  storeRef:
    name: akeyless
  syncPolicy: OnRemoteChange
  syncInterval: 5m
  target:
    name: app-secret
    creationPolicy: Owner
  rolloutRestartTargets:
    - kind: Deployment
      name: myapp
  data:
    - secretKey: password
      remoteRef:
        key: /prod/app/password
    - secretKey: api-token
      remoteRef:
        key: /prod/app/token
```

Mount the resulting Secret in your workload:

```yaml
env:
  - name: PASSWORD
    valueFrom:
      secretKeyRef:
        name: app-secret
        key: password
```

---

## AkeylessDynamicSecret

Syncs a **single** dynamic or rotated Akeyless item into a Kubernetes Secret.

### Spec fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `storeRef` | Yes | — | Reference to an Akeyless store |
| `path` | Yes | — | Akeyless dynamic-secret or rotated-secret path |
| `type` | No | `Dynamic` | `Dynamic` or `Rotated` |
| `secretKey` | No | `value` | Key written in the Kubernetes Secret |
| `property` | No | — | Extract a field from JSON payloads |
| `syncPolicy` | No | `OnRemoteChange` | Sync policy |
| `syncInterval` | No | `1m` | Polling interval |
| `target` | Yes | — | Destination Kubernetes Secret |
| `rolloutRestartTargets` | No | — | Workloads to restart on data change |

### Example — dynamic database credentials

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessDynamicSecret
metadata:
  name: db-credentials
  namespace: my-app
spec:
  storeRef:
    name: akeyless
  path: /dynamic-secret/prod-db
  type: Dynamic
  secretKey: password
  syncPolicy: OnRemoteChange
  syncInterval: 1m
  target:
    name: db-credentials
  rolloutRestartTargets:
    - kind: Deployment
      name: api-server
```

---

## Sync policies

Set `spec.syncPolicy` on `AkeylessSecret` or `AkeylessDynamicSecret`:

| Policy | Behavior |
|--------|----------|
| `OnRemoteChange` | Poll Akeyless on `syncInterval`; full sync only when remote item version changes (default) |
| `Periodic` | Full sync on every `syncInterval` |
| `OnSpecChange` | Sync when the CR spec or metadata changes |
| `CreatedOnce` | Create the Kubernetes Secret once; never update values |
| `OnWebhook` | Sync immediately when an Akeyless event webhook fires; `syncInterval` is fallback polling |

### Force an immediate sync

Annotate the resource:

```bash
kubectl annotate akeylesssecret app-secret \
  secrets.akeyless.io/force-sync="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --overwrite
```

---

## Rollout restart

When synced secret **data** changes, the operator can trigger a rolling restart of dependent workloads by setting `secrets.akeyless.io/restartedAt` on the pod template.

### Configuration

```yaml
rolloutRestartTargets:
  - kind: Deployment      # Deployment | StatefulSet | DaemonSet
    name: myapp
```

Supported kinds: `Deployment`, `StatefulSet`, `DaemonSet`.

The operator compares a data hash annotation on the managed Secret (`secrets.akeyless.io/data-hash`). When the hash changes after a successful sync, it patches each listed workload.

---

## Akeyless event webhook

For near-real-time sync, enable the webhook server on the operator and set `syncPolicy: OnWebhook`.

### Operator configuration (Helm)

```yaml
akeylessWebhook:
  enabled: true
  port: 8083
  path: /webhook/akeyless
  token: "<optional-bearer-token>"
```

### Operator configuration (CLI flags)

| Flag | Description |
|------|-------------|
| `--akeyless-webhook-addr` | Listen address (e.g. `:8083`). Empty disables the server |
| `--akeyless-webhook-path` | HTTP path (default `/webhook/akeyless`) |
| `--akeyless-webhook-token` | Optional bearer token for incoming requests |

Expose the webhook port via a Kubernetes `Service` and configure Akeyless to send event notifications to:

```text
http://<service-host>:8083/webhook/akeyless
```

When a valid event is received, the operator enqueues a reconcile for matching `AkeylessSecret` / `AkeylessDynamicSecret` resources.

---

## Status and troubleshooting

### Check resource status

```bash
kubectl get akeylesssecretstore,akeylesssecret,akeylessdynamicsecret -n my-app
kubectl describe akeylesssecret app-secret -n my-app
```

A healthy sync shows `Ready=True` and reason `SecretSynced`.

### Common status conditions

| Condition | Meaning |
|-----------|---------|
| `Ready=True`, `SecretSynced` | Last sync succeeded |
| `Ready=False`, `SecretSyncedError` | Sync failed — check store auth, item path, and operator logs |

### Operator logs

```bash
kubectl logs deployment/akeyless-secrets-operator -n akeyless-secrets-operator
```

---

## Helm chart options (Akeyless-only mode)

Key values for a minimal Akeyless-only deployment:

| Value | Default | Description |
|-------|---------|-------------|
| `processAkeylessSecret` | `true` | Reconcile `AkeylessSecret` |
| `processAkeylessDynamicSecret` | `true` | Reconcile `AkeylessDynamicSecret` |
| `processAkeylessSecretStore` | `true` | Reconcile `AkeylessSecretStore` |
| `processClusterAkeylessSecretStore` | `true` | Reconcile `ClusterAkeylessSecretStore` |
| `processLegacyExternalSecret` | `false` | Legacy ESO `ExternalSecret` support |
| `processClusterStore` | `false` | Legacy ESO cluster stores |
| `processSecretStore` | `false` | Legacy ESO namespace stores |
| `scopedRBAC` | `false` | Limit RBAC to `scopedNamespace` |
| `scopedNamespace` | `""` | Namespace scope for operator |
| `akeylessWebhook.enabled` | `false` | Enable event webhook server |
| `image.repository` | *(set at install)* | Container image you built and pushed |
| `image.tag` | *(set at install)* | Tag of your built image |
| `installCRDs` | `true` | Install CRDs with Helm |

### Operator CLI flags (Akeyless controllers)

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-akeyless-secret-reconciler` | `true` | `AkeylessSecret` controller |
| `--enable-akeyless-dynamic-secret-reconciler` | `true` | `AkeylessDynamicSecret` controller |
| `--enable-akeyless-secret-store-reconciler` | `true` | `AkeylessSecretStore` controller |
| `--enable-cluster-akeyless-secret-store-reconciler` | `true` | `ClusterAkeylessSecretStore` controller |
| `--enable-legacy-external-secrets-reconciler` | `false` | Legacy ESO controllers |
| `--enable-generator-state` | `true` | Legacy GeneratorState controller (disable if CRD not installed) |
| `--namespace` | all | Restrict reconciliation to one namespace |

---

## Migration from External Secrets Operator

| Legacy (ESO) | Akeyless operator |
|--------------|-------------------|
| `SecretStore` | `AkeylessSecretStore` |
| `ClusterSecretStore` | `ClusterAkeylessSecretStore` |
| `ExternalSecret` | `AkeylessSecret` |

Enable legacy reconciliation temporarily with `--enable-legacy-external-secrets-reconciler=true` during migration.

---

## Related documentation

- [Build and install from source](image-publishing.md)
- [Example manifests](../docs/examples/)
