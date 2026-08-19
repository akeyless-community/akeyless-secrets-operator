# Getting Started — Step-by-Step Manual

This manual walks through installing the Akeyless Secrets Operator, syncing your first secret, and fixing common problems.

- **Full API reference:** [User Guide](akeyless-secrets-operator-guide.md)
- **All documentation:** [Documentation index](README.md)

---

## What you are installing

The operator runs in your cluster and:

1. Connects to Akeyless (SaaS or Gateway)
2. Reads items you reference in `AkeylessSecret` / `AkeylessDynamicSecret` resources
3. Creates or updates native Kubernetes `Secret` objects

**Custom resources** (API group `secrets.akeyless.io/v1alpha1`):

| Resource | Purpose |
|----------|---------|
| `AkeylessSecretStore` | Akeyless connection + auth for a namespace |
| `ClusterAkeylessSecretStore` | Shared store (cluster-scoped) |
| `AkeylessSecret` | Sync static secrets from Akeyless |
| `AkeylessDynamicSecret` | Sync dynamic / rotated secrets |

---

## Prerequisites

- Kubernetes **1.25+**
- **Helm 3.8+** (OCI chart support)
- Network from the cluster to Akeyless (`https://api.akeyless.io` or your Gateway URL)
- Akeyless credentials with **read** access to target items
- Tools: `kubectl`, `helm`

---

## Step 1 — Install the operator with Helm

Install from the published OCI chart on GHCR (same pattern as External Secrets Operator):

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --create-namespace
```

No local clone or image build is required. The chart pulls `ghcr.io/akeyless-community/akeyless-secrets-operator:v0.1.1` by default.

Choose **one** variant if your cluster already has related resources:

### Path A — Fresh cluster (default)

Use the command above. It installs **only the four Akeyless CRDs** (not legacy External Secrets CRDs).

### Path B — External Secrets Operator already installed

Use the same command as Path A. Chart defaults skip legacy ESO CRDs so Helm does not conflict with the existing `external-secrets` release.

### Path C — Akeyless CRDs already applied manually

If you previously ran `kubectl apply` for Akeyless CRDs, install **without** CRDs:

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace \
  --set installCRDs=false
```

### Verify the install

```bash
kubectl get pods -n akeyless-secrets-operator
kubectl get crd | grep secrets.akeyless.io
helm list -n akeyless-secrets-operator
```

Expected:

- Pod `akeyless-secrets-operator-*` is **Running**, **0 restarts**
- Four CRDs: `akeylesssecrets`, `akeylessdynamicsecrets`, `akeylesssecretstores`, `clusterakeylesssecretstores`
- Helm release `akeyless-secrets-operator` is `deployed`

---

## Step 2 — Configure Akeyless credentials

Create a Kubernetes Secret in the **application namespace** (not necessarily the operator namespace):

```bash
kubectl create namespace my-app   # if needed
```

Edit and apply [examples/akeyless-creds-secret.example.yaml](examples/akeyless-creds-secret.example.yaml):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: akeyless-creds
  namespace: my-app
type: Opaque
stringData:
  accessId: "<AKEYLESS_ACCESS_ID>"
  accessType: "api_key"
  accessTypeParam: "<AKEYLESS_API_KEY>"
```

```bash
kubectl apply -f docs/examples/akeyless-creds-secret.example.yaml
```

Supported `accessType` values: `api_key`, `aws_iam`, `azure_ad`, `gcp`, `k8s`. See the [User Guide](akeyless-secrets-operator-guide.md) for Gateway and cloud auth examples.

---

## Step 3 — Create an AkeylessSecretStore

The store tells the operator **how to reach Akeyless** and **which credentials to use**.

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecretStore
metadata:
  name: akeyless
  namespace: my-app
spec:
  akeylessGWApiURL: "https://api.akeyless.io"   # or Gateway URL
  auth:
    secretRef:
      accessID:
        name: akeyless-creds
        key: accessId
      accessType:
        name: akeyless-creds
        key: accessType
      accessTypeParam:
        name: akeyless-creds
        key: accessTypeParam
```

```bash
kubectl apply -f docs/examples/akeyless-secret-store.yaml
kubectl get akeylesssecretstore -n my-app
kubectl describe akeylesssecretstore akeyless -n my-app
```

Expected status: **Ready=True**, reason **Valid**.

---

## Step 4 — Sync a secret with AkeylessSecret

```yaml
apiVersion: secrets.akeyless.io/v1alpha1
kind: AkeylessSecret
metadata:
  name: app-secret
  namespace: my-app
spec:
  storeRef:
    name: akeyless
    kind: AkeylessSecretStore
  syncPolicy: OnRemoteChange    # or Periodic, OnSpecChange, CreatedOnce
  syncInterval: 5m
  target:
    name: app-secret            # Kubernetes Secret name to create/update
    creationPolicy: Owner
  data:
    - secretKey: password
      remoteRef:
        key: /path/to/akeyless/item
```

