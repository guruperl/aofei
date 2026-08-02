# Status I03 - External Campaign Management API

State: `[+] Complete`

## Goal

Expose a stable external API for authorized campaign management and reporting
without treating internal Summer JSON routes as a public integration contract.

## Dependencies

- D01 enforceable delivery semantics.
- S02 scoped identities and permissions.
- O01 quotas, overload protection, metrics, and audit visibility.

## Tasks

| Item | State | Notes |
|---|---:|---|
| API boundary | `[+]` | Optional `/api/v1` resources cover the credential-owned advertiser, campaigns, ad groups, creatives, targeting, publication operations, and derived delivery reports without routing through Summer JSON. |
| Authentication | `[+]` | One-time `w8m_v1` tokens persist only as deployment-keyed digests and bind expiry, rotation/revocation, one advertiser, and fixed scopes; recent-MFA Summer actions own lifecycle. |
| Write safety | `[+]` | Strict D01/D02 validation, exact ETags, exclusive-upsert idempotency claims, version triggers, redacted immutable audit, and generation-scoped publication acknowledgement protect mutations. |
| Read behavior | `[+]` | Stable-id cursors, bounded pages/ranges, UTC and inherited-timezone semantics, USD/accounting metadata, partial report state, and fixed envelopes are implemented and specified. |
| Quotas | `[+]` | Atomic Redis credential/account minute quotas use isolated keys and TTLs; an exhausted credential cannot consume another credential's account allowance. |
| OpenAPI and clients | `[+]` | OpenAPI 3.1 defines 10 versioned paths and 20 strict schemas; the checked typed Go client, 180-day deprecation policy, migration, rollout, retention, and rollback contracts are published and tested. |

## Acceptance Criteria

- No public integration depends on internal Genelet route names or templates.
- Cross-account access is denied at every resource boundary.
- Retried writes are idempotent and auditable.
- Version compatibility and deprecation are documented before launch.
- API documentation states D01's UTC daily reset, cache propagation bound,
  fail-closed limited-demand behavior, and non-adaptive pacing semantics.

## Verification

- Contract/golden, authz, idempotency, concurrency, pagination, quota, audit,
  compatibility, and full closeout suites.

## Result

- `managementapi` and its typed client implement an independently disabled
  advertiser control plane. `cmd/unify` mounts it before the Genelet catch-all
  only after the S02 identity and shared key boundaries initialize.
- The current baseline has 79 tables, 6 routines, and 33 triggers. Four API
  tables own digested credentials, 24-hour exact response replay, publication
  operations, and immutable redacted evidence; three resource triggers advance
  optimistic versions for API and portal edits.
- Cache jobs enroll pending operations in an opaque generation before reading
  configuration and acknowledge only that generation after the configured
  serving backend publishes. A later commit waits for the next generation, and
  acknowledgement never changes `New`/`Prepare` review eligibility.
- Advertiser/admin Summer pages issue, rotate, revoke, and list least-privilege
  credentials under named permission and recent-MFA checks. The restricted
  `identity-admin -action=prune-api-audit` path applies S02 retention with a
  connection-local gate that is reset or the connection is discarded.
- Evolution v24 records the new public API, schema, configuration, cache, and
  operator boundaries. The deployment remains opt-in and no live service,
  database, credential, or DNS state was changed.

## Reconciliation From S04

- API strings and error details are data, never trusted HTML. Generated API
  documentation and clients must escape advertiser names, creative source,
  report dimensions, and validation messages and must not offer an executable
  creative preview endpoint under the management API contract.

## Reconciliation From O01

- API quotas may reuse O01 admission primitives, but must isolate API traffic
  from auction and UI capacity. Credential and account identifiers must never
  appear in expvar metric names or labels.
- The public contract must document bounded bodies and the relevant 413, 429,
  and 503 retry behavior. API load evidence must use O01's recorded hardware,
  configuration, workload, dependency, and SLO format.

## Reconciliation From A01

- Report resources must label response prices as USD CPM and spend as the
  reconciled per-impression USD amount, with six-decimal currency semantics and
  A01 freshness/source metadata. Generic campaign writes cannot mutate
  statements, adjustments, settlements, or Redis accounting floors.
- Financial endpoints remain out of the first management API unless they add
  dedicated S02 permissions, trusted actor identity, maker/checker transitions,
  idempotency, immutable audit, and sensitive-field denial tests.

## Reconciliation From D02

- The first write API exposes reviewed positive USD CPM only; legacy
  CPC/CPA/ROI rows are read-only migration records. Creative writes use explicit
  Banner/Video/Native types, positive rotation weight, URL-only Banner/Video
  sources, and versioned structured Native fields with the same URL/size/MIME
  checks as Summer.
- API acceptance is distinct from cache activation. Responses must report
  validation and bounded publication state without offering raw executable
  previews or bypassing the D02 cache-first rollout.

## Reconciliation From O02

- API health and readiness use the shared lifecycle endpoints, but API quotas
  and dependency/error evidence remain isolated from auction capacity. A
  failover may retry reads; writes require their public idempotency key and
  must never rely on a load balancer retry for correctness.
