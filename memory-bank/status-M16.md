# Status M16 - Middleman AdX Identity And Schema

## Goal

Create the database and admin identity foundation for middleman AdX fallback
without changing bid serving behavior yet.

## Completed

- `[+]` Downstream DSP account role.
  - Added `mid_dsp` as a separate partner account table.
  - Added `def_entitytype` row `6 -> mid_dsp.dsp_id`.
  - Added Summer/Genelet `dsp` role config mapped to `mid_dsp`.

- `[+]` Endpoint metadata.
  - Added `mid_bidder` for DSP-owned OpenRTB endpoint metadata.
  - DSP users can manage endpoint metadata; admins control credential refs,
    credential status, and activation.
  - Registered Summer `dsp` and `bidder` component modules.

- `[+]` Route and reporting schema.
  - Added `mid_route_group`, `mid_route_bidder`, and `mid_route_target` for
    future fallback route assignment.
  - Added `daily_mid_bidder` for future daily middleman reporting.

- `[+]` Docs and memory bank.
  - Added `docs/middleman-adx.md`.
  - Updated product, architecture, tech-stack, database, DSP workflow,
    Summer UI, milestone, and evolution docs.

## Carry Forward

- `[ ]` Build route cache and downstream OpenRTB fanout client in M17.
- `[ ]` Integrate fallback auction behavior into `ServeBid` in M18.
- `[ ]` Add callback proxying, audit, operations, and DSP reporting in M19/M20.

## Verification

- `[+]` `GOWORK=off go test ./summer ./summer/bidder ./summer/registry ./genelet`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load && ./scripts/aofei-local.sh diff-schema`
- `[+]` `GOWORK=off staticcheck ./summer ./summer/registry ./summer/dsp ./summer/bidder`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

Focused staticcheck for changed Summer packages passes. A broader exploratory
`GOWORK=off staticcheck ./summer ./summer/registry ./genelet` still reports
pre-existing Genelet findings and is not used as the M16 gate.
