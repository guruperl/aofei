# OpenRTB Measurement

This document records the current DSP measurement flow. It is descriptive, not
a new runtime contract.

## HTTP Flow

`../pzdesign/cmd/unify` registers these DSP endpoints:

| Method | Path | Handler | Purpose |
|---|---|---|---|
| `POST` | `/bid/{domain}` | `dsp.Controller.ServeBid` | Parse an OpenRTB bid request and return a bid or no-bid status. |
| `GET` | `/win` | `dsp.Controller.ServeWinLoss` | Record an exchange win callback. |
| `GET` | `/loss` | `dsp.Controller.ServeWinLoss` | Record an exchange loss callback. |
| `GET` | `/imp` | `dsp.Controller.ServeWinLoss` | Record an impression tracker callback and refresh impression caps. |
| `GET` | `/clk` | `dsp.Controller.ServeWinLoss` | Record a click callback, refresh click caps, and redirect when a valid `redirect` target is present. |
| `GET` | `/mid/win` | `dsp.Controller.ServeMiddlemanCallback` | Record and proxy a middleman win callback. |
| `GET` | `/mid/loss` | `dsp.Controller.ServeMiddlemanCallback` | Record and proxy a middleman loss callback. |
| `GET` | `/mid/bill` | `dsp.Controller.ServeMiddlemanCallback` | Record and proxy a middleman billing callback. |
| `GET` | `/mid/click` | `dsp.Controller.ServeMiddlemanCallback` | Record a cooperative downstream middleman click notification. |
| `POST` | `/action` | `dsp.Controller.ServeAction` | Persist a signed, idempotent analytical conversion/action and apply same-lineage click/view attribution. |

`POST /bid/{domain}` implements the W8M OpenRTB 2.5 profile. An explicit
non-2.5 `x-openrtb-version`, unsupported content type, duplicate/empty
impression ID, non-USD request currency, non-finite/negative floor, or request
without any supported Banner/Video/Native format is rejected before matching.
Standard multi-format impressions remain compatible; external fanout narrows
each impression to the media intent selected during matching.
Request gzip and successful JSON response gzip are negotiated and bounded by
the traffic policy; `204` remains an empty uncompressed no-bid.

External middleman responses must echo the request ID and approved seat, use
USD CPM, meet the raw forwarded floor, arrive before the deadline, and pass the
D02 media/size/secure-markup/callback gate. Fixed reason and latency metrics
are operational evidence only and do not change impression-based billing.

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
them. Tracker prices use the same selected USD CPM value returned in
`Bid.price`. Under `usd-cpm-impression-v2`, the tracker retains CPM while the
reservation and ledger convert one accepted impression to USD spend as
`CPM / 1000`; spend therefore follows the served bid price rather than the raw
item cost field.

For a budget-limited local winner, all generated win/loss/impression/click URLs
also carry a signed opaque `delivery_reservation` token. Response construction
has already converted the selected CPM to one-impression USD spend and reserved
that amount plus one impression atomically across the campaign/ad-group total
and daily scopes. A published impression finalizes
the reservation, a published click applies its click count once, and a
published loss releases an active reservation. Response/materialization
failure also releases it. Measurement publication failure keeps the
reservation active so the same signed callback can retry without incrementing
budget state twice. The token metadata remains available for the full accepted
signature lifetime.

DSP-generated measurement URLs include HMAC `sig` and Unix-second `sig_ts`
parameters when `tracking_secret` or `TRACKING_SECRET` is configured. `sig_ts`
must be present, inside `tracking_signature_ttl_seconds`, and part of the
signature. `/imp` and `/clk` signatures cover the full concrete query payload,
including `redirect` on click URLs. `/win` and `/loss` sign immutable packed
demand/supply fields and `sig_ts` so exchanges can still replace auction macros.
Click redirects, win/loss notifications, and Redis cap mutations require a
valid non-expired signature. Every `/imp` and `/clk` request is validated even
when the served item has no cap or the `cap` query value is missing; unsigned,
expired, or modified URLs return `400`.
Duplicate `/win` and `/loss` notifications for the same auction bid are
short-circuited until the signed timestamp's exact validity deadline when Redis
is available. Signed `/imp` and `/clk` events are also deduplicated independently
until that deadline using a bounded hash of status, auction ID, bid ID, and
impression ID. This preserves the full accepted lifetime of a signature whose
timestamp is within the five-minute future-clock-skew allowance instead of
starting a fresh configured TTL at callback time.