- The contract names shared-dependency 503/timeout behavior, partial report
  freshness, readiness drain, and recovery compatibility. Load and availability
  claims require the O02 regional measurement window, N-1 headroom, and edge
  coverage rather than a single process test.

## Reconciliation From D03

- Generic advertiser campaign credentials cannot approve bidders, assign
  credential references, edit or publish operator routes, or enable fallback or
  `Always` traffic. Any future external-bidder resource is a separate contract
  with dedicated S02 permissions, audit, quotas, and activation state.
- Campaign/report resources may expose only authorized middleman aggregates and
  partial-data freshness. They never return partner endpoints, credential
  references or values, callback URLs, raw bids, route internals, or an API
  operation that bypasses D03's database/Redis preflight and staged canary.

## Reconciliation From R02

- Public report resources use the R02 metric registry, UTC interval bounds,
  USD six-decimal/accounting version, dimensions, current/partial/unavailable
  states, action-retention warning, and account scope. Summer's authenticated
  `/goto/<role>/json/ledger` chartag remains an internal UI route and is never
  published or versioned as this API.
- Reporting credentials receive explicit S02 scopes and cannot request another
  advertiser/publisher, operator-only cost/margin/route data, experiment salts
  or subject/outcome digests. Pagination/quotas must be measured against R02's
  MySQL query baseline; public API pressure remains isolated from UI and auction
  capacity and does not itself justify OLAP.

## Reconciliation From P02

- Report resources expose only the closed P02 supply categories and an
  operator-authorized public seller id/type already permitted by R02. They do
  not return seller approval history, raw `source.schain`, public-review URLs,
  private partner data, or another publisher's inventory.
- The first advertiser campaign-management API cannot create/edit publisher
  inventory, approve sellers, or submit a trusted supply chain. Any later
  publisher API needs separate S02 scopes, idempotent writes, audit, cache
  publication state, and the same controlled validation as Summer.

## Reconciliation From S02

- The public API uses a distinct service-principal credential, not the Summer
  role cookie, database session, TOTP, recovery code, or internal JSON route.
  Store only a keyed credential digest; bind the principal to one advertiser,
  expiry, revocation state, and explicit `api.*` permissions. Credential
  creation, rotation, and revocation require a reauthenticated S02 human actor
  and immutable redacted audit event.
- Derive advertiser scope from the authenticated service principal before
  loading or mutating a resource. Never accept `adv_id`, role, permission,
  resource grant, audit actor, or audit reason as authority from the request.
  Cross-account denial must occur before validation side effects, idempotency
  claims, cache publication, or report queries.
- API audits may record stable credential/principal ids, object hashes,
  idempotency digests, result, and safe reasons, but never raw bearer tokens,
  S02 session tokens, TOTP/recovery material, request bodies, or creative
  source. API authentication failures use the S02 security retention class and
  fixed-cardinality operational metrics.

## Verification Results

- Go 1.23.5 full tests and vet passed in Aofei, Pzdesign, and Genelet. The
  documented Aofei management/DSP/cache race suite, full Pzdesign Summer race
  suite, and full Genelet race suite passed.
- Pinned staticcheck v0.5.1 passed for Aofei and Pzdesign with its documented
  legacy style exclusions; Genelet passed with its established
  naming/simplification exclusions only.
- A clean MySQL 8.0.41 baseline repeatedly exercised one-time credential
  issuance, rotation/revocation, cross-account denial, strict campaign/item/
  creative/targeting writes, exact replay, changed-body conflict, six-way
  concurrent retry, stale ETag, cursor pagination, inherited timezone,
  immutable/bounded audit retention, and cache generation activation against a
  disposable Redis 7 instance. The inventory was 79 tables, 6 routines, and 33
  triggers; all API/account/balance fixtures were zero afterward and both
  containers were removed.
- The concurrency proof first exposed an InnoDB shared-lock upgrade deadlock.
  The replacement exclusive upsert plus random claim token then passed ten
  consecutive lifecycle runs; a final three-run clean restore covered the
  expanded contract assertions.
- Redocly validated all 10 OpenAPI paths and 20 schemas (with only the expected
  non-blocking license-metadata warning because this checkout declares no
  license file). All 279 Pzdesign templates, Chinese public-copy/data guards,
  Aofei documentation/public-data/SQL guards, actionlint, and all three
  repository diff-hygiene checks passed.

## Review Closeout

- Deep review fixed idempotency deadlocks, prevented a mutation committed during
  a cache build from receiving that generation's acknowledgement, restricted
  acknowledgement to the configured serving backend, and ordered nested scope
  checks before ownership queries.
- Review also added a bounded/discard-safe audit retention path, distinguished
  credential-store outages from invalid tokens, isolated exhausted credential
  quota from account quota, normalized delimiter-safe URLs, enforced exact
  ETags and database-storable limits/times, and returned the real inherited
  campaign timezone for ad groups.
- S03, A02, and the demand-gated I02 milestone now carry explicit I03
  reconciliation. No commit was created because the active goal's commit
  policy is `none`.
