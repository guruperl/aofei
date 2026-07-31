# Status M41 - Measurement Replay Idempotency

State: `[+]` Completed

## Tasks

- `[+]` Add signed event identity and Redis replay keys for `/imp` and `/clk`.
- `[+]` Suppress duplicates before cap mutation and ledger publication.
- `[+]` Keep Redis failures and incomplete identities fail-open.
- `[+]` Add replay suppression, Redis-error, and unkeyed-event metrics.
- `[+]` Document the once-per-signed-event reporting policy.
- `[+]` Run closeout verification and deep review.
- `[+]` Replace the pre-side-effect replay marker with an owned processing
  claim that is finalized only after successful publication.
- `[+]` Make frequency-cap mutation idempotent across failed publication
  retries.
- `[+]` Add failure/retry and concurrent replay regression coverage.
- `[+]` Run review-remediation verification and close the reopened milestone.
- `[+]` Replace implicit replay results with owner, duplicate, unkeyed, and
  Redis-fail-open outcomes that retain complete event keys.
- `[+]` Continue valid publication after claim/cap failures while keeping cap
  writes idempotent and finalizing owned claims only after publication.
- `[+]` Add the cap-update fail-open metric and cancellation/failure/skew
  regressions.
- `[+]` Run follow-up review-remediation verification and close the reopened
  milestone.

## Acceptance

- `[+]` Duplicate impressions and clicks publish and mutate cap state once per
  status within the signature TTL.
- `[+]` Impression and click identities remain independent.
- `[+]` Redis errors preserve normal callback and redirect availability.
- `[+]` Keyed claim failures still attempt the transactional cap marker;
  unkeyed events publish without non-idempotent cap mutation.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./internal/jobs/ledger`
- `[+]` `GOWORK=off go test -race ./dsp ./internal/jobs/ledger`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off go vet ./...`
- `[+]` `GOWORK=off staticcheck ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` `GOWORK=off go test -race ./dsp ./match`
- `[+]` M39-M44 full Go, race, vet, staticcheck, docs, actionlint, sibling,
  Docker cache-smoke, and diff-hygiene gates.
- `[+]` Go 1.23.5 follow-up full/scoped race gates plus staged claim, cap,
  publication, redirect, unkeyed, cancellation, and future-skew regressions.

## Notes

- Finding: B3, with fail-open suppression selected as the product policy.
- Replay keys hash status plus auction, bid, and impression IDs to bound key
  length without trusting identifier delimiters.
- No schema or ledger payload change was required; only duplicate side effects
  within the configured signature TTL are removed.
- No `evolution/` entry was added because this implements the explicitly
  selected measurement integrity policy without changing product boundaries.
- Reopened after the M39-M44 review found that a cap or publication failure
  after replay-key creation permanently suppressed a legitimate retry.
- Review remediation now uses a short owner-token claim, releases it on
  pre-publication failure, finalizes it after successful publication, and
  commits cap mutation with a per-event marker so retries do not double count.
- Reopened again because cap failures still returned errors and a failed claim
  discarded the event key needed for idempotent cap mutation.
- Follow-up closeout adds `aofei_tracking_cap_update_fail_open_total` and
  verifies only successfully published events with missed cap updates count it.
