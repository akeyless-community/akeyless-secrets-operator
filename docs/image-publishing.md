# Build and install from source

Install the operator by building the container image from this repository and deploying the local Helm chart. There is no public chart or image registry — you build, push to a registry your cluster can reach, then install with Helm.

## Overview

| Step | What you do |
|------|-------------|
| 1 | Clone the repo and build the Linux binary + container image |
| 2 | Push the image to Docker Hub, a private registry, or load it into a local cluster |
| 3 | `helm upgrade --install` using `./deploy/charts/external-secrets` and your image coordinates |

---

## 1. Build the image

From the repository root:

```bash
# amd64 (most cloud clusters)
ARCH=amd64 make build-amd64
docker build --platform linux/amd64 -f Dockerfile \
  -t docker.io/<your-user>/akeyless-secrets-operator:dev .

# arm64 (Apple Silicon nodes, Graviton, etc.)
ARCH=arm64 make build-arm64
docker build --platform linux/arm64 -f Dockerfile \
  -t docker.io/<your-user>/akeyless-secrets-operator:dev .
```

Or use Make with explicit registry variables:

```bash
ARCH=amd64 \
  IMAGE_NAME=docker.io/<your-user>/akeyless-secrets-operator \
  IMAGE_TAG=dev \
  make docker.build
```

---

## 2. Make the image available to the cluster

### Push to a registry

```bash
docker login
docker push docker.io/<your-user>/akeyless-secrets-operator:dev
```

Use any registry your cluster can pull from (Docker Hub, ECR, ACR, GCR, Harbor, etc.). Replace `docker.io/<your-user>/...` with your coordinates throughout.

### Private registry

If the registry requires authentication, create a pull secret in the operator namespace:

```bash
kubectl create secret docker-registry akeyless-operator-registry \
  --docker-server=<registry-host> \
  --docker-username=<user> \
  --docker-password=<token-or-password> \
  -n akeyless-secrets-operator
```

Pass it to Helm with `--set imagePullSecrets[0].name=akeyless-operator-registry` or in a values file.

### Local clusters (kind / minikube)

Load the image directly instead of pushing:

```bash
kind load docker-image docker.io/<your-user>/akeyless-secrets-operator:dev
# or
minikube image load docker.io/<your-user>/akeyless-secrets-operator:dev
```

Use `imagePullPolicy: IfNotPresent` or `Never` in Helm values when the image is pre-loaded.

---

## 3. Install with Helm

Always install from the chart in this repository:

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set installCRDs=true \
  --set image.repository=docker.io/<your-user>/akeyless-secrets-operator \
  --set image.tag=dev
```

### Namespace-scoped install

For a single application namespace, use the example values file:

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  -f docs/examples/helm-values-scoped.example.yaml \
  --namespace my-app --create-namespace \
  --set image.repository=docker.io/<your-user>/akeyless-secrets-operator \
  --set image.tag=dev
```

Edit `docs/examples/helm-values-scoped.example.yaml` to match your image and scope before applying.

### Co-existing with External Secrets Operator

When ESO is already installed, use the chart defaults — only the four `secrets.akeyless.io` CRDs are installed. Legacy ESO generator and `external-secrets.io` CRDs are disabled (`crds.createGenerators: false`, `crds.createExternalSecret: false`, etc.) so Helm does not try to adopt CRDs owned by the existing `external-secrets` release.

---

## Example manifests

After the operator is running, apply the Akeyless resources:

```bash
kubectl apply -f docs/examples/akeyless-creds-secret.example.yaml
kubectl apply -f docs/examples/akeyless-secret-store.yaml
kubectl apply -f docs/examples/akeyless-secret.yaml
```

See also [rollout-restart-example.yaml](examples/rollout-restart-example.yaml) for rollout restart testing.

---

## Makefile image defaults

`make docker.build` / `make docker.push` use `IMAGE_REGISTRY` and `IMAGE_REPO` from the [Makefile](../Makefile). Override them when building for your registry:

```bash
ARCH=amd64 \
  IMAGE_NAME=docker.io/<your-user>/akeyless-secrets-operator \
  IMAGE_TAG=dev \
  make docker.build docker.push
```

The Helm chart `values.yaml` contains placeholder defaults; **always set `image.repository` and `image.tag` at install time** to match the image you built.

---

## Checklist

- [ ] Image built from a known git commit or tag
- [ ] Image pushed (or loaded) to a registry the cluster can reach
- [ ] `imagePullSecrets` configured if the registry is private
- [ ] Helm install uses `./deploy/charts/external-secrets` with your image coordinates
- [ ] Helm chart `appVersion` in `Chart.yaml` matches the source you built (informational)

---

## Related docs

- [User guide](akeyless-secrets-operator-guide.md)
- [Example manifests](examples/)
- [Helm chart values](../deploy/charts/external-secrets/values.yaml)
- [Contributing — dev guide](contributing/devguide.md)
