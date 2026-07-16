# GHCR packages and releases (maintainers)

Published installs use GitHub Container Registry (GHCR):

| Package | URL |
|---------|-----|
| Container image | `ghcr.io/akeyless-community/akeyless-secrets-operator` |
| Helm chart (OCI) | `oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator` |

Packages are **public** — users install without registry credentials.

## Cut a new release

1. **Actions** → **Release** → **Run workflow** → input tag (e.g. `v0.1.2`)
2. Or create a [GitHub Release](https://github.com/akeyless-community/akeyless-secrets-operator/releases) with the same tag
3. Verify:
   ```bash
   helm show chart oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator --version 0.1.2
   docker pull ghcr.io/akeyless-community/akeyless-secrets-operator:v0.1.2
   ```
4. Update docs if the default `--version` in examples should change

The release workflow builds multi-arch images, bumps chart `version` / `appVersion` from the tag, and pushes the OCI chart.

## Make packages public (new repository)

GHCR packages are **private by default** when first created. For a **new** public repo:

1. Open [github.com/orgs/akeyless-community/packages](https://github.com/orgs/akeyless-community/packages)
2. For each package (image + chart): **Package settings** → **Change visibility** → **Public**
3. Link the package to the repository if prompted

Verify without logging in:

```bash
helm show chart oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator --version 0.1.1
```

A `403 Forbidden` on `helm install oci://...` means the chart package is still private.

## Private registry fallback (not recommended)

If packages must stay private, clusters need `imagePullSecrets` and authenticated `helm registry login`. Prefer public packages for the community install path documented in [getting-started.md](getting-started.md).
