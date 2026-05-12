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
- `docs/genelet-manual.md` and `docs/summer-ui-structure.md` describe the
  current framework/operator contracts.

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
