# Akeyless Secrets Operator

Kubernetes operator that syncs secrets from [Akeyless](https://www.akeyless.io/) into native Kubernetes `Secret` objects.

This project is a focused fork of [External Secrets Operator](https://github.com/external-secrets/external-secrets), retaining only the Akeyless provider and the core reconciliation engine. It is maintained by the [Akeyless Community](https://github.com/akeyless-community).

## Why fork?

- **Single provider** — smaller binary, simpler CRDs, faster releases aligned with Akeyless features
- **Akeyless-first UX** — documentation, Helm chart, and defaults tailored to Akeyless deployments
- **Change-aware sync** — roadmap includes tighter integration with Akeyless secret lifecycle (see [docs/ROADMAP.md](docs/ROADMAP.md))

## Compatibility

CRDs and API group remain `external-secrets.io/v1` for now, so existing `ExternalSecret`, `SecretStore`, and `ClusterSecretStore` manifests from ESO's Akeyless provider continue to work.

## Akeyless features included

Includes upstream ESO Akeyless support plus pending upstream PR [#6507](https://github.com/external-secrets/external-secrets/pull/6507):

- Static, rotated, and certificate secrets
- Gateway URL, custom CA, `caProvider`
- Auth: `api_key`, `aws_iam`, `gcp`, `azure_ad`, `k8s`
- **`ignoreCache`** — bypass Gateway cache on reads (`get-secret-value`, `get-rotated-secret-value`, `get-certificate-value`)
- **`serviceAccountRef`** for `azure_ad` — namespace-scoped AKS Workload Identity

See [docs/provider/akeyless.md](docs/provider/akeyless.md) for configuration examples.

## Quick start

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: akeyless
spec:
  provider:
    akeyless:
      akeylessGWApiURL: "https://api.akeyless.io"
      authSecretRef:
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
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: akeyless
    kind: SecretStore
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

## Origin & license

Forked from External Secrets Operator (Apache 2.0). See [LICENSE](LICENSE) and upstream attribution in commit history.
