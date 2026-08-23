# Status D05 - Post-D04 Auction Compatibility And Hot-Path Remediation

State: `[ ]` Planned

## Goal

Resolve the confirmed post-D04 frequency-cap, OpenRTB compatibility, and bid
hot-path findings without weakening signed-callback validation, the documented
at-least-once publication boundary, cache-generation failure visibility, or
shared impression/click/throttle cap state.

## Dependencies

- D01 delivery limits, D02 creative/media validation, and completed D04
  tracking/cap wire contracts remain authoritative.
- Version-2 `BothCap` UTC trailers, signed callback shapes, and public WinLoss
  URL method signatures remain compatible.
- D05 is repository-only work and does not depend on S06 Cloudflare activation.
  Complete it before P03 begins.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Legacy cap migration | `[+]` | `Fcap.legacyStartIn`/`legacyStartLocal` now interpret legacy-only calendar fields in the worker's `time.Local` (the pre-D04 reader's wall-clock location) before `ensureUTC` freezes the absolute instant into UTC epoch minutes. Version-2 epoch-minute trailers remain authoritative. Added `TestLegacyBothCapReadsUseLocalWallClockAndUpgradeToUTC` (non-UTC, restores `time.Local`) and `TestLegacyStartInIsLocationAwareWithoutGlobalTimezone` (fixed-zone, no global mutation). `docs/multiple-cache.md` now records the same-host-timezone rolling requirement until touched legacy state upgrades. |
| Invalid-cap isolation and tracker hardening | `[+]` | `RAdvs.FilterByCaps` now partitions candidates first, excludes only the invalid ones (counted by the fixed-cardinality `aofei_invalid_cap_candidates_total` expvar), and skips their Redis reads while leaving valid candidates in the same slot untouched. DB cache compilation (`dbRAdvsBySizeIDSlotID` → `updateRow`) retains fail-fast with the exact item ID and no partial generation. `WinLoss.packURLValues`/`trackerURLWithError` are now error-aware: an invalid cap yields no signed tracker (`ImpURL`/`PackURLString` return `""`) and `DSP.NewBid` aborts materialization. Added `TestInvalidCapIsolatedFromValidCandidates` and `TestWinLossTrackerRejectsInvalidCap`. |
| OpenRTB dimension compatibility | `[+]` | `NativeFormat.validatedSizes` now treats Native video `0x0` as omitted, lets an incomplete preferred image pair fall back to a complete `wmin/hmin` via `nativeImageSize`, and still rejects negative/oversized values and incomplete minimum pairs. `getSizeIDNativeForImp` uses the first exact Banner `format` entry when scalar `w/h` is `0x0` (skipping ratio-only/oversized entries) and returns a compatible no-size result for video `0x0`. Updated `size_test.go` with fallback/omitted/rejection cases and `TestBannerZeroUsesFirstExactFormat`. |
| Audience hot-path cleanup | `[+]` | Removed the five synchronous `log.Printf` calls in `Audience.Has` for expected geo/demo/ua/dh/acl mismatches, along with the now-unused `log` import. Nil-safety and match results are unchanged, and no per-mismatch metric or serialized side effect was added. |
| Macro replacement plan | `[ ]` | Build the deduplicated longest-key-first macro replacement slice once per `applyMacro` call and reuse it for every query value. Preserve standard-over-custom precedence, repeated query values, and deterministic output; add allocation/latency evidence. |

## Finding Disposition (2026-08-23)

The source review is `review2.md`, an untracked scratch input. This table is the
durable disposition; implementation must not depend on retaining that file.

| Review2 item | Disposition | Owning task |
|---:|---|---|
| 2 | Reject as a defect. Publish followed by uncertain completion can replay, but D04 deliberately retains an observable at-least-once boundary. Exactly-once needs a separately approved outbox/idempotent-consumer design. | Exclusion |
| 3 | Partially confirmed P3. The packing error is discarded, but the claimed normal cache-to-cap-bypass path is stopped by upstream validation. Harden materialization defensively. | Invalid-cap isolation and tracker hardening |
| 4 | Confirmed P2 legacy compatibility defect on non-UTC deployments. | Legacy cap migration |
| 5 | Confirmed P2 runtime blast radius. Isolate the invalid candidate, but retain fail-fast cache compilation instead of silently publishing a partial generation. | Invalid-cap isolation and tracker hardening |
| 6 | Confirmed P2 Native compatibility regression for omitted dimensions. | OpenRTB dimension compatibility |
| 7 | Confirmed P2 Banner compatibility regression for `0x0` plus exact `format`. | OpenRTB dimension compatibility |
| 8 | Reject as a defect. Malformed signed click state deliberately returns `400` without redirect; only validated callbacks with retryable local dependency failures preserve the landing redirect. | Exclusion |
| 12 | Reject as a correctness defect. Bid-time deletion was intentionally removed because one shared item field can contain a live sibling click window or throttle; whole-hash idle TTL remains the documented lifecycle. | Exclusion |
| 13 | Confirmed P2 bid-path logging/lock-contention defect. | Audience hot-path cleanup |
| 15 | Confirmed P3 repeated allocation/sort defect. | Macro replacement plan |

## Acceptance Criteria

- Legacy-only cap state preserves the instant the old reader would observe in
  UTC and representative positive/negative-offset deployment timezones; new
  version-2 writes remain timezone-independent.
- One invalid RAdv cannot suppress valid candidates in the same slot, cannot be
  treated as uncapped, and cannot produce a signed tracker. Cache compilation
  still reports and blocks an invalid source generation.
- Missing optional Native dimensions and Banner `0x0` with a valid exact
  `format` follow compatible fallback/no-bid behavior; malformed explicit
  dimensions still fail before cache lookup.
- Expected audience mismatches do not write process logs, and macro expansion
  retains deterministic byte-for-byte output with fewer allocations.
- Public OpenRTB response fields, signed callback query shapes, Redis key/wire
  formats, and WinLoss URL method signatures remain unchanged.

## Verification

- Focused legacy timezone/rolling-format, invalid mixed-candidate, no-Redis,
  tracker-materialization, Native/Banner format, audience, and macro tests.
- Benchmarks for audience mismatch and multi-value macro expansion before and
  after the hot-path changes.
- `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`, pinned staticcheck,
  the documented Aofei race suite, DSP/cache/documentation guards, and
  `git diff --check`.
- After implementation and automated verification, run the bounded complete
  D05 review-fix gate. Iteration 1 has not started.

## Exclusions

- No exactly-once broker/outbox redesign and no reversal of D04's accepted
  at-least-once callback boundary.
- No redirect after malformed signed click data.
- No bid-time deletion of a complete shared `BothCap` item field and no new
  per-field retention/storage redesign.
- No Cloudflare, production deployment, schema, or external-service mutation.