The first event acquires a short owner-token processing claim, performs cap
mutation and log publication, then converts the claim to the deadline-bound
completion marker. A publication failure releases the claim for retry; cap
mutation uses a separate transactional event marker so that retry does not
increment the counter twice. A cap-state failure does not release an owned claim
before publication. Successful publication still finalizes replay suppression.
Duplicates preserve the normal `204` or click redirect response without
repeating either side effect.
Win/loss callbacks use the same owner-token lifecycle without cap mutation. In
particular, a transient win/loss publication failure releases its processing
claim, and a loss keeps the delivery reservation active until a retry publishes
the loss durably. Only that successful publication attempts the idempotent
reservation release; concurrent or completed duplicates publish nothing and do
not touch the reservation.

This path is deliberately fail-open: claim or cap Redis errors do not reject an
otherwise valid `/imp` or `/clk`. Keyed events still attempt the idempotent cap
transaction when claim acquisition fails. Events with incomplete replay
identity publish but skip non-idempotent cap mutation. Tracking Redis work has a
two-second bound detached from HTTP cancellation, so one disconnected request
cannot cancel shared Redis work or return watched/transactional state to the
connection pool.

Valid signed capped events with no user suffix in `auction_bid_id` still enter
the measurement log but skip Redis cap mutation. Cap counters saturate at the
packed `uint8` maximum instead of wrapping. `bothcap:<user_id>` hashes receive
the configured `cap_state_ttl_seconds` idle TTL, default 90 days; refreshes do
not shorten a longer existing TTL.

Replay operations expose `aofei_tracking_replay_suppressed_total`,
`aofei_tracking_replay_fail_open_total`,
`aofei_tracking_replay_redis_errors_total`, and
`aofei_tracking_replay_unkeyed_total` through `/debug/vars`.
`aofei_tracking_cap_update_fail_open_total` counts valid events published
without a successful cap update.

Successful signed `/imp` and `/clk` publication also attempts a bounded,
detached MySQL `measurement_touch` write for R01 attribution. This write is
fail-open relative to the tracker and never changes cap, delivery-reservation,
or billing behavior. Local click landing URLs carry a separate HMAC-protected
`w8m_action_token`; the internal delivery reservation token is never exposed as
an action identity. The action endpoint, exact taxonomy, retry/idempotency
rules, attribution windows, retention, and reporting are specified in
[conversion-attribution.md](conversion-attribution.md).

Current tracker embedding is format-dependent: native and native-video markup
include `/imp` trackers and use `/clk` as a redirecting primary link. Banner
iframe markup embeds `/imp` pixels and only uses `/clk` when creative content
opts in with the `{CLICK_URL}` macro. `/clk` records best-effort click state and
redirects only when the normal tracking fields are present and the `redirect`
target is valid HTTP(S).

Middleman bids use a separate callback proxy instead of generated creative
markup. The selected downstream bid's original callback URLs are stored in Redis
with a TTL, and the upstream response receives signed `/mid/*` URLs. Aofei
forwards downstream `nurl`, `burl`, and `lurl` when present, replacing
`${AUCTION_PRICE}` with the downstream payable price. `burl` is the preferred
billable event; when the selected downstream bid has no `burl`, `/mid/win`
becomes the billable fallback. Cooperative click notification is exposed in the
forwarded request as `ext.aofei_middleman.click_notify_urls`; downstream ad
markup must opt in, because Aofei does not rewrite arbitrary middleman `adm`.

Downstream forwarding state and local NATS publication state use separate
idempotency markers. A retry after a local publish failure therefore reuses the
recorded downstream result without sending the downstream callback again, then
retries only the unpublished local fact. `/mid/click` uses the same local
publication guard. An in-flight duplicate does nothing until its owner
finishes, and a completed duplicate performs neither side effect. If a
downstream callback succeeds but writing its completed forwarding state fails,
the marker is cleared and the request returns an error; a retry may forward the
callback again. This narrow post-forward persistence boundary is intentionally
at-least-once, so downstream endpoints must be idempotent by auction and
impression identity.

