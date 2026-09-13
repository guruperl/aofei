# Retired milestone M36 - Runtime Safety And Test/Observability Hardening

**Milestone.** M36
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M36.md
**Source specification.** memory-bank/milestone.md#m36---runtime-safety-and-testobservability-hardening
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
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
````

## Status record

````markdown
# Status M36 - Runtime Safety And Test/Observability Hardening

State: `[+]` Completed

Fix the meaningful confirmed risks from the post-M35 whole-repo review while
preserving current schema/cache/response contracts unless a task explicitly
records and verifies a contract decision.

## Tasks

- `[+]` Create the M36 status file and milestone scope.
- `[+]` Refactor or wrap `cmd/spread` service behavior for signal-aware
  shutdown, NATS drain/close, and non-blocking callback reporting.
- `[+]` Add focused tests for spread message handling and shutdown/extracted
  job behavior.
- `[+]` Harden middleman bidder fanout so custom clients cannot bypass endpoint
  validation, nil-client paths use the safe callback client, and unsafe
  endpoints are rejected before outbound network I/O.
- `[+]` Add cap-refresh contention/latency observability and document how to
  interpret it.
- `[+]` Add middleman callback retry backlog/staleness observability and
  operator alerting notes.
- `[+]` Document audit queue drop alerting and any additional fanout/callback
  failure counters added during the milestone.
- `[+]` Define the test taxonomy for package, integration, race, staticcheck,
  Docker smoke, and schema checks; update README, AGENTS, and memory-bank
  commands.
- `[+]` Decide local/spread cache staleness policy and implement the chosen
  runtime behavior with tests.
- `[+]` Triage low-risk adjacent cleanup items: `HhLock`, `DSP.impID` bounds,
  native macro invariant comments/tests, ADR cross-links, and `cron_halfhour`
  schema-hook status.
- `[+]` Check whether `evolution/` needs a new version after the policy and
  boundary decisions are final.
- `[+]` Run closeout verification and mark the milestone complete only after
  verification passes.

## Acceptance

- `[+]` `cmd/spread` exits cleanly on normal service signals without requiring
  `SIGKILL`.
- `[+]` Spread shutdown and message handling are covered by focused tests or by
  tests on extracted `internal/jobs/spread` logic.
- `[+]` Middleman bidder fanout always uses safe HTTP behavior or rejects unsafe
  endpoint URLs before outbound network I/O, including nil-client and custom
  client construction cases.
- `[+]` Operators can observe cap contention, audit drops, callback retry
  backlog/staleness, and relevant middleman callback/fanout failures.
- `[+]` README, AGENTS, and memory-bank verification commands distinguish
  package tests, integration tests, race tests, staticcheck, Docker smoke
  checks, and schema checks.
- `[+]` Local/spread static-cache freshness has a documented runtime policy,
  scrape-time loaded-at/age/stale metrics, and test coverage for fresh, stale,
  and idle-traffic age advancement states.
- `[+]` Schema-affecting cleanup around `cron_halfhour`, if performed, follows
  the normal schema baseline workflow; otherwise it is explicitly deferred.

## Verification

- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off go vet ./...`
- `[+]` `GOWORK=off staticcheck ./...`
- `[+]` `GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` No new integration-tag, Docker smoke, or schema-drift command was added
  as an extra mandatory closeout command; the milestone documented the explicit
  existing command families instead.

## Notes

- Confirmed review risks intentionally in scope: `cmd/spread` shutdown/job
  boundary, middleman bidder fanout HTTP safety fallback, cap/audit/retry
  observability, integration test taxonomy, and local/spread cache staleness.
- Rejected or stale review claims are not M36 scope: MaxMind FD leak as stated,
  silent non-USD floor handling as stated, ledger scanner 64 KiB limit, and a
  missing `adminapi` package comment.
- M36 uses explicit verification command families rather than adding build tags.
- Local/spread cache staleness is alert-only. Stale snapshots set expvar metrics
  but do not fail closed by age alone.
- 2026-05-14 closeout follow-up resolved the two remaining review findings:
  middleman bidder endpoint validation now applies even with custom HTTP
  clients, and local cache freshness metrics compute age/stale at scrape time
  from `aofei_local_cache_loaded_at_unix` so idle traffic cannot freeze the
  reported age.
- `cron_halfhour` cleanup is deferred because it changes the schema baseline and
  should be handled through the normal schema workflow.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M36 status file and milestone scope. | `[+]` | |
| Refactor or wrap `cmd/spread` service behavior for signal-aware | `[+]` | |
| Add focused tests for spread message handling and shutdown/extracted | `[+]` | |
| Harden middleman bidder fanout so custom clients cannot bypass endpoint | `[+]` | |
| Add cap-refresh contention/latency observability and document how to | `[+]` | |
| Add middleman callback retry backlog/staleness observability and | `[+]` | |
| Document audit queue drop alerting and any additional fanout/callback | `[+]` | |
| Define the test taxonomy for package, integration, race, staticcheck, | `[+]` | |
| Decide local/spread cache staleness policy and implement the chosen | `[+]` | |
| Triage low-risk adjacent cleanup items: `HhLock`, `DSP.impID` bounds, | `[+]` | |
| Check whether `evolution/` needs a new version after the policy and | `[+]` | |
| Run closeout verification and mark the milestone complete only after | `[+]` | |
| `cmd/spread` exits cleanly on normal service signals without requiring | `[+]` | |
| Spread shutdown and message handling are covered by focused tests or by | `[+]` | |
| Middleman bidder fanout always uses safe HTTP behavior or rejects unsafe | `[+]` | |
| Operators can observe cap contention, audit drops, callback retry | `[+]` | |
| README, AGENTS, and memory-bank verification commands distinguish | `[+]` | |
| Local/spread static-cache freshness has a documented runtime policy, | `[+]` | |
| Schema-affecting cleanup around `cron_halfhour`, if performed, follows | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off go vet ./...` | `[+]` | |
| `GOWORK=off staticcheck ./...` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| No new integration-tag, Docker smoke, or schema-drift command was added | `[+]` | |

````
