# Milestones

This file stays at milestone level. Detailed task rows live in the matching
`memory-bank/status-M*.md` files. Do not recreate an aggregate
`memory-bank/status.md` file.

Status markers:

| Symbol | Meaning |
|---|---|
| `[ ]` | Pending |
| `[+]` | Completed |
| `[~]` | In progress |
| `[!]` | Blocked |
| `[X]` | Cancelled |

## Closeout Checklist

Use this order when closing a milestone:

1. Run the milestone's required verification commands.
2. Update code-adjacent docs and the memory-bank files that changed behavior,
   contracts, tools, or operator workflow.
3. Resolve review findings or carry them forward explicitly in the matching
   `status-M*.md` file with a dated note.
4. Check whether `evolution/` needs a new prompt/result version.
5. Mark the matching `status-M*.md` tasks, review findings, and milestone state
   complete only after verification passes.
6. Commit the milestone only when the execution request includes commit
   handling.

## M0 - Agentic Harness Bootstrap `[+]`

Create the project operating layer for future agent work.

Scope:

- Add `AGENTS.md`, `memory-bank/`, `evolution/`, and `docs/`.
- Rewrite the root README as a clean current operator entry point.
- Move or preserve old long-form documentation under `docs/`.
- Record milestone-level direction without inventing detailed task breakdowns.

Acceptance:

- A new agent can read `AGENTS.md` and know where to start.
- Product, architecture, tech stack, milestones, and coarse status are captured.
- Historical docs are not lost, but the root README no longer presents legacy
  manual setup as the active workflow.

## M1 - Local Docker Runtime Stabilization `[+]`

Make the Docker-backed development runtime the unquestioned local baseline.

Scope:

- MySQL, Redis, and NATS run from Docker using repository helpers.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated and ignored.
- Local service auth uses Docker helper credentials only.
- Reset/load/sample/status commands are documented and repeatable.

Acceptance:

- A clean checkout can start local services, load the database baseline, load
  sample data, populate Redis, and report service status.
- No active workflow depends on the retired root config directory or legacy
  database credentials.

## M2 - Schema Baseline Stewardship `[+]`

Make `etc/step4_init.sql` the durable schema and baseline-data contract.

Scope:

- Define a repeatable schema comparison path between Docker MySQL and
  `etc/step4_init.sql`.
- Keep views, routines, triggers, and table definitions covered.
- Strip legacy definers and production auth from baseline SQL.
- Document how to update the baseline after intentional schema changes.

Acceptance:

- Docker MySQL schema can be recreated from `etc/step4_init.sql`.
- Drift between Docker MySQL and the baseline can be detected and reviewed.

## M3 - Redis And NATS Cache Pipeline Reliability `[+]`

Prove that cache and message-bus flows work from the Docker services.

Scope:

- Validate `cmd/redis-cache` Redis mode, spread mode, and combined mode.
- Verify `PubMap`, `RAdv`, audience, and creative cache structures after sample
  data is loaded.
- Confirm NATS connectivity for paths that require it.
- Keep Redis key inspection and cache-read commands documented.

Acceptance:

- Cache population succeeds against Docker MySQL and Docker Redis.
- NATS-required cache/spread flows connect to Docker NATS.
- Cache inspection commands show expected sample objects.

## M4 - Bid Path Smoke Coverage `[+]`

Add a reliable local proof that the DSP request path still works.

Scope:

- Use existing `etc/samples/` OpenRTB fixtures where possible.
- Exercise config loading, Redis cache reads, matching, bid response generation,
  and win/loss-adjacent paths.
- Keep tests deterministic and independent of production services.

Acceptance:

- A documented command proves the seeded local runtime can process representative
  bid-path samples.
- Failures point to config, schema, cache, fixture, or matching boundaries.

Result:

- `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'`
  covers the Redis-backed `ServeBid` path with local sample data and controlled
  malformed, oversized, and no-bid failure modes.

## M5 - Summer/Genelet Admin Compatibility `[+]`

Align admin models, filters, and components with the active Docker schema.

Scope:

- Check Summer model/filter tests against the current schema.
- Confirm active tables, views, and stored routines still match admin model
  expectations.
- Keep generated local config and upload/template paths out of git.

Acceptance:

- Admin model smoke tests run against the Docker schema.
- Known admin/schema mismatches are either fixed or explicitly listed for later
  product decisions.

Result:

- Summer root/pub/slot model and filter tests and Genelet framework tests pass
  against Docker MySQL via `SUMMER="$PWD/etc/summer.local.json"`.
- Stale slot/weight schema assumptions were corrected; larger Genelet
  query-builder hardening remains tracked as future architecture work.

## M6 - Ledger, Logs, And Operational Commands `[+]`

Clarify and verify the non-bid operational commands.

Scope:

- Review `cmd/ledger`, `cmd/nats-client`, `cmd/winloss`, `cmd/spread`, and
  related SQL/log paths.
- Define what local data each command needs.
- Separate active command workflows from historical deployment notes.

Acceptance:

- Each active operational command has a documented local invocation or a clearly
  recorded blocker.
- Log and ledger flows do not require production credentials.

Result:

- `docs/operational-commands.md` documents local prerequisites, invocations,
  outputs, and M6/M7 blockers for `cmd/ledger`, `cmd/nats-client`,
  `cmd/winloss`, `cmd/spread`, and `cmd/maxmind`.
- DSP controller startup now has typed options for disabling NATS and MaxMind;
  ledger, win/loss simulation, and MaxMind inventory commands use those options
  instead of string modes.
- NATS log consumption uses a copied-message queue and a single writer loop for
  file handles and rotation.
- Ledger interval and daily writes are transactional, missing win/loss files are
  retryable missing-input errors, and win/loss statistics aggregate demand by
  creative id.
- Win/loss simulation validates no-bid and malformed native tracker responses
  before indexing response slices.

## M7 - MaxMind And Geo Runtime `[+]`

Make geodata expectations explicit and locally testable.

Scope:

- Keep `etc/maxmind.json` as the active geodata config reference.
- Document which MaxMind database files are external runtime inputs.
- Verify existing maxmind/ipsearch tests with local fixtures or clear skips.

Acceptance:

- Developers know which geo assets are required and where config points.
- Geodata tests are either runnable locally or marked with explicit input
  requirements.

