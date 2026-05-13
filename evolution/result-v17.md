# Result V17

M34 selected a future direct SSP taxonomy direction without changing runtime
contracts.

Implemented direction:

- Added ADR 0001 for richer supply taxonomy.
- Kept `pub`, `pub_site`, and `pub_slot` as the publisher and inventory
  ownership boundary.
- Recommended additive nullable/defaulted future fields for site/app identity,
  integration mode, slot/media taxonomy, and quality/source taxonomy.
- Kept `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` as the current
  runtime direct SSP boundary.
- Deferred schema, cache payload, audit payload, ledger, and Summer/Genelet UI
  implementation to later milestones.

Deferred direction:

- M35 remains the separate ADR milestone for deciding whether an SSP
  account/schema boundary is still needed.
