# Status M24 - Middleman Operations Reliability

## Goal

Improve route-cache visibility, route health checks, and downstream callback
forwarding reliability for the existing middleman fallback system without
changing auction winner selection or adding MySQL work to `/bid`.

## Tasks

- `[+]` Route-only cache command.
  - Added `cmd/redis-cache -cache=routes` and `-cache=routes -read`.
  - Route-only mode updates or reads middleman route cache keys without
    rebuilding publisher, demand, audience, creative, or spread cache families.

- `[+]` Route-cache metadata.
  - Added optional version-1 JSON metadata with generation time, entry count,
    source, route-table high-water timestamp, and route-entry checksum.
  - Older version-1 payloads without metadata remain readable with unknown
    freshness.

- `[+]` Route-cache visibility.
  - Added Redis cache freshness details to admin `midroute` topics HTML and
    JSON output.
  - The UI reports status only; it does not run cache refresh.

- `[+]` Route health checks.
  - Added admin `/goto/admin/g/midroute?action=health` and JSON equivalent.
  - Health rows cover active groups with no active targets/bidders, inactive or
    unapproved route bidders, missing `credential_ref` names, and invalid
    synthetic campaign/item/creative chains.
  - Credential secrets remain outside MySQL/Redis/UI.

- `[+]` Durable downstream callback retry.
  - Added `mid_callback_retry` to `etc/step4_init.sql`.
  - Added `internal/jobs/midcallback` and `cmd/mid-callback-retry`.
  - `/mid/win`, `/mid/loss`, and `/mid/bill` enqueue only retryable downstream
    forwarding failures: network/request errors, HTTP 429, and HTTP 5xx.
  - Missing URLs, invalid URLs, duplicate callbacks, and HTTP 4xx responses
    other than 429 are not queued.
  - Retryable failures keep the notify idempotency key after a durable retry row
    is recorded, so duplicate exchange callbacks do not re-forward downstream.
  - Retry processing claims due rows as `Processing` before forwarding.
  - Retry processing forwards downstream only and does not republish win/loss
    or billable delivery records.

- `[+]` Documentation and memory.
  - Updated middleman, operational command, database, production, memory-bank,
    and evolution notes for M24.

- `[+]` Deep review fixes.
  - Adjusted route-cache freshness so a newly generated empty route cache is
    fresh when the route DB is also empty, while older metadata-less payloads
    remain unknown.

## Carry Forward

- `[ ]` Spread/local snapshots for bidder routes remain deferred; middleman
  routes are still Redis-only runtime cache data.
- `[+]` `trigger_mode='Always'` runtime behavior moved to M25 and is
  implemented behind `middleman_always_enabled`.
- `[ ]` Real invoicing/payment execution remains future settlement work.
- `[X]` Arbitrary downstream markup impression/click rewrite remains a
  non-goal unless future reporting requires reopening it.

## Verification

- `[+]` `GOWORK=off go test ./match ./internal/jobs/cache ./internal/jobs/midcallback ./summer/midroute ./dsp ./cmd/redis-cache ./cmd/mid-callback-retry`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./internal/jobs/cache ./internal/jobs/midcallback ./summer/midroute ./summer/registry ./cmd/redis-cache ./cmd/mid-callback-retry`
- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh diff-schema`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes -read`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/mid-callback-retry -read`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
- `[+]` `cd ../pzdesign && go run ./tools/check-templates.go -ext=.g`
