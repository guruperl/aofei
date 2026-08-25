# Product

## What This Project Is

`aofei` / `winter` is a Go DSP package for OpenRTB-style real-time bidding. It
combines bid request handling, advertiser/campaign matching, publisher and slot
mapping, audience and creative selection, frequency capping, admin integration,
and local service orchestration.

The package is currently runnable from a clean local Docker harness: MySQL is
the active relational backend, Redis stores mutable runtime state and Redis-mode
cache payloads, NATS is the local message bus for spread/log flows, and
local/spread bid mode can serve static cache reads from in-process snapshots.

## Primary Users

- Developers and agents maintaining the DSP package.
- Operators who need to reset and inspect a local database/cache/message-bus
  runtime.
- Admin UI users managing advertisers, publishers, campaigns, slots, creatives,
  ledger records, and targeting data.
- Bid path integrations that send OpenRTB requests and expect deterministic
  low-latency responses.

## Core Domain

- Advertisers (`adv`) own campaigns, items, creatives, balances, and targeting.
- Publishers (`pub`) expose sites, slots, publisher attributes, and traffic
  metadata.
- Advertisers (`adv`) can own middleman bidder endpoints for fallback exchange
  fanout. Existing advertiser auth and reporting remain the account boundary.
  Advertisers can create and edit safe endpoint metadata in Summer/Genelet,
  while operators retain route, credential, synthetic reporting row, and traffic
  activation control. Operators manage route groups, route bidders, and traffic
  targets through the admin Summer/Genelet `midroute` UI. The synthetic
  campaign/item chain reuses the existing ACL and channel matching model to
  decide which original publisher/site inventory may be forwarded to a bidder.
- External exchange and DSP interoperability is fixed at a bounded OpenRTB 2.5
  profile: independently scoped impressions, USD CPM/floors, negotiated gzip,
  strict response identity/media/callback/lateness validation, and safe fixed
  diagnostics without public raw bid captures.
- Matching code turns database state into Redis and spread/static cache
  structures such as `PubMap`, `RAdv`, audience maps, and creative maps.
- DSP runtime code reads request data, config, cache entries, mutable Redis
  state, MaxMind lookup data, and logging paths to produce bid responses and
  win/loss records.
- Redis cache refresh and ledger aggregation remain singleton scheduled jobs,
  normally on dedicated cache and log aggregation nodes rather than every
  `cmd/unify` node.
- Single-region production serving uses at least two identical `cmd/unify`
  nodes behind health-checked load balancing. Lifecycle readiness is withdrawn
  before graceful drain, local/static nodes also reject missing or stale
  generations, singleton job leases renew while work runs, and interval/daily
  ledger identities remain database-unique through a Redis partition.
- Accounting contract `usd-cpm-impression-v3` keeps public OpenRTB/tracking
  prices in USD CPM while authoritative caches, reservations, and ledgers use
  exact integer micro/nano-USD through the statement boundary. Authorized
  operators create immutable advertiser/publisher
  statement snapshots, adjustments, approvals, settlements, corrections, and
  CSV exports through `cmd/accounting`; the public Summer funding routes and
  full card/bank credential columns are retired.
- A02 adds a disabled-by-default hosted payment boundary: Stripe Checkout
  collects advertiser funding and Stripe Connect Express onboards publisher
  payout destinations without full credentials entering W8M. Exact A01
  statements, aggregate movement limits, maker/checker/recent-MFA controls,
  stable provider idempotency inside a conservative replay window, mandatory
  approved bindings, an immutable first-submission binding selection, signed
  durable webhooks, immutable opaque object mappings and payout country, and
  explicit fee/refund/dispute/payout-failure reconciliation remain
  authoritative. Live provider enablement is a separate
  legal/finance/risk decision, not a repository default.
- Signed R01 actions measure conversions, purchases, downloads, video
  completion, and namespaced custom events. A separate HMAC lineage token,
  advertiser-scoped event idempotency, click-over-view attribution, bounded
  retention/reconciliation, and scoped reporting keep these facts analytical;
  they never mutate CPM billing, balances, or D01 delivery reservations.
