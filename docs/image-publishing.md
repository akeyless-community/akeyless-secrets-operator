# Install and publish

## Recommended — install from GHCR

Install the published Helm chart and image (no local build):

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --create-namespace
```

The chart defaults pull `ghcr.io/akeyless-community/akeyless-secrets-operator` at the chart `appVersion` (e.g. `v0.1.1` for chart `0.1.1`). See [GitHub Releases](https://github.com/akeyless-community/akeyless-secrets-operator/releases) for the latest version.

### Fresh cluster vs existing CRDs

```bash
# Default — install Akeyless CRDs (not legacy ESO CRDs)
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace

# CRDs already applied manually — skip CRD install
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace \
  --set installCRDs=false
```

### Co-existing with External Secrets Operator

Use the same command. Chart defaults install only `secrets.akeyless.io` CRDs and skip legacy ESO CRDs.

### Upgrade

```bash
helm upgrade akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --reuse-values
```

### Namespace-scoped install

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -f docs/examples/helm-values-scoped.example.yaml \
  -n my-app --create-namespace
```

---

## Releases (maintainers)

See [ghcr-visibility.md](ghcr-visibility.md) for publishing a new version.

---

## Advanced — build and install from source

Use for development, air-gapped clusters, or a private registry.

### 1. Build the image

```bash
git clone https://github.com/akeyless-community/akeyless-secrets-operator.git
cd akeyless-secrets-operator

ARCH=amd64 make build-amd64
docker build --platform linux/amd64 -f Dockerfile \
  -t docker.io/<your-user>/akeyless-secrets-operator:dev .
```

For arm64: `ARCH=arm64`, `--platform linux/arm64`.

### 2. Push or load the image

```bash
docker login
docker push docker.io/<your-user>/akeyless-secrets-operator:dev
```

For kind/minikube: `kind load docker-image docker.io/<your-user>/akeyless-secrets-operator:dev`

### 3. Install from the local chart

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set image.repository=docker.io/<your-user>/akeyless-secrets-operator \
  --set image.tag=dev
```

---

## Example manifests

```bash
kubectl apply -f docs/examples/akeyless-creds-secret.example.yaml
kubectl apply -f docs/examples/akeyless-secret-store.yaml
kubectl apply -f docs/examples/akeyless-secret.yaml
```

---

## Related docs

- [Getting started](getting-started.md)
- [User guide](akeyless-secrets-operator-guide.md)
- [Documentation index](README.md)
- [Helm chart INSTALL](../deploy/charts/external-secrets/INSTALL.md)
