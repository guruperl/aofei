# Architecture

## Package Layout

| Path | Role |
|---|---|
| `internal/jobs/cache`, `cmd/redis-cache` | Populates Redis or spread cache files from MySQL state. |
| `cmd/nats-client` | Local NATS client/log consumer command. |
| `cmd/spread` | Spread/cache command support. |
| `internal/jobs/ledger`, `internal/jobs/action`, `cmd/ledger`, `cmd/action-measurement`, `cmd/winloss`, `cmd/maxmind` | Operational commands for ledger, analytical action reconcile/retention, win/loss, and geodata workflows. |
| `dsp/` | DSP config, controller, bid handling, and win/loss logic. |
| `match/` | Runtime matching models for advertisers, creatives, audience maps, caps, sizes, and Redis/spread serialization. |
| `acl/` | Access/control and publisher mapping helpers used by bid and cache paths. |
| `maxmind/` | Geo/IP lookup helpers and tests. |
| `etc/` | Active SQL baseline, sample configs, generated local configs, samples, and data-load helper code. |
| `scripts/` | Local Docker service helper scripts. |
| `backup/` | Policy only; operational snapshots and third-party data stay outside Git. |
| `docs/` | Stable long-form references. |
| `memory-bank/` | Current product, architecture, tech stack, milestone, and status memory. |
| `evolution/` | Versioned history of direction changes. |

The sibling `../pzdesign` checkout is the Go module
`github.com/guruperl/pzdesign`. It owns `cmd/unify`, `genelet/`, `summer/`,
`tmpls/`, `www/`, and Summer/Genelet docs under `docs/`; its command and Summer
packages consume Aofei domain packages such as `dsp/`, `acl/`, `match/`, and
`uploaded/`.

## Runtime Data Flow

1. `scripts/aofei-local.sh` starts Docker MySQL, Redis, and NATS, then writes
   local configs into `etc/aofei.local.json` and `etc/summer.local.json`.
2. `etc/step4_init.sql` initializes the active MySQL schema and baseline data.
3. `etc/demand.sql` plus `go run ./etc pub` can load sample local demand and
   publisher data.
4. The cache job reads MySQL through `dsp.Config`, builds `PubMap`, the derived
   direct-SSP publisher-by-id lookup, `RAdv`, audience, creative, and
   Redis-only middleman route caches, discovers active creative size IDs from
   the schema, then replaces Redis cache entries or publishes spread/NATS reset
   and snapshot messages. It runs through standalone `cmd/redis-cache`;
   `cmd/unify` does not run cache refreshers. Route-only middleman cache
   publication is available through
   `cmd/redis-cache -cache=routes`. `cmd/spread` must be running when spread
   messages should become `.local/spread/` file snapshots; on startup it
   best-effort bootstraps those snapshots from Redis when Redis and MySQL are
   reachable.
5. `../pzdesign/cmd/unify` reads `SUMMER` and `AOFEI`, wires Summer/Genelet
   admin routes, and serves DSP bid paths using the same MySQL/Redis/NATS
   config. `GET /healthz` reports process liveness and `GET /readyz` reports
   lifecycle/local-generation readiness without leaking dependency detail.
   SIGINT/SIGTERM withdraw readiness before stopping new HTTP work, allow a
   15-second in-flight drain, then close the controller and its audit/service
   resources. A single-region deployment runs at least two identical nodes
   behind a load balancer that admits only ready nodes.
