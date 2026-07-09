# Repository standards — akeyless-community public repos

Every **public** repository in the `akeyless-community` org must follow this baseline before we announce or accept external contributions.

## Why does this live in `akeyless-secrets-operator`?

This repository is the **reference implementation** for org-wide standards — not because the script is operator-specific. The operator was the first public community repo, so it hosts:

| Artifact | Purpose |
|----------|---------|
| `scripts/configure-github-protection.sh` | Canonical script — copy to new repos or run remotely with `REPO=<name>` |
| `docs/repository-standards.md` | Policy doc — copy to new repos |
| `.github/CODEOWNERS` | Template for team review routing |

**For a new public repo**, either:

1. **Copy** `scripts/configure-github-protection.sh` and `docs/repository-standards.md` into the new repo, then run locally, or
2. **Run remotely** from a clone of this repo:
   ```bash
   OWNER=akeyless-community REPO=<new-repo> ./scripts/configure-github-protection.sh
   ```

The script is **idempotent** — safe to re-run on repos that are already public (including this one) to refresh settings after policy changes.

> **Note:** GitHub only allows push restrictions and some protection features on **public** repositories in our org tier. The prerequisite is that the **target** repo visibility is public — not that you run the script only once at go-live.

## Goals

- **Public read, team write** — anyone can fork and open PRs; only org teams merge to `main`
- **Review before merge** — PR required with approval from `@akeyless-community/cs-admin` (not arbitrary public accounts)
- **CI gate** — required checks must pass
- **Supply chain hygiene** — secret scanning, dependency alerts, restricted Actions permissions

## Teams

| Team | Role |
|------|------|
| `@akeyless-community/cs-admin` | Default merge team — maintain access, default CODEOWNERS for most paths |
| `@akeyless-community/security` | Security-sensitive paths (workflows, deploy, CRDs) — auto-requested, not required to merge |

## One-command setup

From the repository root (target repo must be **public**):

```bash
./scripts/configure-github-protection.sh
```

For another repo (run from this clone):

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
4. **Branch protection on `main`** (classic layer):
   - Required PR with **1+ approving review**
   - **Re-approval** after new commits; **conversation resolution** required
   - **CI check** must pass (`test-and-build` by default)
   - **Push restricted** to `@akeyless-community/cs-admin` only
   - **Admins included** in rules (`enforce_admins: true`)
   - **CODEOWNERS not required** to merge by default (`REQUIRE_CODE_OWNER_REVIEWS=false`)
5. **Ruleset on `main`** (self-sufficient layer — does not rely on classic protection alone):
   - `non_fast_forward` — no force-push
   - `update` — block direct pushes (PR path required)
   - `pull_request` — reviews + **required reviewer team** (`cs-admin` for all paths)
   - `required_status_checks` — same CI gate as branch protection
6. **Verification** — script **exits with error** if branch protection or ruleset is incomplete after apply

## Required files in each repo

| File | Purpose |
|------|---------|
| `.github/CODEOWNERS` | **Required on `main`** — auto-requests reviewers; mandatory only if `REQUIRE_CODE_OWNER_REVIEWS=true` |
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
- [ ] Add `.github/CODEOWNERS` **before** running the protection script
- [ ] Run `./scripts/configure-github-protection.sh` (fails loudly if protection incomplete)
- [ ] CI workflow exposes required check name (default: `test-and-build`)
- [ ] Enable **Automatically delete head branches** (Settings → General → Pull Requests)
- [ ] Run `gitleaks detect --source . --redact` before first public push
- [ ] No real credentials in docs/examples or git history

## Who can merge?

| Actor | Can merge to `main`? |
|-------|----------------------|
| External contributor (fork PR) | **No** — needs `cs-admin` team approval + CI |
| Org member not in `cs-admin` | **No** — ruleset `required_reviewers` blocks merge without team approval |
| `cs-admin` team member | **Yes** — after team review + CI (re-approval required on new commits) |
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
| `REQUIRE_CODE_OWNER_REVIEWS` | `false` | Set `true` to require CODEOWNERS approval before merge |
| `CI_CHECK` | `test-and-build` | Required status check context |

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Script exits after step 4 | Branch protection failed to apply — check org admin scope / plan tier |
| Ruleset incomplete error | Re-run script; confirm `required_status_checks` and `update` rules present |
| PR merges without review | Re-run script; ruleset `required_reviewers` must include merge team |
| CODEOWNERS not requested | File must exist on `main` before protection run |
| Want mandatory CODEOWNERS | Set `REQUIRE_CODE_OWNER_REVIEWS=true` (script errors if CODEOWNERS missing) |