Result:

- `docs/maxmind-runtime.md` documents `etc/maxmind.json`, the external
  GeoLite2 City `.mmdb` path, ignored local geodata assets, generation, and
  verification commands.
- `cmd/maxmind` now loads only DSP config/database access, generates the
  country/state maps without loading existing MaxMind runtime data, and writes
  the configured JSON atomically.
- Asset-backed lookup tests skip explicitly when `etc/GeoLite2-City.mmdb` or
  `etc/qq-pz.dat` is absent; compile and pure utility tests remain local-safe.

## M8 - Full Repository Test Hygiene `[+]`

Move from scoped smoke checks to a clean repository-level verification target.

Scope:

- Resolve the `backup/` Go package discovery issue.
- Decide the canonical command for full Go verification.
- Add or update CI-style verification notes.
- Keep historical files preserved without polluting active package tests.

Acceptance:

- `GOWORK=off go test ./...` is clean and is the canonical package verification
  command.
- The root README and `AGENTS.md` point to the same verification target.

Result:

- Historical Go helpers under `backup/` are build-ignored and no longer appear
  in `GOWORK=off go list ./...`.
- Advice enum tests now treat zero values as wildcard `"All"` and invalid
  out-of-range values as `"Unknown"` where the stringers support it.
- Native fixture tests read from `etc/samples/` and assert `Native.AdM`
  round-trip behavior instead of emitting debug failures.
- Summer/Genelet DB-backed tests skip cleanly when generated local config is
  absent or the configured DB cannot be reached; malformed configs still fail.
- Staticcheck, Docker smoke/admin checks, and schema drift checks remain
  documented non-gating follow-ups for M8.

## M9 - Production Deployment Runbook `[+]`

Rebuild production/operator docs from current reality rather than legacy notes.

Scope:

- Review historical systemd/NATS/Redis/TDengine notes under
  `docs/legacy-operations.md`.
- Decide what still applies to production.
- Separate local Docker development from production deployment.
- Document secrets, backup/restore, service ownership, and rollout assumptions
  without committing credentials.

Acceptance:

- Operators have one current deployment/runbook document.
- Historical notes are either retired or clearly marked as retained context.

## M10 - Documentation And Agent Stewardship `[+]`

Keep the harness useful as the project changes.

Scope:

- Maintain `README.md`, `AGENTS.md`, `memory-bank/`, `evolution/`, and `docs/`.
- Add new evolution versions only for real direction or boundary changes.
- Keep milestone docs summary-level and detailed task lists in `status-M*.md`.

Acceptance:

- A future agent can resume from the memory bank without rediscovering local
  runtime, schema, or documentation decisions.
- Milestones, status, and docs remain consistent after each substantial change.

## M11 - Genelet And Summer UI Stewardship `[+]`

Review and harden the Genelet/Summer admin framework surface without changing
DSP bid behavior, database schema, cache payload contracts, or production config
shape.

Scope:

- Move the remaining M5 Genelet/Summer hardening findings into active work.
- Replace active component-loading panics with error-returning setup paths.
- Add a central Genelet SQL identifier/query-building validation seam for
  component metadata and request-derived fields, filters, and ordering.
- Guard controller/model/filter reflection dispatch, auth header parsing,
  login/logout error handling, static files, multipart input, and Summer option
  state.
- Centralize the Summer module registry used by `cmd/unify`.
- Add Genelet and Summer UI maintenance documentation under `docs/`.

Acceptance:

- `cmd/unify` initializes Summer modules from one registry and reports
  component/setup errors instead of panicking.
- Active CRUD/query paths reject unsafe identifiers and request-provided raw SQL
  fragments before database execution.
- Controller dispatch and forwarded group handling return framework errors for
  malformed inputs instead of panics.
- Summer request-specific UI selections do not mutate shared `LARGES` option
  data, and cache side effects use typed storage helpers.
- `../pzdesign/docs/genelet-manual.md` and
  `../pzdesign/docs/summer-ui-structure.md` describe the current
  framework/operator contracts.

## M12 - OpenRTB And Audience Matching Review `[+]`

Run a documentation-first review of the OpenRTB bid path, attribute extraction,
audience matching, cache contracts, selection, and measurement flow.

Scope:

- Review `advice`, `demo`, `dh`, `maxmind`, `acl`, `match`, and `dsp`.
- Document request, response, win, loss, impression, click, NATS, log, and
  ledger flow.
- Document attribute extraction, audience data sources, matching predicates, and
  Redis/spread cache contracts.
- Record findings with severity, evidence, impact, recommended fix, and
  disposition.
- Do not refactor runtime code, schema, Redis/spread payloads, or production
  config shape in this pass.

Acceptance:

- `memory-bank/status-M12.md` contains the review findings and verification.
- `docs/openrtb-measurement.md`, `docs/audience-matching.md`, and
  `docs/dsp-workflow.md` describe the current behavior and known gaps.
- Deferred implementation work is carried into M13 instead of being hidden in
  prose.

## M13 - OpenRTB And DSP Refactor Backlog `[X]`

Resolve the concrete implementation and design work discovered during M12.

Scope:

- Fix or explicitly product-scope OpenRTB bid-path gaps from M12.
- Add targeted tests around ACL, geo/date-hour enrichment, multi-impression
  behavior, app/video/native rendering, uploaded audience priority, selection
  math, spread cache publishing, and measurement edge cases.
- Keep Redis/spread cache and OpenRTB wire contract changes explicit and
  documented.

Acceptance:

- Each M12 deferred finding is fixed, cancelled with rationale, or carried
  forward with a named owner milestone.
- Behavior changes include focused tests and updated docs/memory bank entries.

## M14 - Redis And Spread Cache Reliability `[+]`

Harden the Redis/static spread cache path after the post-M13 cache review.

Scope:

- Make spread subscriptions receive cache subjects whose payload keys contain
  dots, especially publisher domains.
- Replace in-place spread file writes with atomic snapshot replacement.
- Let `cmd/spread` bootstrap local files from Redis on startup when Redis/DB are
  available, so a restarted receiver can recover current static cache state.
- Make full Redis and spread refreshes remove stale static cache records.
- Recompute item-level RAdv cache refreshes from MySQL slot state rather than
  merging against local spread files.
