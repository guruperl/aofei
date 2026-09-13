# Retired milestone M25 - Middleman Auction Expansion

**Milestone.** M25
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M25.md
**Source specification.** memory-bank/milestone.md#m25---middleman-auction-expansion
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M25 - Middleman Auction Expansion `[+]`

Allow explicitly gated middleman fanout to compete with local bids after M24 is
closed and reviewed.

Scope:

- Add `middleman_always_enabled`, default false.
- Include `trigger_mode` in route cache behavior for `Always` routes.
- Keep `Fallback` routes limited to local no-bid impressions.
- Let `Always` middleman bids compete with local bids on effective CPM after
  margin markup, while preserving local-wins fallback when comparison is unsafe.

Acceptance:

- `middleman_enabled` remains required.
- `middleman_always_enabled=false` ignores `Always` route fanout.
- Existing timeout, bidder limit, credential, ACL/channel, USD, floor, and
  callback-proxy controls continue to apply.
- Mixed local and middleman winners can be returned in one OpenRTB response.
````

## Status record

````markdown
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

- `[+]` Optional Redis-independent route-snapshot ownership is consolidated in
  D03 as a cache-mode parity decision, not a bidder-runtime behavior change.
- `[X]` Arbitrary downstream markup impression/click rewriting remains closed;
  R01 may open a new milestone only if cooperative notification cannot satisfy
  a concrete measurement requirement.

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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Add `middleman_always_enabled`, default false. | `[+]` | |
| Include `trigger_mode` in route-cache runtime behavior. | `[+]` | |
| Keep legacy `middleman:routes` fallback-only and publish M25 | `[+]` | |
| Preserve `Fallback` routes as local-no-bid-only. | `[+]` | |
| Let `Always` route bids compete with local bids on effective CPM after | `[+]` | |
| Preserve existing callback proxying, credential, timeout, bidder-limit, | `[+]` | |
| Update docs, memory, tests, and verification after implementation. | `[+]` | |
| Optional Redis-independent route-snapshot ownership is consolidated in | `[+]` | |
| Arbitrary downstream markup impression/click rewriting remains closed; | `[X]` | |
| Runtime tests where local wins over lower marked-up middleman bid. | `[+]` | |
| Runtime tests where `Always` middleman bid beats local by effective CPM. | `[+]` | |
| Tests proving `Fallback` behavior is unchanged. | `[+]` | |
| Tests proving `middleman_always_enabled=false` ignores `Always` fanout. | `[+]` | |
| Mixed-impression tests with local and middleman winners in one response. | `[+]` | |
| `GOWORK=off go test ./dsp ./match ./internal/jobs/cache` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./dsp ./match ./internal/jobs/cache ./summer/midroute ./cmd/redis-cache` | `[+]` | |
| `./scripts/aofei-local.sh check-sql` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes` | `[+]` | |
| `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=routes -read` | `[+]` | |

````
