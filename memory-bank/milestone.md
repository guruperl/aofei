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

## M29 - Publisher Tag UI And Download `[ ]`

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

## M30 - SSP Measurement, Cookie, And Reporting Semantics `[ ]`

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

## M31 - SSP Hardening And Product Boundary `[ ]`

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

## Post-M25 Middleman Backlog

These items remain intentionally outside M25:

- Add spread/local snapshots for middleman bidder routes if `cmd/unify` should
  support middleman fallback without Redis static-cache reads.
- Add real invoicing/payment execution from `daily_mid` settlement facts.
- Keep arbitrary downstream markup impression/click rewriting closed unless a
  future reporting requirement makes cooperative click notify insufficient.
