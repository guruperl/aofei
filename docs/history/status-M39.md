# Retired milestone M39 - Tracking And Runtime Integrity

**Milestone.** M39
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M39.md
**Source specification.** memory-bank/milestone.md#m39---tracking-and-runtime-integrity
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M39 - Tracking And Runtime Integrity `[+]`

Close the confirmed tracking-signature, frequency-cap, and adjacent request
correctness findings without changing schema or response formats.

Scope:

- Require valid configured-TTL signatures for every `/imp` and `/clk` event.
- Bound Redis cap-state lifetime, saturate packed cap counters, and keep valid
  signed measurement events recordable when no cap user identity exists.
- Remove adjacent audit initialization, weighted-selection, and SSP cookie
  correctness hazards.

Acceptance:

- Unsigned or expired impression/click events are rejected whether or not a
  `cap` value is present.
- Cap counters cannot wrap and every cap-state write has a positive TTL without
  shortening a longer existing TTL.
- Empty-user trackers still publish but do not mutate cap state; audit startup,
  weighted selection, and SSP cookie resolution have focused regression tests.

Result:

- Impression and click signatures are unconditional and use the configured
  replay TTL on tracker and redirect paths.
- Packed cap counters saturate, Redis cap hashes receive bounded idle expiry,
  and empty-user events remain measurable without attempting cap mutation.
- Audit initialization is serialized, weighted selection has a deterministic
  final-positive fallback, and SSP cookie creation happens once per request.
- Reopened after the M39-M44 review found callback-time TTL reuse, request-bound
  Redis transaction contexts, and an unconditional bulk cap expiry could break
  validity and TTL guarantees.
- Follow-up remediation now uses exact signature deadlines, detached two-second
  tracking Redis contexts with confirmed transaction cleanup, and one atomic
  bulk cap/conditional-expiry script.
````

## Status record

````markdown
# Status M39 - Tracking And Runtime Integrity

State: `[+]` Completed

## Tasks

- `[+]` Require configured-TTL signatures for every `/imp` and `/clk` event.
- `[+]` Make tracking signature TTL parameters explicit at every call site.
- `[+]` Saturate packed frequency-cap counters at 255.
- `[+]` Add bounded Redis cap-state TTL behavior and config.
- `[+]` Publish valid empty-user tracking events without cap mutation.
- `[+]` Serialize lazy audit publisher initialization.
- `[+]` Prevent lazy audit publisher creation after controller shutdown starts.
- `[+]` Make weighted selection robust to floating-point fallthrough.
- `[+]` Resolve the SSP browser cookie once per request.
- `[+]` Update code-adjacent docs, config examples, and memory-bank contracts.
- `[+]` Run closeout verification and deep review.
- `[+]` Derive every tracking marker TTL from the signature's exact validity
  deadline, including accepted future skew.
- `[+]` Validate signatures before Redis work and detach bounded Redis
  transactions from HTTP cancellation.
- `[+]` Make bulk cap data/conditional-expiry updates one atomic Redis script.
- `[+]` Run follow-up review-remediation verification and close the reopened
  milestone.

## Acceptance

- `[+]` Unsigned, expired, and modified `/imp` and `/clk` URLs are rejected
  with or without cap data.
- `[+]` Cap counters saturate and cap-state keys receive a positive TTL without
  shortening a longer TTL.
- `[+]` Empty-user signed trackers publish successfully and skip cap refresh.
- `[+]` Audit initialization, weighted selection, and cookie behavior have
  focused regression coverage.
- `[+]` Expired callbacks perform no Redis work, valid callbacks cannot poison
  a shared connection through cancellation, and bulk writes preserve longer
  TTLs while adding expiry to persistent/new keys.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./match`
- `[+]` `GOWORK=off go test -race ./dsp ./match`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off go vet ./...`
- `[+]` `GOWORK=off staticcheck ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` Go 1.23.5 full tests/vet, pinned staticcheck v0.5.1, scoped race,
  documentation, actionlint, benchmarks, sibling gates, Docker cache smoke,
  and both repository diff-hygiene checks.

## Notes

- Findings: A1-A4, C1, C2, C5, and C7.
- No schema, cache payload, `/pz` response, or middleman semantics change.
- The deep review made the compatibility bulk cap writer transactional so no
  successful write can omit expiry after a partial pipeline failure.
- The M39-M44 review's shutdown finding was resolved by closing audit
  initialization under the same mutex used by lazy publisher creation.
- Reopened after the follow-up review found callback-time TTL reuse,
  request-bound Redis transaction cleanup, and unconditional bulk expiry.
- Follow-up remediation passes expired/no-Redis, one-connection cancellation,
  exact-deadline/skew, and bulk atomic-TTL regression coverage.
- No `evolution/` entry was added because the product and architecture
  boundaries are unchanged.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Require configured-TTL signatures for every `/imp` and `/clk` event. | `[+]` | |
| Make tracking signature TTL parameters explicit at every call site. | `[+]` | |
| Saturate packed frequency-cap counters at 255. | `[+]` | |
| Add bounded Redis cap-state TTL behavior and config. | `[+]` | |
| Publish valid empty-user tracking events without cap mutation. | `[+]` | |
| Serialize lazy audit publisher initialization. | `[+]` | |
| Prevent lazy audit publisher creation after controller shutdown starts. | `[+]` | |
| Make weighted selection robust to floating-point fallthrough. | `[+]` | |
| Resolve the SSP browser cookie once per request. | `[+]` | |
| Update code-adjacent docs, config examples, and memory-bank contracts. | `[+]` | |
| Run closeout verification and deep review. | `[+]` | |
| Derive every tracking marker TTL from the signature's exact validity | `[+]` | |
| Validate signatures before Redis work and detach bounded Redis | `[+]` | |
| Make bulk cap data/conditional-expiry updates one atomic Redis script. | `[+]` | |
| Run follow-up review-remediation verification and close the reopened | `[+]` | |
| Unsigned, expired, and modified `/imp` and `/clk` URLs are rejected | `[+]` | |
| Cap counters saturate and cap-state keys receive a positive TTL without | `[+]` | |
| Empty-user signed trackers publish successfully and skip cap refresh. | `[+]` | |
| Audit initialization, weighted selection, and cookie behavior have | `[+]` | |
| Expired callbacks perform no Redis work, valid callbacks cannot poison | `[+]` | |
| `GOWORK=off go test ./dsp ./match` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off go vet ./...` | `[+]` | |
| `GOWORK=off staticcheck ./...` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| Go 1.23.5 full tests/vet, pinned staticcheck v0.5.1, scoped race, | `[+]` | |

````
