# Milestones

This file stays at milestone level. Detailed task rows live in the matching
`memory-bank/status-<lane><number>.md` files. Do not recreate an aggregate
`memory-bank/status.md` file.

## Status ID Pattern

The original single-digit M-lane history is normalized to `M00` through `M09`;
`M10` and later already satisfy the two-digit minimum. New product work is
organized into zero-padded domain lanes:

```text
D01, D02, ...  DSP demand, campaigns, auctions, creatives, and middleman bidding
P01, P02, ...  Publisher inventory, direct SSP, floors, and supply transparency
R01, R02, ...  Measurement, attribution, reporting, and experimentation
I01, I02, ...  OpenRTB integrations, management APIs, and mobile SDKs
S01, S02, ...  Privacy, identity, authorization, and traffic quality
A01, A02, ...  Accounting, billing, settlement, funding, and payouts
O01, O02, ...  Operations, observability, capacity, availability, and recovery
```

The two-digit minimum keeps each lane sorting naturally after it reaches `10`.
Always use the zero-padded form for new lane files, and never reuse an ID after
its status file exists. Cancelled work keeps its file and is marked cancelled.

## W8M Marketplace Roadmap

The W8M marketplace roadmap prioritizes commercial correctness, privacy,
accounting safety, and production controls before expanding automation or
scale. D01 through A02 in the original sequence below, D04, and D05 are
complete. A 2026-08-23 follow-up review opened D05 for post-D04 auction
compatibility and hot-path remediation; D05 is now complete. S06 repository
hardening, managed Cloudflare widget, constrained Free-plan edge rule,
production deployment, and live proof are also complete. The remaining
deep-review horizon resumes with P03, S05, O03, R03, and A03. I02 remains
demand-gated and must not start until P03/S05
are complete and a named Android or iOS integration supplies supported
OS/version and lifecycle requirements. Matching lane status files are the
authoritative completion record; completed M-lane files retain earlier runtime
history.

Delivery sequence:

1. Foundation prerequisites: D01, S01, S04, and O01.
2. Foundation completion: A01 after D01, then P01 after D01/S01/A01/O01.
3. Core expansion: D02, I01, R01, and O02, then staged D03 after I01.
4. Product expansion: R02, P02, S02, and S03, then I03/A02 after S02.
5. Follow-up review remediation: D04 callback/runtime history, D05
   auction/cap/hot-path fixes, and S06 repository plus production activation
   are complete; the remaining sequence starts with P03 direct-SSP
   authenticity, then S05 trust boundaries, O03
   job/cache/filesystem reliability, R03 experiment/report integrity, then A03
   exact monetary sources.
6. Demand-gated mobile delivery: I02 after P03/S05 and a named mobile
   integration requires supported native SDKs.

The strict serial order is:

```text
D01 -> S01 -> S04 -> O01 -> A01 -> P01
-> D02 -> I01 -> R01 -> O02 -> D03
-> R02 -> P02 -> S02 -> I03 -> S03 -> A02
-> D04 -> D05 -> S06 -> P03 -> S05 -> O03 -> R03 -> A03
-> I02 (only after P03/S05 and a named mobile integration)
```

Controlled direct-SSP and middleman staging may begin with existing runtime
features, but revenue-bearing activation must satisfy the prerequisite lane
acceptance criteria recorded in the corresponding status files.

