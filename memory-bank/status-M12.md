# Status M12 - OpenRTB And Audience Matching Review

Milestone status: `[+]` Completed

Goal: review the OpenRTB bid path and audience matching architecture without
runtime refactors, document current behavior, and move implementation work into
M13.

## Tasks

- `[+]` Review bid request parsing, publisher lookup, candidate filtering,
  creative selection, response generation, and audit publishing.
- `[+]` Review attribute extraction across `advice`, `demo`, `dh`, `maxmind`,
  and `acl`.
- `[+]` Review audience loading, serialization, uploaded audience checks, and
  `Audience.Has` predicates.
- `[+]` Review frequency-cap filtering, bid-floor weighting, response URLs, and
  win/loss/impression/click measurement.
- `[+]` Add documentation:
  - `docs/openrtb-measurement.md`
  - `docs/audience-matching.md`
  - `docs/dsp-workflow.md`
- `[+]` Add evolution entry for the M12/M13 milestone-direction change.
  - Files: `evolution/prompt-v3.md`, `evolution/result-v3.md`.
- `[+]` Run M12 verification.

## Review Findings - 2026-05-12

- `[+]` High: app/web site-type targeting is not populated on request ACLs.
  - Evidence: `acl.NewOpenRTBACL` chooses app vs web defaults but never assigns
    `ACL.SiteType` (`acl/openrtb.go:24`, `acl/openrtb.go:30`), while
    `ACLAudience.Has` rejects nonzero site-type targets when
    `self.SiteTypes != a.SiteType` (`acl/audience.go:65`).
  - Impact: campaigns/items targeting only Web or only App can be incorrectly
    filtered out because runtime request ACLs remain `SiteTypeUnknown`.
  - Recommended fix: set `ACL.SiteType` in `NewOpenRTBACL`, add app/web unit
    tests, and verify sample bid behavior.
  - Disposition: `M13 deferred`.

- `[+]` High: IP geo enrichment only runs when `device.geo` exists.
  - Evidence: `maxmind.NewOpenRTBGeo` enters all geo/IP enrichment under
    `if geo := device.Geo; geo != nil` (`maxmind/openrtb.go:13`), and the
    MaxMind fallback is nested inside that block (`maxmind/openrtb.go:35`).
  - Impact: requests with `device.ip` but no `device.geo` get empty country,
    state, city, DMA, and UTC offset attributes, reducing geo and date/hour
    match accuracy.
  - Recommended fix: attempt IP lookup whenever `device.IP` exists and a
    requested geo field is missing, regardless of whether `device.Geo` exists.
  - Disposition: `M13 deferred`.

- `[+]` High: date/hour targeting treats OpenRTB UTC offset minutes as a local
  enum.
  - Evidence: the OpenRTB dependency documents `geo.utcoffset` as minutes from
    UTC (`openrtb2/geo.go:129` in module
    `github.com/prebid/openrtb/v20@v20.3.0`); this code copies it directly
    (`maxmind/openrtb.go:22`), casts it to `uint8` (`match/attribute.go:85`),
    then `dh.DH.dhw` interprets values 1..13 and 14..24 as a custom hour-offset
    enum (`dh/dh.go:30`).
  - Impact: exchange-provided offsets such as `60` or `-300` do not map to the
    intended visitor local time, so weekday/hour targeting can match the wrong
    bucket.
  - Recommended fix: normalize OpenRTB minute offsets to either real
    `time.Duration` offsets or the existing stored enum at a single boundary,
    and add tests for positive, negative, and zero offsets.
  - Disposition: `M13 deferred`.

- `[+]` Medium: bid processing is first-impression only.
  - Evidence: response `ImpID` reads `bid.Imp[0]` (`dsp/dsp.go:42`), attribute
    extraction uses `Imp[0]` for video/native/size (`match/attribute.go:57`,
    `match/size.go:17`), ACL slot extraction uses `Imp[0].TagID`
    (`acl/openrtb.go:80`), and selection uses `bid.Imp[0].BidFloor`
    (`dsp/controller.go:274`).
  - Impact: multi-impression bid requests are accepted but every impression
    after index 0 is ignored. This is valid only if the product contract is
    intentionally single-impression.
  - Recommended fix: either reject multi-impression requests explicitly or
    implement per-impression matching and response construction.
  - Disposition: `M13 deferred`.

- `[+]` Medium: app/video/native creative rendering branches can choose the
  wrong markup format.
  - Evidence: `Creative.AdM` returns image native markup whenever
    `attr.NativeFormat != nil || attr.IsApp` before checking `attr.IsVideo`
    (`match/creative.go:267`).
  - Impact: app banner or app video impressions can receive default native image
    markup even when the offered impression is not native.
  - Recommended fix: branch by offered impression type first, then apply app/web
    differences inside each format.
  - Disposition: `M13 deferred`.

- `[+]` Medium: uploaded-audience matches suppress otherwise eligible
  non-upload candidates.
  - Evidence: `ServeBid` first appends only candidates whose
    `UploadAudience.Has` succeeds (`dsp/controller.go:237`), and skips the
    broader audience filter when at least one direct upload match exists
    (`dsp/controller.go:258`).
  - Impact: uploaded audiences act as an implicit priority tier, not just a
    predicate. That may be intentional, but it is not encoded as a documented
    campaign ranking rule.
  - Recommended fix: product-scope whether uploaded audiences should be hard
    priority or normal predicates, then make the code and docs match.
  - Disposition: `M13 deferred`.

