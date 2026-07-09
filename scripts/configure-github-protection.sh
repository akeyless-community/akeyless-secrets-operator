#!/usr/bin/env bash
# Configure GitHub repository protections for akeyless-community public repos.
#
# Enforces team-only merges on the default branch via classic branch protection
# AND a self-sufficient ruleset (CI gate, PR reviews, push restriction).
# CODEOWNERS auto-requests reviewers when present; optional mandatory flag.
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

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed."
}

require_cmd gh
require_cmd jq

if ! [[ "$REQUIRED_REVIEWS" =~ ^[0-9]+$ ]]; then
  die "REQUIRED_REVIEWS must be a non-negative integer (got: ${REQUIRED_REVIEWS})"
fi

require_code_owner_bool=false
if [[ "$REQUIRE_CODE_OWNER_REVIEWS" == "true" ]]; then
  require_code_owner_bool=true
elif [[ "$REQUIRE_CODE_OWNER_REVIEWS" != "false" ]]; then
  die "REQUIRE_CODE_OWNER_REVIEWS must be 'true' or 'false' (got: ${REQUIRE_CODE_OWNER_REVIEWS})"
fi

echo "Target: ${OWNER}/${REPO} (branch: ${BRANCH})"
echo "Merge team: @${OWNER}/${MERGE_TEAM} | Reviews required: ${REQUIRED_REVIEWS} | CODEOWNERS required: ${require_code_owner_bool}"
echo

visibility="$(gh api "repos/${OWNER}/${REPO}" --jq .visibility)"
if [[ "$visibility" != "public" ]]; then
  die "repository visibility is '${visibility}'. Make the repository public first:
  gh repo edit ${OWNER}/${REPO} --visibility public"
fi

if [[ "$require_code_owner_bool" == "true" ]]; then
  if ! gh api "repos/${OWNER}/${REPO}/contents/.github/CODEOWNERS?ref=${BRANCH}" >/dev/null 2>&1; then
    die ".github/CODEOWNERS is missing on ${BRANCH}. require_code_owner_reviews would be a no-op.
  Add CODEOWNERS before enabling REQUIRE_CODE_OWNER_REVIEWS=true."
  fi
fi

merge_team_id="$(gh api "orgs/${OWNER}/teams/${MERGE_TEAM}" --jq .id 2>/dev/null || true)"
if [[ -z "$merge_team_id" || "$merge_team_id" == "null" ]]; then
  die "could not resolve team id for @${OWNER}/${MERGE_TEAM} (needs org read: gh auth refresh -s read:org)"
fi

build_protection_payload() {
  jq -n \
    --arg ci_check "$CI_CHECK" \
    --arg merge_team "$MERGE_TEAM" \
    --argjson required_reviews "$REQUIRED_REVIEWS" \
    --argjson require_code_owner_reviews "$require_code_owner_bool" \
    '{
      required_status_checks: {
        strict: true,
        contexts: [$ci_check]
      },
      enforce_admins: true,
      required_pull_request_reviews: {
        dismiss_stale_reviews: true,
        require_code_owner_reviews: $require_code_owner_reviews,
        required_approving_review_count: $required_reviews,
        require_last_push_approval: true
      },
      required_conversation_resolution: true,
      required_linear_history: true,
      allow_force_pushes: false,
      allow_deletions: false,
      block_creations: false,
      restrictions: {
        users: [],
        teams: [$merge_team],
        apps: []
      }
    }'
}