| ID | State | Status file | Summary |
|---|---|---|---|
| D01 | Completed | [status-D01.md](status-D01.md) | Campaign delivery guardrails. |
| D02 | Completed | [status-D02.md](status-D02.md) | Auction, pricing, and creative correctness. |
| D03 | Completed; activation-gated | [status-D03.md](status-D03.md) | External DSP / AdX middleman activation. |
| D04 | Completed | [status-D04.md](status-D04.md) | Delivery, tracking, and auction integrity. |
| D05 | Completed | [status-D05.md](status-D05.md) | Post-D04 auction compatibility and hot-path remediation. |
| P01 | Completed; publisher activation-gated | [status-P01.md](status-P01.md) | Direct SSP commercial readiness and activation. |
| P02 | Completed | [status-P02.md](status-P02.md) | Supply metadata and seller transparency. |
| P03 | In progress | [status-P03.md](status-P03.md) | Direct SSP request authenticity; threat contract, versioned locator codec/runtime reader, and SDK/server request authentication complete. |
| R01 | Completed | [status-R01.md](status-R01.md) | Conversion, action, and attribution measurement. |
| R02 | Completed | [status-R02.md](status-R02.md) | Marketplace analytics and experimentation. |
| R03 | Planned | [status-R03.md](status-R03.md) | Experiment and reporting integrity. |
| I01 | Completed | [status-I01.md](status-I01.md) | OpenRTB partner interoperability. |
| I02 | Planned; demand-gated | [status-I02.md](status-I02.md) | Android and iOS publisher SDKs. |
| I03 | Completed; disabled by default | [status-I03.md](status-I03.md) | External campaign management API. |
| S01 | Completed | [status-S01.md](status-S01.md) | Privacy, consent, and data disclosure. |
| S02 | Completed; disabled by default | [status-S02.md](status-S02.md) | Identity, two-factor authentication, and RBAC. |
| S03 | Completed; disabled by default | [status-S03.md](status-S03.md) | Traffic quality and anti-fraud. |
| S04 | Completed | [status-S04.md](status-S04.md) | Template escaping and XSS audit. |
| S05 | Planned | [status-S05.md](status-S05.md) | Runtime trust-boundary hardening. |
| S06 | Completed; active on W8M | [status-S06.md](status-S06.md) | Public account abuse protection. |
| A01 | Completed | [status-A01.md](status-A01.md) | Billing and manual settlement safety. |
| A02 | Completed; disabled by default | [status-A02.md](status-A02.md) | Hosted funding and publisher payout integration. |
| A03 | Planned | [status-A03.md](status-A03.md) | Exact monetary source migration. |
| O01 | Completed | [status-O01.md](status-O01.md) | Production traffic controls and observability. |
| O02 | Completed; production claims evidence-gated | [status-O02.md](status-O02.md) | Single-region availability, recovery, and SLO. |
| O03 | Planned | [status-O03.md](status-O03.md) | Job, cache, and filesystem reliability. |

Historical M-lane status index:

| ID | Status file | Summary |
|---|---|---|
| M00 | [status-M00.md](status-M00.md) | Agentic harness bootstrap. |
| M01 | [status-M01.md](status-M01.md) | Local Docker runtime stabilization. |
| M02 | [status-M02.md](status-M02.md) | Schema baseline stewardship. |
| M03 | [status-M03.md](status-M03.md) | Redis and NATS cache pipeline reliability. |
| M04 | [status-M04.md](status-M04.md) | Bid path smoke coverage. |
| M05 | [status-M05.md](status-M05.md) | Summer/Genelet admin compatibility. |
| M06 | [status-M06.md](status-M06.md) | Ledger, logs, and operational commands. |
| M07 | [status-M07.md](status-M07.md) | MaxMind and geo runtime. |
| M08 | [status-M08.md](status-M08.md) | Full repository test hygiene. |
| M09 | [status-M09.md](status-M09.md) | Production deployment runbook. |
| M10 | [status-M10.md](status-M10.md) | Documentation and agent stewardship. |
| M11 | [status-M11.md](status-M11.md) | Genelet and Summer UI stewardship. |
| M12 | [status-M12.md](status-M12.md) | OpenRTB and audience matching review. |
| M13 | [status-M13.md](status-M13.md) | OpenRTB and DSP refactor backlog. |
| M14 | [status-M14.md](status-M14.md) | Redis and spread cache reliability. |
| M15 | [status-M15.md](status-M15.md) | DSP serving hardening. |
| M16 | [status-M16.md](status-M16.md) | Middleman AdX advertiser-owned bidder schema. |
| M17 | [status-M17.md](status-M17.md) | Advertiser bidder portal. |
| M18 | [status-M18.md](status-M18.md) | Summer template modernization. |
| M19 | [status-M19.md](status-M19.md) | Maintenance job package refactor. |
| M20 | [status-M20.md](status-M20.md) | Middleman bidder runtime. |
| M21 | [status-M21.md](status-M21.md) | Middleman callback proxy and price reconciliation. |
| M22 | [status-M22.md](status-M22.md) | Middleman reporting and settlement views. |
| M23 | [status-M23.md](status-M23.md) | Middleman route operations UI. |
| M24 | [status-M24.md](status-M24.md) | Middleman operations reliability. |
| M25 | [status-M25.md](status-M25.md) | Middleman auction expansion. |
| M26 | [status-M26.md](status-M26.md) | Middle-term review. |
| M27 | [status-M27.md](status-M27.md) | SSP contract and cache lookup foundation. |
| M28 | [status-M28.md](status-M28.md) | SSP runtime adapter. |
| M29 | [status-M29.md](status-M29.md) | Publisher tag UI and download. |
| M30 | [status-M30.md](status-M30.md) | SSP measurement, cookie, and reporting semantics. |
| M31 | [status-M31.md](status-M31.md) | SSP hardening and product boundary. |
| M32 | [status-M32.md](status-M32.md) | SSP mobile/API contract and response formats. |
| M33 | [status-M33.md](status-M33.md) | SSP middleman fallback. |
| M34 | [status-M34.md](status-M34.md) | Richer supply taxonomy ADR. |
| M35 | [status-M35.md](status-M35.md) | SSP account/schema ADR. |
| M36 | [status-M36.md](status-M36.md) | Runtime safety and test/observability hardening. |
| M37 | [status-M37.md](status-M37.md) | Operational follow-up hardening. |
| M38 | [status-M38.md](status-M38.md) | Prebid/OpenRTB pattern adoption review. |
| M39 | [status-M39.md](status-M39.md) | Tracking and runtime integrity. |
| M40 | [status-M40.md](status-M40.md) | Redis cache availability and route efficiency. |
| M41 | [status-M41.md](status-M41.md) | Measurement replay idempotency. |
| M42 | [status-M42.md](status-M42.md) | Unified HTTP graceful shutdown. |
| M43 | [status-M43.md](status-M43.md) | Repository CI baseline. |
| M44 | [status-M44.md](status-M44.md) | Bid-path logging and benchmark cleanup. |
| M45 | [status-M45.md](status-M45.md) | Open-source security and privacy hygiene. |

