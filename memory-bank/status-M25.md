# Status M25 - Middleman Auction Expansion

## Goal

Plan and implement explicitly gated middleman `Always` fanout after M24 closes,
allowing eligible downstream bids to compete with local bids on effective CPM.

## Tasks

- `[+]` Add `middleman_always_enabled`, default false.
- `[+]` Include `trigger_mode` in route-cache runtime behavior.
- `[+]` Keep legacy `middleman:routes` fallback-only and publish M25
  `middleman:routes:v2` to avoid old runtimes treating `Always` as fallback.
- `[+]` Preserve `Fallback` routes as local-no-bid-only.
- `[+]` Let `Always` route bids compete with local bids on effective CPM after
  margin markup.
- `[+]` Preserve existing callback proxying, credential, timeout, bidder-limit,
  ACL/channel, USD, and floor safety controls.
- `[+]` Update docs, memory, tests, and verification after implementation.

## Carry Forward

- `[ ]` Optional Redis-independent route snapshots remain a cache-mode parity
  decision, not a bidder-runtime behavior change.
- `[ ]` Arbitrary downstream markup impression/click rewrite remains closed.

## Verification

- `[+]` Runtime tests where local wins over lower marked-up middleman bid.
- `[+]` Runtime tests where `Always` middleman bid beats local by effective CPM.
- `[+]` Tests proving `Fallback` behavior is unchanged.
- `[+]` Tests proving `middleman_always_enabled=false` ignores `Always` fanout.
- `[+]` Mixed-impression tests with local and middleman winners in one response.
- `[+]` `GOWORK=off go test ./dsp ./match ./internal/jobs/cache`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./internal/jobs/cache ./summer/midroute ./cmd/redis-cache`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes -read`
