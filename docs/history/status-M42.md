# Retired milestone M42 - Unified HTTP Graceful Shutdown

**Milestone.** M42
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M42.md
**Source specification.** memory-bank/milestone.md#m42---unified-http-graceful-shutdown
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M42 - Unified HTTP Graceful Shutdown `[+]`

Add signal-aware graceful shutdown to the sibling `../pzdesign/cmd/unify`
service and drain Aofei audits after in-flight HTTP requests finish.

Scope:

- Use a standard-library signal context in `pzdesign` and extract a testable
  HTTP server lifecycle.
- Allow 15 seconds for graceful shutdown, then force close and report failure.
- Update the Aofei production runbook for the new service behavior.

Acceptance:

- SIGINT/SIGTERM stop new work, wait for in-flight handlers, and close the
  controller only after HTTP shutdown.
- Timeout and normal shutdown paths have focused tests in `pzdesign`.

Result:

- `cmd/unify` now owns a standard-library SIGINT/SIGTERM context and a testable
  listener/server lifecycle.
- Normal shutdown drains in-flight HTTP for up to 15 seconds before controller
  close; timeout forces close and returns the joined shutdown error.
- Controller/logger defers execute inside the run function even when startup or
  serving returns an error.
````

## Status record

````markdown
# Status M42 - Unified HTTP Graceful Shutdown

State: `[+]` Completed

## Tasks

- `[+]` Add SIGINT/SIGTERM context handling in `../pzdesign/cmd/unify`.
- `[+]` Extract and test normal, error, and timeout server lifecycle behavior.
- `[+]` Drain HTTP before closing the Aofei controller and audit publisher.
- `[+]` Update production service documentation.
- `[+]` Run cross-repository closeout verification and deep review.

## Acceptance

- `[+]` Normal signals stop acceptance and wait up to 15 seconds for handlers.
- `[+]` Shutdown timeout forces server close and returns an error.
- `[+]` Controller close occurs after HTTP draining.

## Verification

- `[+]` `(cd ../pzdesign && GOWORK=off go test ./...)`
- `[+]` `(cd ../pzdesign && GOWORK=off go test -race ./cmd/unify)`
- `[+]` `(cd ../pzdesign && GOWORK=off go vet ./...)`
- `[+]` `(cd ../pzdesign && GOWORK=off staticcheck ./cmd/unify)`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
- `[+]` M39-M44 follow-up rerun: Go 1.23.5 sibling full tests/vet,
  `cmd/unify` race, pinned staticcheck, templates, actionlint, and diff hygiene.

## Notes

- Finding: B4.
- Go's `internal` boundary requires pzdesign to use `os/signal` directly.
- The server owns an explicit listener so real in-flight drain behavior is
  covered without a fixed test port.
- No `evolution/` entry was added because service ownership and public HTTP
  contracts are unchanged.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Add SIGINT/SIGTERM context handling in `../pzdesign/cmd/unify`. | `[+]` | |
| Extract and test normal, error, and timeout server lifecycle behavior. | `[+]` | |
| Drain HTTP before closing the Aofei controller and audit publisher. | `[+]` | |
| Update production service documentation. | `[+]` | |
| Run cross-repository closeout verification and deep review. | `[+]` | |
| Normal signals stop acceptance and wait up to 15 seconds for handlers. | `[+]` | |
| Shutdown timeout forces server close and returns an error. | `[+]` | |
| Controller close occurs after HTTP draining. | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./...)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test -race ./cmd/unify)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go vet ./...)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off staticcheck ./cmd/unify)` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |
| M39-M44 follow-up rerun: Go 1.23.5 sibling full tests/vet, | `[+]` | |

````
