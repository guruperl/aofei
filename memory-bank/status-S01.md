# Status S01 - Privacy, Consent, And Data Disclosure

State: `[+]` Completed

## Goal

Centralize lawful, minimal, and testable use and disclosure of user and device
data across direct SSP, DSP, tracking, audits, and external bidder fanout.

## Dependencies

- Foundation work with no new lane dependency.
- Legal/policy review supplies applicable market and partner requirements.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Data inventory | `[+]` | `docs/privacy-data-governance.md` classifies request identity, cookies, geo/demographics, regulatory signals, audiences, cap/tracking state, callback context, logs, and account records by source, purpose, recipient, and retention. |
| Consent contract | `[+]` | One most-restrictive decision covers COPPA, GPC, DNT/LMT, US Privacy, GPP, GDPR/TCF v2.3, missing, invalid, and conflicting signals; personalization requires configured vendor/purposes, current CMP metadata, compatible publisher restrictions, and disclosed-vendor evidence. |
| Runtime policy | `[+]` | `/bid` and `/pz` decide before matching; contextual/restricted requests lose identity and detailed data, all modes lose exact coordinates, accepted identities are domain-separated HMAC pseudonyms, and trackers retain only signed short-lived pseudonymous state. |
| Bidder sanitation | `[+]` | Every endpoint receives a fresh typed, contextual, impression-scoped request; raw identity, IP/UA, detailed geo/demographics, search data, unknown fields, and uncontrolled extensions are removed, and ambiguous impression IDs fail before fanout. |
| Data lifecycle | `[+]` | Audience writes install conditional TTL atomically; scoped deletion helpers never enumerate neighbors; generated NATS logs prune by configured retention; the governance contract assigns encryption, key rotation, backup, access/export/correction, and deletion ownership. |
| Operator evidence | `[+]` | Fixed-cardinality privacy metrics, privacy mode/reason audits, request/response/attribute/winloss redaction, bounded configuration, integration guidance, incident steps, and regression fixtures are in place. |

## Acceptance Criteria

- `[+]` Every collected or disclosed identity field has a documented source, purpose,
  consent rule, retention, and recipient.
- `[+]` Missing or denied consent follows explicit behavior and cannot leak through
  middleman passthrough fields or logs.
- `[+]` Sensitive stored data and backups have documented encryption and key-rotation
  ownership.
- `[+]` Privacy decisions are testable without exposing personal data in metrics or
  public responses.

## Verification

- `[+]` Consent matrix, data-scrubbing, retention/deletion, audit-redaction,
  publisher/partner isolation, public-data, security, and full closeout suites.

## Deep Review

- Corrected TCF vector consumption when the configured vendor is below the
  vector maximum, validated range ordering and zero padding, rejected duplicate
  segments, and made restrictive publisher entries monotonic. Exact
  coordinates are removed universally so no special-feature opt-in is assumed.
- Eliminated the legacy IP-plus-UA identity fallback at the runtime boundary;
  browser body identifiers are ignored and cookies are read or set only after a
  configured personalization grant and complete request validation.
- Removed privacy-sensitive creative macro output and opaque markup/tracking
  data from audits. Win/loss publication also removes the HMAC user suffix
  embedded in a local auction bid id after cap and replay processing completes.
- Made bidder isolation fail closed for empty, duplicate, missing, or
  cross-assignment impression identities while keeping external fanout
  contextual and behind its separate default-false disclosure gate.
- Kept infrastructure encryption and legal-market interpretation outside the
  code's claims: the operator contract explicitly requires named counsel, key,
  secret-store, volume, transport, backup, and incident ownership.

## Closeout Evidence

- Go 1.23.5 full tests and vet passed in Aofei and pzdesign; staticcheck v0.5.1
  passed with each workflow's exact check set.
- The documented Aofei scoped race suite and pzdesign `cmd/unify` race suite
  passed, as did DSP/match benchmarks and focused consent, sanitation,
  pseudonym, cookie, audience TTL/deletion, log-pruning, and audit fixtures.
- Aofei documentation/public-data guards and pzdesign template/public-copy/
  public-data guards passed; gitleaks v8.30.1 scanned both repositories' full
  histories with no findings; both worktrees pass `git diff --check`.
- The existing D01 disposable Docker cache smoke remains valid because S01 did
  not change schema or cache-population contracts; S01 Redis lifecycle behavior
  is covered independently with miniredis atomicity and isolation tests. The
  pre-existing local Docker stack was not reset or flushed.
- No evolution entry was added because S01 implements the privacy boundary
  already approved in the current roadmap/evolution direction. No commit was
  created because the active goal's commit policy is `none`.
