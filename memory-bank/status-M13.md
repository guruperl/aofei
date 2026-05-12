# Status M13 - OpenRTB And DSP Refactor Backlog

Milestone status: `[X]` Complete

Goal: resolve the implementation and design backlog found during M12.

## Deferred Work From M12

- `[+]` Fix app/web site-type targeting in runtime ACL extraction.
  - Source finding: M12 high severity ACL site-type gap.
  - Result: `acl.NewOpenRTBACLForImp` sets `ACL.SiteType` to web/app and uses
    per-impression `tagid` for slot extraction; `NewOpenRTBACL` remains the
    `imp[0]` compatibility wrapper.
  - Tests: `acl/openrtb_test.go`.

- `[+]` Fix IP geo enrichment when `device.geo` is absent.
  - Source finding: M12 high severity MaxMind fallback gap.
  - Result: `maxmind.NewOpenRTBGeo` now performs IP fallback independently of
    `device.geo` and preserves request-provided fields before filling gaps.
    `CreatePzGeo` also records timezone offset minutes from MaxMind timezone
    names when available.
  - Tests: `maxmind/openrtb_test.go`.

- `[+]` Normalize OpenRTB UTC offsets before date/hour matching.
  - Source finding: M12 high severity UTC offset unit mismatch.
  - Result: `dh.NewDHFromMinutes` handles OpenRTB minute offsets for visitor
    local time; stored audience `utcoffset` values remain the existing enum and
    still override visitor local time when configured.
  - Tests: `dh/dh_test.go`.

- `[+]` Decide and implement the multi-impression contract.
  - Source finding: M12 first-impression-only behavior.
  - Decision: support all impressions.
  - Result: `ServeBid` loops over impressions, produces bids for each matched
    impression, groups bids by campaign seat, and returns no content only when
    no impression can bid.
  - Tests: `dsp/m13_test.go`.

- `[+]` Correct app, banner, native, and video creative markup selection.
  - Source finding: M12 creative rendering branch gap.
  - Decision: format precedence is native, then video, then banner.
  - Result: size/format resolution and creative markup branch on impression
    format, not app/web context. App banner returns banner iframe markup.
    Banner markup embeds impression pixels but does not rewrite click tracking.
  - Tests: `match/attribute_m13_test.go`, `match/creative_m13_test.go`.

- `[+]` Product-scope uploaded-audience candidate priority.
  - Source finding: M12 uploaded-audience priority gap.
  - Decision: uploaded-audience direct matches remain a priority tier.
  - Result: `ServeBid` keeps uploaded matches ahead of otherwise eligible
    candidates; only when no uploaded match exists does it run combined
    audience predicates.

- `[+]` Define and enforce bid-floor, cost-type, and currency semantics.
  - Source finding: M12 bid-floor/currency gap.
  - Decision: USD eCPM only; empty `bidfloorcur` is treated as USD and
    unsupported currencies produce no bid for that impression.
  - Result: selection uses `RAdvs.PickIndexPrice`; CPM uses `cost`, CPC and ROI
    use `100*cost`, CPA uses `0.01*cost`, and all-zero weights return no match.
  - Tests: `match/radv_test.go`, `dsp/m13_test.go`.

- `[+]` Fix spread/NATS RAdv slot publishing.
  - Source finding: M12 spread cache publishing gap.
  - Result: NATS spread refresh publishes every nonempty slot for a refreshed
    size and applies cleanup only to the first published slot.
  - Tests: `cmd/spread/main_test.go`.

- `[+]` Align Redis and IO audience cache miss semantics.
  - Source finding: M12 audience cache miss gap.
  - Decision: missing audience means wildcard; malformed data remains an error.
  - Result: `RAdvs.AudiencesFromIO` leaves missing audience files as nil
    wildcard matches, matching Redis `HMGET` miss semantics.
  - Tests: `match/audience_m13_test.go`.

- `[+]` Add focused tests for measurement edge cases.
  - Source finding: M12 measurement documentation gaps.
  - Result: M13 adds focused coverage for multi-impression response IDs,
    NATS-disabled bid path behavior, unsupported currency partial fill, bad
    win/loss price parsing, and banner impression tracker markup. Existing
    tests continue to cover cap refresh mechanics and ledger
    tracker-only aggregation behavior.

## Verification

- `[+]` `GOWORK=off go test ./acl ./maxmind ./dh ./match ./dsp ./cmd/spread`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'`

## Post-Review Fixes

- `[+]` Align response price, tracker auction price, and ledger spend input.
  `DSP.WinLoss` now carries the selected USD eCPM into the win/loss record
  instead of using raw item cost for tracker URLs.
- `[+]` Avoid panics on empty native impressions with valid fallback formats.
  Native format selection now falls through to video/banner when native request
  parsing produces no usable size.
- `[+]` Avoid forcing MaxMind lookup solely because UTC offset is zero.
  Zero remains a valid visitor offset; IP fallback is triggered by missing geo
  IDs instead.