- Add an in-process local static cache for local/spread bid serving, while
  keeping caps and uploaded audiences Redis-backed.

Acceptance:

- Focused command/package tests cover spread subject routing, reset/cleanup
  handling, creative file map keys, and the local bid path.
- Docs describe the Redis, NATS, file, and in-memory cache roles clearly.

## M15 - DSP Serving Hardening `[+]`

Resolve the post-M14 DSP serving review findings without reopening M14.

Scope:

- Sign DSP-generated tracker and click URLs and require valid signatures for
  click redirects and cap mutations.
- Remove request-path filesystem generation checks from local static cache
  reads.
- Make frequency-cap refresh atomic while preserving Redis key and payload
  shape.
- Move request/response/attribute audit publishing off the request goroutine.
- Preserve creative URL query parameters and repeated values during macro
  replacement.
- Clean the focused staticcheck finding in DSP tests.

Acceptance:

- Focused tests cover signed click redirect acceptance/rejection, local static
  cache read behavior, async audit queue drops, repeated-query macro expansion,
  and cap refresh behavior.
- Measurement, workflow, cache, audience, and memory-bank docs describe the new
  serving contracts.

## M16 - Middleman AdX Advertiser-Owned Bidder Schema `[+]`

Establish the advertiser-owned endpoint, routing, and synthetic reporting schema
needed before fallback fanout changes the bid path.

Scope:

- Add downstream endpoint metadata under `adv_bidder`, owned by `adv`.
- Link each bidder endpoint to optional synthetic campaign, item, and creative
  IDs so existing advertiser ledger/report joins can be reused.
- Add `mid_route_*` tables for future fallback route groups and inventory
  assignment.
- Document that runtime fanout is still disabled after this milestone.

Acceptance:

- The active schema baseline recreates empty `adv_bidder` and `mid_route_*`
  middleman tables.
- Summer registry includes the advertiser-owned bidder endpoint module.
- Docs and memory bank describe the advertiser-owned endpoint/reporting boundary
  ACL/channel eligibility reuse, and future milestone sequence.

## M17 - Advertiser Bidder Portal `[+]`

Make `adv_bidder` usable from Summer/Genelet without enabling DSP runtime
fanout.

Scope:

- Expose advertiser HTML routes for bidder list, new, insert, edit, and update.
- Expose admin HTML routes for bidder list, edit, update, and approval.
- Keep JSON routes under `/goto/{role}/json/bidder`.
- Restrict advertiser writes to safe endpoint metadata and expose credential
  status plus active status as read-only.
- Validate bidder endpoint URLs and timeout values before writes.
- Let admin approval create or validate inactive synthetic reporting rows,
  store a credential ref, and mark the bidder active.
- Move active Summer templates to the sibling `../pzdesign/tmpls` tree and point
  generated local Summer config away from ignored `.local/templates`.

Acceptance:

- Advertisers cannot set credential refs, credential state, activation, or
  synthetic IDs.
- Admin approval requires `bidder_id` and `credential_ref`, runs in one DB
  transaction, creates or validates the synthetic campaign/item/creative chain,
  and rejects partial or wrong-advertiser synthetic state.
- DSP bid serving, route cache, downstream OpenRTB fanout, auctions, and
  callback proxying remain unchanged.

## M18 - Summer Template Modernization `[+]`

Make the sibling `pzdesign` UI tree the active Summer/Genelet template and asset
source, and move rendering to Go `html/template`.

Scope:

- Use `../pzdesign/tmpls` for Summer HTML templates and `../pzdesign/www` for
  static UI assets in generated local config.
- Keep `.g` templates as the primary runtime surface; keep `.e` variants
  parse-clean where practical.
- Convert Genelet HTML, login, error, and mail template rendering to
  `html/template`.
- Add advertiser and admin bidder `.g` pages to `pzdesign/tmpls`.
- Add bidder navigation links to the existing advertiser and admin sidebars.

Acceptance:

- All active `.g` action templates in `../pzdesign/tmpls` parse as
  `html/template`.
- Bidder advertiser/admin pages render in tests against the sibling template
  tree when it is present.
- Existing `.e` variants are parse-clean as best-effort coverage but remain
  secondary to the active `.g` templates.

## M19 - Maintenance Job Package Refactor `[+]`

Refactor cache and ledger jobs for reuse, keep Redis cache population as a
singleton scheduled command, and keep ledger on the log aggregation node.

Scope:

- Refactor `cmd/redis-cache` and `cmd/ledger` logic into reusable internal job
  packages while keeping the standalone command flags.
- Keep Redis cache refresh, ledger, `cmd/nats-client`, `cmd/spread`, and
  `cmd/winloss` separate commands.

Acceptance:

- `cmd/unify` behavior remains HTTP UI and ADX serving only.
- Redis cache refresh remains a singleton cron/timer job on one dedicated node.
- Ledger runs only on the node where `cmd/nats-client` aggregates win/loss log
  files.
- Standalone cache and ledger commands remain available for cron, timers, and
  manual operation.

## M20 - Middleman Bidder Runtime `[+]`

Wire approved bidders into fallback runtime after local campaign matching
returns no bid.

Scope:

- Build the route/bidder cache from `adv_bidder` plus `mid_route_*`.
- Require active route group, active route bidder, active credential-ready
  bidder, valid synthetic chain, route match, and synthetic item ACL/channel
  eligibility before fanout.
- Fan out downstream OpenRTB requests within the minimum of request `tmax`,
  route timeout, and DSP config timeout.
- Discard late, invalid, inactive, and non-USD responses.
- Aggregate surviving responses by price, apply route/bidder margin, and return
  the best upstream bid only after local no-bid fallback.

Acceptance:

- Approved bidders are considered only through configured routes and existing
  ACL/channel eligibility.
- Local campaign bids still win before any fallback fanout.
- No callback proxying, win/loss reconciliation, or middleman reporting changes
  are introduced until later milestones.

## M21 - Middleman Callback Proxy And Price Reconciliation `[+]`

Keep Aofei in the callback path for middleman bids and establish the
charge/pay accounting facts needed before reporting.

Scope:

- Store short-lived selected-bid callback context in Redis for middleman winners.
- Replace upstream-facing middleman `nurl`, `burl`, and `lurl` with signed
  Aofei `/mid/*` proxy URLs.
