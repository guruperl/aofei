# Status M5 - Summer/Genelet Admin Compatibility

Milestone status: `[ ]` Pending

Goal: Align admin models, filters, and components with the active Docker schema.

## Tasks

- `[ ]` Inventory Summer and Genelet test coverage.
  - Files: `summer/**/*_test.go`, `genelet/**/*_test.go`.
  - Command:
    ```bash
    find summer genelet -name '*_test.go' | sort
    ```
  - Acceptance: test files are grouped by package and schema dependency.

- `[ ]` Verify generated Summer config works against Docker MySQL.
  - Files: `etc/summer.local.json`, `scripts/aofei-local.sh`, `summer/*`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer -run '^$'
    ```
  - Acceptance: package loads with generated config and no active `conf/`
    dependency.

- `[ ]` Run focused Summer model tests.
  - Files: `summer/model_test.go`, `summer/*/model_test.go`.
  - Command:
    ```bash
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot -run 'Model|DB|Get|List'
    ```
  - Acceptance: failures are fixed or recorded with exact table/column mismatch.

- `[ ]` Run focused Summer filter tests.
  - Files: `summer/filter_test.go`, `summer/*/filter_test.go`.
  - Command:
    ```bash
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot -run 'Filter'
    ```
  - Acceptance: filters match active schema columns and expected query behavior.

- `[ ]` Compare Summer component JSON to active model fields.
  - Files: `summer/**/component.json`, `summer/**/model.go`,
    `summer/**/filter.go`.
  - Command:
    ```bash
    rg -n '"field"|"name"|"column"' summer --glob 'component.json'
    ```
  - Acceptance: any stale component-field references are corrected or listed.

- `[ ]` Verify Genelet framework tests that do not need production services.
  - Files: `genelet/*_test.go`, `genelet/test.conf`.
  - Command:
    ```bash
    GOWORK=off go test ./genelet
    ```
  - Acceptance: framework tests pass or config-dependent skips are explicit.

- `[ ]` Document active admin runtime assumptions.
  - Files: `docs/local-docker-runtime.md`, `memory-bank/architecture.md`,
    `memory-bank/tech-stack.md`.
  - Acceptance: docs explain how Summer config is generated and how admin tests
    connect to Docker MySQL.

- `[ ]` Run M5 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot
    GOWORK=off go test ./genelet
    git diff --check
    ```
  - Acceptance: commands pass or remaining blockers are exact and assigned to a
    later milestone.

## Review Findings

- `[ ]` Resolve Summer slot `size_id` contract drift. Tests and edit paths use a
  `size_id` column, while the active `pub_slot` schema does not store that
  column and slot filters synthesize it.

- `[ ]` Stop loading DSP config through Genelet config paths in Summer tests.
  Tests that point at `etc/aofei.local.json` fail because Genelet expects
  `ConnectArray`.

- `[ ]` Make component and fixture paths independent of package working
  directory. Several tests assume relative paths that only work from a specific
  cwd.

- `[ ]` Replace component-loader panics with actionable errors where practical.
  Missing component JSON currently crashes tests and commands instead of
  returning setup diagnostics.

- `[ ]` Review `cmd/unify` model, storage, and filter registries. The command
  hardcodes duplicated registrations, making missing Summer module coverage
  hard to verify.
