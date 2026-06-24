# Akeyless Secrets Operator

Kubernetes operator that syncs secrets from [Akeyless](https://www.akeyless.io/) into native Kubernetes `Secret` objects.

This project is a focused fork of [External Secrets Operator](https://github.com/external-secrets/external-secrets), retaining only the Akeyless provider and the core reconciliation engine. It is maintained by the [Akeyless Community](https://github.com/akeyless-community).

## Why fork?

- **Single provider** — smaller binary, simpler CRDs, faster releases aligned with Akeyless features
- **Akeyless-first UX** — documentation, Helm chart, and defaults tailored to Akeyless deployments
- **Change-aware sync** — roadmap includes tighter integration with Akeyless secret lifecycle (see [docs/ROADMAP.md](docs/ROADMAP.md))

## Compatibility

**Use the Akeyless-owned CRDs** under API group `secrets.akeyless.io/v1alpha1`:

| Legacy (ESO) | Akeyless operator |
|--------------|-------------------|
| `SecretStore` | `AkeylessSecretStore` |
| `ClusterSecretStore` | `ClusterAkeylessSecretStore` |
| `ExternalSecret` | `AkeylessSecret` |

Legacy `external-secrets.io` reconciliation is **disabled by default**. Enable with `--enable-legacy-external-secrets-reconciler=true` if needed during migration.

See [docs/examples/akeyless-secret.yaml](docs/examples/akeyless-secret.yaml).

## Sync policies

`AkeylessSecret.spec.syncPolicy` controls when Kubernetes Secrets are updated:

| Policy | Behavior |
|--------|----------|
| `OnRemoteChange` (default) | Polls Akeyless `last_version` on `syncInterval`; full sync only when a remote item changes |
| `Periodic` | Sync on a fixed interval |
| `OnSpecChange` | Sync when the CR spec/metadata changes |
| `CreatedOnce` | Create once, never update values |

## Akeyless features included

Includes upstream ESO Akeyless support plus pending upstream PR [#6507](https://github.com/external-secrets/external-secrets/pull/6507):

- Static, rotated, and certificate secrets
- Gateway URL, custom CA, `caProvider`
- Auth: `api_key`, `aws_iam`, `gcp`, `azure_ad`, `k8s`
- **`ignoreCache`** — bypass Gateway cache on reads (`get-secret-value`, `get-rotated-secret-value`, `get-certificate-value`)
- **`serviceAccountRef`** for `azure_ad` — namespace-scoped AKS Workload Identity

See [docs/provider/akeyless.md](docs/provider/akeyless.md) for configuration examples.

## Quick start

See the full [User Guide](docs/akeyless-secrets-operator-guide.md) for installation, all configuration options, and examples.

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecretStore
metadata:
  name: akeyless
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
---
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecret
metadata:
  name: app-secret
spec:
  syncPolicy: OnRemoteChange
  syncInterval: 5m
  storeRef:
    name: akeyless
  target:
    name: app-secret
  data:
    - secretKey: password
      remoteRef:
        key: /path/to/secret
```

## Build

```bash
make build          # builds with -tags akeyless (default)
make test           # unit tests
make reviewable     # full PR gate (generate, lint, tests, CRD snapshots)
```

## Container image

See [docs/image-publishing.md](docs/image-publishing.md) for building, pushing to a personal registry, migrating to `ghcr.io/akeyless-community/akeyless-secrets-operator`, and which files to update when the image coordinates change.

Example manifests: [docs/examples/](docs/examples/).

## Origin & license

Forked from External Secrets Operator (Apache 2.0). See [LICENSE](LICENSE) and upstream attribution in commit history.
