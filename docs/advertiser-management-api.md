# Advertiser Management API

I03 adds an optional external advertiser control plane at `/api/v1`. Its
contract is [management-api-openapi.yaml](management-api-openapi.yaml), and the
checked generated Go client is `managementapi/client`. It is not a supported
alias for `/goto/<role>/json/...`; Summer/Genelet component names, templates,
cookies, sessions, TOTP values, and CSRF tokens are not API contracts.

## Boundary And Lifecycle

The first version exposes only the credential-owned advertiser and its
campaigns, ad groups (`items`), creatives, targeting, cache-activation
operations, and delivery reports. It cannot manage publishers, statements,
adjustments, settlements, bidders, credential references, middleman routes,
seller approval, experiments, or account permissions.

The feature is disabled in `etc/aofei.json`. Enabling it requires all of:

1. Apply the reviewed I03 migration: four API tables, `api_version` on campaign,
   item, and creative, five triggers, and the indexes/checks from the clean
   baseline. Never replay `step4_init.sql` over a populated database.
2. Complete and enable S02 identity in Summer. The advertiser or administrator
   signs in with TOTP and recent reauthentication, opens **管理 API 凭证**, and
   chooses the minimum required scopes. A token is shown once.
3. Provision one common random 32-byte key to every `cmd/unify` node under the
   environment name in `management_api.key_env`. The same key is needed only
   by nodes that issue or validate credentials. Store it outside Git and JSON.
4. Set `management_api.enabled=true`, review request/body/quota/cache bounds,
   deploy a canary, and prove a read, a repeated idempotent write, an account
   denial, a quota rejection, and an activation transition before wider use.

Tokens use the `w8m_v1_<public-id>_<secret>` shape. MySQL stores the lookup
public id and a domain-separated HMAC-SHA-256 digest, never the bearer token.
Rotation replaces the digest immediately. Revocation and expiry fail closed.
Browser sessions authorize credential issuance/rotation/revocation but cannot
authenticate API calls; API tokens cannot log in to the browser portal.

## Scopes

| Scope | Access |
|---|---|
| `api.campaign.read` | Advertiser profile, campaigns, items, and their operation status. |
| `api.campaign.write` | Create and replace campaign/item configuration; cannot approve delivery. |
| `api.creative.read` / `api.creative.write` | Read or author source-only creatives. |
| `api.targeting.read` / `api.targeting.write` | Read or replace bounded item targeting fields. |
| `api.report.read` | Read account-scoped derived delivery intervals. |

The server derives `adv_id` from the verified credential before loading a
resource. Request JSON rejects unknown fields, including caller-supplied
advertiser, role, permission, audit, activation, or accounting authority.
Missing and cross-account ids both return the same `404` envelope.

## Delivery And Commercial Semantics

- All start/end values and daily hard-limit resets are UTC. A campaign's IANA
  timezone applies only to its 168-hour Monday-through-Sunday calendar; item
  calendars inherit it.
- `Fast` and `Even` are deterministic non-adaptive pacing modes. No endpoint
  performs automatic bidding or budget optimization.
- Each campaign and item has separate total and UTC-daily spend, impression,
  and click hard-limit scopes. An omitted individual limit is unlimited; zero
  is a hard stop.
- Prices are finite positive USD CPM. Spend report values are reconciled
  per-impression USD strings with six-decimal accounting semantics and the
  `usd-cpm-impression-v2` version.
- Limited demand fails closed when the shared Redis reservation boundary or a
  current delivery cache is unavailable. API acceptance never changes this.
- Campaign/item creation remains in the existing review states (`New` and
  `Prepare`). The management API cannot self-approve delivery.
- Banner and Video creatives are HTTPS source URLs with reviewed file types;
  Native uses the structured v1 source object. The API has no executable
  preview endpoint.

## Writes, Versions, And Activation

Every `POST` and `PATCH` requires `Idempotency-Key` with 8–128 visible
characters. The database stores only its keyed digest plus a request digest, a
random per-attempt claim token, and the bounded completed response for 24
hours. A single exclusive upsert serializes concurrent repeats without an
InnoDB lock-upgrade race. The same key and exact request replay the original
`202`; the same key with different input returns `409`.

Every `PATCH` also requires the latest `ETag: "vN"` in `If-Match`. Database
triggers advance `api_version` for API and portal updates, so stale writers get
`409` instead of silently overwriting a human edit.

A successful mutation means configuration is durably accepted, not yet
acknowledged by a cache publication. The `202` body contains an operation with
a bounded activation deadline. Before a cache build reads configuration, it assigns an opaque
publication generation to the pending operations already visible in MySQL.
Only those operations become `Active` after that generation reaches the
configured serving backend (Redis in remote mode, spread in local mode, or
both for `all`). A mutation committed during the build waits for the next
generation rather than receiving a false activation claim.
`GET /api/v1/operations/{id}` reports `Delayed` after the deadline without
claiming publication. Normal cache propagation is at most the reviewed
`management_api.cache_activation_seconds` (900 seconds in the example).
An `Active` operation confirms that a later cache generation completed; it does
not change `New`/`Prepare` review state and does not itself prove that the
campaign, item, or creative is eligible to serve.

