#!/usr/bin/env bash
# Configure GitHub repository protections for akeyless-community public repos.
#
# Enforces team-only merges on the default branch: required PR reviews,
# CI checks, and push restrictions to the merge team. CODEOWNERS may still
# auto-request reviewers but is not required to merge unless enabled.
#
# Run AFTER the repository is public. Some features are unavailable on
# private repos in the akeyless-community org (GitHub Free plan).
#
# Usage:
#   ./scripts/configure-github-protection.sh
#   ./scripts/configure-github-protection.sh --dry-run
#   OWNER=akeyless-community REPO=my-repo MERGE_TEAM=cs-admin ./scripts/configure-github-protection.sh
set -euo pipefail

OWNER="${OWNER:-akeyless-community}"
REPO="${REPO:-akeyless-secrets-operator}"
BRANCH="${BRANCH:-main}"
MERGE_TEAM="${MERGE_TEAM:-cs-admin}"
REVIEW_TEAM="${REVIEW_TEAM:-security}"
REQUIRED_REVIEWS="${REQUIRED_REVIEWS:-1}"
REQUIRE_CODE_OWNER_REVIEWS="${REQUIRE_CODE_OWNER_REVIEWS:-false}"
CI_CHECK="${CI_CHECK:-test-and-build}"
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
echo "Merge team: @${OWNER}/${MERGE_TEAM} | Reviews required: ${REQUIRED_REVIEWS} | CODEOWNERS required: ${REQUIRE_CODE_OWNER_REVIEWS}"
echo

visibility="$(gh api "repos/${OWNER}/${REPO}" --jq .visibility)"
if [[ "$visibility" != "public" ]]; then
  echo "ERROR: repository visibility is '${visibility}'."
  echo "Make the repository public first, then re-run this script."
  echo
  echo "  gh repo edit ${OWNER}/${REPO} --visibility public"
  exit 1
fi

echo "1/6 Granting team access to the repository..."
for team in "$MERGE_TEAM" "$REVIEW_TEAM"; do
  permission="maintain"
  if [[ "$team" == "$REVIEW_TEAM" && "$team" != "$MERGE_TEAM" ]]; then
    permission="triage"
  fi
  if run gh api \
    --method PUT \
    "orgs/${OWNER}/teams/${team}/repos/${OWNER}/${REPO}" \
    -f permission="$permission" 2>/dev/null; then
    echo "  @${OWNER}/${team} → ${permission}"
  else
    echo "  WARN: could not set @${OWNER}/${team} access (needs org admin: gh auth refresh -s admin:org)"
    echo "        Manually: Settings → Collaborators and teams → add @${OWNER}/${team} with ${permission}"
  fi
done

echo "2/6 Tightening GitHub Actions permissions..."
run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/actions/permissions" \
  --input - <<'EOF'
{
  "enabled": true,
  "allowed_actions": "selected"
}
EOF

run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/actions/permissions/selected-actions" \
  --input - <<'EOF'
{
  "github_owned_allowed": true,
  "verified_allowed": true
}
EOF

run gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}/actions/permissions/workflow" \
  --input - <<'EOF'
{
  "default_workflow_permissions": "read",
  "can_approve_pull_request_reviews": false
}
EOF

echo "3/6 Enabling security analysis features..."
run gh api \
  --method PATCH \
  -H "Accept: application/vnd.github+json" \
  "repos/${OWNER}/${REPO}" \
  --input - <<'EOF'
{
  "security_and_analysis": {
    "dependency_graph": { "status": "enabled" },
    "dependabot_alerts": { "status": "enabled" },
    "secret_scanning": { "status": "enabled" },
    "secret_scanning_push_protection": { "status": "enabled" },
    "dependabot_security_updates": { "status": "enabled" }
  }
}
EOF

echo "4/6 Applying branch protection on ${BRANCH} (team-only push + required reviews)..."
protection_payload="$(cat <<EOF
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "${CI_CHECK}"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": ${REQUIRE_CODE_OWNER_REVIEWS},
    "required_approving_review_count": ${REQUIRED_REVIEWS},
    "require_last_push_approval": true
  },
  "required_conversation_resolution": true,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "restrictions": {
    "users": [],
    "teams": ["${MERGE_TEAM}"],
    "apps": []
  }
}
EOF
)"
if [[ "$DRY_RUN" -eq 1 ]]; then
  printf 'DRY RUN: apply branch protection\n%s\n' "$protection_payload"
else
  printf '%s' "$protection_payload" | gh api \
    --method PUT \
    -H "Accept: application/vnd.github+json" \
    "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" \
    --input -
fi

echo "5/6 Upserting repository ruleset..."
ruleset_id="$(gh api "repos/${OWNER}/${REPO}/rulesets" --jq '.[] | select(.name == "Protect '"${BRANCH}"'") | .id' 2>/dev/null | head -1 || true)"
ruleset_payload="$(cat <<EOF
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
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": ${REQUIRED_REVIEWS},
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": ${REQUIRE_CODE_OWNER_REVIEWS},
        "require_last_push_approval": true,
        "required_review_thread_resolution": true
      }
    }
  ]
}
EOF
)"
if [[ -n "$ruleset_id" ]]; then
  echo "Updating existing ruleset (id: ${ruleset_id})..."
  if [[ "$DRY_RUN" -eq 1 ]]; then
    printf 'DRY RUN: update ruleset %s\n' "$ruleset_id"
  else
    printf '%s' "$ruleset_payload" | gh api \
      --method PUT \
      -H "Accept: application/vnd.github+json" \
      "repos/${OWNER}/${REPO}/rulesets/${ruleset_id}" \
      --input -
  fi
else
  run gh api \
    --method POST \
    -H "Accept: application/vnd.github+json" \
    "repos/${OWNER}/${REPO}/rulesets" \
    --input - <<<"$ruleset_payload"
fi

echo "6/6 Verifying repository settings..."
gh api "repos/${OWNER}/${REPO}" --jq '{visibility, delete_branch_on_merge, allow_forking}'
gh api "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" --jq '{
  required_approving_review_count: .required_pull_request_reviews.required_approving_review_count,
  require_code_owner_reviews: .required_pull_request_reviews.require_code_owner_reviews,
  enforce_admins: .enforce_admins.enabled,
  restrictions: .restrictions
}' || true

cat <<EOF

Done.

Team merge policy on ${BRANCH}:
  - Direct pushes: only @${OWNER}/${MERGE_TEAM}
  - Merges: PR required, ${REQUIRED_REVIEWS}+ approving review(s) from any reviewer with write access
  - CI: ${CI_CHECK} must pass
  - Fork PRs: external contributors cannot merge (no write access)

Apply this standard to every new public repo:
  OWNER=${OWNER} REPO=<repo-name> ./scripts/configure-github-protection.sh

See docs/repository-standards.md for the full checklist.

Manual follow-ups (Settings UI):
  - Settings → General → Pull Requests: enable "Automatically delete head branches"
  - Settings → General → Features: disable Wiki if unused
  - Add .github/CODEOWNERS to auto-request reviews (optional; not required to merge by default)
EOF
