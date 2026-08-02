# Conversion, Action, And Attribution Measurement

R01 adds a server-to-server analytical action contract for conversions,
purchases, downloads, completed video views, and namespaced custom actions.
Action facts support advertiser reporting and attribution only. They are not a
source for CPM billing, balance changes, delivery-budget use, settlement, or a
D01 reservation transition.

## Lineage Token

For a local creative with an advertiser landing URL, W8M adds a
`w8m_action_token` query parameter to the landing URL inside the signed `/clk`
redirect. The HMAC-protected token contains only its version, issuance and
expiry times, random token id, auction/bid/impression ids, and the selected
advertiser/campaign/ad-group/creative/publisher/site/slot ids. It never contains
the Redis delivery-reservation token, a raw user/device identifier, consent
text, IP address, user agent, or an accounting identity.

The advertiser should retain the token with its own order/action workflow and
send the resulting action from its backend. The token proves W8M delivery
lineage; the request MAC binds the exact callback bytes to that token. It is not
a replacement for advertiser authentication, fraud controls, or authorization
on the advertiser's own conversion workflow. Do not expose the token in logs,
support tickets, analytics dimensions, or referrer forwarding.

Tokens default to the 30-day click window. A token is accepted from its issued
time, including the configured request clock skew, until the exact exclusive
expiry time. Rotating `tracking_secret` invalidates outstanding tokens.

## HTTP Contract

Send `POST /action` with `Content-Type: application/json`. The request body is
limited to 64 KiB and has this versioned shape:

```json
{
  "version": 1,
  "token": "<w8m_action_token>",
  "event_id": "order:20260801-42",
  "event_type": "purchase",
  "occurred_at": "2026-08-01T12:34:56.123Z",
  "value_usd": "25.500000"
}
```

Required headers are:

- `X-W8M-Action-Timestamp`: current Unix time in decimal seconds;
- `X-W8M-Action-Signature`: lowercase hexadecimal HMAC-SHA256. The HMAC key is
  the exact token string and the message is the UTF-8 bytes
  `w8m-action-request-v1\n<TIMESTAMP>\n<RAW_JSON_BODY>`.

Generate the signature from the exact body bytes that are transmitted. The
default request-clock allowance is five minutes. Unknown JSON fields, trailing
JSON values, malformed timestamps, invalid/expired tokens, and invalid MACs are
rejected before database work.

The taxonomy is deliberately closed:

| `event_type` | Additional fields |
|---|---|
| `conversion` | No `action_name` or value. |
| `purchase` | Positive decimal `value_usd`, at most six fractional digits; currency is fixed to USD. |
| `download` | No `action_name` or value. |
| `video_complete` | No `action_name` or value. |
| `custom` | Required lowercase `namespace.name`, each segment beginning with a letter and containing only letters, digits, and underscore. No value. |

`event_id` is an advertiser-scoped idempotency identity of 1–128 characters,
starting with an ASCII letter or digit and then using letters, digits, `.`,
`_`, `:`, or `-`. A retry with the same advertiser, event id, and exact fact
returns success without another row. Reusing the id for a different fact
returns `409 Conflict`.

Responses are:

- `204 No Content`: stored or already stored identically;
- `400 Bad Request`: invalid event shape or lifecycle;
- `401 Unauthorized`: invalid, expired, or incorrectly signed lineage;
- `409 Conflict`: event-id reuse with different content;
- `413` or `415`: body/media-type violation;
- `503 Service Unavailable`: durable storage was unavailable. Retry the exact
  body and event id with a fresh timestamp/signature while the token is valid.

## Attribution Rules

Signed `/imp` and `/clk` publication makes one durable `measurement_touch` for
the exact auction/bid/impression/selected-creative lineage. Touch persistence is
detached, bounded, and fail-open relative to measurement publication; a MySQL
failure does not turn an otherwise valid tracker into an HTTP error.

When an action is accepted, attribution uses only touches with the same signed
lineage at or before `occurred_at`:

1. the latest click in the configured click window (default 720 hours);
2. otherwise the latest view in the view window (default 168 hours);
3. otherwise `unattributed`.

Window starts are inclusive, action time is exclusive of future times beyond
the accepted clock skew, and click always wins even when a view is newer. An
action older than `action_max_age_hours` (default 2160) is rejected. `late`
means the action was received after the click window, but does not alter the
stored occurrence time. `cmd/action-measurement -action=reconcile` repairs only
facts still marked `unattributed`; it never rewrites an existing click/view
decision.

The current contextual contract does not synthesize cross-device or general
view-through identity. View attribution is possible only when the same signed
lineage token is legitimately retained by the advertiser workflow and a view
touch exists without a qualifying click. Adding identity-based attribution or
automatic bidding requires a separate reviewed privacy/product contract.

## Storage, Retention, And Operations

`measurement_touch` and `measurement_action` are direct MySQL facts. A unique
`(adv_id,event_id)` key provides concurrency-safe idempotency. Action rows carry
only fixed privacy mode/reason values and an HMAC domain-separated pseudonym
derived from the random token id. Raw identity and consent are never accepted.
Both tables receive an `expires_at` based on `action_retention_hours` (default
2160 hours). Configuration validation requires this lifetime to cover
`action_max_age_hours`, so retention cannot silently shorten an advertised
attribution/action lifecycle.

Run maintenance from a trusted operator host with an ignored production config:

```bash
GOWORK=off AOFEI=/run/secrets/aofei.json \
  go run ./cmd/action-measurement -action=reconcile -limit=1000

GOWORK=off AOFEI=/run/secrets/aofei.json \
  go run ./cmd/action-measurement -action=prune -limit=10000
```

Authorized pseudonym-scoped export and erasure are available with
`-action=export|delete -pseudonym=<64-lowercase-hex>`. Export deliberately omits
token hashes, auction ids, and publisher ids. Follow the verified privacy-case
workflow before either operation; never put a pseudonym in shared shell logs.

Advertiser reporting is exposed at
`/goto/adv/g/ledger?action=topicsAdvActions`. It is scoped by the authenticated
advertiser id and reconciles action/attribution counts and purchase value
against aggregate impressions, clicks, and spend. All supplied labels are
ordinary escaped text; they are never placed in executable chart or URL
contexts.

Fixed-cardinality `/debug/vars` evidence includes action requests, accepted
facts, duplicates, rejection reasons, attribution outcomes, touch types, and
touch-write errors under the `aofei_action_*` names.
