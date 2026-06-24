# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| latest `main` | Yes |
| released tags | Yes, per [GitHub Releases](https://github.com/akeyless-community/akeyless-secrets-operator/releases) |

## Reporting a vulnerability

**Do not open a public GitHub issue** for security vulnerabilities.

Email **security@akeyless.io** with:

- Description of the issue
- Steps to reproduce
- Impact assessment (if known)
- Affected version or commit SHA (if known)

We aim to acknowledge reports within 3 business days.

## Vulnerability management

We monitor for dependency and workflow issues via:

1. GitHub Dependabot alerts
2. GitHub secret scanning and push protection
3. CodeQL static analysis (`.github/workflows/codeql.yml`)
4. Dependency review on pull requests (`.github/workflows/dependency-review.yml`)

## Helm chart security

The Helm chart is intended for general-purpose deployments. Review and harden the default values for your environment (RBAC scope, webhook TLS, network policies, admission controls, and credential storage).

Misconfiguration of the chart in your cluster is outside the scope of this project's support policy, even when it leads to a security incident.

## Security incident response

For maintainers, follow [SECURITY_RESPONSE.md](SECURITY_RESPONSE.md).