- Treat `burl` as the preferred billable event and use win notification as the
  billable fallback only when no downstream `burl` exists.
- Reconcile upstream charge price and downstream pay price before forwarding
  callbacks downstream.
- Add cooperative click notify URLs in forwarded request `ext` without
  rewriting downstream ad markup.

Acceptance:

- Middleman callbacks publish existing winloss records tied to synthetic
  campaign/item/creative IDs.
- Billable middleman events are idempotent per selected bid.
- Downstream callbacks receive net payable `${AUCTION_PRICE}` values.
- Arbitrary `adm` rewrite and advertiser/operator reporting remain later work.

## M22 - Middleman Reporting And Settlement Views `[+]`

Turn M21 callback facts into advertiser and operator reporting.

Scope:

- Add middleman-specific interval and daily ledger tables.
- Extend `cmd/ledger` to aggregate `WinLoss.Middleman` charge, pay, margin,
  route, bidder, synthetic demand, publisher, win/loss, billable impression,
  click, and callback health facts.
- Add advertiser Summer/Genelet reports that show pay-side middleman spend by
  hour, bidder, and slot.
- Add admin Summer/Genelet reports that show charge, pay, margin, route,
  bidder, publisher, and callback health views.

Acceptance:

- Existing local campaign ledger semantics remain unchanged.
- Advertiser middleman reports are scoped to the logged-in advertiser and do not
  expose charge or margin fields.
- Admin reports expose charge/pay/margin and route dimensions.
- Bid fanout, callback proxying, durable callback retries, and arbitrary markup
  rewriting remain unchanged.

## M23 - Middleman Route Operations UI `[+]`

Make middleman route assignment operable from Summer/Genelet without changing
bid runtime behavior or cache refresh ownership.

Scope:

- Add an admin-only Summer/Genelet `midroute` module for route-group CRUD.
- Add nested admin actions for `mid_route_bidder` membership and optional
  bidder timeout/margin overrides.
- Add nested admin actions for `mid_route_target` global, publisher, site,
  slot, and optional size assignment.
- Validate route knobs before writes: fallback/always trigger mode, bounded
  timeouts, bounded margins, active flags, target entity pairs, and nullable
  optional overrides.
- Add server-rendered Go `html/template` pages in `../pzdesign/tmpls`.

Acceptance:

- Operators can create and edit active route groups, attach approved bidders,
  and assign traffic targets without direct SQL edits.
- `cmd/unify` still reads route state only through the Redis
  `middleman:routes` cache; it does not refresh route cache data.
- `cmd/redis-cache -cache=redis|all` remains the singleton route-cache refresh
  path after route edits.
- M23 does not add spread route snapshots, durable callback retries, real
  settlement execution, arbitrary markup rewriting, or `Always` fanout runtime.

## M24 - Middleman Operations Reliability `[+]`

Make the current fallback system easier to operate and more reliable without
changing auction winner semantics.

Scope:

- Add route-only `cmd/redis-cache -cache=routes` refresh and read support.
- Add additive metadata to Redis `middleman:routes` payloads: generation time,
  entry count, source, route-table high-water timestamp, and entry checksum.
- Show route-cache freshness on the admin `midroute` topics and health views
  without running refresh from the UI.
- Add admin `midroute?action=health` HTML/JSON output for route groups with no
  active targets/bidders, inactive or unapproved route bidders, missing
  credential refs, and invalid synthetic chains.
- Add `mid_callback_retry` and singleton `cmd/mid-callback-retry` for retryable
  downstream `/mid/win`, `/mid/loss`, and `/mid/bill` forwarding failures.

Acceptance:

- `/bid` does not write MySQL retry rows or perform new slow operational work.
- Route cache publication remains owned by the singleton cache node.
- Retry queues contain only retryable post-auction downstream callback failures:
  network/request errors, HTTP 429, and HTTP 5xx.
- Retry execution forwards downstream only and does not republish win/loss or
  billable delivery records.

## M25 - Middleman Auction Expansion `[+]`

Allow explicitly gated middleman fanout to compete with local bids after M24 is
closed and reviewed.

Scope:

- Add `middleman_always_enabled`, default false.
- Include `trigger_mode` in route cache behavior for `Always` routes.
- Keep `Fallback` routes limited to local no-bid impressions.
- Let `Always` middleman bids compete with local bids on effective CPM after
  margin markup, while preserving local-wins fallback when comparison is unsafe.

Acceptance:

- `middleman_enabled` remains required.
- `middleman_always_enabled=false` ignores `Always` route fanout.
- Existing timeout, bidder limit, credential, ACL/channel, USD, floor, and
  callback-proxy controls continue to apply.
- Mixed local and middleman winners can be returned in one OpenRTB response.

## M26 - Middle-Term Review `[+]`

Track and remediate the independent deep code quality and architecture review
without reopening completed milestones.

Scope:

- Turn the independent review into an explicit status backlog covering security,
  reliability, configuration, cache compatibility, operations, observability,
  testing seams, schema hygiene, and documentation gaps.
- Prioritize callback signature/replay hardening, SSRF protection, config
  validation, callback retry recovery, and singleton operations safety.
- Record historical lineage to earlier milestones where issues belong
  conceptually, while keeping M26 as the active remediation milestone.
- Keep reviewed non-issues documented so future reviews do not re-triage the
  same false positives.

Acceptance:

- `memory-bank/status-M26.md` lists every independent-review finding with
  status, affected area, disposition, and verification expectation.
- Critical and high findings have clear remediation ordering.
- Findings tied to M21-M25, M19, M11/M18, pzdesign, or architecture gaps are
  classified without reopening those closed milestones.
- Review fixes are implemented as focused changes with tests and verification
  recorded in `status-M26.md`.

## M27 - SSP Contract And Cache Lookup Foundation `[+]`

Make direct publisher SSP request identity and publisher lookup unambiguous
before runtime serving.

Scope:

- Define the v1 `POST /pz` request/response contract for browser tags.
- Keep `site` as packed `(pub_id, site_id)` and `adUnits[].slot` as packed
  `(slot_id, size_id)`.
- Treat `adUnits[].code` as a DOM element id only in the v1 contract.
- Add an additive publisher cache lookup by `pub_id`, derived from `pubmap`,
  with reverse site/slot metadata for future ACL matching.
