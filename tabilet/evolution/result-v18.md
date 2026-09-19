# Result V18

M35 closed the direct SSP account/schema boundary decision without changing
runtime contracts.

Implemented direction:

- Added ADR 0002 for the SSP account and schema boundary.
- Decided not to add a separate `ssp` account role or separate SSP-owned
  inventory schema for the current `/pz` path.
- Kept `pub`, `pub_site`, and `pub_slot` as the publisher account and inventory
  ownership boundary.
- Kept `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` as the current
  runtime direct SSP source boundary.
- Deferred any separate SSP account model until concrete legal, settlement,
  intermediary, permission, compliance, or partner-credential requirements
  exist.

Deferred direction:

- Future SSP schema work should follow the additive M34 taxonomy direction
  unless a later milestone reopens the account-boundary decision.