Status markers:

| Symbol | Meaning |
|---|---|
| `[ ]` | Pending |
| `[+]` | Completed |
| `[~]` | In progress |
| `[!]` | Blocked |
| `[X]` | Cancelled |

## Review Finding Severity

P1 and P2 are engineering review priorities, not product-domain lane priority,
milestone execution order, or status markers. When a narrower linked review
policy does not define them:

- **P1** is a severe defect in milestone acceptance, correctness,
  security/privacy, data integrity, or a public compatibility contract.
- **P2** is a material defect in supported behavior, reliability,
  compatibility, operations, or required verification/documentation that does
  not rise to P1.

Classify from impact, likelihood, and affected scope rather than implementation
or fix size. P1, P2, and any higher-severity finding block milestone closure and
cannot be carried into a later milestone. A lower-severity finding may be
carried only with a named pending owner and explicit rationale.

## Milestone Review Procedure

After implementation and automated verification, review the complete milestone
for correctness, failure semantics, security/privacy, compatibility,
operations, tests, and documentation. The initial review is iteration 1. Record
each iteration number and its findings in the active status notes before fixing
anything. After every P1/P2-or-higher fix, rerun affected verification and
review the whole milestone again.

The gate passes only when an iteration finds no P1, P2, or higher-severity
issue. Run no more than 10 iterations, without resetting for a new session or
reviewer. If iteration 10 still has a blocking finding, leave the milestone
incomplete, mark it through the status file's blocked mechanism, report the
limit, and do not begin downstream reconciliation without explicit user
direction. [GOAL.md](../GOAL.md) owns the full multi-milestone execution
protocol.

## Closeout Checklist

Use this order when closing a milestone:

1. Run the milestone's required verification commands.
2. Update code-adjacent docs and the memory-bank files that changed behavior,
   contracts, tools, or operator workflow.
3. Pass the bounded milestone review gate. Resolve every P1/P2-or-higher
   finding; carry a lower-severity finding only with a named pending owner and
   explicit rationale in the matching lane status file.
4. Reconcile every affected pending status file and recompute the remaining
   dependency order according to [GOAL.md](../GOAL.md).
5. Check whether `evolution/` needs a new prompt/result version.
6. Mark the matching lane status tasks, review findings, and milestone state
   complete only after verification and a clean review-fix iteration pass.
7. Commit the milestone only when the execution request includes commit
   handling.

## M00 - Agentic Harness Bootstrap `[+]`

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

## M01 - Local Docker Runtime Stabilization `[+]`

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

## M02 - Schema Baseline Stewardship `[+]`

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