- Add parser and validation tests for current Pzdesign tags and historical
  Holiday samples.

Acceptance:

- No `/pz` runtime serving is wired yet.
- Direct tag tokens can be parsed and validated against cached
  publisher/site/slot data without MySQL on the request path.
- Existing `/bid/{domain}` behavior and cache reads remain unchanged.

## M28 - SSP Runtime Adapter `[+]`

Serve direct browser ad-tag requests through the existing Aofei bid engine.

Scope:

- Add `dsp.Controller.ServeSSP` and wire `POST /pz` in `../pzdesign/cmd/unify`.
- Convert valid SSP ad units into internal OpenRTB impressions.
- Reuse the existing local bid flow for candidates, caps, audiences, creative
  rendering, trackers, and audit publishing.
- Return a JSON HTML-string array in input order.
- Add SSP request, malformed, fill, no-fill, and validation expvar counters.

Acceptance:

- Multi-ad-unit SSP requests return renderable HTML in request order.
- Direct SSP impressions and clicks carry real publisher, site, slot, and size
  IDs.
- `/pz` does not write MySQL or refresh caches.

Result:

- `dsp.Controller.ServeSSP` reads the v1 browser JSON body with the existing bid
  body limit, validates direct tokens through local/Redis `pubmap:by-id`, and
  returns a JSON HTML-string array in input order.
- The adapter synthesizes OpenRTB browser metadata from request headers and
  cache-derived site/slot strings, then reuses the existing local candidate,
  cap, audience, creative rendering, tracker, and audit paths.
- `../pzdesign/cmd/unify` registers `POST /pz` before the Genelet catch-all.
- M28 keeps middleman fallback, cookies, CORS/origin policy changes, publisher
  tag UI, and reporting-semantics separation out of scope.

## M29 - Publisher Tag UI And Download `[+]`

Make direct publisher tags usable from the existing `pub` UI.

Scope:

- Fix publisher slot snippets to use an absolute Aofei endpoint.
- Update `www/js/ads.js` to derive the ad-server origin from the script URL or
  an explicit endpoint option.
- Add visible tag download/link actions on publisher slot pages.
- Align browser and API examples with the M28 request contract.
- Keep the existing `pub` role and site/slot CRUD.

Acceptance:

- Publishers can copy or download a working browser tag for each slot.
- External sites can embed the sample tag and call Aofei `/pz`.
- Existing publisher admin tests still pass.

Result:

- `pub_slot.size_id` is stored in the SQL baseline and Summer slot metadata, so
  generated direct tags use each slot's configured width/height.
- Publisher slot topics generate M28-compatible browser/API samples with
  absolute `/pz` endpoints, direct `site` and `slot` tokens, DOM-only ad-unit
  codes, and banner `mediaTypes`.
- `www/js/ads.js` defaults to the origin of the loaded script plus `/pz`,
  supports explicit endpoint overrides, preserves ad-unit order, and omits
  credentials by default.
- Publisher slot topics expose copy actions and downloadable
  `aofei-slot-<slot_id>.html` browser samples.
- `cmd/unify` handles `OPTIONS /pz` and applies permissive CORS headers only to
  `/pz`.

## M30 - SSP Measurement, Cookie, And Reporting Semantics `[+]`

Make direct SSP traffic observable and compatible with existing ledgers.

Scope:

- Identify direct SSP traffic separately from ADX `/bid` in request, response,
  and attribute audits.
- Add best-effort browser user-cookie handling with IP+UA fallback.
- Verify `/imp` and `/clk` records from SSP markup feed existing win/loss and
  ledger inputs.
- Add direct web tag, app-like API, partial-fill, all-no-fill, and invalid-token
  smoke fixtures.
- Document direct SSP measurement behavior.

Acceptance:

- Operators can distinguish ADX and direct SSP traffic in logs/audits.
- Existing ledger aggregation continues without schema change unless a concrete
  reporting gap is found.
- Browser cookie absence does not break serving.

Result:

- `/pz` request and response audits use explicit SSP envelopes with
  `source:"ssp"` and `contract:"pz-v1"` while ADX `/bid` request/response audits
  remain raw OpenRTB payloads.
- Attribute audits include additive `source` and `contract` fields for ADX and
  SSP traffic.
- `/pz` uses a valid browser-only `aofei_pz_uid` cookie as OpenRTB user identity
  when present; missing or invalid browser cookies keep current-request serving
  on the existing IP+UA fallback and set a best-effort cookie for later browser
  requests. `platform:"sdk"` requests do not read, set, rotate, or propagate the
  cookie.
- SSP markup reuses the existing signed `/imp` and `/clk` tracker path, and
  tracker `WinLoss` records aggregate through the current ledger schema.

## M31 - SSP Hardening And Product Boundary `[+]`

Finish safety, operations, and product separation after the basic path works.

Scope:

- Add origin/referrer policy controls for browser tags.
- Validate token tampering, inactive inventory, mismatched size, and unsupported
  media-type cases.
- Decide whether publisher inventory needs an explicit supply-source field.
- Keep browser HTML arrays as stable v1 while planning optional API/mobile
  response formats.
- Update production runbook, local runtime docs, memory bank status, and
  closeout verification.

Acceptance:

- Direct SSP is safe to expose outside local development.
- ADX `/bid/{domain}` and direct `/pz` are documented as separate traffic
  entrypoints.
- Remaining advanced API/mobile/native response work is carried forward
  explicitly.

Result:

- `/pz` now enforces exact cached-site-host `Origin`/`Referer` policy after
  token/cache validation and before cookies, bidding, or audit publishing.
  Browser traffic must send a matching `Origin` or `Referer`; `platform:"sdk"`
  may omit both headers, but any supplied header must still match.
- Policy rejections return `403` and increment
  `aofei_ssp_policy_rejections_total`.
- Direct SSP remains bounded by the `/pz` entrypoint plus audit
  `source:"ssp"`/`contract:"pz-v1"` metadata; no supply-source schema/cache
  field, `ads.js` change, credentialed CORS, or response-format change was
  added.
- API/mobile/native response formats and richer supply taxonomy remain future
  product work.

## M32 - SSP Mobile/API Contract And Response Formats `[+]`

