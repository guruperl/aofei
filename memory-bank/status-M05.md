# Status M05 - Summer/Genelet Admin Compatibility

Milestone status: `[+]` Completed

Goal: Align admin models, filters, and components with the active Docker schema.

## Tasks

- `[+]` Inventory Summer and Genelet test coverage.
  - Files: `summer/**/*_test.go`, `genelet/**/*_test.go`.
  - Command:
    ```bash
    find summer genelet -name '*_test.go' | sort
    ```
  - Result: Summer tests are concentrated in root, `pub`, `slot`, and `weight`
    packages; Genelet has framework tests plus DB-backed model/crud/procedure
    tests now driven by `SUMMER`.

- `[+]` Verify generated Summer config works against Docker MySQL.
  - Files: `etc/summer.local.json`, `scripts/aofei-local.sh`, `summer/*`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer -run '^$'
    ```
  - Result: `./summer -run '^$'` passes with `SUMMER` pointing at
    `etc/summer.local.json`.

- `[+]` Run focused Summer model tests.
  - Files: `summer/model_test.go`, `summer/*/model_test.go`.
  - Command:
    ```bash
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot -run 'Model|DB|Get|List'
    ```
  - Result: model tests pass against Docker MySQL. Fixed stale `pub_slot.size_id`
    expectations, package-relative component paths, isolated test table names,
    and stale sample IDs.

- `[+]` Run focused Summer filter tests.
  - Files: `summer/filter_test.go`, `summer/*/filter_test.go`.
  - Command:
    ```bash
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot -run 'Filter'
    ```
  - Result: filter tests pass. Tests now use the generated Summer config instead
    of the DSP config.

- `[+]` Compare Summer component JSON to active model fields.
  - Files: `summer/**/component.json`, `summer/**/model.go`,
    `summer/**/filter.go`.
  - Command:
    ```bash
    rg -n '"field"|"name"|"column"' summer --glob 'component.json'
    ```
  - Result: corrected `summer/weight/component.json` to read creative `size_id`
    through `adv_creative` instead of stale `adv_item.size_id`.

- `[+]` Verify Genelet framework tests that do not need production services.
  - Files: `genelet/*_test.go`, `genelet/test.conf`.
  - Command:
    ```bash
    GOWORK=off go test ./genelet
    ```
  - Result: `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./genelet`
    passes. DB-backed tests use generated Docker MySQL config and skip
    explicitly when `SUMMER` or Docker MySQL is unavailable.

- `[+]` Document active admin runtime assumptions.
  - Files: `docs/local-docker-runtime.md`, `memory-bank/architecture.md`,
    `memory-bank/tech-stack.md`.
  - Result: docs and memory bank explain `SUMMER` vs `AOFEI` and admin test
    commands.

- `[+]` Run M05 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot
    GOWORK=off go test ./genelet
    git diff --check
    ```
  - Result: verification passed on 2026-05-12.

## Review Findings

- `[+]` Resolve Summer slot `size_id` contract drift. Tests and edit paths use a
  `size_id` column, while the active `pub_slot` schema does not store that
  column and slot filters synthesize it.

- `[+]` Stop loading DSP config through Genelet config paths in Summer tests.
  Tests that point at `etc/aofei.local.json` fail because Genelet expects
  `ConnectArray`.

- `[+]` Make component and fixture paths independent of package working
  directory. Several tests assume relative paths that only work from a specific
  cwd.

- `[+]` Replace component-loader panics with actionable errors where practical.
  Missing component JSON currently crashes tests and commands instead of
  returning setup diagnostics.
  - Carried to and resolved in M11: active setup uses `genelet.LoadComponent`;
    `NewComponent` remains as the legacy panic wrapper.

- `[+]` Review `cmd/unify` model, storage, and filter registries. The command
  hardcodes duplicated registrations, making missing Summer module coverage
  hard to verify.
  - Carried to and resolved in M11: `summer/registry` owns component-backed
    model, storage-model, and filter registration.

### Second Review Pass - 2026-05-12

- `[+]` Add a central SQL identifier whitelist for Genelet/Summer query
  builders. Request or component-derived fields, table names, and `_gsql`
  clauses are currently interpolated into SQL outside a single validation seam.
  - Carried to and resolved in M11: active CRUD/query builders validate
    identifiers and reject request-provided `_gsql` fragments.

- `[+]` Lock down Summer access-control SQL inputs. `summer/ac` uses request
  `table` and `idname` values directly in `SELECT` and `UPDATE` statements.

- `[+]` Sanitize admin upload filenames and enforce upload size limits.
  Multipart handling and creative upload moves use client-provided filenames in
  filesystem paths without `filepath.Base`/path traversal checks.

- `[+]` Replace fragile type assertions and header indexing with guarded error
  paths. Login/logout error handling and forwarded group parsing can panic on
  ordinary database errors or malformed auth headers.
  - Carried to and resolved in M11: login/logout check error types and
    forwarded group count mismatches return framework errors.

## Milestone Review - 2026-05-12

- `[+]` Reviewed M05 after verification. No blocking regressions found in the
  Summer/Genelet config usage, active-schema test updates, upload path
  hardening, or access-control table/id whitelisting.
- `[+]` Residual hardening was carried to M11 and resolved there: central
  Genelet SQL identifier validation, component loading as errors instead of
  active setup panics, `cmd/unify` registry deduplication, and broader
  login/logout error guard cleanup.