## M03 - Redis And NATS Cache Pipeline Reliability `[+]`

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

## M04 - Bid Path Smoke Coverage `[+]`

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

## M05 - Summer/Genelet Admin Compatibility `[+]`

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

## M06 - Ledger, Logs, And Operational Commands `[+]`

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
  outputs, and M06/M07 blockers for `cmd/ledger`, `cmd/nats-client`,
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

## M07 - MaxMind And Geo Runtime `[+]`

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

## M08 - Full Repository Test Hygiene `[+]`

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
  documented non-gating follow-ups for M08.

## M09 - Production Deployment Runbook `[+]`

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
- Keep milestone docs summary-level and detailed task lists in matching lane
  status files.

Acceptance:

- A future agent can resume from the memory bank without rediscovering local
  runtime, schema, or documentation decisions.
- Milestones, status, and docs remain consistent after each substantial change.

## M11 - Genelet And Summer UI Stewardship `[+]`

Review and harden the Genelet/Summer admin framework surface without changing
DSP bid behavior, database schema, cache payload contracts, or production config
shape.

Scope:

- Move the remaining M05 Genelet/Summer hardening findings into active work.
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

## M45 - Open-Source Security And Privacy Hygiene `[+]`

Remove exposed credentials and production-derived data from both public
repositories, make email disablement fail before account mutation, and prevent
future repository leaks.

Scope:

- Disable compromised SMTP configuration and fail closed for public account
  mail workflows while preserving login and authenticated portals.
- Separate schema/reference catalogs from deterministic synthetic fixtures and
  remove tracked backups, customer sources, captured traffic, and personal
  deployment identifiers.
- Rewrite every branch and tag in Aofei and pzdesign, then add full-history
  Gitleaks and tracked-data gates.

Acceptance:

- No known credential, customer identifier, private path, or production-derived
  payload remains in tracked files or reachable repository history.
- Schema/load/cache/bid/admin checks pass against disposable Docker resources
  without resetting the database used by w8m.com.
- Existing deployed accounts still log in, while disabled registration and
  password retrieval fail before writes.

Result:

- Summer account mail actions now require a complete SMTP block before model
  mutation; the deployed block was removed and the service restarted.
- The database baseline contains schema/reference catalogs only; one synthetic
  fixture set owns documented local advertiser, publisher, and admin logins.
- Historical backups and captured OpenRTB data were removed, customer DOCX
  sources moved outside Git, and both repositories gained privacy/secret gates.
- Every branch and tag was rewritten and independently verified before the
  sanitized refs were published.

## D01 - Campaign Delivery Guardrails `[+]`

Make budgets and schedules authoritative bid eligibility, not merely stored UI
or ledger data. Exhausted total/daily budgets and out-of-window campaigns or
items must not bid, including under concurrent traffic and stale-cache risks.
Detailed tasks and verification are in [status-D01.md](status-D01.md).

## D02 - Auction, Pricing, And Creative Correctness `[+]`

Align public pricing claims with runtime behavior, select the highest qualified
campaign effective CPM, reserve weights for creative rotation inside the
winner, complete native authoring, and validate media/size/secure markup.
Detailed tasks and verification are in [status-D02.md](status-D02.md).

## D03 - External DSP / AdX Middleman Activation `[+]`

The config-gated middleman path now has read-only database/Redis/credential
preflight, stricter topology and header safety, and a staged production runbook.
Fallback and optional `Always` require distinct evidence gates; checked-in and
deployed traffic remains off until a named partner is approved. Detailed tasks
and verification are in [status-D03.md](status-D03.md).

## D04 - Delivery, Tracking, And Auction Integrity `[+]`

Correct confirmed ACL SQL, callback publication/idempotency, CPM type, cap-time,
and bounded matching defects while preserving D01/A01 reconciliation and the
documented deterministic auction policy. Detailed tasks and verification are in
[status-D04.md](status-D04.md).

## D05 - Post-D04 Auction Compatibility And Hot-Path Remediation `[+]`

Correct legacy cap-time interpretation, isolate invalid capped demand without
weakening cache publication, restore compatible optional OpenRTB dimension
handling, and remove avoidable audience logging and macro-plan rebuilds from
the bid path. Preserve D04 callback, redirect, cap-lifecycle, and wire-format
contracts. Detailed tasks and verification are in
[status-D05.md](status-D05.md).