- `[+]` Medium: bid-floor and currency handling are underspecified.
  - Evidence: `RAdv.GetItemWeight` compares local CPM values directly to
    `bidFloor` while ignoring `bidFoorCur` (`match/radv.go:610`), and responses
    always declare `"USD"` (`dsp/controller.go:316`).
  - Impact: non-USD floors and campaign cost types can be compared incorrectly,
    leading to underbids, over-filtering, or inconsistent spend semantics.
  - Recommended fix: define supported currencies and cost-type math; reject or
    convert unsupported floors before selection.
  - Disposition: `M13 deferred`.

- `[+]` Medium: spread/NATS slot cache refresh appears to publish only one slot
  per size.
  - Evidence: `radvHashToRedisSpreadBySizeID` loops over every slot but calls
    `ToSpread` only when `i == 0` (`match/radv.go:369`,
    `match/radv.go:379`).
  - Impact: spread/local snapshot mode can miss most slot RAdv files after a
    full size refresh, while Redis mode writes all slots.
  - Recommended fix: publish every slot and attach cleanup semantics only to
    the first message or a separate control subject.
  - Disposition: `M13 deferred`.

- `[+]` Medium: Redis and IO audience cache miss behavior differs.
  - Evidence: Redis `HMGET` skips empty audience entries
    (`match/audience.go:333`), but IO mode returns an error if any candidate's
    audience file is missing (`match/audience.go:348`).
  - Impact: local/spread mode can no-bid a slot for a missing audience snapshot
    that Redis mode would treat as no targeting.
  - Recommended fix: make missing IO audience files match Redis semantics or
    document and test the difference.
  - Disposition: `M13 deferred`.

- `[+]` Medium: accepted-bid audit publishing is best effort after the HTTP
  response is sent.
  - Evidence: `ServeBid` writes the response before publishing request,
    response, and attribute events (`dsp/controller.go:331`), and publish errors
    are logged after the response has already been returned (`dsp/controller.go:336`).
  - Impact: bidders can receive `200 OK` while request/response/attribute audit
    events are lost.
  - Recommended fix: no M12 code change. Keep this as the documented contract
    unless a future reliability milestone changes acceptance semantics.
  - Disposition: `M12 documented`.

- `[+]` Medium: ledger inputs count impression and click trackers, not bare win
  or loss callbacks.
  - Evidence: `cmd/ledger.Statistics` increments impressions only for
    `StatusTrackImp` and clicks only for `StatusTrackClk`
    (`cmd/ledger/ledger.go:149`).
  - Impact: spend and delivery accounting require tracker callbacks; a win-only
    exchange callback records a win/loss event but does not become ledger spend.
  - Recommended fix: document the measurement contract and decide in M13 only
    if win callbacks should drive spend.
  - Disposition: `M12 documented`.

- `[+]` Medium: banner iframe responses do not embed DSP impression/click
  trackers.
  - Evidence: `Creative.AdM` builds `impTrackers` and `clickTrackers` for
    native/native-video markup, but the banner branch returns only an iframe
    using creative content (`match/creative.go:246`, `match/creative.go:273`).
  - Impact: banner responses can be accepted without feeding `/imp` and `/clk`
    measurement or frequency-cap refresh through DSP tracker URLs.
  - Recommended fix: product-scope banner measurement expectations, then either
    embed tracker pixels/click wrappers or document banner delivery as externally
    measured.
  - Disposition: `M13 deferred`.

- `[+]` Low: focused tests do not yet cover the riskiest matching seams.
  - Evidence: current DSP tests cover controller options, bid ID packing, and
    local smoke shapes (`dsp/controller_test.go:5`, `dsp/dsp_test.go:9`,
    `dsp/smoke_test.go:23`); package tests largely cover enum/stringer and
    serialization helpers.
  - Impact: app/web ACL, absent-geo IP enrichment, date/hour offsets,
    multi-impression behavior, app video/native branching, uploaded-audience
    priority, and spread cache publishing can regress without focused failures.
  - Recommended fix: add narrow tests with local fixtures before or during M13
    fixes.
  - Disposition: `M13 deferred`.

## Verification

Passed on 2026-05-12:

```bash
GOWORK=off go test ./advice ./demo ./dh ./maxmind ./maxmind/ipsearch ./acl ./match ./dsp
GOWORK=off go test ./...
./scripts/aofei-doc-check.sh
git diff --check
```

Optional Docker/local-config smoke also passed because `etc/aofei.local.json`
exists:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
```

## Result

- M12 completed as a documentation and review-only milestone.
- Current measurement, audience matching, and DSP workflow behavior is recorded
  under `docs/`.
- Implementation work found during review is explicitly deferred to
  `memory-bank/status-M13.md`.
- Evolution V3 records the review-to-refactor milestone handoff.
- Post-review fixes expanded the documentation guard to future status/evolution
  files, corrected the OpenRTB dependency memory, and documented the banner
  tracker gap.
