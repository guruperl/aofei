# Status P03 - Direct SSP Request Authenticity

State: `[~]` In progress

## Goal

Replace enumerable direct-SSP inventory locators with a versioned integrity
contract and give non-browser `/pz` traffic a real publisher/application
authentication boundary without treating a public browser token as proof of
traffic origin.

## Dependencies

- P01 commercial inventory validation and P02 server-owned seller metadata.
- S01 privacy/consent, S02 credential lifecycle, O01 traffic controls, and O02
  rotation/recovery requirements.
- D04 callback integrity, D05 follow-up remediation, and S06 repository plus
  production activation are complete. P03 is now the next strict-order
  milestone; authenticated traffic still cannot be used for revenue acceptance
  until this milestone's own contract and gates are complete.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Threat and compatibility contract | `[+]` | [The accepted contract](../docs/direct-ssp-authenticity.md) separates browser locator integrity/provenance from publisher/App request authentication; covers enumeration, replay, compromise, key/credential rotation, scoped revocation, and cache withdrawal; and preserves active-cache authority. It explicitly treats HMAC-protected browser values as observable/replayable public locators, not publisher authentication. |
| Versioned browser inventory tokens | `[+]` | The default-off `pz2.<kind>.<key_id>.<epoch>.<payload>.<mac>` codec uses a deployment-only 32-byte HMAC-SHA-256 key, binds site and complete publisher/site/slot/size identities, verifies a bounded current/previous epoch ring, rejects tamper/mix-and-match/unknown-version downgrade, revalidates active cache state, counts fixed legacy/v2 outcomes, and supplies explicit `legacy_read_mode: allow|deny`. Existing UI/manifest output remains v1 for the later integration/rollout rows. |
| SDK/server request authentication | `[+]` | Default-off `publisherauth` requires an App-scoped Ed25519 signature over canonical `POST /pz` context and the exact decompressed body, a 300-second freshness window, an immutable MySQL-derived public-key snapshot, exact cached publisher/site scope, and a hashed one-use Redis nonce claim. S02 Summer controls use named `publisher.credential.*` permissions, publisher resource scope, and recent MFA to issue/show once, rotate with bounded overlap, revoke, list safe metadata, and transactionally write immutable lifecycle audits. MySQL stores only the public verifier; private seeds, raw signatures, bodies, and nonces never enter MySQL/Redis/audits/samples. Generic pre-auction `401`/`503` failures and concurrent replay tests cover missing, stale, body-mismatched, cross-App, unavailable, and duplicate proofs while Browser behavior stays unchanged. |
| Browser and App enforcement | `[+]` | Valid SDK proofs remain subordinate to cached Web/App type, active inventory, exact App identity, media/size/floor, browser provenance, privacy, admission quotas, and server-owned seller chains. Invalid inventory/App input returns generic `400`, exact browser-policy failure generic `403`, and unavailable/corrupt publisher cache generic retryable `503`; focused tests prove no cookie, audit, auction, callback, or middleman side effect. |
| Client-claim disposition | `[ ]` | Re-evaluate request-supplied geo precedence, uploaded-audience marker normalization, and other client-controlled targeting fields under the authenticated/unauthenticated split. Preserve documented behavior only when its trust and abuse limits are explicit and tested. |
| Portal, API, and cache integration | `[ ]` | Update publisher tag/API samples and cache records for versioned tokens and credential metadata without making Summer sessions or I03 advertiser credentials runtime credentials. Cross-account requests fail before generation, download, rotation, or audit disclosure. |
| Rollout and abuse evidence | `[ ]` | Add legacy/v2/authenticated outcome metrics, bounded replay evidence, rate controls, rotation/rollback instructions, and a cache-first canary. No production publisher is cut over until regenerated tags or App credentials and rollback ownership are confirmed. |

## Acceptance Criteria

- An unauthenticated party cannot mint a new accepted publisher/site/slot/size
  combination by encoding database IDs.
- Non-browser requests cannot omit browser-origin evidence without a valid,
  scoped, fresh publisher/App request proof.
- Public browser tokens are never described or trusted as proof that traffic
  came from a human browser or from the publisher.
- Token/credential rotation and legacy withdrawal are observable, reversible,
  account-scoped, and do not require embedding a repository or database secret.
- Existing `/pz` media, floor, privacy, response, measurement, and seller-chain
  behavior remains server-owned.

## Verification

- Forge, enumeration, tamper, expiry/epoch, replay, rotation, revocation,
  cross-account, missing-header, cache-staleness, and concurrent-request tests.
- Browser tag and authenticated SDK-style contract tests through `cmd/unify`,
  including local and middleman fill/no-fill behavior.
- Full Aofei, pzdesign, and Genelet verification, public-data/secret scans,
  schema/cache compatibility, documentation/template checks, and diff hygiene.

## Exclusions

- This milestone does not claim device attestation, human-viewability proof, or
  general fraud prevention; S03 owns reviewed traffic-quality controls.
- Maintained Android and iOS client packages remain conditional I02 work.
