# Status M35 - SSP Account/Schema ADR

State: `[+]` Completed

Decide whether direct SSP needs a separate account or schema ownership boundary
after the M32-M34 evidence.

## Tasks

- `[+]` Create the M35 status file.
- `[+]` Write the SSP account/schema boundary ADR.
- `[+]` Update direct SSP docs and memory-bank direction.
- `[+]` Check evolution history and add a new version for the account/schema
  decision.
- `[+]` Run closeout verification.
- `[+]` Mark milestone complete after verification.

## Acceptance

- `[+]` The ADR records that no separate `ssp` account role or schema boundary
  is needed for the current `/pz` direct SSP path.
- `[+]` The ADR keeps `pub`, `pub_site`, and `pub_slot` as the publisher and
  inventory ownership boundary.
- `[+]` The ADR explains why existing auth, ACL/cache/ledger joins, admin UI,
  audit source separation, and M34 taxonomy cover the known needs.
- `[+]` The ADR lists concrete future triggers for reconsidering a separate SSP
  boundary.
- `[+]` M35 changes docs and memory only; tracked schema, cache payload,
  runtime, audit payload, ledger, and admin UI implementation files stay
  unchanged.

## Verification

- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `git diff --check`
- `[+]` Manual acceptance check: ADR 0002 records "keep `pub`" as the decision,
  lists future split triggers, and only docs/memory/evolution files changed.

## Notes

- M35 is ADR-only. Future schema work should follow the additive M34 taxonomy
  direction unless a later milestone reopens the account-boundary decision.
