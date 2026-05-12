# OpenRTB Measurement

This document records the current DSP measurement flow. It is descriptive, not
a new runtime contract.

## HTTP Flow

`cmd/unify` registers these DSP endpoints:

| Method | Path | Handler | Purpose |
|---|---|---|---|
| `POST` | `/bid/{domain}` | `dsp.Controller.ServeBid` | Parse an OpenRTB bid request and return a bid or no-bid status. |
| `GET` | `/win` | `dsp.Controller.ServeWinLoss` | Record an exchange win callback. |
| `GET` | `/loss` | `dsp.Controller.ServeWinLoss` | Record an exchange loss callback. |
| `GET` | `/imp` | `dsp.Controller.ServeWinLoss` | Record an impression tracker callback and refresh impression caps. |
| `GET` | `/clk` | `dsp.Controller.ServeWinLoss` | Record a click tracker callback and refresh click caps. |

`ServeBid` reads and limits the request body, unmarshals
`openrtb2.BidRequest`, validates that at least one impression and a device are
present, resolves the publisher from the route domain, builds runtime
attributes per impression, filters candidates, chooses creatives for the served
impressions, writes the OpenRTB response, then publishes audit events if NATS is
available.

## Response Measurement Fields

The response contains one bid for each impression that can be matched. Bids are
grouped into `SeatBid` entries by campaign seat. `dsp.WinLoss` builds:

- `nurl`: `/win` with OpenRTB auction macros for the exchange to replace.
- `lurl`: `/loss` with OpenRTB auction macros for the exchange to replace.
- impression tracker URL: `/imp` embedded in native markup and banner
  impression pixels.
- click tracker URL: `/clk` embedded in native markup.

Tracker URLs are generated with concrete auction values and cap state. Win and
loss URLs intentionally use standard auction macros until the exchange resolves
them. Tracker prices use the same selected USD eCPM value returned in
`Bid.price`, so ledger spend follows the served bid price rather than the raw
item cost field.

Current tracker embedding is format-dependent: native and native-video markup
include `/imp` and `/clk` tracker URLs, while banner iframe markup embeds
`/imp` pixels only and does not rewrite click behavior.

## NATS And Log Flow

The bid path publishes these NATS subjects after a successful HTTP response:

| Subject | Payload | Consumer |
|---|---|---|
| `request` | Raw request body | `cmd/nats-client` writes `log_request/request.<stamp>`. |
| `response` | Raw response body | `cmd/nats-client` writes `log_response/response.<stamp>`. |
| `attribute` | `match.AttributePlus` JSON, one per served impression | `cmd/nats-client` writes `log_attribute/attribute.<stamp>`. |
| `winloss` | `dsp.WinLoss` JSON | `cmd/nats-client` writes `log_winloss/winloss.<stamp>`. |

Audit publish is best effort after the bid response is sent. If NATS is missing
or publish fails, the accepted bidder response is not rolled back.

`cmd/spread` ignores the four log subjects and handles only cache/spread
subjects.

## Ledger Inputs

`cmd/ledger` reads `log_winloss/winloss.<stamp>`. It aggregates only:

- `StatusTrackImp` as impressions and spend.
- `StatusTrackClk` as clicks.

Bare `StatusWin` and `StatusLoss` events are written to the win/loss log when
callbacks arrive, but current ledger statistics do not count them as delivery or
spend.

## Known Measurement Gaps

- Accepted bids can miss request/response/attribute audit logs if NATS fails
  after the HTTP response is sent.
- Ledger spend depends on the impression tracker firing; a win callback alone
  is not billable in the current aggregation code.
- Cap refresh runs on `/imp` and `/clk`, not on `/win` or `/loss`.
- Banner iframe responses embed DSP impression pixels but do not embed or
  rewrite click tracking. Banner click measurement still requires creative or
  landing-flow integration.
- Multi-impression requests can produce partial responses. Impressions skipped
  for targeting, unsupported currency, or missing cache entries have no bid and
  no attribute audit record.
- Unsupported or unresolved `auction_price` values cause win/loss status
  handling to return a bad request.