6. Bid/win/loss/log flows use Redis for mutable runtime state and NATS/spread/log
   paths for message and log transport. Local/spread bid mode loads static
   cache snapshots into memory at controller startup and periodically reloads
   the files at one-third of the tightest configured cache-age bound; request
   handlers read only the current in-memory snapshot.
   OpenRTB bid requests are matched per impression; response bids are grouped
   by campaign seat. DSP-generated `/imp` and `/clk` tracker URLs are HMAC
   signed over concrete query payloads, and click redirects plus cap mutations
   require valid signatures. `/win` and `/loss` remain analytics callbacks with
   signatures over immutable packed fields so exchange auction macros can still
   be resolved by the exchange. Middleman `/mid/*` callback proxy URLs are
   signed by token and store selected-bid context in Redis. Native click links
   use `/clk` as a tracking redirect with a direct advertiser fallback; banner
   creatives opt into the same redirect through `{CLICK_URL}`.
   D01 compiles campaign/ad-group UTC windows, campaign-timezone weekly
   calendars, deterministic pacing, and four budget scopes into RAdv payload
   version 2. The request path rejects stale/out-of-window/exhausted candidates
   and uses one Redis script to reserve every configured scope atomically.
   Budgeted local demand fails closed when reservation state is unavailable;
   response/materialization failures and signed loss callbacks release active
   reservations, published impressions finalize them, and click callbacks
   update click state idempotently. Total-budget Redis state is conservative
   and persistent; daily state is keyed by UTC date.
   S01 evaluates privacy signals before matching or disclosure. Missing,
   denied, opt-out, invalid, and currently unmapped GPP requests are
   contextual; COPPA is restricted; personalization requires an explicitly
   configured W8M TCF vendor/purpose contract and a current service-specific
   grant. Exact coordinates are removed in every mode. Contextual/restricted
   requests lose identity, IP, raw UA, precise geo, demographics, search data,
   and extensions before matching. Accepted
   local identity is HMAC-pseudonymized before cap keys and tracking tokens.
   Audit payloads are independently redacted, and external middleman fanout is
   always independently contextual, impression-scoped, and protected by a
   separate default-false disclosure gate.
   S03 traffic quality is another independent default-off boundary. Trusted
   workers submit bounded aggregate windows to the `trafficquality` domain,
   which derives only Replay, ImpossibleSequence, InvalidOriginApp,
   MalformedIdentity, AbnormalRate, AbnormalCTR, Automation, and PartnerPolicy.
   Raw internal keys are HMAC-digested before persistence. Versioned rules
   create immutable decisions, bounded evidence, scoped cases/history,
   aggregate counters, reviewed enforcement, and separate billing
   recommendations. Missing/partial evidence always reduces action and billing
   to Observe. `cmd/unify` loads a detached immutable enforcement snapshot at
   startup and refreshes it independently; publisher checks precede candidate
   lookup, advertiser checks filter local candidates, and partner checks filter
   middleman assignments. Refresh errors retain the last valid snapshot only
   through its maximum age, then serving fails open.
   Direct publisher SSP traffic is a separate `POST /pz` entrypoint. The
   browser contract uses `site` packed as `(pub_id, site_id)` and
   `adUnits[].slot` packed as `(slot_id, size_id)`; the browser DOM `code` is
   not trusted as supply identity. The `/pz` adapter validates these tokens
   against the direct publisher cache, including Web/App type, configured slot
   size, and server-owned USD CPM floor. The adapter requires a bounded safe
   code and exactly one matching media type, rejects browser/App or SDK/Web
   combinations, and uses the greater of configured and request floors,
   synthesizes internal OpenRTB browser impressions from headers and
   cache-derived site/slot strings, reuses the local Aofei bid path, and returns
   a JSON HTML-string array in ad-unit order with `""` for no-fill units.
   M29 publisher tags are generated from the `pub` slot UI using configured
   `ServerURL`, stored `pub_slot.size_id`, DOM ids of the form
	   `pz-slot-<slot_id>`, and banner `mediaTypes` samples. Web sites receive
	   browser samples and App sites receive SDK/API samples. `www/js/ads.js`
	   derives its default `/pz` endpoint from the loaded script origin and can be
	   overridden per call; it materializes filled markup in an opaque-origin
	   sandboxed `srcdoc` iframe and records filled/no-fill/error state without
	   assigning server markup to host-page `innerHTML`. `cmd/unify` applies permissive CORS headers only to
	   `POST/OPTIONS /pz`.
	   M30 identifies SSP traffic in audits with `source:"ssp"` and
	   `contract:"pz-v1"`; S01 supersedes its original fallback behavior so a
	   browser `aofei_pz_uid` is read or set only for an accepted personalization
	   grant, and IP+UA is never a contextual identity fallback. M31 keeps `/pz`
	   CORS credentialless and
	   enforces validated `POST /pz` policy instead: browser requests must send a
	   valid `Origin` or `Referer` whose host exactly matches the cached site
	   string, any present `Origin` or `Referer` must match, and only
	   `platform:"sdk"` may omit both headers. SDK/in-app requests do not read,
	   set, or propagate `aofei_pz_uid`. The direct SSP cache includes only
	   active publisher site/slot tuples, and `/pz` trusts forwarded IP headers
	   only from peers configured in `trusted_proxy_cidrs`. M32 lets SDK
	   requests include OpenRTB-like `app`, `device`, `user`, and `regs` objects,
	   synthesizes `BidRequest.App` from the validated cached site string,
	   rejects supplied app id/bundle/domain mismatches, applies SDK body identity
	   only after the S01 decision, and renders explicit
	   `responseFormat:"json"` fill objects or `"openrtb"` `BidResponse`
	   payloads while preserving omitted/`"html"` browser array responses. M33
	   runs the existing middleman runtime for valid `/pz` auctions after local
	   matching. Local no-fill impressions use `Fallback` candidates; local
	   filled impressions use `Always` candidates only when all middleman and
	   privacy gates are enabled. SSP fanout sends a bidder-specific scrubbed
	   OpenRTB request, while SSP request/response audits wrap scrubbed `/pz`
	   request and final response payloads. M34 selects the richer supply taxonomy
	   direction: add defaulted fields
	   to existing publisher tables for site/app identity, integration mode,
	   slot/media intent, and quality/source metadata; extend direct publisher
	   cache and audits additively; and keep `source:"ssp"`/`contract:"pz-v1"`
	   as the runtime entrypoint boundary. M35 closes the account/schema boundary
	   question by keeping the
	   existing `pub`, `pub_site`, and `pub_slot` ownership model for the current
	   `/pz` path and deferring any separate SSP account model until concrete
	   legal, settlement, intermediary, permission, compliance, or
	   partner-credential requirements exist. P02 implements the schema, cache,
	   Summer UI, privacy-safe attribute, and report fields. Seller authorization
	   belongs to the existing publisher account; publisher edits revoke it.
	   Direct `/pz` ignores client source claims and generates `source.schain`
	   from approved cache state. Middleman sanitation retains only a valid
	   bounded standard chain and strips `source.pchain` and node extensions.
