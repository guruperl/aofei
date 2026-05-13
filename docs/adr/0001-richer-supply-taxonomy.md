# ADR 0001: Richer Supply Taxonomy

Date: 2026-05-13

Status: Accepted

## Context

Direct SSP traffic currently enters Aofei through `POST /pz` while ADX/OpenRTB
traffic enters through `/bid/{domain}`. The v1 `/pz` contract uses packed
tokens for trusted supply identity:

- `site` packs `(pub_id, site_id)`;
- `adUnits[].slot` packs `(slot_id, size_id)`;
- `adUnits[].code` is only a publisher-page DOM/debug id.

The existing `pub` role owns publisher accounts and inventory. `pub_site` and
`pub_slot` own site/app and slot records. M31-M33 deliberately kept richer
supply source fields out of runtime schema/cache work and identified direct SSP
traffic with the `/pz` entrypoint plus audit metadata:
`source:"ssp"` and `contract:"pz-v1"`.

The current schema has useful legacy fields such as `site_type`, `site_url`,
`foreign_id`, `qa_slot`, `qa_device`, and `qa_position`, but these fields pack
multiple concepts together. Future reporting, policy, and admin workflows need
clearer taxonomy without replacing the working `/pz` v1 contract or creating a
new account role prematurely.

## Decision

Keep `pub`, `pub_site`, and `pub_slot` as the future ownership boundary for
direct publisher inventory. Add richer supply taxonomy later as nullable or
defaulted fields on the existing publisher tables, not as a replacement for the
current `/pz` tokens and not as a new account role in this ADR.

M34 does not change schema, cache payloads, runtime behavior, audit payloads, or
Summer/Genelet admin code. Until a later schema/cache milestone implements this
taxonomy, `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` remains the
runtime boundary for direct SSP traffic.

## Future Taxonomy Direction

### Site And App Identity

Normalize existing site/app hints into explicit inventory identity semantics on
`pub_site`.

Recommended future fields:

- inventory environment, such as web, app, connected TV, digital out-of-home,
  or other future environments;
- canonical inventory identity, derived from current `site_url` and
  `foreign_id` usage, such as host/domain for web inventory and bundle/app id
  for app inventory;
- optional display name or label separate from the canonical identity;
- optional store URL or app/site URL for review workflows.

`site_type`, `site_url`, and `foreign_id` can seed these fields during
migration, but the new fields should make the meaning explicit instead of
requiring callers to infer whether a value is a host, app bundle, external
platform id, or display label.

### Integration Mode

Record how inventory reaches Aofei without changing account ownership.

Recommended future values distinguish:

- ADX/domain traffic served through `/bid/{domain}`;
- direct browser tags served through `/pz`;
- SDK/app traffic served through `/pz`;
- future server-to-server or API traffic;
- future special integrations only when they carry different validation or
  operational semantics.

Integration mode is taxonomy, not authorization by itself. Runtime policy still
depends on entrypoint, packed tokens, cache validation, origin/referrer checks,
and configuration gates until later milestones intentionally change that.

### Slot And Media Taxonomy

Keep request-supplied `mediaTypes` as runtime request metadata and add durable
slot intent separately on `pub_slot`.

Recommended future fields:

- intended media support, such as banner, video, native, audio, or multi-format;
- placement context, such as above fold, in-feed, interstitial, rewarded,
  sticky, popup, or other controlled values;
- player or render context where it affects policy or pricing;
- slot refresh behavior and ad-density hints where needed for quality review;
- allowed configured sizes or size policy separate from the existing packed
  slot token.

Runtime validation can continue to trust the packed `size_id` and cached slot
metadata while the taxonomy is introduced. Later runtime work can decide which
taxonomy fields become bidding, pricing, or policy inputs.

### Quality And Source Taxonomy

Preserve the intent of legacy `qa_slot` categories as explicit future fields
rather than continuing to pack unrelated ideas into one value.

Recommended future field groups:

- traffic quality, such as human-reviewed, sampled, suspicious, or blocked;
- source quality, such as owned-and-operated, partner, network, resale, or
  unknown;
- page/app experience flags, including popup usage, ad density, auto-refresh,
  incent/rewarded behavior, and intrusive placement controls;
- management control, such as publisher-managed, operator-managed,
  partner-managed, or unknown;
- device and position defaults seeded from existing `qa_device` and
  `qa_position` where they are still useful.

These fields should be controlled enum-like values in admin forms and stable
cache payloads, not free-form strings added ad hoc to request handling.

## Cache Impact

Future cache work should extend the direct publisher cache additively.
`pubmap:by-id` can include a taxonomy object or clearly named optional fields
inside existing publisher/site/slot metadata. Old consumers must be able to
ignore missing or unknown taxonomy fields until runtime starts using them.

Spread/local mode should continue to derive direct publisher lookup data from
the existing `pubmap` snapshot unless a later milestone identifies a concrete
need for a separate spread family. Cache versioning is required only when a
future change makes old readers unable to safely ignore the new fields.

## Audit And Reporting Impact

M34 adds no ledger or audit schema change.

After schema and cache support exists, request/response/attribute audit payloads
may include a taxonomy object or normalized fields so operators can separate
web/app, browser tag/SDK/API, media intent, and quality/source dimensions in
analysis. That later work must preserve the current entrypoint boundary:
`source:"ssp"` and `contract:"pz-v1"` identify current direct SSP traffic until
a future contract is explicitly introduced.

Ledger aggregation should not change only because taxonomy fields exist. A
future reporting milestone can decide whether any taxonomy dimensions belong in
ledger tables, derived reports, or offline analytics only.

## Admin UI Impact

Later Summer/Genelet work should add controlled selects and constrained inputs
to the existing `pub`, `pub_site`, and `pub_slot` forms. It should not introduce
a separate `ssp` account role as part of taxonomy collection.

Recommended admin changes:

- site/app forms expose inventory environment and canonical identity fields;
- slot forms expose intended media, placement, refresh, density, and quality
  controls;
- defaults preserve current behavior for existing publishers;
- help text and validation describe operational meaning without changing the
  `/pz` tag contract.

## Migration Path

1. Add nullable or defaulted taxonomy columns to existing publisher tables in a
   future schema milestone.
2. Backfill from existing `site_type`, `site_url`, `foreign_id`, `qa_slot`,
   `qa_device`, and `qa_position` using conservative mappings and explicit
   unknown/default values where historical data is ambiguous.
3. Publish taxonomy in the direct publisher cache additively.
4. Compare request, response, and attribute audit output before and after cache
   publication to confirm no behavior change.
5. Add Summer/Genelet controlled selects and validation to the existing
   publisher/site/slot forms.
6. Decide, after runtime and reporting consumers have migrated, whether legacy
   packed fields remain compatibility-only, stay as primary fields, or can be
   retired in a later cleanup.

## Consequences

This keeps the current direct SSP implementation stable and preserves the `pub`
ownership model while giving later schema and UI milestones a concrete target.
It also avoids overloading `source:"ssp"` with inventory taxonomy: audit source
continues to identify the runtime entrypoint, while future taxonomy fields
describe the inventory and integration details behind that entrypoint.
