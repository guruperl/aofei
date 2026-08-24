# Direct SSP Request Authenticity

This document is the P03 security and compatibility contract for `POST /pz`.
It separates public browser inventory locators from publisher/App request
authentication and defines the invariants that P03 implements and future
changes must preserve.

The threat contract, versioned browser-token codec/runtime reader, default-off
SDK/server request-authentication boundary, independent browser/App
enforcement, client-claim disposition, portal/cache integration, evidence
tooling, and rollout/rollback operations are implemented and reviewed.
Production activation remains pending a separately authorized named-publisher
canary. Publisher pages and the
readiness manifest now use the same controller-configured issuer as `/pz`: the
checked-in disabled state emits v1, while an enabled v2 deployment emits only
current-epoch v2 samples. The checked-in v2 block stays disabled and legacy
reads stay allowed until the rollout gate is complete.

## Current Boundary And Non-Claims

The currently generated browser `site` token packs `(pub_id, site_id)` and
each `slot` token packs `(slot_id, size_id)` as unkeyed base32. Anyone who
observes or guesses the numeric ids can encode another candidate tuple.
Acceptance still requires the tuple to exist in the active direct-publisher
cache, but the historical token itself provides neither integrity nor
authentication.

Current browser requests also require an exact cached-site-host match for all
present `Origin` and `Referer` values, with at least one required. This is a
useful browser provenance policy, but it is not publisher authentication: a
non-browser client can set those headers, and a valid tag copied from an
approved page remains observable and replayable.

The checked-in `direct_ssp_auth.enabled:false` compatibility state still lets
`platform:"sdk"` requests omit both headers without a request proof. When that
gate is enabled, every SDK/server request must pass the implemented scoped,
body-bound, fresh, one-use proof before entering auction side effects. App
identity and active inventory remain independent checks in both states. Do not
treat the default-off compatibility behavior as an authenticated production
contract.

The browser `aofei_pz_uid` cookie is privacy-gated cap/tracking identity. It is
not a publisher credential, inventory capability, request signature, or
human-viewability proof.

## Trust Contracts

| Mechanism | What it may prove | What it does not prove |
|---|---|---|
| Versioned browser locator set | A W8M issuer minted the site locator and each complete publisher/site/slot/size ad-unit locator, and neither was altered. | That the request came from the publisher, an authorized user, a human, or a unique browser; confidentiality or non-replay. |
| Exact browser `Origin`/`Referer` policy | A conforming browser exposed the configured page host for this request. | Publisher identity against a custom HTTP client, device integrity, viewability, or non-replay. |
| SDK/server request proof | A currently authorized publisher/App credential signed the bounded request body within the accepted freshness and replay window. | Consent, human activity, inventory activity, seller authorization, traffic quality, or creative safety. |
| Active direct-publisher cache | The publisher/site/slot/size and Web/App relationship is active and commercially complete in the generation loaded by the worker. | Who sent the request or whether a previously issued locator/credential is uncompromised. |
| S02 Summer session | Who may manage, generate, rotate, revoke, or inspect publisher-scoped security material in the control plane. | Runtime `/pz` request authentication. |
| I03 advertiser credential | One advertiser's scoped `/api/v1` control-plane authority. | Publisher, App, inventory, or `/pz` authority. |

No one mechanism substitutes for another. In particular, adding an HMAC to a
public browser value prevents undetected minting or modification; it does not
turn that value into a secret or a request credential.

## Threats And Required Disposition

