# Retired milestone M07 - MaxMind And Geo Runtime

**Milestone.** M07
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M07.md
**Source specification.** memory-bank/milestone.md#m07---maxmind-and-geo-runtime
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Milestone closed with its recorded acceptance evidence preserved in the retired status record.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
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
````

## Status record

````markdown
# Status M07 - MaxMind And Geo Runtime

Milestone status: `[+]` Completed

Goal: Make geodata expectations explicit and locally testable.

## Tasks

- `[+]` Inspect active MaxMind config.
  - Files: `etc/maxmind.json`, `maxmind/*`, `maxmind/ipsearch/*`.
  - Command:
    ```bash
    sed -n '1,160p' etc/maxmind.json
    ```
  - Acceptance: every configured path and data source is understood.

- `[+]` Identify external geodata assets that are not in git.
  - Files: `etc/maxmind.json`, `.gitignore`, `docs/maxmind-runtime.md`.
  - Command:
    ```bash
    rg -n 'mmdb|GeoLite|GeoIP|csv|ip' etc maxmind .gitignore
    ```
  - Acceptance: required external files are listed with expected local paths.

- `[+]` Verify maxmind package tests that do not require external assets.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch -run '^$'
    ```
  - Acceptance: packages compile without external assets or blockers are exact.

- `[+]` Run available MaxMind tests with current local assets.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch
    ```
  - Acceptance: passing tests are recorded; failing tests identify missing
    assets or schema/config mismatch.

- `[+]` Add skips or fixture paths for asset-dependent tests.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`,
    `docs/maxmind-runtime.md`.
  - Acceptance: tests fail only for real code problems, not absent proprietary
    or large local assets.

- `[+]` Verify `cmd/maxmind` build and local invocation.
  - Files: `cmd/maxmind/main.go`, `etc/maxmind.json`.
  - Command:
    ```bash
    GOWORK=off go test ./cmd/maxmind -run '^$'
    GOWORK=off go run ./cmd/maxmind -h || true
    ```
  - Acceptance: command requirements are known and documented.

- `[+]` Create MaxMind runtime documentation.
  - Files: `docs/maxmind-runtime.md`, `README.md`,
    `memory-bank/tech-stack.md`.
  - Acceptance: docs explain config file, external assets, ignored paths, and
    test commands.

- `[+]` Run M07 verification.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch ./cmd/maxmind -run '^$'
    git diff --check
    ```
  - Acceptance: compile-level verification passes and full asset-dependent test
    behavior is documented.

## Review Findings

- `[+]` Make MaxMind asset-dependent tests explicit. Current full tests expect
  local GeoLite/IP data files that are not present in the repository.

- `[+]` Document or parameterize the active geodata path. `etc/maxmind.json`
  points at an external `/media` database path.

- `[+]` Fix `cmd/maxmind` generation flow. The command creates a controller that
  loads configured IP data before generating or writing that target data.

- `[+]` Replace remaining panic-style error handling in the IP search path with
  returned errors or documented fatal command behavior.

### Second Review Pass - 2026-05-12

- `[+]` Make MaxMind data generation atomic. `cmd/maxmind` writes to the
  configured IP data path directly, so interrupted generation can corrupt the
  active runtime file.

- `[+]` Remove the existing-IP-data dependency from generation startup.
  `cmd/maxmind` still creates a DSP controller before generating data, which
  loads MaxMind state before the command has produced it.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Inspect active MaxMind config. | `[+]` | |
| Identify external geodata assets that are not in git. | `[+]` | |
| Verify maxmind package tests that do not require external assets. | `[+]` | |
| Run available MaxMind tests with current local assets. | `[+]` | |
| Add skips or fixture paths for asset-dependent tests. | `[+]` | |
| Verify `cmd/maxmind` build and local invocation. | `[+]` | |
| Create MaxMind runtime documentation. | `[+]` | |
| Run M07 verification. | `[+]` | |
| Make MaxMind asset-dependent tests explicit. Current full tests expect | `[+]` | |
| Document or parameterize the active geodata path. `etc/maxmind.json` | `[+]` | |
| Fix `cmd/maxmind` generation flow. The command creates a controller that | `[+]` | |
| Replace remaining panic-style error handling in the IP search path with | `[+]` | |
| Make MaxMind data generation atomic. `cmd/maxmind` writes to the | `[+]` | |
| Remove the existing-IP-data dependency from generation startup. | `[+]` | |

````