7. `cmd/nats-client` consumes NATS log subjects into `.local/logs/log_*`
   interval files, runs under signal-aware shutdown, drains NATS on exit, and
   flushes its queued log messages before closing files. Generated log
   directories are tightened to `0750` and generated log files to `0640`.
   Generated subject files older than `privacy_log_retention_hours` are pruned
   at startup and rotation without touching unrelated files. The
   ledger job consumes `winloss.<stamp>` files into interval and daily ledger
   tables through standalone `cmd/ledger`; missing input remains retryable
   command input. Middleman callback metadata also populates `ledger_mid` and
   `daily_mid` for advertiser pay-side reports and admin settlement views.
   A01 converts CPM to one-impression USD spend (`CPM/1000`) in D01 reservations
   and ledger aggregation while leaving wire/tracker prices unchanged.
   `cmd/accounting` snapshots completed daily advertiser or publisher facts
   into DECIMAL statements, then enforces immutable adjustments,
   maker-checker transitions, source reconciliation, corrections, audit, and
   scoped CSV export without mutating ledger or Redis floors.
   R01 local click redirects add a separately signed, time-bounded action
   lineage token to the advertiser landing URL. `POST /action` validates the
   token and an exact-body request MAC before MySQL work, then inserts one
   `measurement_action` for each advertiser/event id. Successful impression
   and click publication separately attempts a bounded, detached
   `measurement_touch` write; failure is observable and fail-open relative to
   tracking. Attribution uses exact lineage, click before view, configured
   inclusive windows, and an unattributed fallback. The action tables expire
   under the privacy-log lifetime; `cmd/action-measurement` reconciles only
   unattributed rows, prunes expiry, and provides pseudonym-scoped privacy
   export/deletion. None of these flows reads or changes delivery reservation
   or A01 statement state.
   The same interval ledger transaction aggregates a separate
	   `report_delivery` map keyed by UTC interval plus a deterministic SHA-256
	   digest of the complete local/fallback/always demand, route, P02 supply,
	   authorized-seller, coarse country/state, and device OS/type tuple. Every
	   dimension remains an explicit report column. Local facts
   keep advertiser spend and publisher revenue equal to the billable
   per-impression USD amount; middleman facts retain charge, pay, nonnegative
   margin, raw downstream CPM, returned CPM, route, trigger, callback errors,
   and the A01 accounting version. Summer selects advertiser/publisher/operator
   SQL from the authenticated `_grole` and never from caller-supplied account-
   looking parameters. Actions are queried separately to avoid dimensional
   fan-out.
   The `reporting` package owns metric registry and controlled-experiment
   contracts. A privileged runtime loads a version, deterministically assigns
   a 32-byte pseudonym under a per-experiment salt, records one immutable
   exposure, then may record append-only idempotent primary/guardrail outcomes.
   Only hashes/digests cross the storage boundary. `cmd/report-experiment`
   provides explicit audited create/start/stop/complete transitions, bounded
   expiry prune, and exact audited subject deletion, with no serving mutation.
   D02 accepts only positive finite local USD CPM, selects the highest demand
   unit with deterministic cross-unit ties, and applies positive creative
   weights only within that winner. It validates local source/media/size/MIME,
   HTTPS and requested Native assets before D01 reservation; failed creatives
   are removed and the auction is rerun. Local Banner sources become iframe
   markup, Video sources become VAST 3.0, and Native version-1 source JSON is
   mapped only to explicitly requested assets with compatible MIME. Campaign
   `foreign_id` remains an external business identifier and is not repurposed
   as a Native fallback URL. Middleman bids pass strict ID, price, dimensions,
   media, callback, secure and contained Banner/VAST/Native checks, including
   absolute VAST resource URLs and Native version/asset compatibility, before
   markup and competition. Response failure or a selected middleman replacement
   idempotently releases the displaced local reservation.
