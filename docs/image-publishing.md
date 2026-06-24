# Container image — build, publish, and registry migration

This guide covers how to build the operator image, publish it for development, and later move it to the official Akeyless community registry. Use it for local testing today and for public releases later.

## Image naming conventions

| Registry | Example image | When to use |
|----------|---------------|-------------|
| Personal Docker Hub (dev) | `docker.io/<your-user>/akeyless-secrets-operator:dev` | Private development and cluster testing |
| Official community registry (release) | `ghcr.io/akeyless-community/akeyless-secrets-operator:v0.1.0` | Public Helm installs and documentation |

The **official default** in this repository is:

```text
ghcr.io/akeyless-community/akeyless-secrets-operator:<tag>
```

Tags should follow semver for releases (`v0.1.0`, `v0.2.0`, …). Use descriptive dev tags (`dev-test`, `main-<sha>`) only for personal registries.

---

## 1. Build the image locally

From the repository root:

```bash
# Build the Linux amd64 binary (AKS / most clusters)
ARCH=amd64 make build-amd64

# Build the container image
docker build --platform linux/amd64 -f Dockerfile \
  -t docker.io/<your-dockerhub-user>/akeyless-secrets-operator:dev-test .
```

Or use Make with explicit registry variables:

```bash
ARCH=amd64 \
  IMAGE_NAME=docker.io/<your-dockerhub-user>/akeyless-secrets-operator \
  IMAGE_TAG=dev-test \
  make docker.build
```

For arm64 clusters, replace `amd64` with `arm64`.

---

## 2. Push to a personal Docker Hub account (current setup)

```bash
docker login
docker push docker.io/<your-dockerhub-user>/akeyless-secrets-operator:dev-test
```

If the repository is **private**, create a pull secret in every namespace that runs the operator:

```bash
kubectl create secret docker-registry akeyless-docker-hub \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=<your-dockerhub-user> \
  --docker-password=<token-or-password> \
  -n <namespace>
```

Reference that secret in the Deployment (`imagePullSecrets`) or Helm values (`imagePullSecrets`).

---

## 3. Move the image to the official Akeyless registry

When you are ready to publish under the Akeyless community org, choose **one** of these approaches.

### Option A — Retag an existing image (fastest)

Use this when the image already on Docker Hub is the exact build you want to release.

```bash
# Pull the tested image from your personal account
docker pull docker.io/<your-dockerhub-user>/akeyless-secrets-operator:dev-test

# Retag for the official registry (pick a semver tag)
export RELEASE_TAG=v0.1.0
docker tag \
  docker.io/<your-dockerhub-user>/akeyless-secrets-operator:dev-test \
  ghcr.io/akeyless-community/akeyless-secrets-operator:${RELEASE_TAG}

# Authenticate to GitHub Container Registry (needs packages:write on the org)
echo "<GITHUB_PAT>" | docker login ghcr.io -u <github-user> --password-stdin

# Push
docker push ghcr.io/akeyless-community/akeyless-secrets-operator:${RELEASE_TAG}
```

### Option B — Build and push directly to GHCR (recommended for releases)

Build from a clean git tag so the image matches source control.

```bash
git checkout v0.1.0   # or your release tag

ARCH=amd64 \
  IMAGE_NAME=ghcr.io/akeyless-community/akeyless-secrets-operator \
  IMAGE_TAG=v0.1.0 \
  make docker.build docker.push
```

### After the image is on the official registry

1. Update the files listed in [Files to update](#files-to-update-when-changing-the-image) below.
2. Open a PR with those changes (or include them in the release PR).
3. Install or upgrade with Helm using the new coordinates (see [Install with Helm](#install-with-helm)).
4. Delete or archive old dev tags on your personal Docker Hub account when no longer needed.

---

## Files to update when changing the image

Update these locations so Helm defaults and local build tooling stay consistent.

| File | What to change | Used by |
|------|----------------|---------|
| [`deploy/charts/external-secrets/values.yaml`](../deploy/charts/external-secrets/values.yaml) | `image.repository` and `image.tag` | **Primary Helm default** — all `helm install` / `helm upgrade` commands that do not override image |
| [`Makefile`](../Makefile) | `IMAGE_REGISTRY` and `IMAGE_REPO` (lines ~19–21) | `make docker.build` / `make docker.push` defaults |
| [`docs/examples/helm-values-scoped.example.yaml`](../docs/examples/helm-values-scoped.example.yaml) | `image.repository` and `image.tag` | Example scoped Helm install (optional) |

### Example: switching from personal Docker Hub to official GHCR

**Before (dev):**

```yaml
# deploy/charts/external-secrets/values.yaml
image:
  repository: docker.io/<your-user>/akeyless-secrets-operator
  tag: dev
```

**After (release `v0.1.0`):**

```yaml
# deploy/charts/external-secrets/values.yaml
image:
  repository: ghcr.io/akeyless-community/akeyless-secrets-operator
  tag: "v0.1.0"
```

```makefile
# Makefile
export IMAGE_REGISTRY ?= ghcr.io
export IMAGE_REPO     ?= akeyless-community/akeyless-secrets-operator
```

No code changes are required beyond these configuration files unless you add a CI publish workflow later.

---

## Install with Helm

Official / production install:

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set image.repository=ghcr.io/akeyless-community/akeyless-secrets-operator \
  --set image.tag=v0.1.0
```

Scoped namespace install (see `docs/examples/helm-values-scoped.example.yaml`):

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  -f docs/examples/helm-values-scoped.example.yaml \
  --namespace my-app --create-namespace
```

Override image at install time without editing files:

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --set image.repository=ghcr.io/akeyless-community/akeyless-secrets-operator \
  --set image.tag=v0.1.0
```

---

## Example manifests

Apply the examples under `docs/examples/` after installing the operator:

```bash
kubectl apply -f docs/examples/akeyless-creds-secret.example.yaml
kubectl apply -f docs/examples/akeyless-secret-store.yaml
kubectl apply -f docs/examples/akeyless-secret.yaml
```

See also [rollout-restart-example.yaml](../docs/examples/rollout-restart-example.yaml) for rollout restart testing.

---

## Checklist before a public release

- [ ] Image built from a tagged git commit
- [ ] Image pushed to `ghcr.io/akeyless-community/akeyless-secrets-operator:<semver>`
- [ ] `deploy/charts/external-secrets/values.yaml` updated
- [ ] `Makefile` `IMAGE_REGISTRY` / `IMAGE_REPO` confirmed
- [ ] Helm chart `appVersion` in `Chart.yaml` matches the release tag
- [ ] Release notes mention required operator flags for Akeyless-only mode
- [ ] If the registry is private, document required `imagePullSecrets`

---

## Related docs

- [User guide](akeyless-secrets-operator-guide.md)
- [Example manifests](../docs/examples/)
- [Helm chart values](../deploy/charts/external-secrets/values.yaml)
- [Contributing — dev guide](contributing/devguide.md)
