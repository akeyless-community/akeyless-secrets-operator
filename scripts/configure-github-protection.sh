#!/usr/bin/env bash
# Configure GitHub repository protections for akeyless-secrets-operator.
#
# Run AFTER the repository is public. Branch protection and several security
# features are unavailable on private repos in the akeyless-community org
# (GitHub Free plan).
#
# Usage:
#   ./scripts/configure-github-protection.sh
#   ./scripts/configure-github-protection.sh --dry-run
set -euo pipefail

OWNER="akeyless-community"
REPO="akeyless-secrets-operator"
BRANCH="main"
DRY_RUN=0

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=1
fi

run() {
  if [[ "$DRY_RUN" -eq 1 ]]; then
    printf 'DRY RUN: %q' "$@"
    printf '\n'
  else
    "$@"
  fi
}

echo "Target: ${OWNER}/${REPO} (branch: ${BRANCH})"
echo

visibility="$(gh api "repos/${OWNER}/${REPO}" --jq .visibility)"
if [[ "$visibility" != "public" ]]; then
  echo "ERROR: repository visibility is '${visibility}'."
  echo "Make the repository public first, then re-run this script."
  echo
  echo "  gh repo edit ${OWNER}/${REPO} --visibility public"
  exit 1
fi

echo "1/5 Tightening GitHub Actions permissions..."
run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/actions/permissions" \
  -f "enabled=true" \
  -f "allowed_actions=selected" \
  -f "github_owned_allowed=true" \
  -f "verified_allowed=true"

run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/actions/permissions/workflow" \
  -f "default_workflow_permissions=read" \
  -f "can_approve_pull_request_reviews=false"

echo "2/5 Enabling security analysis features..."
run gh api \
  --method PATCH \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}" \
  --input - <<'EOF'
{
  "security_and_analysis": {
    "advanced_security": { "status": "enabled" },
    "secret_scanning": { "status": "enabled" },
    "secret_scanning_push_protection": { "status": "enabled" },
    "dependabot_security_updates": { "status": "enabled" }
  }
}
EOF

echo "3/5 Applying branch protection on ${BRANCH}..."
run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "test-and-build",
      "dependency-review"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "required_approving_review_count": 1,
    "require_last_push_approval": true
  },
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "restrictions": null
}
EOF

echo "4/5 Creating repository ruleset (signed commits + no force-push)..."
run gh api \
  --method POST \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/rulesets" \
  --input - <<EOF
{
  "name": "Protect ${BRANCH}",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/${BRANCH}"],
      "exclude": []
    }
  },
  "rules": [
    { "type": "non_fast_forward" },
    { "type": "required_signatures" }
  ]
}
EOF

echo "5/5 Verifying repository settings..."
gh api "repos/${OWNER}/${REPO}" --jq '{visibility, delete_branch_on_merge, allow_forking}'
gh api "repos/${OWNER}/${REPO}/actions/permissions" --jq .
gh api "repos/${OWNER}/${REPO}/actions/permissions/workflow" --jq .
gh api "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" --jq '{required_pull_request_reviews, enforce_admins, required_status_checks}' || true

cat <<'EOF'

Done.

Manual follow-ups (Settings UI):
  - Settings → General → Pull Requests: enable "Automatically delete head branches"
  - Settings → General → Features: disable Wiki if unused
  - Settings → Collaborators and teams: confirm @akeyless-community/cs-admin and @akeyless-community/security have appropriate access
  - Re-run CodeQL once (Actions → CodeQL Advanced → Run workflow) to confirm GHAS is active

Pre-public hygiene (do before flipping visibility):
  - Run: gitleaks detect --source . --redact
  - Review docs/examples/ for placeholder credentials before publishing
  - Confirm no real credentials in git history
EOF
