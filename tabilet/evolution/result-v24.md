# Result V24

I03 establishes W8M's optional external advertiser management plane.

Implemented direction:

- `cmd/unify` mounts Aofei's versioned management handler ahead of Genelet only
  when `management_api.enabled` is explicitly activated. Internal HTML/JSON
  routes, role cookies, sessions, TOTP, and CSRF values are not public API
  credentials or contracts.
- A one-time `w8m_v1` bearer token resolves through a public id and
  constant-time comparison of a deployment-keyed digest. Each credential is
  bound to one advertiser, fixed scopes, expiry, rotation, and revocation;
  Summer credential lifecycle actions retain S02 permission and recent-MFA
  enforcement.
- Strict account-derived reads and writes cover campaigns, ad groups,
  source-only creatives, targeting, publication operations, and derived
  delivery reports. D01 delivery limits/pacing, D02 creative rules, A01/R02
  accounting labels, and publisher/middleman/operator boundaries remain in
  force.
- Every mutation uses a keyed idempotency identity, an exclusive per-attempt
  claim token, trigger-backed optimistic versions, a bounded stored response,
  and immutable redacted audit. Redis atomically isolates credential and
  advertiser request quotas.
- A cache job enrolls already-visible pending operations into an opaque
  generation before reading configuration and acknowledges only that
  generation after the configured serving backend publishes. Later commits
  wait for the next generation; acknowledgement never bypasses commercial
  review eligibility.

Contract consequences:

- The clean baseline gains four API tables, three version columns, five
  triggers, exclusive idempotency claims, and cache-publication generation
  markers. Populated systems require a reviewed additive migration before API
  activation.
- `/api/v1` is the major compatibility boundary. Incompatible changes require
  a new major path; ordinary deprecation has a documented 180-day notice and
  migration window.
- The OpenAPI 3.1 source and checked typed Go client are reviewed together.
  Request bodies, quotas, timeouts, retries, reports, errors, and source-only
  creative behavior are explicit rather than inherited from Summer.
- API evidence uses the S02 retention class through the restricted
  `identity-admin -action=prune-api-audit` path and a separate maintenance
  database principal. The HTTP principal must not receive audit deletion.