| Threat | Required P03 disposition |
|---|---|
| Locator enumeration, forgery, or mix-and-match | The versioned browser locator set binds each full publisher/site/slot/size identity under a server-only integrity key. Unknown versions, key ids, malformed values, modified tuples, and a valid slot moved under another site fail before auction side effects. |
| Replay of an observed browser locator | Accepted as an explicit public-browser limitation. Exact host policy, active inventory, admission controls, privacy policy, and S03 traffic-quality controls remain independent defenses. Browser locator replay must not be reported as authenticated publisher traffic. |
| Browser-header spoofing | `Origin`/`Referer` remains an exact browser policy, not a general client identity proof. Web inventory cannot be used through the SDK omission path, and App inventory cannot be used through the browser path. |
| SDK/server impersonation or replay | A distinct publisher/App-scoped credential must produce a body-bound proof with bounded freshness and shared replay detection. Missing, stale, duplicated, revoked, cross-App, or body-mismatched proof fails closed before inventory use or auction side effects. |
| Publisher or App credential compromise | Revoke and replace the affected credential independently, retain redacted account-scoped audit evidence, and reject it across all HTTP nodes after the bounded shared-state propagation window. Do not rotate unrelated advertiser or Summer credentials as a substitute. |
| Browser-locator signing-key compromise | Stop issuing with the affected key id, deploy a replacement, measure migration, and disable the compromised verification key after the explicit overlap. Active-cache withdrawal remains available for narrower publisher/site/slot containment. |
| Inventory deactivation or stale cache | Every request, including one with valid integrity/authentication, must still resolve the complete tuple from the active loaded cache. A valid locator or request proof never reactivates inactive, incomplete, stale, or mismatched inventory. |
| Cross-account control-plane access | Generation, download, credential display, rotation, revocation, and audit access require the verified S02 publisher scope and named permission; client `_grole`, `_gadmin`, or account ids are never trusted. |
| Credential, token, or request leakage | Secrets and raw proofs stay out of MySQL plaintext, Redis payloads, logs, audits, samples, downloaded browser tags, mobile source, and repository files. Public locators may be logged only under the existing privacy/retention contract. |

S03, not P03, owns general fraud detection, automation classification,
viewability, and billing recommendations. P03 does not claim device attestation
or proof that an authorized publisher behaved honestly.

## Versioned Browser Locator Implementation

The implemented token is an ASCII value with this fixed shape:

```text
pz2.<site|slot>.<key_id>.<epoch>.<payload>.<mac>
```

`payload` uses unpadded URL-safe base64 over fixed-width big-endian integers.
A `site` payload contains `pub_id` and `site_id`. A `slot` payload contains the
complete `pub_id`, `site_id`, `slot_id`, and `size_id` tuple, so a valid slot
cannot be moved beneath another site. `mac` is the complete HMAC-SHA-256 output
over a domain-separated canonical token prefix. Non-canonical epochs/base64,
wrong token kinds, unknown versions, unknown key selectors, modified payloads,
modified signatures, and mixed v1/v2 site-slot requests fail before auction
side effects.

Each configured key has a URL-safe `key_id`, a positive rotation `epoch`, and
an environment reference containing a base64 or hexadecimal encoding of an
exact 32-byte key. The immutable runtime codec emits only with `current` and
verifies at most `current` plus one distinct `previous` key. The epoch is the
explicit validity boundary: removing a selector from that bounded key ring
withdraws every token from that epoch. There is deliberately no per-token
expiry timestamp.

The key stays deployment-owned and never appears in the tag, cache payload,
schema, repository, or publisher UI. A disabled example is checked in:

```json
"direct_ssp_tokens": {
  "enabled": false,
  "legacy_read_mode": "allow",
  "current": {
    "key_id": "primary",
    "epoch": 1,
    "key_env": "DIRECT_SSP_TOKEN_KEY"
  }
}
```

When enabled, `legacy_read_mode:"allow"` accepts both complete v1 requests and
complete v2 requests. `"deny"` is the explicit legacy-disable gate and is
valid only while v2 is enabled. Unknown versions never enter the historical
base32 decoder, and a request cannot combine a v1 site with a v2 slot or the
reverse. The fixed-cardinality
`aofei_ssp_inventory_token_outcomes_total` map distinguishes accepted v1/v2,
legacy-gate denial, invalid v1/v2, mixed-version, and unknown-version outcomes.

Every successfully verified v2 tuple still resolves against the active loaded
publisher cache. Integrity cannot reactivate inactive, incomplete, stale, or
mismatched inventory. V2 locators also remain public and replayable; their
acceptance is never recorded as publisher authentication.

