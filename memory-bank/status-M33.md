# Status M33 - SSP Middleman Fallback

State: `[+]` Completed

Let valid direct SSP `/pz` auctions use existing middleman fallback and gated
`Always` behavior after local matching while preserving M32 response formats.

## Tasks

- `[+]` Share the local auction and middleman winner materialization path
  between ADX `/bid` and direct SSP `/pz`.
  - Local no-bid impressions become `Fallback` candidates.
  - Local filled impressions become `Always` candidates only when both
    `middleman_enabled` and `middleman_always_enabled` are true.
  - Callback proxy setup remains the materialization gate, with local fallback
    when callback setup fails and a local winner exists.
- `[+]` Preserve SSP response formats for middleman winners.
  - `"html"` remains an ordered `[]string`.
  - `"json"` remains ordered fill/no-fill objects; middleman fills omit
    local-only `impressionUrl` and `clickUrl`.
  - `"openrtb"` remains a standard `BidResponse`, grouped by final winner seat,
    with empty `seatbid` on all-no-fill.
- `[+]` Keep invalid SSP traffic out of middleman fanout.
  - Malformed JSON, invalid tokens, unsupported media, and browser policy
    rejections return before middleman runtime invocation.
- `[+]` Update direct SSP docs and memory-bank product, architecture, tech
  stack, and milestone notes.
- `[+]` Run closeout verification.
- `[+]` Mark milestone complete after verification.

## Acceptance

- `[+]` Valid local no-fill `/pz` requests can invoke middleman fallback and
  render middleman winners in `html`, `json`, and `openrtb`.
- `[+]` Gated `Always` middleman bids can beat local winners only when
  `middleman_always_enabled` is true and the marked-up bid is higher.
- `[+]` Lower middleman bids and disabled gates keep local winners.
- `[+]` Callback proxy setup failure falls back to local winners where
  available.
- `[+]` Malformed, invalid-token, unsupported-media, and browser-policy rejected
  requests do not invoke middleman.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./acl ./match ./internal/jobs/ledger ./internal/jobs/cache -count=1`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Notes

- 2026-05-13 review resolution: SSP `responseFormat:"openrtb"` now sets
  `BidResponse.bidid` from the materialized winner's auction bid id while
  leaving `SeatBid.Bid.ID` as the concrete response-bid id; local and middleman
  OpenRTB tests cover the distinction.
- No schema drift checks are required for M33 because this milestone does not
  intentionally change schema.
- No new evolution version was added; M33 implements the already recorded SSP
  middleman direction without changing schema, cache shape, account roles, or
  the public M32 response formats.
