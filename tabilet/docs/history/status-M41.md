# Retired milestone M41 - Measurement Replay Idempotency

**Milestone.** M41
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M41.md
**Source specification.** memory-bank/milestone.md#m41---measurement-replay-idempotency
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M41 - Measurement Replay Idempotency `[+]`

Suppress duplicate signed impression and click callbacks within the tracking
signature lifetime while preserving request availability when Redis fails.

Scope:

- Deduplicate `/imp` and `/clk` independently by signed auction event identity.
- Apply replay suppression before cap mutation and ledger publication.
- Expose suppression, fail-open, and unkeyed-event metrics.

Acceptance:

- A repeated signed event publishes and mutates cap state once within the TTL.
- Impression and click identities remain independent, normal HTTP/redirect
  responses are preserved, and Redis failures remain fail-open.

Result:

- `/imp` and `/clk` use independent signature-TTL replay keys hashed from the
  signed auction event identity before cap or ledger side effects.
- Duplicate events preserve normal callback/redirect responses while skipping
  cap refresh and publication; Redis and identity failures remain fail-open.
- Suppression, fail-open, Redis-error, and unkeyed-event expvars document the
  operational effect.
- Reopened to finalize replay identity only after successful side effects and
  make cap mutation idempotent when publication is retried.
- Review remediation completed with owned processing claims, post-publication
  completion markers, and transactional per-event cap markers.
- Reopened after the follow-up review found implicit claim outcomes and cap
  errors could still reject valid events or fall back to non-idempotent writes.
- Follow-up remediation now retains keyed fail-open identity, skips unkeyed cap
  mutation, publishes through claim/cap Redis errors, and finalizes owned claims
  only after successful publication.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Add signed event identity and Redis replay keys for `/imp` and `/clk`. | `[+]` | |
| Suppress duplicates before cap mutation and ledger publication. | `[+]` | |
| Keep Redis failures and incomplete identities fail-open. | `[+]` | |
| Add replay suppression, Redis-error, and unkeyed-event metrics. | `[+]` | |
| Document the once-per-signed-event reporting policy. | `[+]` | |
| Run closeout verification and deep review. | `[+]` | |
| Replace the pre-side-effect replay marker with an owned processing | `[+]` | |
| Make frequency-cap mutation idempotent across failed publication | `[+]` | |
| Add failure/retry and concurrent replay regression coverage. | `[+]` | |
| Run review-remediation verification and close the reopened milestone. | `[+]` | |
| Replace implicit replay results with owner, duplicate, unkeyed, and | `[+]` | |
| Continue valid publication after claim/cap failures while keeping cap | `[+]` | |
| Add the cap-update fail-open metric and cancellation/failure/skew | `[+]` | |
| Run follow-up review-remediation verification and close the reopened | `[+]` | |
| Duplicate impressions and clicks publish and mutate cap state once per | `[+]` | |
| Impression and click identities remain independent. | `[+]` | |
| Redis errors preserve normal callback and redirect availability. | `[+]` | |
| Keyed claim failures still attempt the transactional cap marker; | `[+]` | |
| `GOWORK=off go test ./dsp ./internal/jobs/ledger` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./internal/jobs/ledger` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off go vet ./...` | `[+]` | |
| `GOWORK=off staticcheck ./...` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match` | `[+]` | |
| M39-M44 full Go, race, vet, staticcheck, docs, actionlint, sibling, | `[+]` | |
| Go 1.23.5 follow-up full/scoped race gates plus staged claim, cap, | `[+]` | |

````
