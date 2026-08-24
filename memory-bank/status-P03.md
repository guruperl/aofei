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
| Versioned browser inventory tokens | `[ ]` | Define a versioned server-issued integrity format binding the complete publisher/site/slot/size identity, bounded validity or explicit rotation epoch, and key identifier without exposing a secret. Preserve a measured dual-read migration for existing generated tags, then provide an explicit legacy-disable gate. |
| SDK/server request authentication | `[ ]` | Require a publisher/App-scoped request credential and body-bound freshness/replay proof for `platform:"sdk"` traffic before accepting the omitted Origin/Referer path. Credentials are generated, shown, rotated, revoked, scoped, and audited through S02 controls and never stored in plaintext in MySQL, Redis, logs, samples, or mobile source. |
| Browser and App enforcement | `[ ]` | Keep exact browser Origin/Referer checks, App identity, media/size/floor, active inventory, consent, quota, and seller-chain derivation independent of client claims. Define deterministic failures before auction, cap, callback, or middleman side effects. |
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
