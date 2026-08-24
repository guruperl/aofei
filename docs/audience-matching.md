# Audience Matching

This document describes the current runtime matching model across attribute
extraction, audience cache data, predicates, and known correctness risks.

## Attribute Extraction

`match.NewAttribute` requires a nonnil bid request, at least one impression, a
device, and a publisher cache object. It is the compatibility wrapper for
`imp[0]`; the bid path uses `match.NewAttributeForImp` to build one
`match.Attribute` per impression:

| Package | Runtime role |
|---|---|
| `advice` | Extract device type, make/platform, OS, and OS version from OpenRTB device fields and user-agent fallback. |
| `demo` | Extract gender, year-of-birth bucket, and language from `user`, `wlang`, and device language fields. |
| `dh` | Convert request time plus OpenRTB geo UTC-offset minutes into day, hour, and weekday predicates. Stored audience UTC-offset values keep the existing enum contract. |
| `maxmind` | Copy OpenRTB geo fields and enrich missing geo fields from configured IP search data, including IP fallback when `device.geo` is absent. |
| `acl` | Extract publisher, web/app site type, site/app, per-impression slot, category, and blocklist attributes. |
| `match` | Combine all attributes with resolved `RPub`, size ID, native format, user ID, IFA, and app/video flags. |

Before extraction, S01 removes identity and sensitive fields from contextual or
restricted requests. For a configured personalized grant, the transient local
identity precedence remains `user.buyeruid`, `user.id`, then IFA/device IDs,
but the resulting IFA and cap user key are domain-separated HMAC pseudonyms.
IP plus user-agent is no longer a runtime identity fallback.
For direct SSP, an unauthenticated SDK compatibility request cannot reach this
personalized path even when it supplies an otherwise valid consent string. A
valid P03 publisher/App proof plus the independent S01 grant is required before
SDK body identity, publisher-asserted IP/coarse geo, demographics, or uploaded-
audience values survive to matching. Exact coordinates are always removed.

## Audience Sources

`match.DBGetAudience` builds a runtime `Audience` per active item. It combines:

- `acl.ACLAudience` from advertiser, campaign, item, publisher/site/app, and
  category access-control tables.
- `dh.DHAudience` from `fullday`, `fullhour`, `weekday`, and `utcoffset`
  targeting values.
- `maxmind.GeoAudience` from country, state, DMA, city, ISP, zip, lon, lat, and
  connection-type targeting.
- `demo.DemoAudience` from gender, age/year bucket, and language targeting.
- `advice.UaAudience` from OS, OS version, platform/browser, and device
  targeting.
- `uploaded.UploadAudience` from uploaded user/device identifier sets in Redis.

Uploaded audience keys use only canonical markers `buyeruid`, `userid`, `ip`,
`ifa`, `did`, `dpid`, and `mac`. Writers normalize case/whitespace plus the
historical `buyerid` and `user` aliases; readers and scoped deletion retain a
bounded alias fallback so older TTL-bound sets remain usable and removable.

Nil or empty subaudiences generally mean wildcard targeting. Uploaded audiences
are different: when upload targeting is configured, every configured identifier
type must be present in the bid and present in the advertiser upload set.
The combined matcher rejects a nil request attribute, and an ACL subaudience
rejects a missing request ACL rather than dereferencing it. Empty, nonnil
audiences remain wildcards for valid attributes. Missing request demographics
are treated as unknown: they match an empty demographic audience, but fail any
configured gender, age, or language constraint.

Middleman bidder fallback reuses this same ACL and channel eligibility surface
through each bidder's synthetic item. Admin approval creates or validates that
synthetic chain while leaving those rows inactive. The Redis middleman route
cache chooses the coarse fanout pool, and the synthetic advertiser/campaign/item
chain decides whether the original publisher/site request is allowed for a
specific bidder.

## Cache Contracts

Runtime matching reads these Redis families:

