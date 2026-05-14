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
