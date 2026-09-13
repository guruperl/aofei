# Retired milestone M35 - SSP Account/Schema ADR

**Milestone.** M35
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M35.md
**Source specification.** memory-bank/milestone.md#m35---ssp-accountschema-adr
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M35 - SSP Account/Schema ADR `[+]`

Decide whether a separate SSP account boundary is still needed after M32-M34
evidence. Do not implement a separate account in this milestone.

Scope:

- Decide the account/schema boundary for current direct SSP `/pz` traffic.
- Keep this milestone ADR-only with no schema, runtime, cache payload, audit
  payload, ledger, or Summer/Genelet admin UI change.
- Record concrete future triggers for reopening the separate SSP account
  question.

Acceptance:

- ADR 0002 records the decision to keep `pub`, `pub_site`, and `pub_slot` as the
  publisher account and inventory ownership boundary.
- No separate `ssp` account role or separate SSP-owned inventory schema is added
  for the current `/pz` path.
- Future SSP schema work follows the additive M34 taxonomy direction unless a
  later milestone reopens the account-boundary decision.

Result:

- [ADR 0002](../docs/adr/0002-ssp-account-schema-boundary.md) decides not to add
  a separate `ssp` account role or separate SSP-owned inventory schema for the
  current direct SSP path.
- `pub`, `pub_site`, and `pub_slot` remain the publisher account and inventory
  ownership boundary.
- Future reconsideration requires concrete legal, settlement, intermediary,
  permission, compliance, or partner-credential requirements.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M35 status file. | `[+]` | |
| Write the SSP account/schema boundary ADR. | `[+]` | |
| Update direct SSP docs and memory-bank direction. | `[+]` | |
| Check evolution history and add a new version for the account/schema | `[+]` | |
| Run closeout verification. | `[+]` | |
| Mark milestone complete after verification. | `[+]` | |
| The ADR records that no separate `ssp` account role or schema boundary | `[+]` | |
| The ADR keeps `pub`, `pub_site`, and `pub_slot` as the publisher and | `[+]` | |
| The ADR explains why existing auth, ACL/cache/ledger joins, admin UI, | `[+]` | |
| The ADR lists concrete future triggers for reconsidering a separate SSP | `[+]` | |
| M35 changes docs and memory only; tracked schema, cache payload, | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `git diff --check` | `[+]` | |
| Manual acceptance check: ADR 0002 records "keep `pub`" as the decision, | `[+]` | |

````
