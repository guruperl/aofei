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
| `GET` | `/clk` | `dsp.Controller.ServeWinLoss` | Record a click callback, refresh click caps, and redirect when a valid `redirect` target is present. |

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
- click redirect URL: `/clk?...&redirect=<advertiser-url>` embedded as the
  primary native click link and available to banner creatives through
  `{CLICK_URL}`.

Tracker URLs are generated with concrete auction values and cap state. Win and
loss URLs intentionally use standard auction macros until the exchange resolves
them. Tracker prices use the same selected USD eCPM value returned in
`Bid.price`, so ledger spend follows the served bid price rather than the raw
item cost field.

DSP-generated measurement URLs include an HMAC `sig` when `tracking_secret` or
`TRACKING_SECRET` is configured. `/imp` and `/clk` signatures cover the full
concrete query payload, including `redirect` on click URLs. `/win` and `/loss`
sign only immutable packed demand/supply fields so exchanges can still replace
auction macros. Click redirects and Redis cap mutations require a valid
signature; unsigned or modified redirect URLs return `400`.

Current tracker embedding is format-dependent: native and native-video markup
include `/imp` trackers and use `/clk` as a redirecting primary link. Banner
iframe markup embeds `/imp` pixels and only uses `/clk` when creative content
opts in with the `{CLICK_URL}` macro. `/clk` records best-effort click state and
redirects only when the normal tracking fields are present and the `redirect`
target is valid HTTP(S).

## NATS And Log Flow

The bid path publishes these NATS subjects after a successful HTTP response:

| Subject | Payload | Consumer |
|---|---|---|
| `request` | Raw request body | `cmd/nats-client` writes `log_request/request.<stamp>`. |
| `response` | Raw response body | `cmd/nats-client` writes `log_response/response.<stamp>`. |
| `attribute` | `match.AttributePlus` JSON, one per served impression | `cmd/nats-client` writes `log_attribute/attribute.<stamp>`. |
| `winloss` | `dsp.WinLoss` JSON | `cmd/nats-client` writes `log_winloss/winloss.<stamp>`. |

Audit publish is best effort after the bid response is sent. Request, response,
and attribute audit messages are enqueued to a bounded in-process queue and
published by a background worker without flushing in the HTTP request goroutine.
If NATS is missing, the queue is full, or publish fails, the accepted bidder
response is not rolled back.

`cmd/spread` ignores the four log subjects and handles only cache/spread
subjects.

## Ledger Inputs

`cmd/ledger` reads `log_winloss/winloss.<stamp>`. It aggregates only:

- `StatusTrackImp` as impressions and spend.
- `StatusTrackClk` as clicks.

Bare `StatusWin` and `StatusLoss` events are written to the win/loss log when
callbacks arrive, but current ledger statistics do not count them as delivery or
spend. Win/loss callbacks are analytics only; unresolved or unsigned auction
price macros are not spend authority.

## Known Measurement Gaps

- Accepted bids can miss request/response/attribute audit logs if NATS fails
  after the HTTP response is sent.
- Ledger spend depends on the impression tracker firing; a win callback alone
  is not billable in the current aggregation code.
- Cap refresh runs on `/imp` and `/clk`, not on `/win` or `/loss`.
- Banner iframe responses embed DSP impression pixels. Banner click redirect
  measurement requires the creative content URL/template to opt in with
  `{CLICK_URL}`; arbitrary iframe contents are not wrapped or rewritten.
- Multi-impression requests can produce partial responses. Impressions skipped
  for targeting, unsupported currency, or missing cache entries have no bid and
  no attribute audit record.
- Unsupported or unresolved `auction_price` values still cause `/imp` and
  `/clk` tracking status handling to return a bad request because those paths
  can mutate caps and feed ledger delivery. `/win` and `/loss` remain analytics
  callbacks.