Add mobile/API serving to the existing `/pz` endpoint without changing browser
defaults or account/schema/cache boundaries.

Scope:

- Accept explicit `responseFormat` values: omitted/`"html"`, `"json"`, and
  `"openrtb"`.
- For `platform:"sdk"`, accept OpenRTB-like body `app`, `device`, and `user`
  fields.
- Synthesize app traffic from the validated direct SSP cache while keeping
  `site` and `adUnits[].slot` tokens authoritative.
- Keep `ads.js`, schema, cache shape, CORS credentials, and account roles out of
  scope.

Acceptance:

- Existing browser `/pz` HTML-array behavior remains compatible.
- SDK requests synthesize `BidRequest.App`, do not use cookies, and honor body
  device/user identity.
- App identity mismatches against the validated site token return `400`.
- JSON responses return ordered fill/no-fill objects with native JSON where
  applicable.
- OpenRTB responses return a valid `BidResponse`, including `200` with empty
  `seatbid` on all-no-fill.

## M33 - SSP Middleman Fallback `[+]`

Let direct SSP local no-fill requests use existing middleman fallback and gated
`Always` behavior while preserving the M32 response formats.

Acceptance:

- Valid `/pz` requests can fan out to middleman only after token, media, cache,
  and browser policy validation succeeds.
- Local no-fill impressions use `Fallback` candidates; local filled impressions
  use gated `Always` candidates.
- Middleman winners preserve ordered `html`, `json`, and `openrtb` SSP response
  formats and keep SSP audits wrapped around the original `/pz` request and
  final SSP response.

## M34 - Richer Supply Taxonomy ADR `[+]`

Keep runtime on the `pub` role and write an ADR for taxonomy fields,
cache/audit impact, admin UI changes, and migration path before schema work.

Scope:

- Keep `pub`, `pub_site`, and `pub_slot` as the publisher and inventory
  ownership boundary.
- Recommend additive nullable/defaulted future fields on existing publisher
  tables.
- Cover site/app identity, integration mode, slot/media taxonomy,
  quality/source taxonomy, cache impact, audit impact, admin UI impact, and
  migration path.
- Do not change schema, cache payloads, runtime behavior, audit payloads,
  ledger tables, or Summer/Genelet admin code.

Acceptance:

- ADR 0001 records the future taxonomy direction.
- `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` remains the current
  runtime direct SSP boundary until a later schema/cache milestone.
- M35 remains the separate SSP account/schema ADR.

Result:

- [ADR 0001](../docs/adr/0001-richer-supply-taxonomy.md) keeps `pub`,
  `pub_site`, and `pub_slot` as the publisher and inventory ownership boundary.
- Future taxonomy is additive on existing publisher tables and covers site/app
  identity, integration mode, slot/media intent, and quality/source metadata.
- M34 changes docs and memory only; schema, cache payloads, runtime behavior,
  audit payloads, ledgers, and Summer/Genelet admin code remain unchanged.

## M35 - SSP Account/Schema ADR `[+]`

Decide whether a separate SSP account boundary is still needed after M32-M34
evidence. Do not implement a separate account in this milestone.

Scope:

- Decide the account/schema boundary for current direct SSP `/pz` traffic.
- Keep this milestone ADR-only with no schema, runtime, cache payload, audit
  payload, ledger, or Summer/Genelet admin UI change.
- Record concrete future triggers for reopening the separate SSP account
  question.

Acceptance:

- ADR 0002 records the decision to keep `pub`, `pub_site`, and `pub_slot` as the
  publisher account and inventory ownership boundary.
- No separate `ssp` account role or separate SSP-owned inventory schema is added
  for the current `/pz` path.
- Future SSP schema work follows the additive M34 taxonomy direction unless a
  later milestone reopens the account-boundary decision.

Result:

- [ADR 0002](../docs/adr/0002-ssp-account-schema-boundary.md) decides not to add
  a separate `ssp` account role or separate SSP-owned inventory schema for the
  current direct SSP path.
- `pub`, `pub_site`, and `pub_slot` remain the publisher account and inventory
  ownership boundary.
- Future reconsideration requires concrete legal, settlement, intermediary,
  permission, compliance, or partner-credential requirements.

## M36 - Runtime Safety And Test/Observability Hardening `[+]`

Fix the meaningful confirmed risks from the post-M35 whole-repo review without
changing the active schema shape, cache payload shape, `/pz` response shape, or
middleman product semantics unless a task explicitly records the decision.

Scope:

- Move `cmd/spread` service behavior toward the M19 job pattern by adding
  signal-aware context shutdown, non-blocking or buffered callback reporting,
  graceful NATS drain/close behavior, and focused tests around message handling
  and shutdown.
- Harden middleman bidder fanout so a missing controller HTTP client cannot
  fall back to unwrapped `http.DefaultClient`, and add request-time URL
  validation or an equivalent safe transport invariant for bidder endpoints.
- Add cap-refresh and middleman/callback retry observability where the current
  runtime can silently degrade: retry/conflict counters, basic latency/backlog
  metrics, and operator documentation for alerting on audit drops, callback
  retry backlog, and cap contention.
- Define an integration-test taxonomy with build tags or an equivalent explicit
  command split so package tests, Docker-backed tests, Redis/MySQL-dependent
  tests, race tests, and smoke tests are honest and documented.
- Decide and implement the local/spread static-cache staleness policy: either
  expose/alert on age only or enforce a request-time max-age fail-closed guard,
  then document the operational consequences.
- Triage low-risk confirmed cleanup items opportunistically only when they are
  adjacent to the above work: dead `HhLock`, defensive `DSP.impID` bounds,
  native macro invariants, and ADR cross-links.

Acceptance:

- `cmd/spread` can exit cleanly on normal service signals without requiring
  `SIGKILL`, and tests cover the shutdown path or extracted spread job logic.
- Middleman bidder fanout always uses the safe HTTP path or rejects unsafe
  endpoint URLs before outbound network I/O, including nil-client construction
  cases.
- Operators can see and alert on cap contention, audit drops, callback retry
  backlog/staleness, and relevant middleman callback/fanout failures through
  documented metrics or commands.
- The README, AGENTS guide, and memory bank distinguish local package tests,
  integration tests, race tests, staticcheck, Docker smoke checks, and schema
  checks with concrete commands.