- R02/R03 marketplace analytics derive account-scoped UTC interval facts and
  reconciled action ratios for advertisers, publishers, and operators. Every
  metric has a fixed source/formula/currency/freshness contract; dependency
  gaps remain partial or unavailable rather than false zero. Controlled
  experiments use server-owned v2 namespaces (with explicit legacy v1
  compatibility), deterministic cross-experiment-unlinkable pseudonymous
  assignment, immutable version/allocation contracts, append-only validated
  exposure and idempotent declared-metric outcomes, bounded expiry/exact
  deletion, aggregate-only export, and renewable-lease OS-principal state
  transitions. Reports and experiments cannot modify bids, budgets, delivery,
  accounting, or settlement.
- Local commercial demand supports reviewed positive USD CPM only. The highest
  qualified campaign/ad-group CPM wins; creative weight rotates only inside the
  winning demand unit. Legacy CPC/CPA/ROI rows remain readable migration data
  but cannot bid or be silently converted. Banner/Video use URL-only sources,
  Native uses a versioned structured source, and local/middleman creative
  compatibility is enforced before delivery reservation or winner replacement.
- Campaign and ad-group start/end windows, weekly hours, hard total/daily
  spend/impression/click limits, and deterministic pacing are authoritative
  auction eligibility. Limited demand uses atomic shared Redis reservations;
  stale delivery policy or unavailable reservation state fails closed.
- Bid privacy is contextual by default. COPPA is restricted, opt-outs override
  grants, and personalization requires an explicitly configured current TCF
  vendor/purpose grant. Raw identity is transient and local only; cap/tracking
  identity is HMAC-pseudonymous, audits are redacted, and external bidders see
  independently scrubbed contextual requests only after a separate disclosure
  gate. The operator contract is
  [docs/privacy-data-governance.md](../docs/privacy-data-governance.md).
- The sibling `github.com/guruperl/pzdesign` module provides Summer/Genelet
  admin-model plumbing over the same schema and imports this module's domain
  packages where needed.
- S02 provides an opt-in higher-assurance account boundary for administrator,
  advertiser, publisher, agent, and read-only analyst roles: opaque
  database-backed sessions, required TOTP with one-time recovery, named
  action/resource permissions, recent-MFA checks, and immutable redacted
  security evidence. The checked-in example remains disabled until operators
  apply the schema migration, provision one shared deployment key, verify mail
  recovery, and execute the staged rollout in
  [docs/identity-access-security.md](../docs/identity-access-security.md).
- I03 provides an independently opt-in `/api/v1` advertiser control plane.
  Digested, expiring, revocable service credentials bind every request to one
  advertiser and fixed `api.*` scopes; Redis enforces isolated account/token
  quotas. Campaign/item/creative/targeting writes use durable idempotency,
  trigger-backed optimistic versions, immutable redacted audit, and explicit
  pending-to-active cache operations. Internal Summer JSON and browser identity
  are not public integration contracts.
- S03 provides independently opt-in, explainable traffic-quality controls. A
  closed eight-signal rule taxonomy progresses through Draft, Observe, Canary,
  and Active with immutable decision/version evidence, scoped account review
  and appeal, recent-MFA enforcement/rollback, and separate maker/checker
  billing recommendations. Incomplete evidence remains observe-only;
  infrastructure failures are never IVT signals. Serving reads only a bounded
  immutable enforcement snapshot and fails open after that snapshot expires.
  Automatic or learned scoring remains deferred.
- Public and authenticated control-plane pages render account, campaign,
  publisher, report, and stored creative-review values under contextual
  escaping. Creative review is source-only; executable creative materialization
  is confined to the auction delivery contract and remains subject to D02
  validation.
- Public advertiser/publisher registration and password-recovery mail requests
  have a default-off S06 abuse boundary. When production-enabled, scoped
  Turnstile validation precedes expensive or mutating work, atomic Redis quotas
  use expiring HMAC-pseudonymous email/IP keys, and only reviewed trusted
  proxies can supply client identity. The owner-selected Cloudflare Free
  exact-path burst rule and the complete boundary are active on W8M; other
  deployments remain default-off and must retain their own activation proof.

## Current Product Direction