| Family | Shape | Producer |
|---|---|---|
| `pubmap` | Hash keyed by publisher domain, gob-encoded `acl.Pub`. | `cmd/redis-cache` or Summer cache side effects. |
| `slot:<size_id>` | Hash keyed by slot id, versioned binary `match.RAdvs`. | `cmd/redis-cache` from active creatives and slots. |
| `audience` | Hash keyed by item id, versioned gob `match.Audience`. | `cmd/redis-cache` from active item targeting. |
| `creative` | Hash keyed by creative id, versioned gob `match.Creative`. | `cmd/redis-cache` from active creatives. |
| `middleman:routes:v2` | Preferred M25 JSON route/bidder cache with trigger mode and synthetic ACL payloads. | `cmd/redis-cache -cache=redis` from active `adv_bidder` and `mid_route_*` rows. |
| `middleman:routes` | Legacy fallback-only JSON route/bidder cache for rolling deploys. | Written by the same cache job. |
| `bothcap:<user_id>` | Hash keyed by item id, binary `match.BothCap`. | Tracker callbacks on `/imp` and `/clk`. |
| `upload:<adv_id>:<marker>` | Redis set of advertiser-provided identifier values with bounded idle retention. | Upload/admin flows; membership and conditional TTL commit in one script. |

Spread/local snapshot mode mirrors the same static data under `.local/spread/`:

- `pubmap/<domain>`
- `slot/<size_id>/<slot_id>`
- `audience/<item_id>`
- `creative/<creative_id>`

Middleman route cache is Redis-only in M20. It is not mirrored into spread
snapshots, so `cmd/unify` nodes that enable middleman fallback need Redis
available even when local/spread static campaign cache is enabled.

When DSP local mode is enabled, these files are loaded into an in-process static
cache at controller startup and refreshed through the bounded background reload
loop; an explicit reload hook remains available for controlled use.
Request handlers read the current immutable in-memory snapshot and do not stat
or walk spread files. Mutable frequency caps and uploaded audience sets remain
Redis-backed. Local static bids that do not use caps or uploaded audiences can
proceed without Redis; bids that need those mutable families fail closed when
Redis is unavailable.

Uploaded matching runs only for a personalized decision. Sets default to 30
days from the last write; `privacy_audience_ttl_seconds` controls the value.
Scoped helpers delete one authorized identifier or an entire advertiser/marker
set without listing its contents.

Frequency-cap callbacks keep the existing `bothcap:<user_id>` hash shape and
use the versioned binary `match.BothCap` payload. Version-2 values retain a
legacy-readable prefix and add authoritative UTC epoch-minute start/last data;
new readers also accept legacy-only values. `/imp` and `/clk` cap mutations require valid
DSP tracking signatures and refresh Redis through `WATCH`/`MULTI`/`EXEC` with
bounded retry to avoid lost updates under concurrent callbacks.

Redis and IO modes treat missing audience entries as wildcard matches. Malformed
audience payloads still fail matching because they indicate cache corruption.

## Matching Order

The current bid path applies filters in this order:

1. Resolve publisher/site/slot and creative size for each impression.
2. Load `RAdvs` for `size_id` and `slot_id`.
3. Filter candidates by frequency caps.
4. Load audiences for the remaining candidates.
5. First try uploaded-audience direct matches.
6. If no uploaded direct match exists, run combined audience predicates.
7. Pick a candidate using bid-floor/cost/weight math.
8. Load the creative and render response markup.
9. Optionally run middleman routing from `middleman:routes:v2`, filtering each
   bidder through trigger mode and synthetic ACL/channel eligibility before any
   downstream OpenRTB call.

Banner, video, and Native width/height values must be positive and fit the
existing 16-bit size-key contract (`1..65535`). Malformed dimensions are
rejected before any wrapped/truncated cache key can be queried.

`Audience.Has` evaluates geo, demo, user-agent, date/hour, and ACL predicates.
If a candidate has no audience object, both Redis and spread/IO modes treat it
as matching.

## Known Correctness Risks

- Uploaded-audience direct matches intentionally form a priority tier. If any
  uploaded match exists, otherwise eligible non-upload candidates are not
  considered for that impression.
- Currency conversion is not implemented. Empty or `USD` floors are accepted;
  unsupported currencies produce no bid for that impression.
- `dh` visitor-local offsets are OpenRTB minutes, while stored advertiser
  timezone overrides remain the legacy enum values shown in the admin target
  list.
