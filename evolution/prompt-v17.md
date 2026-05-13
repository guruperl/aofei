# Prompt V17

Record the richer direct SSP supply taxonomy direction as an ADR-only
milestone.

Keep the existing `pub` role as the publisher account and inventory owner. Do
not change schema, cache payloads, runtime behavior, audit payloads, or
Summer/Genelet admin code in this milestone.

The ADR should recommend additive future fields on existing publisher tables,
cover site/app identity, integration mode, slot/media taxonomy,
quality/source taxonomy, cache impact, audit impact, admin UI impact, and
migration path. `source:"ssp"` and `contract:"pz-v1"` remain the current
runtime audit boundary until a later schema/cache milestone.
