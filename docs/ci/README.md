# CI

Pull requests to `main` run the workflow in [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).

## What runs on every PR

| Job | What it does |
|-----|----------------|
| `test-and-build` | `go test -race` in `providers/v1/akeyless`; `go build -tags akeyless` for linux/amd64 |

Branch protection requires the `test-and-build` check to pass before merge.

## Archived upstream workflows

Full upstream ESO workflows (multi-provider e2e, release automation, etc.) are kept under [`.github/upstream-workflows/`](../../.github/upstream-workflows/) for reference. They are **not** active in this repository.

## Bootstrap template

[`ci.yml`](ci.yml) in this directory is the source template used when CI was first enabled. The live workflow is already in `.github/workflows/ci.yml` — do not copy unless setting up a new repository from scratch.
