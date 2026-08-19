# Akeyless Secrets Operator — Roadmap

## Phase 1 — Public release (done)

- [x] Fork ESO with Akeyless provider + `ignoreCache`, `azure_ad` Workload Identity
- [x] Akeyless-owned CRDs (`secrets.akeyless.io/v1alpha1`)
- [x] Public GHCR chart and image (`oci://ghcr.io/akeyless-community/charts/akeyless-secrets-operator`)
- [x] CRD gating for co-existence with ESO
- [x] User documentation ([getting-started.md](getting-started.md), [user guide](akeyless-secrets-operator-guide.md))
- [x] CI on pull requests (`test-and-build`)

## Phase 2 — Vendor cleanup (ongoing)

Runtime already excludes non-Akeyless providers. Remaining cleanup:

| Area | Action |
|------|--------|
| `providers/v1/*` | Delete all except `akeyless/` |
| CRDs / docs | Remove unused upstream provider pages and generator CRDs from default install |
| Module path | Optional: rename module to `github.com/akeyless-community/akeyless-secrets-operator` |

## Phase 3 — React to Akeyless secret changes

Today, sync is primarily **poll-based** (`syncInterval` / `syncPolicy`). Between polls, Kubernetes Secrets can be stale unless the `AkeylessSecret` spec changes.

### Option A — Shorter polling + `ignoreCache` (available now)

- Set `syncInterval: 30s` (or lower) on `AkeylessSecret`
- Set `ignoreCache: true` on `AkeylessSecretStore` when using Gateway with proactive cache
- **Pros:** No new code; works today
- **Cons:** Poll load on Gateway/SaaS; not event-driven

### Option B — Version-aware skip/refresh (recommended next step)

Akeyless `describe-item` returns `last_version`. `syncPolicy: OnRemoteChange` (default) already compares versions.

Potential enhancement: persist per-key versions in `AkeylessSecret.status` for finer-grained skip logic on multi-key secrets.

### Option C — Akeyless webhook trigger (event-driven)

Enable `akeylessWebhook` in Helm values; configure Akeyless to POST on item changes.

**Pros:** Near real-time updates  
**Cons:** Requires webhook endpoint, auth, HA, path → resource mapping

---

## Recommended sequence

1. **Stabilize public install** — releases, docs, support matrix
2. **Option B enhancements** — richer status / version tracking
3. **Evaluate Option C** with Akeyless product (Event Center / webhooks)
4. **Phase 2 cleanup** in parallel once install path is stable

## Open questions

- Keep legacy ESO reconciliation path long-term or remove in v0.2?
- Minimum supported Kubernetes version formalization (currently 1.25+ in docs)?
- HTTP Helm repo (`charts.akeyless-community.io`) in addition to OCI?