Publisher tag/API generation and the cache readiness manifest now use the same
narrow issuer object and immutable codec as `/pz`. The portal first proves the
authenticated publisher/site owns one active site, then emits v1 in disabled
compatibility mode or current-epoch v2 locators when enabled. App samples show
the four signing headers and safe authentication mode but never a private
credential. The read-only readiness manifest records token version, safe key
selector/epoch, legacy policy, request-auth mode, and bounded credential refresh,
staleness, and rotation-overlap settings without exposing environment names,
key bytes, credential ids, or private values. Do not enable v2 or deny legacy
reads in production merely because generation is integrated; the rollout must
measure both read paths, name rollback ownership, and retain a reversible
compatibility interval.

## SDK And Server Authentication

Non-browser traffic uses a credential class distinct from browser locators,
Summer sessions, I03 advertiser tokens, consent identifiers, and browser
cookies. Each credential belongs to one existing `pub` account and one active,
approved App site whose environment is `App` and integration mode is `SDK` or
`ServerAPI`.

Issue and rotation generate an Ed25519 keypair. Summer shows the private value
once in the form `w8m_pz_v1_<public-id>_<private-seed>` after a verified S02
session, named `publisher.credential.*` permission, exact publisher scope, and
recent MFA. MySQL stores only the public id, 32-byte Ed25519 public key, scope,
expiry, rotation/revocation metadata, and actor metadata. The private seed is
not recoverable from MySQL and is never written to Redis or security audit.
Issue, rotation, and revocation write hashed-object lifecycle events to the
existing immutable `auth_security_audit` table in the same transaction.

An SDK/server caller sends these four headers:

```text
X-W8M-PZ-Credential: w8m_pz_v1_<public-id>
X-W8M-PZ-Timestamp: <canonical Unix seconds>
X-W8M-PZ-Nonce: <canonical unpadded base64url, 16-32 bytes>
X-W8M-PZ-Signature: <canonical unpadded base64url Ed25519 signature>
```

The signature covers this newline-separated canonical message:

```text
w8m-pz-request-v1
POST
/pz
<credential reference>
<timestamp>
<nonce>
<base64url SHA-256 of the exact decompressed JSON body>
```

The default freshness window is 300 seconds. A complete immutable verifier
snapshot is loaded from MySQL before an enabled HTTP process starts, refreshed
every 30 seconds, and rejected after 120 seconds without a successful refresh.
The request path never queries MySQL. After signature and exact publisher/site
scope checks, Redis atomically claims a hash of the public id and nonce through
`SET NX EX`; no raw nonce or signature is retained. Duplicate concurrent
requests have exactly one winner. Missing, malformed, stale, body-mismatched,
unknown, expired, revoked, or cross-App proof returns generic `401`; stale
verifier state or unavailable replay storage returns generic `503`.

A caller can always select the public browser path for active Web inventory if
it satisfies that path's rules. It cannot use the browser path for App
inventory. When `direct_ssp_auth` is enabled, it also cannot use
`platform:"sdk"` to bypass browser provenance without the SDK/server proof;
the disabled compatibility state remains explicitly unauthenticated.

## Rotation And Revocation

Browser signing keys and publisher/App credentials have separate lifecycles:

- Browser signing-key rotation changes the server issuer/verifier key ring and
  regenerated public locators. A bounded current/previous overlap supports
  rollback; new output never uses the previous key after rotation.
- Publisher/App credential rotation creates a new scoped credential, exposes
  its private value once, and either revokes the predecessor immediately or
  permits an explicit bounded overlap. The configured default maximum overlap
  is one day. A revocation removes the credential from the local verifier
  snapshot immediately; other nodes converge on their bounded refresh, and a
  stale snapshot fails closed. Revocation does not require regenerating public
  browser tags.
- Inventory revocation is semantic and independent: deactivate the publisher,
  site, or slot and publish a complete cache generation. All locators and
  credentials continue to fail against the withdrawn tuple even if their own
  cryptographic material remains valid.
