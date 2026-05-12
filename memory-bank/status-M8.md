# Status M8 - Full Repository Test Hygiene

Milestone status: `[+]` Completed

Goal: Move from scoped smoke checks to a clean repository-level verification
target.

## Tasks

- `[+]` Reproduce current package discovery state.
  - `GOWORK=off go list ./...` initially discovered
    `github.com/genelet/winter/backup`.
  - After adding `ignore` build tags to historical helpers, `backup` is absent
    from package discovery.

- `[+]` Resolve `backup/` Go package discovery.
  - `backup/step1.go` and `backup/step2.go` are preserved with
    `//go:build ignore`.
  - `backup/README` records that the directory is historical and outside the
    active package test surface.

- `[+]` Run full package compile tests.
  - Command:
    ```bash
    GOWORK=off go test ./... -run '^$'
    ```
  - Result: passed.

- `[+]` Run full tests where safe.
  - Command:
    ```bash
    GOWORK=off go test ./...
    ```
  - Result: passed in the live checkout and in a fresh-copy simulation with
    `etc/aofei.local.json` and `etc/summer.local.json` removed.

- `[+]` Decide canonical verification command.
  - Canonical package gate:
    ```bash
    GOWORK=off go test ./...
    ```
  - `README.md`, `AGENTS.md`, and `memory-bank/tech-stack.md` all name the same
    command.

- `[X]` Add a repository verification script if command sequencing remains
  complex.
  - Decision: no script for M8. The selected gate is a single Go command.

- `[X]` Add schema-contract coverage for SQL embedded outside the baseline
  file.
  - Decision: deferred and non-gating for M8. Schema checks remain documented
    under the Docker helper flow.

- `[+]` Verify parent workspace behavior.
  - `go list ./...` still fails with:
    `directory prefix . does not contain modules listed in go.work or their selected dependencies`.
  - `GOWORK=off go list ./...` succeeds.
  - No parent `go.work` change was made for M8.

- `[+]` Run M8 verification.
  - Commands:
    ```bash
    GOWORK=off go list ./...
    GOWORK=off go test ./... -run '^$'
    GOWORK=off go test ./...
    git diff --check
    ```
  - Result: passed.

## Review Findings

- `[+]` Record current test baseline: `GOWORK=off go test ./... -run '^$'`
  passed before the functional fixes; full `GOWORK=off go test ./...` initially
  failed in `advice` and `match`.

- `[+]` Remove historical code from active package discovery. `backup` no longer
  appears in `GOWORK=off go list ./...`.

- `[+]` Resolve stale `advice` enum expectations. Tests now expect zero-value
  wildcard string `"All"` and out-of-range values as `"Unknown"` where covered.

- `[+]` Fix stale and non-hermetic test fixtures in `match`; MaxMind asset tests
  already skip when external assets are absent; Summer/Genelet tests now handle
  absent generated configs cleanly.

- `[+]` Add defensive DB-test setup. Summer/Genelet DB-backed tests skip on
  absent config or unavailable DB and fail on malformed configs.

### Second Review Pass - 2026-05-12

- `[X]` Add static correctness checks to the hygiene backlog.
  - Decision: staticcheck remains useful but non-gating for M8. Current
    staticcheck strictness is documented as a follow-up check, not the package
    verification gate.

- `[X]` Keep callback concurrency under test, not only compile/race smoke.
  - Decision: deferred from M8. Existing package tests are the canonical gate;
    deeper NATS callback behavior coverage belongs in a later reliability task.

- `[+]` Replace legacy DSN test fixtures with Docker-aware setup or explicit
  skips. Summer/Genelet DB-backed tests use `SUMMER` or the generated local
  config path and skip only when the config is absent or DB ping fails.