- Local/spread cache freshness has a documented runtime policy and automated
  coverage for stale and fresh states.
- Any schema-affecting decision, especially around retired `cron_halfhour`
  triggers/table data, is either explicitly deferred or handled with the normal
  schema baseline workflow.

Result:

- `cmd/spread` now runs under a signal-aware context, logs callback results
  without unbuffered reporting channels, and drains NATS on shutdown.
- Middleman bidder fanout validates every endpoint URL before request creation
  and uses the safe callback HTTP client when no custom client is supplied.
- Cap refresh, audit publishing, local cache freshness, and middleman callback
  retry backlog/staleness now expose operational signals through expvars or
  command output.
- Local/spread cache staleness is alert-only:
  `local_cache_max_age_seconds` marks scrape-time `aofei_local_cache_stale`,
  `aofei_local_cache_loaded_at_unix` records the loaded snapshot timestamp, and
  old snapshots do not fail closed by age alone.
- README, AGENTS, and the memory bank now distinguish package, runtime
  hardening, Docker smoke, admin integration, and schema verification commands.
- `cron_halfhour` cleanup remains deferred as a schema-baseline decision.

## M37 - Operational Follow-Up Hardening `[+]`

Resolve the confirmed follow-up risks from the `review.md` disposition without
changing schema shape, cache payload shape, or bid/SSP product semantics.

Scope:

- Make `cmd/nats-client` signal-aware and testable, with graceful NATS drain,
  queued-message flush, and file-handle close on shutdown.
- Tighten `cmd/nats-client` generated log directory and file permissions for
  ledger input logs.
- Add stable JSON output to `cmd/mid-callback-retry` for backlog alerting while
  preserving the existing text output.
- Record deferred review findings separately from this operational hardening
  milestone.

Acceptance:

- Context cancellation of the extracted NATS client run path drains the NATS
  connection and writes queued messages before exit.
- Generated log directories have no world permissions and no group/world write
  bits; generated log files have no world permissions and no group/world write
  bits.
- `cmd/mid-callback-retry -json` emits `due`, `stale_processing`, `selected`,
  `succeeded`, `retrying`, and `abandoned`; default text output remains
  unchanged for current operators.
- Operator docs and memory bank describe shutdown, permissions, and JSON
  alerting behavior.

Result:

- `cmd/nats-client` now uses `cmdboot.SignalContext`, drains NATS on
  cancellation, flushes queued log messages, closes files, and is covered by a
  fake-connection shutdown test.
- `cmd/nats-client` creates or tightens log directories to `0750` and log files
  to `0640`, with tests asserting private generated modes.
- `cmd/mid-callback-retry` keeps its existing summary line by default and adds
  `-json` for stable automation.
- Pubmap envelope compatibility, source-specific SSP/middleman fanout metrics,
  RAdv SQL-null cleanup, HMAC allocation benchmarking, auction function
  cleanup, and local-cache pointer-swap work remain deferred.

## M38 - Prebid/OpenRTB Pattern Adoption Review `[+]`

Create a documentation-only review of Prebid Server OpenRTB patterns that may
be worth adopting in `aofei`.

Scope:

- Use Prebid Server as an external design reference, not a dependency.
- Summarize relevant OpenRTB flow: request parsing, bidder splitting, adapter
  calls, bid normalization, validation, targeting/cache/debug response
  assembly, and observability.
- Classify adoption candidates for performance, matching, validation,
  security/privacy, and observability as `Adopt soon`,
  `Research/benchmark first`, `Only for middleman fanout`, or
  `Not applicable to aofei`.
- Name later implementation milestones without changing runtime code, schema,
  cache payloads, config, public APIs, or operator workflow.

Acceptance:

- [docs/prebid-openrtb-adoption.md](../docs/prebid-openrtb-adoption.md)
  records the review and deferred implementation candidates.
- Performance and dependency recommendations are measurement-gated.
- `memory-bank/status-M38.md` tracks the documentation tasks and verification.
- Required verification is `./scripts/aofei-doc-check.sh` and
  `git diff --check`.

## M39 - Tracking And Runtime Integrity `[+]`

Close the confirmed tracking-signature, frequency-cap, and adjacent request
correctness findings without changing schema or response formats.

Scope:

- Require valid configured-TTL signatures for every `/imp` and `/clk` event.
- Bound Redis cap-state lifetime, saturate packed cap counters, and keep valid
  signed measurement events recordable when no cap user identity exists.
- Remove adjacent audit initialization, weighted-selection, and SSP cookie
  correctness hazards.

Acceptance:

- Unsigned or expired impression/click events are rejected whether or not a
  `cap` value is present.
- Cap counters cannot wrap and every cap-state write has a positive TTL without
  shortening a longer existing TTL.
- Empty-user trackers still publish but do not mutate cap state; audit startup,
  weighted selection, and SSP cookie resolution have focused regression tests.

Result:

- Impression and click signatures are unconditional and use the configured
  replay TTL on tracker and redirect paths.
- Packed cap counters saturate, Redis cap hashes receive bounded idle expiry,
  and empty-user events remain measurable without attempting cap mutation.
- Audit initialization is serialized, weighted selection has a deterministic
  final-positive fallback, and SSP cookie creation happens once per request.
- Reopened after the M39-M44 review found callback-time TTL reuse, request-bound
  Redis transaction contexts, and an unconditional bulk cap expiry could break
  validity and TTL guarantees.
- Follow-up remediation now uses exact signature deadlines, detached two-second
  tracking Redis contexts with confirmed transaction cleanup, and one atomic
  bulk cap/conditional-expiry script.

## M40 - Redis Cache Availability And Route Efficiency `[+]`

Remove the serving gap during full Redis cache refresh and avoid fetching and
decoding the complete middleman route cache on every eligible request.

Scope:

- Build static cache generations under shadow keys and atomically swap all
  related live families in one Redis transaction.
- Add a short-lived, single-flight controller snapshot for middleman routes.
- Raise the cache attribute-log scanner limit to the ledger limit.

Acceptance:

- Failed cache builds leave the previous live generation untouched, successful
  swaps cannot expose a partially deleted or mixed generation, and stale slot
  keys are removed atomically.
- Middleman route reads are memoized for a configurable interval and refresh
  errors do not fan out using expired routes.