The completed M-lane established the local runtime and the completed original
D/P/R/I/S/A/O lanes form the implemented W8M advertiser-publisher marketplace
baseline. A 2026-08-21 deep review opened D04, P03, S05, O03, R03, and A03 for
confirmed correctness fixes, request authenticity, trust-boundary hardening,
operational reliability, experiment privacy, and an exact-money migration. D04
is complete. A 2026-08-23 follow-up opened repository-only D05 auction/cap and
hot-path remediation and reopened S06 trusted-marker, anonymous-error, Gmail,
and quota-script work. D05, S06, P03, S05, O03, R03, and A03 are now complete.
A 2026-08-25 cross-lane review opened M46 for confirmed demand-eligibility,
schema, filesystem, concurrency, hot-path, creative-boundary, and
static-analysis remediation without reopening those completed milestones.
P03 provides its
threat contract, default-off versioned locator codec/runtime dual reader,
default-off SDK/server authentication, independent browser/App enforcement,
client-claim disposition, portal/cache integration, and repository rollout/
abuse evidence. Production gates remain default-off and require a separately
authorized named publisher canary. S05 provides the reviewed outbound,
creative-consumer, principal, quality-version, and database-integrity boundary.
A03 provides exact v3 monetary authority across prices, reservations, ledgers,
management, statements, and hosted reconciliation while retaining legacy float
data only as labeled read/drain evidence. I02 maintained Android/iOS SDKs
remains separately demand-gated: its P03/S05/A03 repository prerequisites are
complete, M46 must close first, and no named integration defines supported
platforms and lifecycle requirements. Lane state and strict
dependency order live in
[milestone.md](milestone.md); the guide-to-lane map is
[docs/README.md](../docs/README.md). Explicitly deferred investments and their
reconsideration triggers live in [docs/defer.md](../docs/defer.md).

The established runtime direction remains:

- Local runtime should be Docker-backed and free of historical production auth.
- The active schema should be represented by `etc/step4_init.sql`.
- Redis and NATS should be available locally through the same helper flow as
  MySQL.
- Static publisher, slot, audience, and creative data should be inspectable as
  Redis payloads, atomically selected spread disk generations, and local
  in-process generations.
- Middleman fallback is available only by explicit DSP config after
  advertiser-owned endpoint approval, route cache population, synthetic
  reporting row validation, and ACL/channel eligibility checks.
- Middleman route operations expose cache freshness and health in Summer while
  keeping cache publication on the singleton `cmd/redis-cache` node.
- Middleman revenue activation is staged separately from configuration edits.
  A read-only preflight compares the active MySQL model to the published Redis
  v2 generation, resolves credential references without exposing values, and
  enforces distinct preflight, Fallback, and optional Always gates. Routes stay
  Redis-only so revocation has one shared timeline.
- Retryable downstream middleman callback forwarding failures are queued
  durably after `/mid/*` callbacks and retried by a singleton operations
  command; `/bid` remains cache/Redis-only and does not write MySQL retry rows.
- `trigger_mode='Always'` middleman auction expansion is gated by
  `middleman_always_enabled`; when enabled, eligible marked-up middleman bids
  compete with local bids by effective CPM.
- Auction and creative rollout is cache-first: a D02 cache compiler publishes
  explicit creative media metadata before D02 HTTP readers roll. RAdv stays at
  payload version 2 and creative fields are additive for old-reader rollback.
