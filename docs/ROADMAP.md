# Akeyless Secrets Operator — Roadmap

## Phase 1 — Fork bootstrap (in progress)

- [x] Fork ESO codebase with Akeyless provider + PR #6507 (`ignoreCache`, `azure_ad` Workload Identity)
- [x] Build only Akeyless provider (`PROVIDER=akeyless`, strip `pkg/register/*` except `akeyless.go`)
- [ ] Push initial tree to `github.com/akeyless-community/akeyless-secrets-operator`
- [ ] Verify `make build` and `go test ./providers/v1/akeyless/...`

## Phase 2 — Remove other vendors (CRDs & docs)

Runtime already excludes non-Akeyless providers. Remaining cleanup:

| Area | Action |
|------|--------|
| `providers/v1/*` | Delete all except `akeyless/` |
| `apis/.../secretstore_*_types.go` | Remove non-Akeyless provider structs from `SecretStoreProvider` union |
| CRDs | Regenerate with `make manifests`; drop unused provider schemas |
| `generators/v1/*` | Remove vendor-specific generators (ECR, GCR, Vault, etc.); keep generic ones (password, uuid) if still wanted |
| Docs / Helm | Remove non-Akeyless provider pages; rename chart to `akeyless-secrets-operator` |
| CI | Drop multi-provider e2e; add Akeyless-focused e2e |
| Module path | Optional: rename `github.com/external-secrets/external-secrets` → `github.com/akeyless-community/akeyless-secrets-operator` |

**API group decision:** Keep `external-secrets.io` short-term for manifest compatibility. Rebrand to `akeyless.io` only if we need a clean break.

## Phase 3 — React to Akeyless secret changes

Today, ESO refreshes secrets on a **timer** (`spec.refreshInterval`, default 1h). Between polls, K8s secrets stay stale unless the `ExternalSecret` spec changes or the target secret is deleted.

### Option A — Shorter polling + `ignoreCache` (available now)

- Set `refreshInterval: 30s` (or lower) on `ExternalSecret`
- Set `ignoreCache: true` on `SecretStore` when using Gateway with proactive cache
- **Pros:** No new code; works today
- **Cons:** Poll load on Gateway/SaaS; not event-driven

### Option B — Version-aware skip/refresh (recommended next step)

Akeyless `describe-item` returns `last_version`. The provider already uses this internally.

Add to `ExternalSecret` or `SecretStore`:

```yaml
spec:
  refreshPolicy: OnAkeylessVersionChange  # new policy
```

Controller flow:

1. On reconcile, call lightweight `DescribeItem` for each remote ref
2. Compare `last_version` to value stored in `ExternalSecret.status` (new field `remoteVersions`)
3. Skip full fetch if unchanged; fetch and update K8s secret when version bumps

**Pros:** Efficient polling — cheap metadata check, fetch only on change  
**Cons:** Still poll-based; need to define behavior for certificates/rotated secrets

### Option C — Akeyless Event Center / webhook trigger (event-driven)

If Akeyless can POST events on secret create/update/delete:

1. Deploy a small webhook receiver (sidecar or separate controller)
2. Validate and map Akeyless item path → affected `ExternalSecret` resources (index by remote ref)
3. Enqueue reconcile immediately (patch annotation or direct workqueue)

**Pros:** Near real-time updates, minimal idle API traffic  
**Cons:** Requires Akeyless event integration, auth, HA for webhook endpoint, mapping logic

### Option D — Gateway push / long-poll (if supported)

Investigate whether Akeyless Gateway exposes change notifications or streaming APIs usable from the operator.

---

## Recommended sequence

1. **Ship Phase 1** — working Akeyless-only binary + Helm
2. **Implement Option B** — version-aware refresh (biggest bang for buck)
3. **Evaluate Option C** with Akeyless product team (Event Center, audit logs, or custom webhook)
4. **Phase 2 cleanup** in parallel once binary is stable

## Open questions

- Keep PushSecret / ClusterExternalSecret / generators in v1?
- Target API group rebrand timeline?
- Minimum supported ESO manifest compatibility window?