```bash
kubectl apply -f docs/examples/akeyless-secret.yaml
```

### Verify sync

```bash
kubectl get akeylesssecret -n my-app
kubectl describe akeylesssecret app-secret -n my-app
kubectl get secret app-secret -n my-app
```

Expected:

- `AkeylessSecret` condition **Ready=True**, reason **SecretSynced**
- Kubernetes Secret `app-secret` exists with the keys you defined
- Events show `akeyless-secrets-controller` / `SecretSynced`

Check operator logs if sync fails:

```bash
kubectl logs deployment/akeyless-secrets-operator -n akeyless-secrets-operator --tail=50
```

---

## Step 5 — Use the secret in a workload

Example Deployment consuming the synced secret:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
        - name: app
          image: nginx:1.27-alpine
          env:
            - name: PASSWORD
              valueFrom:
                secretKeyRef:
                  name: app-secret
                  key: password
```

See [examples/rollout-restart-example.yaml](examples/rollout-restart-example.yaml) for rollout restart on secret change.

---

## Optional — Dynamic secrets

For dynamic or rotated Akeyless items, use `AkeylessDynamicSecret`:

```bash
kubectl apply -f docs/examples/akeyless-dynamic-secret.yaml
kubectl get akeylessdynamicsecret -n my-app
```

See [examples/akeyless-dynamic-secret.yaml](examples/akeyless-dynamic-secret.yaml).

---

## Optional — Rollout restart on secret change

Add `rolloutRestartTargets` to your `AkeylessSecret`:

```yaml
rolloutRestartTargets:
  - kind: Deployment
    name: myapp
```

When synced secret **data** changes, the operator triggers a rolling restart of that Deployment. The Deployment must exist in the same namespace.

---

## Sync policies (quick reference)

| Policy | When Kubernetes Secret updates |
|--------|--------------------------------|
| `OnRemoteChange` (default) | When Akeyless item version changes |
| `Periodic` | Every `syncInterval` |
| `OnSpecChange` | When the CR spec changes |
| `CreatedOnce` | Once at creation, never updated |
| `OnWebhook` | When Akeyless webhook fires (requires `akeylessWebhook.enabled`) |

---

## Upgrading the operator

```bash
helm upgrade akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --reuse-values
```

Use `installCRDs=false` if CRDs are already on the cluster. Use `installCRDs=true` only on fresh installs where CRDs are not present.

---

## Advanced — build from source

For development, air-gapped clusters, or a private registry, see [image-publishing.md](image-publishing.md).

---

# Troubleshooting

## Helm install fails — `403 Forbidden` on OCI chart

**Symptom:** `helm install oci://ghcr.io/akeyless-community/charts/...` returns `403 Forbidden`.

**Cause:** GHCR chart package is **private** (default when first published).

**Fix:** A maintainer must set package visibility to **public**. See [ghcr-visibility.md](ghcr-visibility.md).

---

## Helm install fails — CRD owned by `external-secrets`

**Symptom:**

```
CustomResourceDefinition "acraccesstokens.generators.external-secrets.io" ...
annotation validation error: key "meta.helm.sh/release-name" must equal "akeyless-secrets-operator":
current value is "external-secrets"
```

**Cause:** Helm tried to install legacy ESO CRDs that already exist from an External Secrets Operator install.

**Fix:** Use a current chart release. Defaults install only Akeyless CRDs. Do not force legacy CRD flags:

```bash
# Do NOT set these unless you explicitly need legacy ESO CRDs:
# --set crds.createGenerators=true
# --set crds.createExternalSecret=true
```

---

## Helm install fails — Akeyless CRD already exists

**Symptom:**

```
CustomResourceDefinition "akeylesssecrets.secrets.akeyless.io" ... exists and cannot be imported
missing key "app.kubernetes.io/managed-by": must be set to "Helm"
```

**Cause:** CRDs were applied earlier with `kubectl apply`, not Helm.

**Fix (recommended):** Install without CRDs:

```bash
helm install akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator --create-namespace \
  --set installCRDs=false
```

**Alternative:** Adopt existing CRDs into Helm ownership (only if you want Helm to manage them):

```bash
for crd in \
  akeylessdynamicsecrets.secrets.akeyless.io \
  akeylesssecrets.secrets.akeyless.io \
  akeylesssecretstores.secrets.akeyless.io \
  clusterakeylesssecretstores.secrets.akeyless.io
do
  kubectl label crd "$crd" app.kubernetes.io/managed-by=Helm --overwrite
  kubectl annotate crd "$crd" \
    meta.helm.sh/release-name=akeyless-secrets-operator \
    meta.helm.sh/release-namespace=akeyless-secrets-operator \
    --overwrite
done
```

