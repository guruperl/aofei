# Direct SSP Request Authenticity

This document is the P03 security and compatibility contract for `POST /pz`.
It separates public browser inventory locators from publisher/App request
authentication and defines the invariants that the remaining P03 work must
preserve.

This first P03 task changes documentation and milestone state only. The
versioned token format, SDK/server credentials, enforcement, portal/cache
integration, and production rollout remain unimplemented. The current runtime
behavior remains the v1 contract in [ssp-direct-traffic.md](ssp-direct-traffic.md).

## Current Boundary And Non-Claims

The current browser `site` token packs `(pub_id, site_id)` and each `slot`
token packs `(slot_id, size_id)` as unkeyed base32. Anyone who observes or
guesses the numeric ids can encode another candidate tuple. Acceptance still
requires the tuple to exist in the active direct-publisher cache, but the token
itself provides neither integrity nor authentication.

Current browser requests also require an exact cached-site-host match for all
present `Origin` and `Referer` values, with at least one required. This is a
useful browser provenance policy, but it is not publisher authentication: a
non-browser client can set those headers, and a valid tag copied from an
approved page remains observable and replayable.

Current `platform:"sdk"` requests may omit both headers and have no request
credential or replay proof. App identity and inventory are checked against the
active cache, but those checks do not prove which publisher application sent
the request. This credentialless behavior is a pre-P03 compatibility state,
not an authenticated production contract.

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

## Browser Locator Target

The later versioned-token task must define one unambiguous server-issued
format with an explicit version and key/rotation selector. The versioned
`site` value must bind `pub_id` and `site_id`; each versioned `slot` value,
validated with that site value, must bind the complete `pub_id`, `site_id`,
`slot_id`, and `size_id` identity. Valid pieces cannot be moved between sites,
and no field may be taken from an unsigned client claim or inferred through a
downgrade heuristic. The signing key stays deployment-owned and never appears
in the tag, cache payload, schema, repository, or publisher UI.

The exact encoding, validity/epoch representation, and key-ring limits belong
to the next P03 row. Whatever format is selected must preserve these rules:

- only the current version is emitted after rollout begins;
- legacy v1 reads are separately measured and allowed only behind an explicit
  compatibility gate;
- unknown or malformed versions never fall back to the legacy decoder;
- disabling the legacy reader is an explicit, observable, reversible rollout
  action rather than an inferred date;
- a valid v2 locator set is still public and replayable; and
- active-cache validation of the complete tuple remains authoritative on every
  request.

The migration may temporarily read v1 and v2 tags, but it must not change the
meaning of `/pz`, create a second inventory owner, or present v1 traffic as v2.

## SDK And Server Authentication Target

Non-browser traffic needs a credential class distinct from browser locators,
Summer sessions, I03 advertiser tokens, consent identifiers, and browser
cookies. Each credential is owned by one existing `pub` account and scoped to
one approved App identity or a narrower integration scope.

The later authentication task must bind the proof to the exact bounded request
body and enough canonical request context to prevent substitution across
requests. It must enforce a bounded clock window and one-use request evidence
through shared state across all accepting HTTP nodes. Credential lookup,
freshness, and replay dependencies fail closed; a process-local replay cache is
not sufficient for a multi-node deployment.

Only a one-time generated secret may be shown to an authorized publisher. The
server retains a non-reversible verifier or a deployment-encrypted value only
where the chosen proof algorithm strictly requires it. Credential identifiers,
scopes, creation, last-use class, rotation, and revocation state may be stored
and audited; the secret and raw proof may not.

A caller can always select the public browser path for active Web inventory if
it satisfies that path's rules. It cannot use the browser path for App
inventory or use `platform:"sdk"` to bypass browser provenance without the
SDK/server proof.

## Rotation And Revocation

Browser signing keys and publisher/App credentials have separate lifecycles:

- Browser signing-key rotation changes the server issuer/verifier key ring and
  regenerated public locators. A bounded current/previous overlap supports
  rollback; new output never uses the previous key after rotation.
- Publisher/App credential rotation creates a new scoped credential, exposes
  its secret once, permits only an explicitly bounded overlap when migration
  needs it, and then revokes the predecessor. A revocation does not require
  regenerating public browser tags.
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

## Enforcement And Side-Effect Order

The final enforcement implementation must retain a deterministic pre-auction
boundary:

1. apply request-method, encoding, body-size, and bounded JSON checks;
2. classify browser versus SDK behavior without accepting that claim as
   identity;
3. validate browser locator integrity, or authenticate SDK/server body,
   freshness, scope, and replay before using App/inventory claims;
4. resolve the complete tuple from the active publisher cache and enforce
   Web/App, media, size, floor, and exact browser host policy;
5. apply privacy and independent admission/traffic-quality policy; then
6. enter matching, reservations/caps, middleman fanout, response
   materialization, and audit publication.

Any failure before step 6 must not set or rotate `aofei_pz_uid`, mutate caps or
reservations, contact middleman bidders, publish request/response/attribute
audits, or disclose account existence through raw ids or credential details.
Exact HTTP status and public error families belong to the enforcement row, but
they must be deterministic and non-oracular.

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

See [publisher-activation.md](publisher-activation.md) for the current rollout
gate. Production authenticity is not established until the later P03 tasks
implement and verify versioned locators, SDK/server authentication, portal and
cache integration, enforcement, observability, rotation, rollback, and a named
publisher canary.
