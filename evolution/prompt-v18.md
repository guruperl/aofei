# Prompt V18

Close the direct SSP account/schema boundary question as an ADR-only milestone.

The decision is to keep the existing `pub` role as the publisher account and
inventory owner for the current `/pz` direct SSP path. Do not add a separate
`ssp` account role or separate SSP-owned inventory schema in M35.

The ADR should record why existing `pub`, `pub_site`, `pub_slot`, audit
`source:"ssp"`/`contract:"pz-v1"`, and the M34 additive taxonomy direction
cover current needs. It should also list concrete future triggers that would
justify reopening the account-boundary decision.
