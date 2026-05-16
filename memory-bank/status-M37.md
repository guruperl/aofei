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