- Direct publisher SSP traffic was the post-M26 product direction. The
  existing `pub` role remains the publisher account and inventory owner. The v1
  browser contract is `POST /pz` with packed `site` and `adUnits[].slot` tokens
  and a JSON array of HTML strings in ad-unit order. M28 serves valid requests
	  through the existing local Aofei bid path. M29 adds publisher slot tag
	  copy/download UI, stored slot sizes, external `ads.js` endpoint resolution,
	  and endpoint-limited permissive `/pz` CORS. M30 originally added SSP
	  audit-source separation and best-effort browser identity; S01 now permits
	  that cookie only for an accepted personalized grant, removes IP+UA identity
	  fallback, and redacts audits. M31 adds exact cached-site-host
	  origin/referrer enforcement for browser traffic while keeping `/pz` CORS
	  credentialless and keeps `/pz` plus audit `source:"ssp"` as the current
	  direct SSP source boundary. M32 adds
	  mobile/API serving on the same `/pz` and `pub` boundary by accepting
	  SDK `app`, `device`, `user`, and `regs` objects, honoring explicit
	  `responseFormat:"json"` and `"openrtb"` outputs, and preserving omitted or
	  `"html"` browser responses as the existing ordered HTML-string array. M33
	  lets valid `/pz` auctions use existing middleman `Fallback` and gated
	  `Always` fanout after local matching while preserving those M32 response
	  formats and keeping malformed, invalid-token, and policy-rejected traffic
	  out of middleman fanout. M34 records richer supply taxonomy as
	  additive defaulted fields on existing publisher tables while
	  keeping `pub`, `pub_site`, and `pub_slot` as the ownership boundary and
	  keeping `/pz` plus audit `source:"ssp"`/`contract:"pz-v1"` as the runtime
	  entrypoint boundary. M35 decides that no
	  separate `ssp` account role or SSP-owned inventory schema is needed for the
	  current `/pz` path; future schema work should extend the existing
	  publisher model unless new legal, settlement, intermediary, permission,
	  compliance, or partner-credential requirements reopen the boundary.
	  P01 makes that existing path commercially activatable: the cache owns exact
	  Web/App type, slot size, and USD CPM floor; the server uses the greater of
	  configured/request floors; invalid supply fails before auction side effects;
	  the publisher UI separates Web tags from App SDK/API samples; and browser
	  markup renders in an opaque-origin sandbox with deterministic fill states.
	  Production onboarding follows a read-only inventory manifest plus
	  cache-first canary/rollback runbook.
	  P02 implements controlled seller, site, and slot supply metadata on that
	  same publisher boundary. Operator-approved seller state generates a
	  conservative standard `source.schain`; client seller claims are ignored,
	  intermediary chains are not falsely marked complete, and transparency does
	  not change the A01 settlement owner. The additive cache and privacy-safe
	  audit fields feed explicit R02 supply dimensions with unknown defaults.
	  P03 separates public browser locator integrity from non-browser request
	  authentication. The default-off `pz2` HMAC locator format binds the full
	  inventory tuple, uses an explicit key id/rotation epoch and bounded
	  current/previous key ring, and supplies measured dual-read plus an explicit
	  legacy-disable gate. It prevents minting or modification but remains
	  observable and replayable; exact browser
	  Origin/Referer policy is provenance, not publisher proof. SDK/server
	  traffic can now require a publisher/App-scoped Ed25519 credential with an
	  exact-body signature, bounded timestamp, and shared one-use Redis nonce.
	  S02-scoped issue/rotation/revocation shows the private seed once, stores
	  only the public verifier, and writes immutable lifecycle audits. The gate
	  stays off until a separately authorized named-publisher rollout. Valid
	  locators or request proofs cannot override Web/App type, active inventory,
	  App identity,
	  browser provenance, privacy, admission, or server-owned seller-chain
	  policy; pre-auction rejections do not reflect cached ids, hostnames, or
	  credential state. S02 sessions, I03 advertiser credentials, and
	  `aofei_pz_uid` cannot substitute for runtime publisher authentication.
	  Unauthenticated SDK compatibility traffic is always contextual before
	  matching even when it asserts a valid-looking grant. Authenticated SDK
	  body IP/coarse geo and identity remain publisher assertions usable only
	  under a separate S01 personalization grant; exact coordinates are removed.
- Root documentation should be short, current, and operational.
- Marketplace report roles derive from the authenticated Genelet `_grole` and
  session account identifier. Advertisers and publishers see only their own
  commercial side; operators see route/commercial aggregates; agent reporting
  remains limited to its explicit review scope. An analyst sees only exact
  delegated report resources and cannot export JSON or mutate product state
  without separately named permissions. Internal JSON chartags are not the
  external `/api/v1` contract.
- Detailed project memory should live in `memory-bank/`.

## Non-Goals

- Restoring old root config directory runtime patterns.
- Using or documenting legacy named database credentials as active auth.
- Building production deployment automation before local harness correctness is
  settled.
- Reintroducing historical runtime, database, traffic, or geodata snapshots as
  tracked source inputs.
- Introducing a separate `ssp` account role for the v1 direct publisher path.
- Claiming automatic optimization, real payment-card processing, multi-region
  availability, or million-RPM capacity before the evidence gates in
  `docs/defer.md` are satisfied.
- Claiming 99.9% availability, recovery objectives, or production latency from
  unit tests, microbenchmarks, or a disposable local restore drill.
