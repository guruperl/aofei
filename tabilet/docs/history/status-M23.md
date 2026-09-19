# Retired milestone M23 - Middleman Route Operations UI

**Milestone.** M23
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M23.md
**Source specification.** memory-bank/milestone.md#m23---middleman-route-operations-ui
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
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
````

## Status record

````markdown
# Status M23 - Middleman Route Operations UI

## Goal

Make middleman route assignment operable from Summer/Genelet while preserving
the existing runtime boundary: `cmd/unify` serves HTTP and reads Redis route
cache data, and the singleton `cmd/redis-cache` job refreshes that cache.

## Tasks

- `[+]` Route admin module.
  - Added the `summer/midroute` component, registry entry, model, and filter.
  - Exposed admin HTML and JSON routes under `/goto/admin/{g,json}/midroute`.

- `[+]` Route-group operations.
  - Added list, new, create, edit, update, and delete actions for
    `mid_route_group`.
  - Validated trigger mode, total timeout, margin percentage, minimum margin
    CPM, and active state before writes.

- `[+]` Route-bidder operations.
  - Added nested bidder membership actions for `mid_route_bidder`.
  - Supported nullable per-bidder timeout, margin percentage, and minimum margin
    overrides.

- `[+]` Route-target operations.
  - Added nested target actions for `mid_route_target`.
  - Supported global routes plus publisher, site, and slot scopes with optional
    size targeting.

- `[+]` Templates.
  - Added admin `midroute` Go templates in `../pzdesign/tmpls`.
  - Added the Middleman route navigation entry in the admin shell.

- `[+]` Documentation and memory.
  - Updated milestone, architecture, product, Summer UI, middleman, production,
    and evolution notes for the route-operations workflow.

- `[+]` Deep review fixes.
  - Fixed partial route update handling so absent fields preserve existing
    values and present blank non-null fields are rejected.
  - Fixed nullable middleman ledger rates so SQL and templates render zero
    instead of Go `%!f(<nil>)` formatting.
  - Preserved literal OpenRTB price/currency macros in middleman and win/loss
    callback URLs while keeping non-macro query values escaped.

## Carry Forward

- `[+]` Route cache refresh is still a singleton `cmd/redis-cache` operation;
  M24 added route-only refresh and UI freshness visibility while preserving
  cache-node ownership.
- `[+]` Optional spread/local route-snapshot ownership is consolidated in D03;
  middleman routes remain Redis-only until D03 records its decision.
- `[+]` `trigger_mode='Always'` runtime behavior moved to M25 and is
  implemented behind `middleman_always_enabled`.
- `[+]` Durable callback retry queues moved to M24 and are implemented for
  retryable downstream post-auction callback forwarding failures.
- `[+]` Real invoicing/payment ownership is consolidated in A01/A02.
- `[X]` Arbitrary downstream markup impression/click rewrite remains a
  non-goal unless future reporting requires reopening it.

## Verification

- `[+]` `GOWORK=off go test ./summer/midroute`
- `[+]` `GOWORK=off go test ./summer/registry ./cmd/unify`
- `[+]` `GOWORK=off go test ./summer/midroute ./summer/registry ./genelet ./cmd/unify`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./summer/ledger ./summer/midroute ./summer/registry ./cmd/unify`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
- `[+]` `cd ../pzdesign && go run ./tools/check-templates.go -ext=.g`
- `[+]` `cd ../pzdesign && go run ./tools/check-templates.go -ext=.e`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Route admin module. | `[+]` | |
| Route-group operations. | `[+]` | |
| Route-bidder operations. | `[+]` | |
| Route-target operations. | `[+]` | |
| Templates. | `[+]` | |
| Documentation and memory. | `[+]` | |
| Deep review fixes. | `[+]` | |
| Route cache refresh is still a singleton `cmd/redis-cache` operation; | `[+]` | |
| Optional spread/local route-snapshot ownership is consolidated in D03; | `[+]` | |
| `trigger_mode='Always'` runtime behavior moved to M25 and is | `[+]` | |
| Durable callback retry queues moved to M24 and are implemented for | `[+]` | |
| Real invoicing/payment ownership is consolidated in A01/A02. | `[+]` | |
| Arbitrary downstream markup impression/click rewrite remains a | `[X]` | |
| `GOWORK=off go test ./summer/midroute` | `[+]` | |
| `GOWORK=off go test ./summer/registry ./cmd/unify` | `[+]` | |
| `GOWORK=off go test ./summer/midroute ./summer/registry ./genelet ./cmd/unify` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./dsp ./summer/ledger ./summer/midroute ./summer/registry ./cmd/unify` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |
| `cd ../pzdesign && go run ./tools/check-templates.go -ext=.g` | `[+]` | |
| `cd ../pzdesign && go run ./tools/check-templates.go -ext=.e` | `[+]` | |

````
