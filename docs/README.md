# Documentation

Start here for the **Akeyless Secrets Operator**. Upstream External Secrets Operator (ESO) pages under `introduction/`, `provider/` (except `akeyless.md`), `api/`, and `guides/` are inherited from the fork and describe legacy ESO resources — use the Akeyless docs below instead.

## For operators (install and run)

| Document | What it covers |
|----------|----------------|
| [Getting Started](getting-started.md) | **Start here** — install from GHCR, first secret, troubleshooting |
| [User Guide](akeyless-secrets-operator-guide.md) | Full API reference, sync policies, Helm values, migration |
| [Install and publish](image-publishing.md) | OCI install (default), upgrades, build-from-source (advanced) |
| [Akeyless provider](provider/akeyless.md) | Provider-specific options (`ignoreCache`, auth, Gateway) |
| [Example manifests](examples/) | Copy-paste YAML for store, secret, dynamic secret |

### Quick install

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --create-namespace
```

Check [GitHub Releases](https://github.com/akeyless-community/akeyless-secrets-operator/releases) for the latest chart version.

## For contributors

| Document | What it covers |
|----------|----------------|
| [CONTRIBUTING.md](../CONTRIBUTING.md) | How to open PRs; merge policy |
| [Developer guide](contributing/devguide.md) | Local build, test, Helm chart |
| [Contributing process](contributing/process.md) | PR workflow |
| [CI](ci/README.md) | What CI runs on pull requests |
| [Roadmap](ROADMAP.md) | Planned features |

## For maintainers

| Document | What it covers |
|----------|----------------|
| [GHCR visibility](ghcr-visibility.md) | Package visibility and release publishing |
| [scripts/configure-github-protection.sh](../scripts/configure-github-protection.sh) | Branch protection for public repos |
| [SECURITY.md](../SECURITY.md) | Vulnerability reporting |
| [SECURITY_RESPONSE.md](../SECURITY_RESPONSE.md) | Incident response process |

## Helm chart

The chart lives at `deploy/charts/external-secrets/` (historical path from the ESO fork). Install instructions: [INSTALL.md](../deploy/charts/external-secrets/INSTALL.md).