8. `cmd/maxmind` reads country and state IDs from Docker MySQL and atomically
   regenerates the configured MaxMind runtime JSON without loading the existing
   geodata file first.

Middleman fallback is active behind `middleman_enabled` for ADX `/bid` and
validated direct SSP `/pz` auctions. Advertiser-owned
OpenRTB endpoints live in `adv_bidder`, with synthetic campaign, item, and
creative IDs for existing ledger/report joins. Summer/Genelet exposes
advertiser-safe endpoint metadata forms and admin review and approval forms.
Approval creates a missing inactive synthetic chain or validates an existing
complete same-advertiser chain, then marks the bidder credential active and the
bidder active. Operators assign active route groups to publisher/site/slot
inventory through the admin `midroute` Summer module, which writes the
`mid_route_*` tables. The Redis `middleman:routes:v2` cache contains active
route/bidder entries, trigger mode, and synthetic item ACL payloads; the legacy
`middleman:routes` key is kept fallback-only for M24 rolling-deploy safety.
`Fallback` routes apply only to local no-bid impressions. `Always` routes apply
only when both `middleman_enabled` and `middleman_always_enabled` are true, and
then marked-up middleman bids compete with local bids on effective CPM. Callback
proxy setup remains the materialization gate; callback setup failure falls back
to a local winner when one exists, otherwise that impression no-fills. Route
edits do not refresh the cache from `cmd/unify`; the singleton
`cmd/redis-cache -cache=redis|all` job remains the cache publication path, with
`-cache=routes` available for route-only refresh.
Each HTTP worker keeps the decoded route result for the configured short TTL;
concurrent misses share one Redis load with an independent
`middleman_timeout_ms` deadline. Each caller waits with its own context, so
request cancellation does not cancel the shared load or fail other waiters;
the load still populates the result/error snapshot, and refresh failure does
not reuse an expired route snapshot.
The cache JSON includes additive freshness metadata when generated by M24+
cache jobs, and the admin `midroute` topics/health views show Redis freshness
and route health without running refreshes. Selected middleman winners create
Redis callback context under `middleman:cb:<token>` and return signed
`/mid/win`, `/mid/loss`,
and optional `/mid/bill` URLs. `burl` is the preferred billable event and win is
the billable fallback only when no `burl` exists. Downstream callbacks receive
net payable auction prices; Aofei logs charge-side prices through the synthetic
chain and middleman-specific charge/pay/margin facts through `ledger_mid` and
`daily_mid`. Retryable downstream callback forwarding failures are stored in
MySQL `mid_callback_retry` by `/mid/*` handlers only, never by `/bid`; the
singleton `cmd/mid-callback-retry` job claims rows as `Processing` and retries
downstream forwards without republishing ledger events. Its default output
remains a human-readable summary line, and `-json` emits stable fields
`due`, `stale_processing`, `selected`, `succeeded`, `retrying`, and
`abandoned` for automation.

