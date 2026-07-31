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
