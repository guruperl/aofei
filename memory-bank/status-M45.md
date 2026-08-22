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
