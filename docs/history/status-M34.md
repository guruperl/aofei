# Retired milestone M34 - Richer Supply Taxonomy ADR

**Milestone.** M34
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M34.md
**Source specification.** memory-bank/milestone.md#m34---richer-supply-taxonomy-adr
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M34 - Richer Supply Taxonomy ADR `[+]`

Keep runtime on the `pub` role and write an ADR for taxonomy fields,
cache/audit impact, admin UI changes, and migration path before schema work.

Scope:

- Keep `pub`, `pub_site`, and `pub_slot` as the publisher and inventory
  ownership boundary.
- Recommend additive nullable/defaulted future fields on existing publisher
  tables.
- Cover site/app identity, integration mode, slot/media taxonomy,
  quality/source taxonomy, cache impact, audit impact, admin UI impact, and
  migration path.
- Do not change schema, cache payloads, runtime behavior, audit payloads,
  ledger tables, or Summer/Genelet admin code.

Acceptance:

- ADR 0001 records the future taxonomy direction.
- `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` remains the current
  runtime direct SSP boundary until a later schema/cache milestone.
- M35 remains the separate SSP account/schema ADR.

Result:

- [ADR 0001](../docs/adr/0001-richer-supply-taxonomy.md) keeps `pub`,
  `pub_site`, and `pub_slot` as the publisher and inventory ownership boundary.
- Future taxonomy is additive on existing publisher tables and covers site/app
  identity, integration mode, slot/media intent, and quality/source metadata.
- M34 changes docs and memory only; schema, cache payloads, runtime behavior,
  audit payloads, ledgers, and Summer/Genelet admin code remain unchanged.
````

## Status record

````markdown
# Status M34 - Richer Supply Taxonomy ADR

State: `[+]` Completed

Record the future direct SSP supply taxonomy direction without changing schema,
cache payloads, runtime behavior, or Summer/Genelet admin code.

## Tasks

- `[+]` Create the M34 status file.
- `[+]` Write the richer supply taxonomy ADR.
- `[+]` Update direct SSP docs and memory-bank direction.
- `[+]` Check evolution history and add a new version for the taxonomy
  direction decision.
- `[+]` Run closeout verification.
- `[+]` Mark milestone complete after verification.

## Acceptance

- `[+]` The ADR keeps `pub`, `pub_site`, and `pub_slot` as the publisher and
  inventory ownership boundary.
- `[+]` The ADR recommends additive nullable/defaulted future fields instead of
  replacing the current `/pz` contract.
- `[+]` The ADR covers site/app identity, integration mode, slot/media
  taxonomy, quality/source taxonomy, cache impact, audit impact, admin UI
  impact, and migration path.
- `[+]` The ADR states that `source:"ssp"` and `contract:"pz-v1"` remain the
  current runtime audit boundary until a later schema/cache milestone.
- `[+]` M34 changes docs and memory only; tracked schema, cache payload,
  runtime, and admin UI implementation files stay unchanged.

## Verification

- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `git diff --check`
- `[+]` Manual acceptance check: ADR covers taxonomy fields, cache impact,
  audit impact, admin UI changes, and migration path; no tracked schema/runtime
  files changed for implementation.

## Notes

- M34 is ADR-only. M35 remains the separate SSP account/schema ADR and does not
  implement a separate account role.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M34 status file. | `[+]` | |
| Write the richer supply taxonomy ADR. | `[+]` | |
| Update direct SSP docs and memory-bank direction. | `[+]` | |
| Check evolution history and add a new version for the taxonomy | `[+]` | |
| Run closeout verification. | `[+]` | |
| Mark milestone complete after verification. | `[+]` | |
| The ADR keeps `pub`, `pub_site`, and `pub_slot` as the publisher and | `[+]` | |
| The ADR recommends additive nullable/defaulted future fields instead of | `[+]` | |
| The ADR covers site/app identity, integration mode, slot/media | `[+]` | |
| The ADR states that `source:"ssp"` and `contract:"pz-v1"` remain the | `[+]` | |
| M34 changes docs and memory only; tracked schema, cache payload, | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `git diff --check` | `[+]` | |
| Manual acceptance check: ADR covers taxonomy fields, cache impact, | `[+]` | |

````
