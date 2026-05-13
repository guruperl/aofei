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
