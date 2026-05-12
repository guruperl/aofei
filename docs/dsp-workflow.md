# DSP Workflow

This document follows the current end-to-end bid workflow from HTTP request to
response and measurement publishing.

## Request Entry

`cmd/unify` wires `POST /bid/{domain}` to `dsp.Controller.ServeBid`. The route
domain is the primary publisher key. The bid body is limited to 1 MiB before
JSON unmarshalling.

The active validation boundary requires:

- a nonnil bid request,
- at least one impression,
- a nonnil device.

Malformed JSON returns `400`. Missing required runtime shape returns no content.

## Publisher And Attribute Resolution

`ServeBid` looks up `acl.Pub` by route domain from either:

- local spread files when `Config.IsLocal` is true, or
- Redis `pubmap` otherwise.

`match.NewAttributeForImp` then derives these values per impression:

- `RPub` ids from publisher site/slot maps,
- creative size and native format from the current impression,
- app/video booleans,
- user ID and IFA,
- demographic, geo, user-agent, date/hour, and ACL attributes.

Unknown site or slot strings fall back to publisher default site/slot ids.

## Candidate Loading

For each impression, the controller loads `match.RAdvs` for the resolved
`size_id` and `slot_id`. Each candidate contains advertiser, campaign, item,
creative, cost, weight, and frequency-cap fields.

If no candidates exist for an impression's size/slot pair, that impression is
skipped. The bid path returns no content only when no impression produces a bid.

## Filtering

Frequency caps are checked first by reading `bothcap:<user_id>` for capped item
ids. Expired cap entries are deleted from Redis.

Audiences are loaded for remaining candidates. The path then:

1. tries uploaded-audience direct matches,
2. falls back to combined audience predicates only when no direct upload match
   exists,
3. skips the impression if no candidates remain.

Combined predicates cover geo, demographic, user-agent, date/hour, and ACL
audiences. Nil audience objects are treated as wildcard matches in Redis and
spread/IO modes.

## Selection

`RAdvs.PickIndexPrice` computes a selection weight for each surviving candidate:

- local cost semantics are converted to USD eCPM,
- candidates below the current impression's `bidfloor` receive weight zero,
- surviving values are multiplied by campaign/item weight,
- a weighted random index is selected.

The response currency is always `USD`. Empty or `USD` `bidfloorcur` is
accepted. Unsupported currencies are not converted and produce no bid for that
impression.

## Creative And Response

The selected creative is loaded from local spread files or Redis `creative`.
`match.Creative.AdM` expands landing, impression, and click tracker URLs and
returns one of:

- default native image markup,
- default native video markup,
- banner iframe markup with DSP impression pixels.

Native is preferred when an impression offers multiple formats, followed by
video, then banner. App inventory no longer forces native markup; app banner
inventory returns banner markup.

`dsp.DSP.NewBid` creates one OpenRTB bid per served impression with:

- bid id derived from request time, creative id, and impression index,
- `impid` from the current impression id,
- price from selected USD eCPM,
- win and loss URLs,
- ad markup,
- campaign and creative ids,
- bundle and categories from the matched ACL audience,
- width and height from creative size.

`ServeBid` groups bids by campaign seat. Audit events are published to NATS
after the response body is written: request and response once, and one
attribute event per served impression.

## Win, Loss, Impression, And Click

`GET /win`, `/loss`, `/imp`, and `/clk` all enter `ServeWinLoss`. The handler
unpacks demand and supply identifiers from query parameters, parses
`auction_price`, builds a `WinLoss` record, and publishes it to NATS when NATS is
available.

For `/imp` and `/clk`, the handler also unpacks cap state and refreshes Redis
frequency caps for the user/item pair.

## Known Workflow Boundaries

- Request, response, and attribute logs are best effort after response write.
- Ledger spend is based on impression tracker records, not win records.
- Redis is the production runtime cache path; spread/local snapshots are a
  development and cache-propagation path that must remain contract-compatible.