D03 keeps middleman routes Redis-only so activation and revocation have one
shared timeline. The read-only `cmd/redis-cache -validate-middleman` boundary
rebuilds the active MySQL model, requires the published Redis v2 checksum and
route high-water to match, validates config/partner profiles, and resolves each
environment credential reference without returning header values. Its
`preflight`, `fallback`, and `always` stages enforce progressively stronger
runtime/privacy gates; route publication and traffic rollout remain separate
mutations.

I01 fixes partner transport and validation at OpenRTB 2.5. O01 admission bounds
encoded bytes; I01 decodes one gzip layer under an independent limit and
compresses successful JSON only after `Accept-Encoding` negotiation. Each
middleman call carries only assigned unique impressions, USD CPM/floors,
exactly one Banner/Video/Native intent, controlled extensions, and the minimum
available timeout. Runtime accepts only matching request/seat identity, USD,
raw price at or above floor, on-time bids, active synthetic reporting IDs, and
D02-valid media/size/secure markup/callbacks. Fixed rejection/candidate maps,
one latency histogram, and sampled hashed metadata diagnostics avoid partner
label cardinality and raw-body disclosure.

Request, response, and attribute audit messages are best-effort analytics.
`dsp.Controller` enqueues them to a bounded in-process queue after writing the
HTTP bid response, and a background publisher sends them to core NATS without
request-path flushes. Routine successful bids and expected no-bids do not emit
process logs; rejected inputs and operational failures use structured `zap`
fields at debug, warning, or error level according to actionability.

## Admin Runtime Boundary

Summer/Genelet admin code lives in the sibling `../pzdesign` module and uses
the generated `SUMMER` config and Docker MySQL. Admin tests that need a database
read `../aofei/etc/summer.local.json` when run from `../pzdesign`; they must
not use the lower-case DSP `AOFEI` config because Genelet expects
`ConnectArray`, `Template`, and `UploadDir`.

All account, campaign, publisher, report, route, and creative-review values
cross the UI boundary as ordinary data under Go `html/template` contextual
escaping. Genelet's fixed CSRF hidden-input renderer is the sole approved raw
HTML conversion. Summer management/review pages show stored creative markup or
URLs only as escaped source; intentional auction delivery of approved creative
markup remains an Aofei response contract owned by D02 validation, not a UI
safe-HTML exception. The inventory and rules are in
[docs/template-rendering-security.md](../docs/template-rendering-security.md)
and `../pzdesign/docs/rendering-security.md`.

The source/runtime boundary and populated-data rollout are specified in
[docs/auction-pricing-creatives.md](../docs/auction-pricing-creatives.md).

## Active Configuration Boundary

- `etc/aofei.json` and `etc/summer.example.json` are checked-in examples.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated local files
  and must remain ignored.
- Summer/Genelet code, UI templates, and static UI assets live in the sibling
  `../pzdesign` module. Generated local Summer config points `ProjectRoot` at
  that checkout, `Template` at `../pzdesign/tmpls`, and `DocumentRoot` at
  `../pzdesign/www`.
- Production configs default to `/etc/aofei/aofei.json` and
  `/etc/aofei/summer.json`, passed through `AOFEI` and `SUMMER`.
- `etc/maxmind.json` is the active MaxMind config reference.
- `etc/maxmind.json` references an external GeoLite2 City `.mmdb` through
  `city_file`, currently `/media/GeoLite2-City.mmdb`.
- Real geodata payloads are external runtime/test assets. `etc/GeoLite2-City.mmdb`
  and `etc/qq-pz.dat` are ignored and must not be committed.
- The retired root config directory is no longer active and should not be
  recreated.
- Operational commands use the generated `AOFEI` config. `cmd/ledger`,
  `cmd/winloss`, and `cmd/maxmind` disable controller NATS and MaxMind startup
  explicitly when they only need database/config access. Redis cache refresh
  remains a singleton scheduled `cmd/redis-cache` job on one dedicated node.
  Ledger runs as a singleton scheduled `cmd/ledger` job on the node where
  `cmd/nats-client` aggregates win/loss log files.
