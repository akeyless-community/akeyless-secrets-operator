# Manual cluster test — CS-AKS1 / namespace `barak`

End-to-end validation manifests used for the Akeyless Secrets Operator development cycle.

## Prerequisites

- `kubectl` context: **CS-AKS1**
- Namespace: **barak**
- Akeyless API credentials in secret `akeyless-operator-creds` (see `akeyless-creds-secret.example.yaml`)
- Docker pull secret `akeyless-docker-hub` if using a **private** image on Docker Hub

## Image reference

The test Deployment pins a development image:

```text
docker.io/barakdmax/akeyless-secrets-operator:dev-test
```

When moving to the official registry, update `operator.yaml` (and optionally `barak-values.yaml` for Helm-based installs). See [docs/image-publishing.md](../../../docs/image-publishing.md) for the full migration guide and file list.

## Apply order

```bash
kubectl config use-context CS-AKS1
kubectl apply -f hack/test/barak/operator.yaml
# Create akeyless-operator-creds from akeyless-creds-secret.example.yaml first
kubectl apply -f hack/test/barak/test-resources.yaml
kubectl apply -f hack/test/barak/rollout-test.yaml   # optional rollout-restart test
```

## What gets tested

| Resource | Purpose |
|----------|---------|
| `AkeylessSecretStore/barak-store` | Auth + connectivity to `https://api.akeyless.io` |
| `AkeylessSecret/barak-static-test` | Sync `/aso/secret1` and `/aso/secret2` into K8s Secret |
| `Deployment/aso-rollout-test` | `rolloutRestartTargets` — pod restart on secret change |

## Operator flags (Akeyless-only mode)

The test Deployment disables all legacy ESO controllers and GeneratorState:

```text
--enable-legacy-external-secrets-reconciler=false
--enable-generator-state=false
--enable-akeyless-secret-reconciler=true
--enable-akeyless-dynamic-secret-reconciler=true
--enable-akeyless-secret-store-reconciler=true
```

## Verified test results (2026-06-24)

| Test | Result |
|------|--------|
| Static secret sync (`barak-static-test`) | Pass — `Ready=True`, 2 keys synced |
| Rollout restart on secret change | Pass — `secrets.akeyless.io/restartedAt` set, new ReplicaSet |
| Operator stability (post GeneratorState fix) | Pass — 0 restarts over 4+ minutes |
| Webhook endpoint `:8083` | Pass — enqueues reconcile |

## Useful commands

```bash
kubectl get akeylesssecret,akeylesssecretstore -n barak
kubectl get secret barak-static-test -n barak -o yaml
kubectl logs deployment/akeyless-secrets-operator -n barak
kubectl get deploy aso-rollout-test -n barak -o yaml | grep restartedAt
```

## Rebuild and redeploy after code changes

```bash
ARCH=amd64 make build-amd64
docker build --platform linux/amd64 -f Dockerfile \
  -t docker.io/barakdmax/akeyless-secrets-operator:dev-test .
docker push docker.io/barakdmax/akeyless-secrets-operator:dev-test
kubectl rollout restart deployment/akeyless-secrets-operator -n barak
```
