#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p .github/workflows
cp docs/ci/ci.yml .github/workflows/ci.yml

echo "Refreshing GitHub auth with workflow scope (browser prompt may appear)..."
gh auth refresh -h github.com -s workflow

git add .github/workflows/ci.yml
if git diff --cached --quiet; then
  echo "CI workflow already committed."
else
  git commit -m "Add basic CI workflow"
fi

git push origin main
echo "CI enabled: https://github.com/akeyless-community/akeyless-secrets-operator/actions"
