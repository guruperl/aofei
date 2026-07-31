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