- Attribute log lines up to 8 MiB are accepted.

Result:

- Full Redis refreshes build shadow families and atomically install one
  complete generation; failed builds preserve live data and successful swaps
  remove empty and obsolete families in the same transaction.
- Workers memoize decoded middleman routes for a configurable five-second
  default with context-aware single-flight refresh and short error caching.
- Attribute log scanning now accepts lines up to 8 MiB, and miniredis plus
  Docker smoke/serving-loop checks cover the replacement path.
- Reopened to serialize every mutating cache mode on one resource lock and let
  route-cache waiters retry after a canceled refresh leader.
- Review remediation completed with one writer lock, cancellation-aware waiter
  retry, and exact scanner/cap transaction coverage in CI.
- Reopened after the follow-up review found that the initiating HTTP request
  still owned and could cancel the shared route refresh.
- Follow-up remediation now runs the shared load under its own
  `middleman_timeout_ms` context while each caller waits independently.

## M41 - Measurement Replay Idempotency `[+]`

Suppress duplicate signed impression and click callbacks within the tracking
signature lifetime while preserving request availability when Redis fails.

Scope:

- Deduplicate `/imp` and `/clk` independently by signed auction event identity.
- Apply replay suppression before cap mutation and ledger publication.
- Expose suppression, fail-open, and unkeyed-event metrics.

Acceptance:

- A repeated signed event publishes and mutates cap state once within the TTL.
- Impression and click identities remain independent, normal HTTP/redirect
  responses are preserved, and Redis failures remain fail-open.

Result:

- `/imp` and `/clk` use independent signature-TTL replay keys hashed from the
  signed auction event identity before cap or ledger side effects.
- Duplicate events preserve normal callback/redirect responses while skipping
  cap refresh and publication; Redis and identity failures remain fail-open.
- Suppression, fail-open, Redis-error, and unkeyed-event expvars document the
  operational effect.
- Reopened to finalize replay identity only after successful side effects and
  make cap mutation idempotent when publication is retried.
- Review remediation completed with owned processing claims, post-publication
  completion markers, and transactional per-event cap markers.
- Reopened after the follow-up review found implicit claim outcomes and cap
  errors could still reject valid events or fall back to non-idempotent writes.
- Follow-up remediation now retains keyed fail-open identity, skips unkeyed cap
  mutation, publishes through claim/cap Redis errors, and finalizes owned claims
  only after successful publication.

## M42 - Unified HTTP Graceful Shutdown `[+]`

Add signal-aware graceful shutdown to the sibling `../pzdesign/cmd/unify`
service and drain Aofei audits after in-flight HTTP requests finish.

Scope:

- Use a standard-library signal context in `pzdesign` and extract a testable
  HTTP server lifecycle.
- Allow 15 seconds for graceful shutdown, then force close and report failure.
- Update the Aofei production runbook for the new service behavior.

Acceptance:

- SIGINT/SIGTERM stop new work, wait for in-flight handlers, and close the
  controller only after HTTP shutdown.
- Timeout and normal shutdown paths have focused tests in `pzdesign`.

Result:

- `cmd/unify` now owns a standard-library SIGINT/SIGTERM context and a testable
  listener/server lifecycle.
- Normal shutdown drains in-flight HTTP for up to 15 seconds before controller
  close; timeout forces close and returns the joined shutdown error.
- Controller/logger defers execute inside the run function even when startup or
  serving returns an error.

## M43 - Repository CI Baseline `[+]`

Turn the documented verification taxonomy into GitHub Actions gates for both
the Aofei and pzdesign repositories.

Scope:

- Add test, vet, staticcheck, race/documentation, and template gates appropriate
  to each repository.
- Check out the public sibling Aofei repository beside pzdesign so its local Go
  replace directive resolves in CI.
- Pin the Go and staticcheck versions and keep workflow permissions read-only.

Acceptance:

- Pushes and pull requests run clean, reproducible checks in both repositories.
- Pzdesign keeps its documented legacy staticcheck style exclusions rather
  than starting with a permanently failing workflow.

Result:

- Aofei push/PR CI runs package tests, vet, pinned staticcheck, scoped race,
  documentation, and diff-hygiene gates with read-only permissions.
- Pzdesign checks out public Aofei as a sibling, then runs package tests, vet,
  pinned staticcheck with established style exclusions, both template parsers,
  and diff hygiene.
- Both workflows pin Go 1.23.5, cancel superseded runs, and pass actionlint plus
  the exact local command set.
- Reopened after the M39-M44 review found that `git diff --check` on a clean
  checkout did not inspect any committed event range.
- Follow-up remediation fetches full primary history and checks PR
  merge-base-to-head or push before-to-after committed ranges, with an
  empty-tree initial-history fallback.

## M44 - Bid-Path Logging And Benchmark Cleanup `[+]`

Reduce avoidable bid-path logging/allocation work and measurement-gate any
future weighted-selection optimization.

Scope:

- Replace sugared request-path logs with structured logging and remove numbered
  progress messages and expected no-bid noise.
- Simplify local effective-CPM selection without changing auction results.
- Add parallel weighted-selection and representative bid-path benchmarks.

Acceptance:

- Bid responses and auction choices remain unchanged with materially quieter,
  structured request logging.
- Benchmarks record the selection and request-path baseline; `math/rand` is not
  replaced without measured evidence.

Result:

- Bid handling, auction fallback, callback setup, and adjacent tracker failures
  use structured `zap` fields; routine progress, success, NATS-unavailable, and
  expected no-bid messages no longer produce process-log noise.
- Local effective CPM validation retains the existing error metric and returns
  the already-materialized bid price directly.
- Parallel weighted-selection and successful two-impression local HTTP
  benchmarks establish an allocation/time baseline without changing the RNG.
- Deep review corrected the weighted-distribution test's bucket upper bounds,
  and all Aofei/pzdesign closeout gates pass under the documented toolchains.

## Post-M25 Middleman Backlog

These items remain intentionally outside M25:

- Add spread/local snapshots for middleman bidder routes if `cmd/unify` should
  support middleman fallback without Redis static-cache reads.
- Add real invoicing/payment execution from `daily_mid` settlement facts.
- Keep arbitrary downstream markup impression/click rewriting closed unless a
  future reporting requirement makes cooperative click notify insufficient.
