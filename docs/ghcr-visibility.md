# GHCR package visibility (maintainers)

Published installs pull the operator image and Helm chart from GitHub Container Registry (GHCR). Packages are **private by default** when first created by CI. They must be **public** for users to install without credentials (same as External Secrets Operator on `ghcr.io/external-secrets`).

## Packages to publish

| Package | URL pattern |
|---------|-------------|
| Container image | `ghcr.io/akeyless-community/akeyless-secrets-operator` |
| Helm chart (OCI) | `ghcr.io/akeyless-community/charts/akeyless-secrets-operator` |

## Make packages public

You need **admin** on the `akeyless-community` org or **maintain** permission on the repository.

### Option 1 — From each package page

1. Open [github.com/orgs/akeyless-community/packages](https://github.com/orgs/akeyless-community/packages)
2. Click the package (e.g. `akeyless-secrets-operator` or `charts/akeyless-secrets-operator`)
3. Click **Package settings** (right side or gear icon)
4. Scroll to **Danger Zone** → **Change visibility**
5. Select **Public** and confirm

Repeat for both the **container** and **Helm chart** packages.

### Option 2 — Link package to the repository

If the package was created by GitHub Actions but does not appear under the repo:

1. Open the package → **Package settings**
2. Under **Manage Actions access** / **Connect repository**, link `akeyless-community/akeyless-secrets-operator`
3. Then set visibility to **Public**

### Option 3 — Org default for new packages

Org owners can set defaults so future packages are public:

1. **Organization settings** → **Packages** → **Package creation**
2. Configure default visibility for organization members (optional; does not retroactively change existing packages)

## Verify public access

Without logging in to GHCR:

```bash
# Image manifest (should return JSON, not 401/403)
curl -sI "https://ghcr.io/v2/akeyless-community/akeyless-secrets-operator/manifests/latest" | head -5

# Helm chart metadata (after a release)
helm show chart oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator --version 0.1.0
```

A `403 Forbidden` on `helm install oci://...` almost always means the chart package is still private.

## Publish chart 0.1.1 and mark GitHub release as latest

The container image `v0.1.1` can exist while the Helm chart is still `0.1.0` if the release workflow ran before [PR #34](https://github.com/akeyless-community/akeyless-secrets-operator/pull/34) was merged.

1. **Merge PR #34** (GHCR image defaults + chart version bump in release workflow)
2. **Actions** → **Release** → **Run workflow** → input tag `v0.1.1` → wait for success
3. Verify chart: `helm show chart oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator --version 0.1.1`
4. **Releases** → create or edit `v0.1.1` → check **Set as the latest release**

```bash
gh release create v0.1.1 --title v0.1.1 --notes "Public OCI install" --latest
```

## After flipping visibility

1. Cut or re-run a release so image + chart are published (GitHub → **Releases** → publish, or **Actions** → **Release** → **Run workflow** with tag e.g. `v0.1.1`)
2. Confirm install works:

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace
```

## Private registry (not recommended for community installs)

If packages stay private, every cluster needs:

```bash
kubectl create secret docker-registry ghcr-credentials \
  --docker-server=ghcr.io \
  --docker-username=<github-user> \
  --docker-password=<github-pat-with-read:packages> \
  -n akeyless-secrets-operator
```

And Helm values: `imagePullSecrets[0].name=ghcr-credentials`

This is **not** the ESO-like experience; prefer public packages for the community chart.
