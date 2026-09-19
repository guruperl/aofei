# Retired milestone M37 - Operational Follow-Up Hardening

**Milestone.** M37
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M37.md
**Source specification.** memory-bank/milestone.md#m37---operational-follow-up-hardening
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M37 - Operational Follow-Up Hardening `[+]`

Resolve the confirmed follow-up risks from the `review.md` disposition without
changing schema shape, cache payload shape, or bid/SSP product semantics.

Scope:

- Make `cmd/nats-client` signal-aware and testable, with graceful NATS drain,
  queued-message flush, and file-handle close on shutdown.
- Tighten `cmd/nats-client` generated log directory and file permissions for
  ledger input logs.
- Add stable JSON output to `cmd/mid-callback-retry` for backlog alerting while
  preserving the existing text output.
- Record deferred review findings separately from this operational hardening
  milestone.

Acceptance:

- Context cancellation of the extracted NATS client run path drains the NATS
  connection and writes queued messages before exit.
- Generated log directories have no world permissions and no group/world write
  bits; generated log files have no world permissions and no group/world write
  bits.
- `cmd/mid-callback-retry -json` emits `due`, `stale_processing`, `selected`,
  `succeeded`, `retrying`, and `abandoned`; default text output remains
  unchanged for current operators.
- Operator docs and memory bank describe shutdown, permissions, and JSON
  alerting behavior.

Result:

- `cmd/nats-client` now uses `cmdboot.SignalContext`, drains NATS on
  cancellation, flushes queued log messages, closes files, and is covered by a
  fake-connection shutdown test.
- `cmd/nats-client` creates or tightens log directories to `0750` and log files
  to `0640`, with tests asserting private generated modes.
- `cmd/mid-callback-retry` keeps its existing summary line by default and adds
  `-json` for stable automation.
- Pubmap envelope compatibility, source-specific SSP/middleman fanout metrics,
  RAdv SQL-null cleanup, HMAC allocation benchmarking, auction function
  cleanup, and local-cache pointer-swap work remain deferred.
````

## Status record

````markdown
# Status M37 - Operational Follow-Up Hardening

State: `[+]` Completed

Resolve the confirmed follow-up risks from `review.md` without changing schema,
cache payloads, or bid/SSP product semantics.

## Tasks

- `[+]` Create the M37 status file and milestone scope.
- `[+]` Add signal-aware `cmd/nats-client` shutdown through
  `cmdboot.SignalContext`.
- `[+]` Extract a testable NATS client run path and fakeable NATS connection
  boundary.
- `[+]` Drain NATS, flush queued log messages, close file handles, and return
  cleanly on context cancellation.
- `[+]` Tighten generated NATS log directories to `0750` and log files to
  `0640`.
- `[+]` Add tests proving shutdown drains NATS and preserves queued log writes.
- `[+]` Add tests asserting generated log paths are not world-readable or
  group/world-writable.
- `[+]` Add `cmd/mid-callback-retry -json` with stable backlog/result fields
  while preserving the existing text summary.
- `[+]` Update operator docs and memory-bank files for shutdown, permissions,
  and JSON alerting.

## Acceptance

- `[+]` Context cancellation of the extracted NATS client run path drains the
  NATS connection and writes queued messages before exit.
- `[+]` Generated NATS log directories and files have no world permissions and
  no group/world write bits.
- `[+]` `cmd/mid-callback-retry -json` emits `due`, `stale_processing`,
  `selected`, `succeeded`, `retrying`, and `abandoned`.
- `[+]` Default `cmd/mid-callback-retry` text output is unchanged.
- `[+]` Operational docs recommend JSON output for automation instead of parsing
  prose.

## Deferred Review Findings

- Pubmap envelope compatibility needs a later cache-contract milestone with
  legacy-plus-enveloped readers before writers flip.
- Source-specific SSP/middleman fanout counters remain low-priority metrics
  work.
- RAdv SQL-null cleanup remains mild cache-builder/runtime coupling debt.
- HMAC signing allocation work needs benchmarks before a signer refactor.
- Auction function size and local-cache atomic pointer swap remain
  opportunistic cleanup.

## Verification

- `[+]` `GOWORK=off go test ./cmd/nats-client`
- `[+]` `GOWORK=off go test ./cmd/mid-callback-retry ./internal/jobs/midcallback`
- `[+]` `GOWORK=off go test ./dsp ./match`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off go vet ./...`
- `[+]` `GOWORK=off staticcheck ./...`
- `[+]` `GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Notes

- No schema changes were made.
- No cache payload format changes were made.
- `review.md` remains a review artifact and is not treated as runtime
  documentation.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M37 status file and milestone scope. | `[+]` | |
| Add signal-aware `cmd/nats-client` shutdown through | `[+]` | |
| Extract a testable NATS client run path and fakeable NATS connection | `[+]` | |
| Drain NATS, flush queued log messages, close file handles, and return | `[+]` | |
| Tighten generated NATS log directories to `0750` and log files to | `[+]` | |
| Add tests proving shutdown drains NATS and preserves queued log writes. | `[+]` | |
| Add tests asserting generated log paths are not world-readable or | `[+]` | |
| Add `cmd/mid-callback-retry -json` with stable backlog/result fields | `[+]` | |
| Update operator docs and memory-bank files for shutdown, permissions, | `[+]` | |
| Context cancellation of the extracted NATS client run path drains the | `[+]` | |
| Generated NATS log directories and files have no world permissions and | `[+]` | |
| `cmd/mid-callback-retry -json` emits `due`, `stale_processing`, | `[+]` | |
| Default `cmd/mid-callback-retry` text output is unchanged. | `[+]` | |
| Operational docs recommend JSON output for automation instead of parsing | `[+]` | |
| `GOWORK=off go test ./cmd/nats-client` | `[+]` | |
| `GOWORK=off go test ./cmd/mid-callback-retry ./internal/jobs/midcallback` | `[+]` | |
| `GOWORK=off go test ./dsp ./match` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off go vet ./...` | `[+]` | |
| `GOWORK=off staticcheck ./...` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