- Mutating operations commands acquire token-owned Redis singleton locks that
  renew at one-third of their lease while work is active, report uncertain or
  lost ownership, and cannot delete a successor's lease. Database idempotency
  remains the authoritative split-brain backstop. The unified
  HTTP service exposes stdlib expvar metrics at `/debug/vars`. The endpoint
  authorizes only direct peers in `metrics_allowed_cidrs` (loopback by default)
  and must also be blocked at the edge. Authorized scrapes perform bounded
  Redis/MySQL checks and read NATS state.
- Public `/bid/{domain}` and `/pz` requests pass through fixed-config admission
  gates before their controllers. Exact configured ADX partners and `ssp` have
  independent token buckets/concurrency slots; unlisted ADX paths share a
  bounded default pool. Body, timeout, QPS, and concurrency failures return
  413, 503, or 429 without consuming admin/tracker capacity.
- Middleman callback proxying uses Redis TTL keys for selected-bid context,
  cooperative click mapping, and billable-event idempotency. These keys are
  runtime state owned by `cmd/unify`, not cache data populated by
  `cmd/redis-cache`.

## Cache Boundary

The multiple-cache split is documented in
[docs/multiple-cache.md](../docs/multiple-cache.md). Static publisher, slot,
audience, and creative data is local and snapshot-swapped in memory for
local/spread bid serving. Redis remains the shared mutable-state backend for
frequency caps, uploaded audience sets, and delivery reservations/counters. Frequency-cap
tracker updates keep the `bothcap:<user_id>` hash and binary `BothCap` payload,
but refresh through Redis optimistic transactions to avoid concurrent lost
updates. Packed counters saturate at 255, valid signed events without a user
identity skip cap mutation but remain measurable, and each cap write ensures
the configured idle TTL without shortening a longer key TTL. Bulk cap writes
commit data and conditional expiry in one Redis script, preserving any longer
existing TTL while adding expiry to new or persistent keys.
Uploaded audience writes use one Redis script to commit membership and install
`privacy_audience_ttl_seconds` on new, persistent, or shorter-lived sets without
shortening longer retention. Scoped delete helpers remove one identifier or one
advertiser/marker set without exporting neighboring values.
Signed impression and click callbacks acquire independent short Redis
processing claims before cap mutation and ledger publication. A successful
publication converts the owned claim to a marker expiring at the signature's
exact validity deadline; publication failure releases it so a legitimate retry
can proceed. Cap mutation and a per-event cap marker commit in one Redis
transaction, preventing that retry from incrementing cap state twice. A cap
failure does not release the owned claim before publication. Duplicate events
keep normal HTTP behavior but skip both side effects. Claim/cap Redis failures
fail open, keyed events still attempt the idempotent cap transaction after a
claim failure, and unkeyed events publish without cap mutation. Tracking Redis
work is bounded to two seconds and detached from HTTP cancellation.
Full Redis static refreshes build shadow families and replace all live hashes
in one transaction, including empty-family and obsolete-slot deletion. A build
failure leaves the previous live generation untouched.
Direct SSP uses an additive `pubmap:by-id` Redis hash derived from `pubmap`.
The value includes publisher domain, the active publisher object, reverse
active site/slot metadata, site type, slot size, and configured USD CPM floor so
`/pz` can validate commercial packed tokens, enforce platform and floor policy,
and reconstruct site/slot strings without a MySQL read. Full cache publication
validates this metadata first. `cmd/redis-cache -validate-publishers` is a
DB-only, no-mutation readiness mode that emits a deterministic packed-token
manifest. P01 readers reject pre-P01 publisher entries lacking type/floor;
older readers decode the additive fields, so operations publish the new cache
generation before rolling new HTTP workers.
Local/static mode derives the same lookup from the loaded `pubmap` snapshot in
memory; `/bid/{domain}` continues to read the existing domain-keyed publisher
cache.
RAdv version 2 carries delivery snapshot generation time, campaign/ad-group
windows and calendars, pacing modes, and reconciled balance facts. New readers
retain version-1 and unversioned decode support. A candidate with delivery
policy fails closed after `delivery_cache_max_age_seconds`; full cache
publication must therefore run at no more than one-third of that interval.
Redis delivery reservations use `delivery:reservation:*`, persistent
`delivery:budget:total:*`, and UTC-date `delivery:budget:daily:*` keys.

## Database Boundary

