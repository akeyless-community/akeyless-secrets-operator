# Container image and Helm chart — publish under akeyless-community

All official artifacts live under the **[akeyless-community](https://github.com/akeyless-community)** GitHub organization:

| Artifact | Location |
|----------|----------|
| Source | `github.com/akeyless-community/akeyless-secrets-operator` |
| Container image | `ghcr.io/akeyless-community/akeyless-secrets-operator:<tag>` |
| Helm chart (OCI) | `oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator` |

No separate Docker Hub account is required.

---

## Install (after a release is published)

```bash
helm upgrade --install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.0 \
  --namespace akeyless-secrets-operator --create-namespace \
  --set installCRDs=true
```

The chart defaults to `ghcr.io/akeyless-community/akeyless-secrets-operator` with the chart `appVersion` as the image tag.

Install from a git checkout (before a release, or for development):

```bash
git clone https://github.com/akeyless-community/akeyless-secrets-operator.git
cd akeyless-secrets-operator

helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  --namespace akeyless-secrets-operator --create-namespace \
  --set installCRDs=true
```

Scoped namespace install (see `docs/examples/helm-values-scoped.example.yaml`):

```bash
helm upgrade --install akeyless-secrets-operator \
  ./deploy/charts/external-secrets \
  -f docs/examples/helm-values-scoped.example.yaml \
  --namespace my-app --create-namespace
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

## Publish a release (maintainers)

Releases are automated by [`.github/workflows/release.yml`](../.github/workflows/release.yml).

### 1. Prepare the chart version

Ensure `deploy/charts/external-secrets/Chart.yaml` matches the release:

```yaml
version: "0.1.0"      # Helm chart version (no leading v)
appVersion: "v0.1.0"  # Container image tag
```

### 2. Create a GitHub Release

Create a release tagged `v0.1.0` on GitHub. The workflow will:

1. Build `linux/amd64` and `linux/arm64` binaries
2. Push `ghcr.io/akeyless-community/akeyless-secrets-operator:v0.1.0` and `:latest`
3. Package and push the Helm chart to `oci://ghcr.io/akeyless-community/charts`

Or trigger manually: **Actions → Release → Run workflow** with tag `v0.1.0`.

### 3. Make GHCR packages public

After the first publish, set each package to **Public** in GitHub:

**Organization → Packages → akeyless-secrets-operator / charts/akeyless-secrets-operator → Package settings → Change visibility**

Clusters must be able to pull without credentials.

---

## Build locally

```bash
ARCH=amd64 make build-amd64

docker build --platform linux/amd64 -f Dockerfile \
  -t ghcr.io/akeyless-community/akeyless-secrets-operator:dev .

# Push (requires GHCR write access)
echo "$GITHUB_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USER --password-stdin
docker push ghcr.io/akeyless-community/akeyless-secrets-operator:dev
```

Or use Make defaults:

```bash
ARCH=amd64 IMAGE_TAG=dev make docker.build docker.push
```

For arm64 clusters, replace `amd64` with `arm64`.

---

## Files to update when changing image coordinates

| File | What to change |
|------|----------------|
| [`deploy/charts/external-secrets/values.yaml`](../deploy/charts/external-secrets/values.yaml) | `image.repository` (primary Helm default) |
| [`deploy/charts/external-secrets/Chart.yaml`](../deploy/charts/external-secrets/Chart.yaml) | `appVersion` (default image tag when `image.tag` is empty) |
| [`Makefile`](../Makefile) | `IMAGE_REGISTRY` and `IMAGE_REPO` for `make docker.build` |
| [`docs/examples/helm-values-scoped.example.yaml`](../docs/examples/helm-values-scoped.example.yaml) | Example overrides |

---

## Checklist before a public release

- [ ] `Chart.yaml` `version` and `appVersion` updated
- [ ] GitHub Release created with matching `v*` tag
- [ ] Release workflow completed successfully
- [ ] GHCR packages set to **Public**
- [ ] `helm show chart oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator --version X.Y.Z` works
- [ ] Test install from OCI on a cluster

---

## Related docs

- [User guide](akeyless-secrets-operator-guide.md)
- [Example manifests](../docs/examples/)
- [Helm chart values](../deploy/charts/external-secrets/values.yaml)