build_ruleset_payload() {
  jq -n \
    --arg name "Protect ${BRANCH}" \
    --arg branch "$BRANCH" \
    --arg ci_check "$CI_CHECK" \
    --argjson required_reviews "$REQUIRED_REVIEWS" \
    --argjson require_code_owner_reviews "$require_code_owner_bool" \
    --argjson merge_team_id "$merge_team_id" \
    '{
      name: $name,
      target: "branch",
      enforcement: "active",
      conditions: {
        ref_name: {
          include: ["refs/heads/" + $branch],
          exclude: []
        }
      },
      rules: [
        { type: "non_fast_forward" },
        {
          type: "update",
          parameters: {
            update_allows_fetch_and_merge: true
          }
        },
        {
          type: "pull_request",
          parameters: {
            required_approving_review_count: $required_reviews,
            dismiss_stale_reviews_on_push: true,
            require_code_owner_review: $require_code_owner_reviews,
            require_last_push_approval: true,
            required_review_thread_resolution: true,
            required_reviewers: [
              {
                reviewer: {
                  id: $merge_team_id,
                  type: "Team"
                },
                minimum_approvals: $required_reviews,
                file_patterns: ["*"]
              }
            ]
          }
        },
        {
          type: "required_status_checks",
          parameters: {
            strict_required_status_checks_policy: true,
            required_status_checks: [
              { context: $ci_check }
            ]
          }
        }
      ]
    }'
}

verify_branch_protection() {
  local protection team_slugs ci_contexts review_count
  if ! protection="$(gh api "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" 2>/dev/null)"; then
    die "branch protection is not applied on ${BRANCH}.
  Classic protection carries push restrictions (teams) and CI requirements.
  Re-run with org admin scope or set teams manually in Settings."
  fi

  team_slugs="$(jq -r '[.restrictions.teams[]?.slug] | join(",")' <<<"$protection")"
  if [[ ",${team_slugs}," != *",${MERGE_TEAM},"* ]]; then
    die "branch protection push restriction missing team '${MERGE_TEAM}' (got: ${team_slugs:-none})"
  fi

  ci_contexts="$(jq -r '[.required_status_checks.contexts[]?] | join(",")' <<<"$protection")"
  if [[ ",${ci_contexts}," != *",${CI_CHECK},"* ]]; then
    die "branch protection missing required CI check '${CI_CHECK}' (got: ${ci_contexts:-none})"
  fi

  review_count="$(jq -r '.required_pull_request_reviews.required_approving_review_count // 0' <<<"$protection")"
  if [[ "$review_count" -lt "$REQUIRED_REVIEWS" ]]; then
    die "branch protection requires ${review_count} reviews; expected at least ${REQUIRED_REVIEWS}"
  fi

  echo "  branch protection: OK (team=${MERGE_TEAM}, ci=${CI_CHECK}, reviews>=${REQUIRED_REVIEWS})"
}

verify_ruleset() {
  local ruleset_id rules_json has_pr has_ci has_update
  ruleset_id="$(gh api "repos/${OWNER}/${REPO}/rulesets" --jq '.[] | select(.name == "Protect '"${BRANCH}"'") | .id' 2>/dev/null | head -1 || true)"
  if [[ -z "$ruleset_id" ]]; then
    die "ruleset 'Protect ${BRANCH}' was not created"
  fi

  rules_json="$(gh api "repos/${OWNER}/${REPO}/rulesets/${ruleset_id}" --jq '.rules')"
  has_pr="$(jq '[.[] | select(.type == "pull_request")] | length' <<<"$rules_json")"
  has_ci="$(jq '[.[] | select(.type == "required_status_checks")] | length' <<<"$rules_json")"
  has_update="$(jq '[.[] | select(.type == "update")] | length' <<<"$rules_json")"

  if [[ "$has_pr" -lt 1 || "$has_ci" -lt 1 || "$has_update" -lt 1 ]]; then
    die "ruleset ${ruleset_id} is incomplete (pull_request=${has_pr}, required_status_checks=${has_ci}, update=${has_update})"
  fi

  local ci_context team_reviewer
  ci_context="$(jq -r '.[] | select(.type=="required_status_checks") | .parameters.required_status_checks[0].context // empty' <<<"$rules_json")"
  if [[ "$ci_context" != "$CI_CHECK" ]]; then
    die "ruleset CI check is '${ci_context:-missing}'; expected '${CI_CHECK}'"
  fi

  team_reviewer="$(jq -r '.[] | select(.type=="pull_request") | .parameters.required_reviewers[0].reviewer.id // empty' <<<"$rules_json")"
  if [[ "$team_reviewer" != "$merge_team_id" ]]; then
    die "ruleset required reviewer team id is '${team_reviewer:-missing}'; expected '${merge_team_id}'"
  fi

  echo "  ruleset ${ruleset_id}: OK (pull_request + CI + update + team reviewer)"
}