Then reinstall with `--set installCRDs=true`.

---

## Operator pod crash-loops / restarts

**Symptom:** Pod shows `RESTARTS > 0`, events show `Back-off restarting failed container`.

**Check logs:**

```bash
kubectl logs deployment/akeyless-secrets-operator -n akeyless-secrets-operator --previous
```

### Cause A — GeneratorState CRD missing

**Log message:**

```
no matches for kind "GeneratorState" in version "generators.external-secrets.io/v1alpha1"
failed to wait for generatorstate caches to sync
```

**Fix:** Ensure `processGeneratorState=false` (default on current `main`):

```bash
helm upgrade akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --reuse-values \
  --set installCRDs=false \
  --set processGeneratorState=false
```

Or patch the deployment immediately:

```bash
kubectl patch deployment akeyless-secrets-operator -n akeyless-secrets-operator \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--enable-generator-state=false"}]'
```

### Cause B — Image pull failure

**Symptom:** `ErrImagePull` / `ImagePullBackOff`

**Fix:**

- Confirm image exists: `docker pull ghcr.io/akeyless-community/akeyless-secrets-operator:v0.1.1`
- If GHCR is private, make packages public ([ghcr-visibility.md](ghcr-visibility.md)) or add `imagePullSecrets`
- Confirm `image.repository` and `image.tag` in Helm values

---

## AkeylessSecretStore not Ready

**Check:**

```bash
kubectl describe akeylesssecretstore <name> -n <namespace>
```

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Auth / invalid credentials | Wrong `accessId` or API key | Verify `akeyless-creds` Secret values |
| Connection refused / timeout | Network or wrong URL | Check `akeylessGWApiURL`, firewall, Gateway reachability |
| TLS / certificate error | Gateway uses custom CA | Set `caBundle` or `caProvider` on the store |
| Secret not found | Credentials Secret in wrong namespace | Store and creds Secret must be in the **same namespace** |

Test Akeyless connectivity from the cluster if needed:

```bash
kubectl run curl-test --rm -i --restart=Never --image=curlimages/curl:8.5.0 \
  -- curl -sf https://api.akeyless.io
```

---

## AkeylessSecret not syncing

**Check status:**

```bash
kubectl describe akeylesssecret <name> -n <namespace>
kubectl get events -n <namespace> --field-selector involvedObject.name=<name>
```

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Store not Ready | Store validation failed | Fix store first (see above) |
| Item not found | Wrong Akeyless path | Verify `remoteRef.key` in Akeyless console |
| Permission denied | Credentials lack read access | Update Akeyless role/policy for the access ID |
| Secret exists, not owned | Target Secret created manually | Delete the Secret or change `target.name` |
| No events from operator | Operator not watching namespace | Check operator pod is Running; check RBAC |

Force a reconcile:

```bash
kubectl annotate akeylesssecret <name> -n <namespace> \
  reconcile="$(date +%s)" --overwrite
```

---

## Wrong operator reconciling resources

If both **External Secrets Operator** and **Akeyless Secrets Operator** are installed, only the Akeyless operator should reconcile `secrets.akeyless.io` resources.

Verify events reference `akeyless-secrets-controller`:

```bash
kubectl get events -n <namespace> --sort-by='.lastTimestamp' | grep -i akeyless
```

---

## Helm upgrade overwrote manual patch

If you patched the deployment manually (e.g. `--enable-generator-state=false`) and a later `helm upgrade` removed it, add the setting to Helm:

```bash
helm upgrade akeyless-secrets-operator \
  oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator \
  --version 0.1.1 \
  -n akeyless-secrets-operator \
  --reuse-values \
  --set processGeneratorState=false
```

---

## Useful diagnostic commands

```bash
# Operator health
kubectl get pods -n akeyless-secrets-operator
kubectl logs deployment/akeyless-secrets-operator -n akeyless-secrets-operator --tail=100

# CRDs
kubectl get crd | grep akeyless

# All Akeyless resources
kubectl get akeylesssecretstore,akeylesssecret,akeylessdynamicsecret -A
kubectl get clusterakeylesssecretstore -A

# Helm release
helm get values akeyless-secrets-operator -n akeyless-secrets-operator
helm history akeyless-secrets-operator -n akeyless-secrets-operator
```

---

## Related documentation

- [Documentation index](README.md)
- [User Guide](akeyless-secrets-operator-guide.md) — full API and configuration reference
- [Install and publish](image-publishing.md) — OCI install and build from source
- [Helm chart INSTALL](../deploy/charts/external-secrets/INSTALL.md)
- [GHCR releases](ghcr-visibility.md) (maintainers)
- [Example manifests](examples/)
- [Akeyless provider](provider/akeyless.md)