## Versioning, Compatibility, And Deprecation

`/api/v1` is the major compatibility boundary. Within v1, the service may add
new optional response fields, enum-independent endpoints, or opt-in scopes.
It will not remove or rename a field, make an optional request field required,
change an existing field's meaning/type, broaden a credential scope, or relax
account isolation. Request objects remain strict, so a client must not send a
field until the published contract declares it; response clients should ignore
unknown optional fields.

An incompatible change uses a new major path and a separately generated client.
Ordinary deprecation receives at least 180 days' notice in this document and
OpenAPI, with `Deprecation` and `Sunset` response headers plus a migration guide
before removal. Emergency credential revocation or endpoint disablement for an
active security incident is not delayed by that window. The OpenAPI `info`
version follows contract releases; the checked Go client must be regenerated
or reviewed against the same source and pass its safety-header/golden tests in
the same change.

## Reads, Reports, Errors, And Quotas

Lists are ordered by ascending stable id and use `cursor` plus `limit` (default
50, maximum 100). Reports accept RFC3339 `from`/`to`, at most 31 days, and state
UTC, USD, source, freshness, and `partial`. Recent or dependency-incomplete
intervals are partial/unavailable, never a fabricated zero.

Errors always use:

```json
{"error":{"code":"version_conflict","message":"resource version does not match If-Match","request_id":"..."}}
```

The API returns bounded `400`, `401`, `403`, `404`, `409`, `413`, `415`, `428`,
`429`, and `503` behavior described by OpenAPI. A `503` write is retried only
with the same idempotency key. Load balancer retries are not a correctness
mechanism.

Redis atomically applies separate per-minute credential and advertiser quotas.
Once a credential has exhausted its own quota, further denials do not consume
the account quota and cannot starve another credential for that reason. Quota
state has a TTL and is isolated under the management API namespace. A quota
dependency failure returns `503`; no API request bypasses admission.
Expvar metrics use only fixed status/rejection classes and never credential or
advertiser ids.

## Audit, Retention, Rotation, And Rollback

`api_audit` stores actor/service ids, account, object id, keyed idempotency
digest, version transition, outcome, and safe reason. It does not store bearer
tokens, request bodies, creative sources, S02 sessions, TOTP/recovery material,
partner credentials, or payment data. Update is forbidden and deletion is
available only to a separated retention principal through the bounded
connection-local gate. Use the S02 account-security retention period and record
the prune externally; the HTTP principal has no update/delete grant. From the
restricted maintenance host, run the shared operator CLI with its maintenance
database configuration:

```bash
SUMMER=/etc/aofei/summer-maintenance.json \
  /opt/aofei/bin/identity-admin \
  -action=prune-api-audit -limit=1000 \
  -reason='scheduled management API audit retention'
```

The command deletes at most 10,000 expired rows per run, writes a new immutable
prune event, resets the connection-local bypass under a fresh two-second
context, and discards the connection if reset cannot be confirmed.

Rotate the HMAC key only through a coordinated credential rotation: issue new
tokens under the new deployment generation, move clients, revoke the old
tokens, then roll every node with the new key. There is no dual-key decoder.
Losing the key invalidates all credentials; exposure requires revoking all API
credentials, replacing the key, and issuing new tokens.

Emergency rollback sets `management_api.enabled=false` on all nodes and leaves
the additive schema/evidence intact. Existing HTML portals and auction routes
continue. Do not drop version columns/triggers while an older node or API
client may still write; do not delete idempotency rows before their 24-hour
window.

## Verification

Run the ordinary Aofei/Pzdesign closeout gates plus:

```bash
GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./managementapi/...
GOWORK=off GOTOOLCHAIN=go1.23.5 go test -race ./managementapi/...
npx --yes @redocly/cli@1.34.5 lint docs/management-api-openapi.yaml
(cd ../pzdesign && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./cmd/unify ./summer/apicredential ./summer/registry)
(cd ../pzdesign && GOWORK=off GOTOOLCHAIN=go1.23.5 go run ./tools/check-templates.go -ext=.g,.e)
```

Restore the baseline into a uniquely named disposable MySQL 8 instance and
use a disposable Redis instance/key. Prove schema counts, one-time issuance,
old-token failure after rotation, exact account scope, simultaneous same-key
writes, changed-body conflict, stale `If-Match`, immutable audit, and operation
activation after a successful cache publication. Remove all fixtures afterward.
