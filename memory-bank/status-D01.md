# Status D01 - Campaign Delivery Guardrails

State: `[+]` Completed

## Goal

Make advertiser, campaign, and item budgets and schedules authoritative auction
eligibility so paid delivery cannot continue outside configured commercial
limits.

## Dependencies

- No new lane dependency; this is foundation work.
- Preserve existing OpenRTB and `/pz` response contracts.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Delivery contract | `[+]` | Total/daily spend, impression, and click limits; inclusive UTC windows; campaign-timezone weekly calendars; UTC daily reset; and fail-closed limited-demand semantics are defined in `docs/delivery-guardrails.md`. |
| Runtime eligibility | `[+]` | RAdv cache payload v2 carries the policy; the bid path rejects stale, malformed, exhausted, or out-of-window candidates before mutable request work. |
| Concurrent spend safety | `[+]` | One Lua reservation covers all limited scopes atomically, persists monotonic ledger floors, bounds spend/impression admission, and exposes lifecycle/error metrics. |
| Deterministic pacing | `[+]` | `Fast` enforces hard limits; `Even` uses deterministic UTC-day and effective campaign/item interval progress without adaptive optimization. |
| Cache freshness | `[+]` | Delivery snapshots fail closed after 900 seconds by default; the singleton publisher runs on a five-minute contract and local/spread controllers reload at one-third of the tightest bound. |
| Ledger reconciliation | `[+]` | Interval and daily jobs reconcile total/current-UTC-day balances; Redis never decreases below a newer reconciled floor. |
| UI and documentation | `[+]` | Advertiser campaign/item/balance forms validate and explain limits, UTC reset, schedules, and pacing in Chinese and English templates; operator and architecture references are updated. |

## Acceptance Criteria

- `[+]` Exhausted total or daily limits cannot win a new auction.
- `[+]` Campaign/item start/end and daypart windows are enforced at request time,
  and configured pacing cannot exceed the authoritative hard budget.
- `[+]` Pause and budget changes have a documented maximum propagation delay.
- `[+]` Concurrent and Redis/cache failure tests prove the selected availability and
  overspend bounds.

## Verification

- `[+]` Focused `dsp`, `match`, cache, and ledger tests, including concurrency and
  time-boundary cases.
- `[+]` Disposable schema reset/load/check/diff and database-backed Summer
  compatibility tests.
- `[+]` Full Aofei and pzdesign package, vet, pinned staticcheck, race, documentation,
  template, Docker cache-smoke, and diff-hygiene gates.

## Exclusions

- Automatic bid or budget optimization and machine learning remain deferred in
  [docs/defer.md](../docs/defer.md).

## Deep Review

- Made unversioned RAdv payloads unambiguously legacy so an old payload whose
  length is also divisible by the v2 record size cannot be misdecoded.
- Changed malformed pacing and invalid cached balance facts from implicit fast
  delivery to fail-closed policy errors.
- Added monotonic Redis ledger floors and made rejected reservations persist a
  newer floor before returning. Releases now clamp to that floor, including the
  overlapping-reservation/newer-ledger/older-cache interleaving.
- Kept total reservation state persistent. Lost callbacks can conservatively
  underdeliver until explicit reconciliation, but they cannot silently reopen
  a hard budget.
- Kept click-limit enforcement event-driven: once a click is recorded at the
  limit, later bids are rejected. Already served impressions can still click,
  so click totals have an inherent in-flight bound rather than a pre-reserved
  exact bound; pre-reserving every possible click would incorrectly turn a
  click limit into an impression limit.
- Added isolated config/state-path overrides to the local launcher and cache
  smoke so verification does not reset or flush another running local stack.

## Closeout Evidence

- Go 1.23.5 full tests/vet in both repositories; staticcheck v0.5.1 with the
  documented pzdesign legacy-style exclusions.
- Scoped Aofei race suite and pzdesign `cmd/unify` race test.
- D01 concurrency, callback lifecycle, cache compatibility/freshness, ledger,
  and database-backed Summer tests.
- Aofei bid-path benchmarks, documentation/public-copy/template guards, schema
  guard/diff, and both repositories' diff hygiene.
- Disposable Docker MySQL/Redis/NATS cache smoke passed in Redis, spread, and
  combined modes; all disposable resources were removed afterward and the
  pre-existing local stack was left running.
- No commit was created because the active goal's commit policy is `none`.
