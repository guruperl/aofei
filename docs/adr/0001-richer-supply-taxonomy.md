# ADR 0001: Richer Supply Taxonomy

Date: 2026-05-13

Status: Accepted and implemented by P02

Related ADRs:

- ADR 0002 refines the account/schema boundary and keeps the current direct SSP
  path on the existing publisher ownership model.

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

Keep `pub`, `pub_site`, and `pub_slot` as the ownership boundary for direct
publisher inventory. Add richer supply taxonomy as defaulted fields on the
existing publisher tables, not as a replacement for the current `/pz` tokens
and not as a new account role.

M34 made the decision without changing runtime code. P02 implements it
additively in schema, publisher cache, privacy-safe attribute audits, R02 report
facts, Summer/Genelet forms, seller authorization, and OpenRTB
`source.schain`. `/pz` plus audit `source:"ssp"` and `contract:"pz-v1"`
remains the runtime entrypoint boundary for direct SSP traffic.

## Implemented Taxonomy

### Site And App Identity

Normalize existing site/app hints into explicit inventory identity semantics on
`pub_site`.

Implemented controlled fields:

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

Implemented values distinguish:

- ADX/domain traffic served through `/bid/{domain}`;
- direct browser tags served through `/pz`;
- SDK/app traffic served through `/pz`;
- future server-to-server or API traffic;
- future special integrations only when they carry different validation or
  operational semantics.

Integration mode is taxonomy, not authorization by itself. Runtime policy still
depends on entrypoint, packed tokens, cache validation, origin/referrer checks,
and configuration gates. P02 uses approved taxonomy for disclosure and
reporting; it does not let a client-provided category grant inventory access.

### Slot And Media Taxonomy

Keep request-supplied `mediaTypes` as runtime request metadata and add durable
slot intent separately on `pub_slot`.

Implemented controlled fields:

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

Preserve the intent of legacy `qa_slot` categories as explicit controlled
fields rather than continuing to pack unrelated ideas into one value.

Implemented controlled field groups:

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

### Seller Transparency And Supply Chain

The existing publisher account owns public seller id, `Publisher` or
`Intermediary` type, advertising-system domain (ASI), name, and domain. A
publisher may propose values, but only an operator-approved exact tuple is
authorized for disclosure; later publisher edits revoke that approval. This
metadata is transparency, not payment authority, and cannot change the A01
publisher settlement owner.

Direct `/pz` requests never supply trusted seller or chain data. An authorized
publisher produces a one-node complete standard `source.schain`; an authorized
intermediary produces an incomplete chain unless a real upstream owner is
recorded. Unauthorized metadata produces no chain. Middleman fanout preserves
only a valid bounded standard chain and removes `source.pchain` and node
extensions.

## Cache Impact

P02 extends both direct publisher Gob cache views additively with seller,
site, and slot supply objects. Old consumers ignore the fields. New consumers
normalize a compatible older generation to explicit `Unknown` categories and
never infer seller authorization.

Spread/local mode continues to derive direct publisher lookup data from the
existing `pubmap` snapshot unless a later milestone identifies a concrete
need for a separate spread family. Cache versioning is required only when a
future change makes old readers unable to safely ignore the new fields.

## Audit And Reporting Impact

P02 adds the allowlisted taxonomy to privacy-safe attribute audits and to the
derived R02 delivery report. It does not add raw request identity, consent,
private partner metadata, or a new financial source. Authorized public seller
id/type and controlled supply categories are reportable; missing legacy values
remain explicit unknowns. `source:"ssp"` and `contract:"pz-v1"` continue to
identify the direct SSP entrypoint.

## Admin UI Impact

P02 adds controlled selects and constrained inputs to the existing `pub`,
`pub_site`, and `pub_slot` forms. It does not introduce a separate `ssp`
account role as part of taxonomy collection.

Implemented UI rules:

- site/app forms expose inventory environment and canonical identity fields;
- slot forms expose intended media, placement, refresh, density, and quality
  controls;
- defaults preserve current behavior for existing publishers;
- help text and validation describe operational meaning without changing the
  `/pz` tag contract.

## Populated-System Migration Contract

1. Add defaulted taxonomy and seller columns to existing publisher tables.
2. Backfill from existing `site_type`, `site_url`, `foreign_id`, `qa_slot`,
   `qa_device`, and `qa_position` using conservative mappings and explicit
   unknown/default values where historical data is ambiguous.
3. Publish taxonomy in the direct publisher cache additively and require
   explicit operator seller authorization.
4. Compare request, response, and attribute audit output before and after cache
   publication to confirm no behavior change.
5. Add Summer/Genelet controlled selects, validation, and hostile-output tests
   to the existing publisher/site/slot forms.
6. Decide, after runtime and reporting consumers have migrated, whether legacy
   packed fields remain compatibility-only, stay as primary fields, or can be
   retired in a later cleanup.

## Consequences

This keeps the direct SSP implementation stable and preserves the `pub`
ownership model while giving reports and external bidders a controlled public
supply description.
It also avoids overloading `source:"ssp"` with inventory taxonomy: audit source
continues to identify the runtime entrypoint, while taxonomy fields
describe the inventory and integration details behind that entrypoint.