## P01 - Direct SSP Commercial Readiness And Activation `[+]`

Make configured publisher slot floors authoritative and prove the existing
browser and SDK-style `/pz` contracts from approved inventory through cache,
auction, tracking, reporting, rollout, and rollback. Detailed tasks are in
[status-P01.md](status-P01.md).

## P02 - Supply Metadata And Seller Transparency `[+]`

Implement the additive supply taxonomy selected by ADR 0001 and introduce
seller/supply-chain metadata without changing the existing publisher account
boundary. Detailed tasks are in [status-P02.md](status-P02.md).

## P03 - Direct SSP Request Authenticity `[~]`

Version direct-SSP inventory token integrity and add a distinct authenticated,
fresh publisher/App request boundary for SDK-style traffic. Public browser
locators remain replayable identifiers rather than publisher authentication.
The accepted threat and compatibility contract separates browser integrity,
browser provenance, SDK/server authentication, replay, compromise, rotation,
and inventory revocation while preserving active-cache authority.
The default-off `pz2` codec now binds complete inventory identity under an
epoch-selected current/previous HMAC key ring, supports measured dual reads,
and exposes an explicit legacy-disable gate without changing generated tags.
The independent default-off SDK/server gate now requires an App-scoped Ed25519
signature over the exact body and canonical request context, bounded freshness,
an immutable public-key snapshot, exact active-cache publisher/App scope, and a
shared one-use Redis nonce claim. S02-scoped lifecycle controls issue the
private value once, store only the public verifier, require named permissions
and recent MFA, and transactionally audit issue, rotation, and revocation.
Detailed tasks and verification are in [status-P03.md](status-P03.md).

## R01 - Conversion, Action, And Attribution Measurement `[+]`

Add signed, idempotent conversion and post-click action collection, attribution
semantics, privacy-aware retention, ledger integration, and advertiser-facing
measurement. Detailed tasks are in [status-R01.md](status-R01.md).

## R02 - Marketplace Analytics And Experimentation `[+]`

Expand reporting dimensions and commercial metrics, define reporting freshness,
and add controlled A/B assignment only after conversion attribution is
reliable. Detailed tasks are in [status-R02.md](status-R02.md).

## R03 - Experiment And Reporting Integrity `[ ]`

Make new experiment assignment namespaces server-owned and cross-experiment
unlinkable, preserve existing assignments through an explicit algorithm
version, and reject malformed or mis-scoped analytical facts. Detailed tasks
and verification are in [status-R03.md](status-R03.md).

## I01 - OpenRTB Partner Interoperability `[+]`

Harden bounded gzip handling, OpenRTB 2.5 partner compatibility, sanitation,
floor/currency/media validation, rejection reasons, and response-time evidence.
Detailed tasks are in [status-I01.md](status-I01.md).

## I02 - Android And iOS Publisher SDKs `[ ]`

Build versioned native wrappers, sample applications, privacy propagation, and
release guidance around the stable `/pz` API after a named mobile integration
justifies maintained SDKs. Detailed tasks are in [status-I02.md](status-I02.md).

## I03 - External Campaign Management API `[+]`

Expose a versioned, scoped, quota-controlled, idempotent, and auditable API for
advertiser campaign management and reporting. Detailed tasks are in
[status-I03.md](status-I03.md).

## S01 - Privacy, Consent, And Data Disclosure `[+]`

Centralize consent interpretation, user-data use and disclosure, middleman
request sanitation, retention, deletion, and audit redaction across `/bid`,
`/pz`, trackers, and external fanout. Detailed tasks are in
[status-S01.md](status-S01.md).

## S02 - Identity, Two-Factor Authentication, And RBAC `[+]`

Add TOTP-based two-factor authentication, recovery controls, granular account
permissions, a read-only analyst role, session hardening, and security audit
events. Detailed tasks are in [status-S02.md](status-S02.md).

## S03 - Traffic Quality And Anti-Fraud `[+]`

Add explainable rule-based invalid-traffic detection, review/quarantine tools,
partner enforcement, and auditable counters without introducing automatic ML
decisions. Detailed tasks are in [status-S03.md](status-S03.md).

## S04 - Template Escaping And XSS Audit `[+]`

Audit every public and authenticated Summer/Genelet rendering path, inventory
intentional stored-markup previews, and centralize the narrow sanitized
safe-HTML boundary without weakening contextual `html/template` escaping.
Detailed tasks are in [status-S04.md](status-S04.md).

