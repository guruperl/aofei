# Retired milestone M45 - Open-Source Security And Privacy Hygiene

**Milestone.** M45
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M45.md
**Source specification.** memory-bank/milestone.md#m45---open-source-security-and-privacy-hygiene
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M45 - Open-Source Security And Privacy Hygiene `[+]`

Remove exposed credentials and production-derived data from both public
repositories, make email disablement fail before account mutation, and prevent
future repository leaks.

Scope:

- Disable compromised SMTP configuration and fail closed for public account
  mail workflows while preserving login and authenticated portals.
- Separate schema/reference catalogs from deterministic synthetic fixtures and
  remove tracked backups, customer sources, captured traffic, and personal
  deployment identifiers.
- Rewrite every branch and tag in Aofei and pzdesign, then add full-history
  Gitleaks and tracked-data gates.

Acceptance:

- No known credential, customer identifier, private path, or production-derived
  payload remains in tracked files or reachable repository history.
- Schema/load/cache/bid/admin checks pass against disposable Docker resources
  without resetting the database used by w8m.com.
- Existing deployed accounts still log in, while disabled registration and
  password retrieval fail before writes.

Result:

- Summer account mail actions now require a complete SMTP block before model
  mutation; the deployed block was removed and the service restarted.
- The database baseline contains schema/reference catalogs only; one synthetic
  fixture set owns documented local advertiser, publisher, and admin logins.
- Historical backups and captured OpenRTB data were removed, customer DOCX
  sources moved outside Git, and both repositories gained privacy/secret gates.
- Every branch and tag was rewritten and independently verified before the
  sanitized refs were published.
````

## Status record

````markdown
# Status M45 - Open-Source Security And Privacy Hygiene

State: `[+]` Completed

## Tasks

- `[+]` Fail closed before public registration/password-retrieval writes when
  account email is disabled, remove the deployed SMTP block, and restart the
  service.
- `[+]` Replace production-derived baseline/business/sample data with one
  deterministic synthetic local fixture.
- `[+]` Remove tracked backup payloads, isolate customer DOCX sources, and
  neutralize personal deployment identifiers in pzdesign.
- `[+]` Add Gitleaks, tracked public-data guards, security policies, and CI
  gates to both repositories.
- `[+]` Rewrite and publish every branch/tag, coordinate cached-reference
  cleanup, and document the new public/private boundary.

## Acceptance

- `[+]` Registration and password-retrieval mail actions fail before account
  mutation when `Blks._gmail` is absent; existing login pages remain healthy.
- `[+]` Baseline load reports 57 tables, 1 view, 6 routines, 18 triggers, zero
  advertiser/publisher rows; sample load adds one synthetic advertiser and one
  synthetic publisher.
- `[+]` Known-secret fingerprints, customer identifiers, production captures,
  and private paths are absent from all rewritten refs.

## Verification

- `[+]` Disposable Docker reset/load/sample, schema diff, Redis population,
  bid-path smoke, and pzdesign database compatibility checks.
- `[+]` Full Aofei and pzdesign tests, vet, pinned staticcheck, scoped race,
  template/public-copy, documentation, actionlint, and diff-hygiene gates.
- `[+]` Gitleaks v8.30.1 plus repository-specific public-data checks on current
  trees, every rewritten ref, and fresh remote clones.
- `[+]` Live service health, public/login HTTP checks, absent SMTP block, and
  preserved deployed database accounts.

## Notes

- Existing Git author/committer names and emails remain as intentional OSS
  attribution. Future commits use the GitHub noreply address.
- No schema migration or production data rewrite is part of M45.
- At M45 closeout, email remained intentionally unavailable pending unrelated
  replacement credentials. On 2026-08-22, operators provisioned a new
  Gmail API OAuth grant with `gmail.send` scope; the follow-up transport keeps
  credentials outside Git and preserves M45's pre-mutation fail-closed gate.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Fail closed before public registration/password-retrieval writes when | `[+]` | |
| Replace production-derived baseline/business/sample data with one | `[+]` | |
| Remove tracked backup payloads, isolate customer DOCX sources, and | `[+]` | |
| Add Gitleaks, tracked public-data guards, security policies, and CI | `[+]` | |
| Rewrite and publish every branch/tag, coordinate cached-reference | `[+]` | |
| Registration and password-retrieval mail actions fail before account | `[+]` | |
| Baseline load reports 57 tables, 1 view, 6 routines, 18 triggers, zero | `[+]` | |
| Known-secret fingerprints, customer identifiers, production captures, | `[+]` | |
| Disposable Docker reset/load/sample, schema diff, Redis population, | `[+]` | |
| Full Aofei and pzdesign tests, vet, pinned staticcheck, scoped race, | `[+]` | |
| Gitleaks v8.30.1 plus repository-specific public-data checks on current | `[+]` | |
| Live service health, public/login HTTP checks, absent SMTP block, and | `[+]` | |

````
