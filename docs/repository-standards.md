# Repository standards — akeyless-community public repos

Every **public** repository in the `akeyless-community` org must follow this baseline before we announce or accept external contributions.

## Goals

- **Public read, team write** — anyone can fork and open PRs; only org teams merge to `main`
- **Review before merge** — no self-merge without approval; CODEOWNERS enforced
- **CI gate** — required checks must pass
- **Supply chain hygiene** — secret scanning, dependency alerts, restricted Actions permissions

## Teams

| Team | Role |
|------|------|
| `@akeyless-community/cs-admin` | Default merge team — maintain access, CODEOWNERS for most paths |
| `@akeyless-community/security` | Security-sensitive paths (workflows, deploy, CRDs) |

## One-command setup

From the repository root (after the repo is **public**):

```bash
./scripts/configure-github-protection.sh
```

For another repo:

```bash
OWNER=akeyless-community REPO=<repo-name> ./scripts/configure-github-protection.sh
```

Dry run:

```bash
./scripts/configure-github-protection.sh --dry-run
```

### What the script configures

1. **Team access** — `cs-admin` (maintain), `security` (triage)
2. **Actions** — selected actions only; workflows default to read-only token
3. **Security analysis** — dependency graph, Dependabot, secret scanning + push protection
4. **Branch protection on `main`**:
   - Required PR with **1+ approving review**
   - **CODEOWNERS** review required
   - **Re-approval** after new commits
   - **Conversation resolution** required
   - **CI check** must pass (`test-and-build` by default)
   - **Push restricted** to `@akeyless-community/cs-admin` only
   - **Admins included** in rules (`enforce_admins: true`)
5. **Ruleset** — no force-push; pull request rules mirrored

## Required files in each repo

| File | Purpose |
|------|---------|
| `.github/CODEOWNERS` | Auto-request reviews from `cs-admin` / `security` |
| `.github/workflows/ci.yml` | CI job named `test-and-build` (or set `CI_CHECK` env var) |
| `scripts/configure-github-protection.sh` | Copy from this repo or symlink |
| `SECURITY.md` | Vulnerability reporting |
| `docs/repository-standards.md` | This document (optional but recommended) |

### Minimal CODEOWNERS template

```
# Security-sensitive
.github/workflows/  @akeyless-community/security
deploy/             @akeyless-community/security

# Default owners
*                   @akeyless-community/cs-admin
```

## New public repo checklist

- [ ] Repository visibility → **Public**
- [ ] Run `./scripts/configure-github-protection.sh`
- [ ] Confirm `@akeyless-community/cs-admin` has **Maintain** on the repo
- [ ] Confirm `@akeyless-community/security` has **Triage** (or Maintain for security-heavy repos)
- [ ] Add `.github/CODEOWNERS`
- [ ] CI workflow exposes required check name (default: `test-and-build`)
- [ ] Enable **Automatically delete head branches** (Settings → General → Pull Requests)
- [ ] Run `gitleaks detect --source . --redact` before first public push
- [ ] No real credentials in docs/examples or git history

## Who can merge?

| Actor | Can merge to `main`? |
|-------|----------------------|
| External contributor (fork PR) | **No** — needs team member approval + CI |
| Org member not in `cs-admin` | **No** — unless added to merge team or given bypass (avoid) |
| `cs-admin` team member | **Yes** — after review + CI (cannot self-approve own PR if last-push approval is on) |
| Admin with bypass | **Avoid** — `enforce_admins: true` applies rules to admins too |

## Customization

Environment variables for `configure-github-protection.sh`:

| Variable | Default | Description |
|----------|---------|-------------|
| `OWNER` | `akeyless-community` | GitHub org |
| `REPO` | `akeyless-secrets-operator` | Repository name |
| `BRANCH` | `main` | Protected branch |
| `MERGE_TEAM` | `cs-admin` | Team slug allowed to push to `main` |
| `REVIEW_TEAM` | `security` | Secondary team granted triage access |
| `REQUIRED_REVIEWS` | `1` | Minimum approving reviews |
| `CI_CHECK` | `test-and-build` | Required status check context |

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Script cannot set team access | Org admin runs `gh auth refresh -h github.com -s admin:org`, or set teams manually in Settings |
| `restrictions` API fails | Repo must be public; org may need GitHub Team plan for some private-repo features |
| PR merges without review | Re-run script; confirm `required_approving_review_count` ≥ 1 |
| CODEOWNERS not requested | File must exist on `main`; enable `require_code_owner_reviews` |