`etc/step4_init.sql` is the source-of-truth schema and reference-catalog
baseline for local MySQL. Mutable account, campaign, publisher, ledger, login,
and traffic rows belong only in the synthetic `etc/demand.sql` fixture or in
deployment-managed data. When Docker MySQL schema changes are intentionally
made, export or otherwise update `etc/step4_init.sql` in the same change. The
baseline must not contain explicit legacy definers, named database auth,
production-derived records, or personal data.
The D01 baseline adds `adv_balance.current_day`, campaign delivery timezone and
weekly schedule/pacing fields, and ad-group weekly schedule/pacing fields.
Interval and daily ledger jobs reconcile those balance baselines before cache
compilation; deployment-managed migrations must precede the version-2 cache
compiler.
The A01 baseline records `usd-cpm-impression-v2` in `acct_contract` and adds
`acct_statement`, `acct_adjustment`, and `acct_audit`. Audit and adjustment
rows are database-immutable. Inactive `pay_*` compatibility tables retain only
non-identity reference/status metadata; full credential, sender, and payment-IP
columns, the `view_payment` view, and the balance-crediting `trig_payment` are
absent. pzdesign no longer registers legacy funding modules or exposes the
retired account-balance action.
The A02 baseline adds `hosted_binding`, `hosted_operation`,
`hosted_provider_object`, `hosted_event`, `hosted_reconciliation`, and
`hosted_audit`. `cmd/unify` registers exact `POST /webhooks/stripe` only when
the disabled-by-default provider service starts, while Summer derives every
human actor/account scope from S02. Provider calls use stable idempotency keys
outside database transactions; a durable `Submitting` claim linearizes the
call, a bounded stale-admin takeover reuses that key, and every attempted call
stops replay before Stripe's 24-hour pruning boundary. Uncertain calls remain
capacity-reserved and cannot be locally canceled. Funding and payout both
require independently approved provider-ready bindings; payout country is an
immutable non-secret binding input. The first submission freezes that binding
ID on the operation, and every retry reloads the same token even after a later
binding replacement. A fast signed webhook may safely win the
response race. Provider events are claimed before mutation,
preserve immutable linked provider objects, and update locked operations under
timestamp plus event-specific transition rules. Read-committed transactions,
exact owner row locks, and unique event/object identities serialize concurrent
deliveries without pre-owner gap-lock deadlocks. Connect events must carry an
exact top-level account owner; only account readiness and payout-failure events
can enter the payout-binding namespace, so connected-account direct charges
cannot claim platform operations through metadata. Dependency-pending signed
events are durably unresolved and return 503; an identical retry may make one
trigger-guarded processing transition after its owner mapping appears, while
the signed envelope/hash remains immutable. Authorized reconciliation
then retrieves each linked Balance Transaction through the pinned provider API
and records a single matched fact or explicit exception after checking status,
source ownership, direction, amount, fee, and net. All stored provider
identities are opaque; immutable evidence and A01 statements—not hosted
redirects, raw bodies, credentials, or UI state—form the reconciliation
boundary.
The R01 baseline adds expiring analytical `measurement_touch` and
`measurement_action` tables. Their unique advertiser/event key provides
concurrency-safe action idempotency; neither table is a financial ledger source
or contains a D01 reservation token.
The R02/P02 baseline adds `report_delivery` plus experiment definition, variant,
exposure, outcome, and audit tables. Delivery interval uniqueness uses a
SHA-256 dimension hash over every explicit demand, route, supply taxonomy,
authorized-seller, geography, and device column. The generated two-byte
OS/type key remains lookup-compatible. Exposure/outcome triggers reject updates;
bounded prune or exact audited erasure deletes them, while experiment audit
triggers reject update/delete. Report storage is derived and reconciles to ledger/action/A01
facts; it is not an auction, reservation, statement, or settlement authority.
O02 makes `ledger_log.timely` and `daily_log.daily` unique durable identities.
The renewable Redis lease prevents normal overlap, while these constraints
reject duplicate interval/day source rows if ownership becomes uncertain.
S02 adds `analyst`, encrypted `auth_mfa`, digested `auth_recovery_code` and
`auth_session`, exact `auth_permission_grant`, and immutable
`auth_security_audit` tables. Genelet pairs its signed role cookie with the
opaque database session, derives the actor/account scope only from verified
authentication, checks a named component action permission and optional exact
resource before component `Preset`/`Before` and model actions, and requires recent MFA for sensitive
actions and report exports. Existing component foreign-key signatures remain
the advertiser/publisher ownership boundary. Security evidence uses redacted
fixed fields and a separate 365–2555-day retention class; TOTP/recovery/session
material never enters logs or audit rows. The encryption/HMAC root is a
deployment-owned 32-byte environment key shared by all HTTP nodes and the
restricted identity maintenance host. The HTTP database principal has no
update/delete grant on the audit table; the bounded retention CLI uses a
separate maintenance principal and configuration in addition to the trigger's
connection-local deletion gate.
I03 adds an isolated management-plane handler before the Genelet catch-all in
`cmd/unify`. It authenticates `w8m_v1` bearer tokens by public lookup plus a
constant-time comparison of a deployment-keyed digest, derives one advertiser
scope, then atomically admits per-credential/per-account Redis quotas. Four API
tables own credentials, 24-hour idempotent response replay, cache-activation
operations, and immutable redacted audit. Campaign, item, and creative
before-update triggers advance `api_version` for portal and API edits. A
cache job first assigns an opaque generation to pending operations visible
before it reads configuration, then marks only that generation Active after the
configured serving backend's Redis/spread publication completes. Mutations
committed during the build wait for the next generation; deadline expiry is
reported as Delayed and never fabricates publication. This acknowledgement
does not bypass `New`/`Prepare` review eligibility. The public contract never
routes through internal Summer JSON.
S03 adds nine `quality_*` tables for rules, immutable decisions, expiring
evidence, scoped cases/events, reviewed enforcement, billing recommendations,
hourly counters, and immutable audit. Ten triggers prevent rule-behavior,
decision, case-history, evidence, and audit rewrites; evidence deletion needs a
connection-local retention gate that is reset on a fresh bounded context or the
connection is discarded. Database checks preserve incomplete-evidence
observe-only and unambiguous canary state. Summer derives actors/scopes from
S02 identity, gives advertiser/publisher self-scope appeal, exact delegated
reads to agent/analyst roles, and recent-MFA administrator mutation. Billing
holds can affect only Draft/Confirmed A01 statements through separate
recommend/approve actors and immutable accounting audit.

