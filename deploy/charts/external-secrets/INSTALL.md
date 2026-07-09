# Install — Akeyless Secrets Operator Helm chart

> The auto-generated `README.md` in this directory is inherited from upstream ESO. **Use this file** for Akeyless install instructions.

## Install from GHCR (recommended)

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --create-namespace
```

Pin the latest release from [GitHub Releases](https://github.com/akeyless-community/akeyless-secrets-operator/releases).

### CRDs already on the cluster

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace \
  --set installCRDs=false
```

### Co-existing with External Secrets Operator

Use the default install command. Only `secrets.akeyless.io` CRDs are installed; legacy ESO CRDs are skipped.

## Install from a local chart (development)

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  -n akeyless-secrets-operator --create-namespace \
  --set image.repository=ghcr.io/akeyless-community/akeyless-secrets-operator \
  --set image.tag=v0.1.1
```

## More documentation

- [Getting Started](../../docs/getting-started.md)
- [User Guide](../../docs/akeyless-secrets-operator-guide.md)
- [Chart values](values.yaml)
