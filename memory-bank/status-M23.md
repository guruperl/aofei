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

- `[ ]` Route cache refresh is still a singleton `cmd/redis-cache` operation;
  route edits do not push cache invalidation from the UI.
- `[ ]` Spread/local snapshots for bidder routes remain deferred after M23;
  middleman routes are still Redis-only runtime cache data.
- `[ ]` `trigger_mode='Always'` remains schema/UI-visible but runtime-inactive;
  bid fanout still uses `Fallback` after local no-bid.
- `[ ]` Durable callback retry queues remain post-M23 reliability work.
- `[ ]` Real invoicing/payment execution remains future settlement work.
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