## S05 - Runtime Trust-Boundary Hardening `[ ]`

Harden special-use address rejection, injected HTTP clients, creative consumers,
principal provenance, quality-rule version selection, and protected database
columns without stripping legitimate sandboxed ad scripts or blocking valid
state transitions. Detailed tasks are in [status-S05.md](status-S05.md).

## S06 - Public Account Abuse Protection `[+]`

Require scoped Turnstile verification before registration/recovery work, apply
atomic pseudonymous Redis quotas, derive client identity only through reviewed
trusted proxies, and layer a Cloudflare edge rate limit over public account
POSTs. Review2 reopened the trusted admin marker, anonymous error rendering,
Gmail MIME/concurrency, and quota-script tasks; all were remediated. The managed
widget, owner-selected Free-plan exact-path 10-second burst rule, production
service configuration, live provider/dependency proof, and rollback/restore
evidence are complete. The Free rule still cannot distinguish GET from POST.
Detailed tasks and verification are in
[status-S06.md](status-S06.md).

## A01 - Billing And Manual Settlement Safety `[+]`

Define charge/pay and CPM/eCPM accounting contracts, support auditable manual
invoicing and publisher settlement, and retire unsafe collection of full card
or bank credentials. Detailed tasks are in [status-A01.md](status-A01.md).

## A02 - Hosted Funding And Publisher Payout Integration `[+]`

Integrate hosted/tokenized external funding and payout providers with
idempotent webhooks, reconciliation, refund/chargeback handling, and secret
management. Aofei must never store full card or bank credentials. Detailed
tasks are in [status-A02.md](status-A02.md).

Result: default-off Stripe Checkout and Connect Express integration now uses
mandatory independently approved opaque bindings, maker/checker operation
states, stable provider idempotency, signed replay/order-safe webhooks,
connected-account isolation, exact Balance Transaction reconciliation, explicit
refund/dispute/payout exceptions, restricted maintenance, fixed-cardinality
metrics, and the A01 manual outage fallback. At A02 closeout the clean baseline
was 94 tables, 6 routines, and 55 triggers. Recorded/disposable verification is complete; live
provider sandbox, migration, governance, and production enablement remain
external go-live gates rather than repository completion claims.

## A03 - Exact Monetary Source Migration `[ ]`

Extend exact money through authoritative demand, reservation, ledger, daily,
management, statement, and hosted reconciliation sources using a versioned,
auditable migration that does not invent precision for historical float data.
Detailed tasks and verification are in [status-A03.md](status-A03.md).

## O01 - Production Traffic Controls And Observability `[+]`

Operationalize protected metrics and alerts, add partner QPS/concurrency and
overload controls, record timeout/rejection/latency evidence, and establish a
repeatable capacity baseline. Detailed tasks are in
[status-O01.md](status-O01.md).

## O02 - Single-Region Availability, Recovery, And SLO `[+]`

Multi-node lifecycle readiness/failover, renewable singleton ownership,
durable ledger identities, dependency semantics, clean-room restore/cache
rebuild, recovery objectives, and an evidence-gated 99.9% SLO contract are
complete. Production 99.9% and provider RPO/RTO achievement remain explicitly
unclaimed until a named production window supplies retained evidence. Detailed
results are in [status-O02.md](status-O02.md).

## O03 - Job, Cache, And Filesystem Reliability `[ ]`

Harden singleton renewal, atomic reusable cache publication, spread-generation
ordering, callback recovery evidence, filesystem permissions/atomicity, and
malformed geodata handling without weakening O02 split-brain safety. Detailed
tasks and verification are in [status-O03.md](status-O03.md).

## Deferred Product Investments

[docs/defer.md](../docs/defer.md) records automatic bidding/ML, internally
operated payment-card processing, multi-region deployment, and million-RPM
engineering together with their current alternatives and evidence-based
reconsideration triggers. Deferred work has no reserved lane ID until its
trigger is satisfied.

## Historical Middleman Carry-Forward

- D03 owns the decision whether middleman routes need spread/local snapshots.
- A01 and A02 own invoicing and payment execution from `daily_mid` facts.
- Arbitrary downstream markup impression/click rewriting remains closed unless
  R01 identifies a measurement requirement that cooperative click notification
  cannot satisfy.
