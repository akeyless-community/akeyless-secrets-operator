# Using HashiCorp Vault alongside Akeyless

The operator image includes the HashiCorp Vault provider in addition to Akeyless, so one operator can sync secrets from both.

| Backend | Store resource | Secret resource |
|---------|----------------|-----------------|
| Akeyless | `AkeylessSecretStore` / `ClusterAkeylessSecretStore` (`secrets.akeyless.io/v1alpha1`) | `AkeylessSecret`, `AkeylessDynamicSecret` |
| Vault | `SecretStore` / `ClusterSecretStore` (`external-secrets.io/v1`) | `ExternalSecret` |

Vault resources use the upstream External Secrets Operator API, so existing ESO Vault manifests work unchanged. The full Vault provider reference (auth methods, KV v1/v2, PKI, namespaces) is in [provider/hashicorp-vault.md](provider/hashicorp-vault.md).

## 1. Enable the Vault resources

The `external-secrets.io` CRDs and reconcilers are off by default. Install or upgrade with [examples/helm-values-vault.example.yaml](examples/helm-values-vault.example.yaml):

```bash
helm upgrade --install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  -n akeyless-secrets-operator --create-namespace \
  -f docs/examples/helm-values-vault.example.yaml
```

Verify:

```bash
kubectl get crd | grep -E 'secrets.akeyless.io|external-secrets.io'
```

You should see the four Akeyless CRDs plus `secretstores`, `clustersecretstores` and `externalsecrets`.

> **Do not install this alongside External Secrets Operator.** Both would own the `external-secrets.io` CRDs and reconcile the same resources. This operator replaces ESO for clusters that need Akeyless and Vault together.

## 2. Create a Vault store

[examples/vault-secret-store.yaml](examples/vault-secret-store.yaml) uses Vault Kubernetes auth with KV v2 mounted at `secret/`:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: vault
  namespace: my-app
spec:
  provider:
    vault:
      server: "https://vault.example.com:8200"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "my-app"
          serviceAccountRef:
            name: "my-app"
```

## 3. Sync a secret from Vault

[examples/vault-external-secret.yaml](examples/vault-external-secret.yaml) reads `secret/my-app/db` and writes the `password` field to the Kubernetes Secret `db-creds`:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: db-creds
  namespace: my-app
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault
    kind: SecretStore
  target:
    name: db-creds
  data:
    - secretKey: password
      remoteRef:
        key: my-app/db
        property: password
```

```bash
kubectl apply -f docs/examples/vault-secret-store.yaml
kubectl apply -f docs/examples/vault-external-secret.yaml
kubectl get secretstore,externalsecret -n my-app
```

Akeyless resources in the same namespace keep working as described in [Getting Started](getting-started.md).
