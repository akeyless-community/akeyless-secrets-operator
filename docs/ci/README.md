# CI setup

Basic CI is defined in `docs/ci/ci.yml`. Copy it to `.github/workflows/ci.yml` to activate:

```bash
mkdir -p .github/workflows
cp docs/ci/ci.yml .github/workflows/ci.yml
```

GitHub requires the **`workflow` OAuth scope** to push workflow files via git:

```bash
gh auth refresh -h github.com -s workflow
git add .github/workflows/ci.yml
git commit -m "Add basic CI workflow"
git push origin main
```

## What CI runs

- `go test ./... -race` in `providers/v1/akeyless`
- `go build -tags akeyless` for linux/amd64

Upstream-only workflows are archived under `.github/upstream-workflows/`.