## Known Architecture Gaps

- The parent `go.work` still does not include this module path, so repository
  package commands require `GOWORK=off` unless that workspace is intentionally
  changed.
- Production deployment now has a Linux systemd-oriented runbook; local Docker
  remains the development workflow, not the production ownership model.
- The single-region topology, recovery objectives, and proposed 99.9% SLO are
  defined, but production achievement remains unclaimed until a named rolling
  measurement window and provider-backed recovery evidence are retained.
- Runtime config parsing needs one validation/defaulting boundary across DSP
  and Summer/Genelet so missing service blocks fail with actionable errors.
- Redis and spread campaign cache payloads use typed version envelopes for
  RAdvs, audience, and creative data while retaining legacy decode support.
  The middleman route Redis payload is versioned JSON.
- Direct SSP richer supply taxonomy is implemented under
  [ADR 0001](../docs/adr/0001-richer-supply-taxonomy.md). P02 extends the
  existing publisher schema/cache/UI additively, derives privacy-safe audits
  and R02 dimensions, and emits seller chains only from approved server state.
  The runtime entrypoint remains `/pz` plus audit `source:"ssp"` and
  `contract:"pz-v1"`.
- Direct SSP account/schema ownership is documented in
  [ADR 0002](../docs/adr/0002-ssp-account-schema-boundary.md). M35 decides not
  to add a separate `ssp` account role or separate SSP-owned inventory schema
  for the current direct SSP path.
- Summer/Genelet admin SQL now has a central identifier/query-building seam for
  component metadata and request-driven filters; handwritten module SQL should
  continue to use narrow allowlists for any interpolated identifiers.
- Summer/Genelet admin auth verifies stored bcrypt password hashes through the
  `Password_hash` issuer field. S02 can immediately replace an exact legacy
  plaintext match with bcrypt during a measured compatibility window; the
  switch is disabled after a credential audit. Direct SQL issuers declare
  ordered `OutPars` matching every selected column, including the password-hash
  column. Identity enablement remains deployment-gated rather than an automatic
  baseline behavior.
