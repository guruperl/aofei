# Retired milestone M11 - Genelet And Summer UI Stewardship

**Milestone.** M11
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M11.md
**Source specification.** memory-bank/milestone.md#m11---genelet-and-summer-ui-stewardship
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
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
````

## Status record

````markdown
# Status M11 - Genelet And Summer UI Stewardship

Milestone status: `[+]` Completed

Goal: review and harden Genelet/Summer admin framework internals and document
the current UI/runtime structure.

## Tasks

- `[+]` Carry M05 residual findings into M11.
  - M05 residuals covered here: component-loader panics, duplicated
    `cmd/unify` registrations, central SQL identifier validation, and broader
    guarded controller/auth dispatch.

- `[+]` Add error-returning component loading.
  - Files: `genelet/component.go`, `cmd/unify/main.go`.
  - Result: `genelet.LoadComponent` returns file, JSON, and validation errors;
    `cmd/unify` uses it during startup. `NewComponent` remains as a legacy
    panic wrapper for existing tests/callers.

- `[+]` Add central Genelet SQL identifier validation.
  - Files: `genelet/sql_ident.go`, `genelet/crud.go`, `genelet/model.go`.
  - Result: active CRUD helpers validate table, alias, key, field, join,
    select-label, filter, and order identifiers. Request-driven `_gsql`
    fragments are rejected by active condition building.

- `[+]` Guard controller/model/filter dispatch.
  - Files: `genelet/utils.go`, `genelet/controller.go`, `genelet/model.go`.
  - Result: reflection dispatch now validates method presence, arity, argument
    assignability, embedded model adapters, and panic recovery. Controller
    paths return framework errors for missing/wrong methods and forwarded group
    count mismatches.

- `[+]` Harden request handling.
  - Files: `genelet/controller.go`.
  - Result: static file serving resolves paths under `DocumentRoot`; multipart
    fields and upload filenames are bounded/validated; login/logout errors are
    typed before status handling.

- `[+]` Centralize Summer module registration.
  - Files: `summer/registry/registry.go`, `cmd/unify/main.go`.
  - Result: model, storage model, filter, and component module names are
    declared once and reused by the unified service.

- `[+]` Fix mutable Summer UI option state.
  - Files: `summer/filter.go`, `summer/item/filter.go`, `summer/slot/filter.go`.
  - Result: request-specific option selections/translations use cloned
    `LARGES` entries, and Redis/NATS/spread storage side effects pass through
    typed helper functions.

- `[+]` Add targeted tests.
  - Files: `genelet/*_test.go`, `summer/filter_test.go`,
    `summer/registry/registry_test.go`.
  - Coverage: component load errors, SQL identifier rejection, dispatch errors,
    static path traversal, forwarded group count mismatch, registry coverage,
    and `LARGES` mutation safety.

- `[+]` Add operator/developer docs.
  - Files: `docs/genelet-manual.md`, `docs/summer-ui-structure.md`,
    `README.md`, `memory-bank/architecture.md`, `memory-bank/tech-stack.md`.

- `[+]` Run M11 verification.
  - Required:
    ```bash
    GOWORK=off go test ./genelet ./summer/... ./cmd/unify
    GOWORK=off go test ./...
    ./scripts/aofei-doc-check.sh
    git diff --check
    ```
  - DB-backed admin compatibility with `SUMMER="$PWD/etc/summer.local.json"` is
    recorded because Docker local config is present.

## Review Findings - 2026-05-12

- `[+]` Component loading panicked on missing/malformed component JSON.
- `[+]` Active CRUD condition building accepted request-derived `_gsql` raw SQL
  fragments.
- `[+]` Request-derived sort/order fields and component table metadata were not
  validated at one Genelet seam before SQL construction.
- `[+]` Controller reflection dispatch could panic on missing methods, wrong
  signatures, or embedded Summer/Genelet model type mismatches.
- `[+]` Forwarded group parsing could index beyond configured role attributes
  or silently accept malformed group counts.
- `[+]` Login/logout handlers assumed every error was `Gerror`.
- `[+]` `cmd/unify` repeated the Summer model/storage/filter registry in three
  maps.
- `[+]` Summer `LARGES` option entries were mutated by request-specific
  selected/translated state.
- `[+]` Summer Redis/NATS/cache side effects used unchecked storage type
  assertions.

## Verification

Passed on 2026-05-12:

```bash
GOWORK=off go test ./genelet ./summer/... ./cmd/unify
GOWORK=off go test ./...
GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  go test ./genelet ./summer ./summer/pub ./summer/slot ./summer/weight
./scripts/aofei-doc-check.sh
git diff --check
```

## Result

- Genelet active setup now reports component and dispatch errors instead of
  panicking.
- Active Genelet CRUD/query construction validates SQL identifiers and rejects
  request-provided `_gsql` fragments.
- Summer module registration is centralized in `summer/registry`, and
  `cmd/unify` uses that registry.
- Summer option and cache side-effect paths no longer mutate shared option state
  or rely on unchecked storage assertions.
- Added [docs/genelet-manual.md](../docs/genelet-manual.md) and
  [docs/summer-ui-structure.md](../docs/summer-ui-structure.md).

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Carry M05 residual findings into M11. | `[+]` | |
| Add error-returning component loading. | `[+]` | |
| Add central Genelet SQL identifier validation. | `[+]` | |
| Guard controller/model/filter dispatch. | `[+]` | |
| Harden request handling. | `[+]` | |
| Centralize Summer module registration. | `[+]` | |
| Fix mutable Summer UI option state. | `[+]` | |
| Add targeted tests. | `[+]` | |
| Add operator/developer docs. | `[+]` | |
| Run M11 verification. | `[+]` | |
| Component loading panicked on missing/malformed component JSON. | `[+]` | |
| Active CRUD condition building accepted request-derived `_gsql` raw SQL | `[+]` | |
| Request-derived sort/order fields and component table metadata were not | `[+]` | |
| Controller reflection dispatch could panic on missing methods, wrong | `[+]` | |
| Forwarded group parsing could index beyond configured role attributes | `[+]` | |
| Login/logout handlers assumed every error was `Gerror`. | `[+]` | |
| `cmd/unify` repeated the Summer model/storage/filter registry in three | `[+]` | |
| Summer `LARGES` option entries were mutated by request-specific | `[+]` | |
| Summer Redis/NATS/cache side effects used unchecked storage type | `[+]` | |

````
