# Status M7 - MaxMind And Geo Runtime

Milestone status: `[ ]` Pending

Goal: Make geodata expectations explicit and locally testable.

## Tasks

- `[ ]` Inspect active MaxMind config.
  - Files: `etc/maxmind.json`, `maxmind/*`, `maxmind/ipsearch/*`.
  - Command:
    ```bash
    sed -n '1,160p' etc/maxmind.json
    ```
  - Acceptance: every configured path and data source is understood.

- `[ ]` Identify external geodata assets that are not in git.
  - Files: `etc/maxmind.json`, `.gitignore`, `docs/maxmind-runtime.md`.
  - Command:
    ```bash
    rg -n 'mmdb|GeoLite|GeoIP|csv|ip' etc maxmind .gitignore
    ```
  - Acceptance: required external files are listed with expected local paths.

- `[ ]` Verify maxmind package tests that do not require external assets.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch -run '^$'
    ```
  - Acceptance: packages compile without external assets or blockers are exact.

- `[ ]` Run available MaxMind tests with current local assets.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch
    ```
  - Acceptance: passing tests are recorded; failing tests identify missing
    assets or schema/config mismatch.

- `[ ]` Add skips or fixture paths for asset-dependent tests.
  - Files: `maxmind/*_test.go`, `maxmind/ipsearch/*_test.go`,
    `docs/maxmind-runtime.md`.
  - Acceptance: tests fail only for real code problems, not absent proprietary
    or large local assets.

- `[ ]` Verify `cmd/maxmind` build and local invocation.
  - Files: `cmd/maxmind/main.go`, `etc/maxmind.json`.
  - Command:
    ```bash
    GOWORK=off go test ./cmd/maxmind -run '^$'
    GOWORK=off go run ./cmd/maxmind -h || true
    ```
  - Acceptance: command requirements are known and documented.

- `[ ]` Create MaxMind runtime documentation.
  - Files: `docs/maxmind-runtime.md`, `README.md`,
    `memory-bank/tech-stack.md`.
  - Acceptance: docs explain config file, external assets, ignored paths, and
    test commands.

- `[ ]` Run M7 verification.
  - Command:
    ```bash
    GOWORK=off go test ./maxmind ./maxmind/ipsearch ./cmd/maxmind -run '^$'
    git diff --check
    ```
  - Acceptance: compile-level verification passes and full asset-dependent test
    behavior is documented.

## Review Findings

- `[ ]` Make MaxMind asset-dependent tests explicit. Current full tests expect
  local GeoLite/IP data files that are not present in the repository.

- `[ ]` Document or parameterize the active geodata path. `etc/maxmind.json`
  points at an external `/media` database path.

- `[ ]` Fix `cmd/maxmind` generation flow. The command creates a controller that
  loads configured IP data before generating or writing that target data.

- `[ ]` Replace remaining panic-style error handling in the IP search path with
  returned errors or documented fatal command behavior.