## NATS And Log Flow

The bid path publishes these NATS subjects after a successful HTTP response:

| Subject | Payload | Consumer |
|---|---|---|
| `request` | Privacy-scrubbed ADX OpenRTB body; SSP JSON envelope with `source:"ssp"`, `contract:"pz-v1"`, privacy evidence, and scrubbed request `payload` | `cmd/nats-client` writes `log_request/request.<stamp>`. |
| `response` | Privacy-scrubbed ADX response; SSP envelope with the selected response payload | `cmd/nats-client` writes `log_response/response.<stamp>`. |
| `attribute` | Identity/precise-data-redacted `match.AttributePlus`, one per served impression, with `source`, `contract`, `privacy_mode`, and `privacy_reason` | `cmd/nats-client` writes `log_attribute/attribute.<stamp>`. |
| `winloss` | `dsp.WinLoss` JSON | `cmd/nats-client` writes `log_winloss/winloss.<stamp>`. |

Audit publish is best effort after the bid response is sent. Request, response,
and attribute audit messages are enqueued to a bounded in-process queue and
published by a background worker without flushing in the HTTP request goroutine.
If NATS is missing, the queue is full, or publish fails, the accepted bidder
response is not rolled back.

ADX `/bid` request and response audits retain the OpenRTB JSON shape but no
longer retain raw identity, precise device/location data, consent strings, or
uncontrolled extensions. Direct SSP `/pz` request and response audits are
wrapped so operators can distinguish the entrypoint. Attribute audits use
`source:"adx", contract:"openrtb"` for `/bid` and
`source:"ssp", contract:"pz-v1"` for `/pz`; their `elapsed` field is an integer
millisecond count. `cmd/nats-client` removes generated interval files older than
`privacy_log_retention_hours` at startup and rotation. See
[privacy-data-governance.md](privacy-data-governance.md).

`cmd/spread` ignores the four log subjects and handles only cache/spread
subjects.

## Ledger Inputs

`cmd/ledger` reads `log_winloss/winloss.<stamp>`. The legacy ledger tables
aggregate only:

- `StatusTrackImp` as impressions and spend.
- `StatusTrackClk` as clicks.

Bare `StatusWin` and `StatusLoss` events are written to the win/loss log when
callbacks arrive, but current ledger statistics do not count them as delivery or
spend. Win/loss callbacks are analytics only; unresolved or unsigned auction
price macros are not spend authority.

Middleman `/mid/bill` publishes a `StatusTrackImp` event with charge-side CPM in
`RAdv.Cost`; the ledger converts it to one-impression USD spend while counting
delivery through the synthetic campaign/item/creative chain. If no downstream `burl` exists,
`/mid/win` publishes the billable `StatusTrackImp` fallback once. Middleman
winloss records also include optional metadata for downstream bid price,
upstream bid price, charge price, pay price, margin, callback source, and
downstream forward status.

M22 reporting consumes `WinLoss.Middleman` into `ledger_mid` and `daily_mid`.
For middleman rows, `StatusTrackImp` contributes billable impressions,
charge/pay/margin spend, `StatusTrackClk` contributes clicks, and
`StatusWin`/`StatusLoss` contribute admin audit counts without becoming spend.
Advertiser-facing middleman spend is pay-side spend; admin reports expose
charge, pay, and margin.

Direct SSP markup reuses the same local `NewBid` rendering path as ADX local
bids. Its signed `/imp` and `/clk` tracker URLs publish normal `WinLoss` records
with direct publisher/site/slot IDs and demand IDs, so `cmd/ledger` aggregates
SSP impressions, clicks, and spend through the existing ledger schema.
Interval ledger completion reconciles campaign/ad-group total delivery
baselines from all `ledger_adv` facts and current-UTC-date daily baselines from
interval facts. Daily aggregation corrects the daily baseline from `daily_adv`
and records its date in `adv_balance.current_day`. Redis request-time state is
seeded monotonically from those facts so a delayed report cannot reopen
already-reserved budget.

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