echo "1/7 Granting team access to the repository..."
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

echo "2/7 Tightening GitHub Actions permissions..."
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

echo "3/7 Enabling security analysis features..."
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

echo "4/7 Applying branch protection on ${BRANCH}..."
protection_payload="$(build_protection_payload)"
if [[ "$DRY_RUN" -eq 1 ]]; then
  printf 'DRY RUN: apply branch protection\n%s\n' "$protection_payload"
else
  printf '%s' "$protection_payload" | gh api \
    --method PUT \
    -H "Accept: application/vnd.github+json" \
    "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" \
    --input -
fi

if [[ "$DRY_RUN" -eq 0 ]]; then
  verify_branch_protection
fi

echo "5/7 Upserting repository ruleset..."
ruleset_id="$(gh api "repos/${OWNER}/${REPO}/rulesets" --jq '.[] | select(.name == "Protect '"${BRANCH}"'") | .id' 2>/dev/null | head -1 || true)"
ruleset_payload="$(build_ruleset_payload)"
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
  if [[ "$DRY_RUN" -eq 1 ]]; then
    printf 'DRY RUN: create ruleset\n%s\n' "$ruleset_payload"
  else
    printf '%s' "$ruleset_payload" | gh api \
      --method POST \
      -H "Accept: application/vnd.github+json" \
      "repos/${OWNER}/${REPO}/rulesets" \
      --input -
  fi
fi

if [[ "$DRY_RUN" -eq 0 ]]; then
  verify_ruleset
fi

echo "6/7 Verifying CODEOWNERS (recommended)..."
if gh api "repos/${OWNER}/${REPO}/contents/.github/CODEOWNERS?ref=${BRANCH}" >/dev/null 2>&1; then
  echo "  .github/CODEOWNERS: present on ${BRANCH}"
else
  echo "  WARN: .github/CODEOWNERS missing on ${BRANCH} — auto-review requests disabled"
  echo "        Add CODEOWNERS (see docs/repository-standards.md)"
fi

echo "7/7 Summary..."
gh api "repos/${OWNER}/${REPO}" --jq '{visibility, delete_branch_on_merge, allow_forking}'
gh api "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" --jq '{
  required_approving_review_count: .required_pull_request_reviews.required_approving_review_count,
  require_code_owner_reviews: .required_pull_request_reviews.require_code_owner_reviews,
  enforce_admins: .enforce_admins.enabled,
  ci_checks: .required_status_checks.contexts,
  push_teams: [.restrictions.teams[].slug]
}' 2>/dev/null || die "final verification: branch protection missing"

cat <<EOF

Done.

Team merge policy on ${BRANCH} (enforced in branch protection AND ruleset):
  - Direct pushes: only @${OWNER}/${MERGE_TEAM} (classic protection) + update rule (ruleset)
  - Merges: PR required with ${REQUIRED_REVIEWS}+ approval(s) from @${OWNER}/${MERGE_TEAM}
  - CI: ${CI_CHECK} must pass (classic protection + ruleset)
  - Fork PRs: external contributors cannot merge (no write access)

Apply this standard to every new public repo:
  OWNER=${OWNER} REPO=<repo-name> ./scripts/configure-github-protection.sh

See docs/repository-standards.md for the full checklist.

Manual follow-ups (Settings UI):
  - Settings → General → Pull Requests: enable "Automatically delete head branches"
  - Settings → General → Features: disable Wiki if unused
  - Ensure .github/CODEOWNERS exists to auto-request reviewers
EOF