- Broad compromise response may combine all three actions, but operator
  evidence must name which boundary was compromised and which containment was
  applied.

No key or credential is considered revoked while another accepting node still
uses unbounded stale verification state. The rollout task must define the
maximum propagation/overlap interval, health evidence, rollback owner, and the
condition that ends compatibility.

The repository abuse proof is `./scripts/aofei-p03-proof.sh`. It exercises the
legacy/v2/tamper/mixed matrix, fixed authentication outcomes, bounded lifecycle
rotation/revocation, O01 SSP admission, and 32 concurrent claims of one signed
nonce with exactly one accepted claim and 31 replays. Replay Redis keys contain
only a domain-separated digest and expire at the end of the accepted timestamp
window (at most twice the configured skew for a future-edge timestamp); normal
rollout and rollback never scan-delete them. The production cache-first canary,
key-ring order, App-withdrawal rollback, and evidence ownership are specified in
[publisher-activation.md](publisher-activation.md).

## Enforcement And Side-Effect Order

The implemented SDK authentication and cross-path enforcement retain this
deterministic pre-auction boundary:

1. apply request-method, encoding, body-size, and bounded JSON checks;
2. classify browser versus SDK behavior without accepting that claim as
   identity;
3. verify SDK/server body and freshness, then resolve the complete tuple from
   the active publisher cache and enforce the exact credential publisher/App
   scope and shared replay claim; browser traffic instead validates locator
   integrity;
4. enforce Web/App, media, size, floor, and exact browser host policy;
5. apply privacy and independent admission/traffic-quality policy; then
6. enter matching, reservations/caps, middleman fanout, response
   materialization, and audit publication.

Any failure before step 6 must not set or rotate `aofei_pz_uid`, mutate caps or
reservations, contact middleman bidders, publish request/response/attribute
audits, or disclose account existence through raw ids or credential details.
The SDK authentication boundary uses generic `401` for proof failures and
generic `503` for unavailable verifier/replay dependencies; later enforcement
work must preserve deterministic, non-oracular failures across both paths.
Malformed, inventory, App-identity, and browser-policy responses therefore use
only generic HTTP status text. A missing or corrupt publisher-cache dependency
returns generic retryable `503`; invalid/inactive inventory remains generic
`400`, and browser provenance policy remains generic `403`.

## Compatibility Invariants

P03 may add version/authentication fields and fixed-cardinality outcome
metadata, but it must preserve the established serving contract unless a later
approved migration says otherwise:

- `POST /pz`, the existing `pub` ownership model, Web/App inventory taxonomy,
  and `pubmap` plus `pubmap:by-id` cache ownership remain;
- omitted/`"html"`, `"json"`, and `"openrtb"` response meanings and ad-unit
  ordering remain;
- server-owned media/size/floor, privacy, seller/supply-chain, local auction,
  callback, middleman, measurement, and settlement rules remain independent of
  client claims;
- browser CORS remains endpoint-limited and credentialless; authentication is
  not moved into `aofei_pz_uid`; and
- request/audit data remains covered by the S01 disclosure and retention
contract. Authentication does not make client-supplied geo, audience,
consent, seller, or targeting claims trustworthy by itself.

An unauthenticated `platform:"sdk"` compatibility request can still carry
restrictive privacy signals, but even a syntactically valid personalization
grant is downgraded to contextual `sdk_unauthenticated`. Its body identity,
geo, demographic, and uploaded-audience values are removed before matching.
After a valid publisher/App proof and an independent S01 personalization grant,
the publisher may assert bounded `device`/`user` values for local matching:
body IP/coarse geo takes precedence over transport fallback when supplied, but
it remains a publisher assertion rather than verified device location, and
latitude/longitude/accuracy/fix age are always removed.

See [publisher-activation.md](publisher-activation.md) for the current rollout
gate. Repository completion is not production authenticity: observability,
rotation/rollback operations, and a named-publisher canary must be executed and
retained for that deployment before either default-off gate is activated.
