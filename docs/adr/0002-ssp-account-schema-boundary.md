# ADR 0002: SSP Account And Schema Boundary

Date: 2026-05-13

Status: Accepted

## Context

Direct SSP traffic currently enters Aofei through `POST /pz`. ADX/OpenRTB
traffic remains on `/bid/{domain}`. The v1 `/pz` contract uses packed direct
publisher tokens for supply identity, and request/response/attribute audits
separate direct SSP traffic with `source:"ssp"` and `contract:"pz-v1"`.

The active schema already has publisher account and inventory ownership through
`pub`, `pub_site`, and `pub_slot`. Summer/Genelet exposes publisher-facing
site/slot flows through the existing `pub` role, and runtime matching, ACL,
cache, tracking, and ledger joins already use the publisher/site/slot identity
chain. M34 accepted ADR 0001 for richer supply taxonomy as additive future
fields on those existing publisher tables.

M35 decides whether a separate `ssp` account role or separate SSP-owned schema
boundary is still needed after M32-M34.

## Decision

Do not add a separate `ssp` account role or separate SSP-owned inventory schema
for the current direct SSP path. Keep `pub`, `pub_site`, and `pub_slot` as the
publisher account and inventory ownership boundary.

`/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` remains the current
runtime source boundary. Future taxonomy and reporting metadata should be added
to the existing publisher model additively, following ADR 0001, unless a later
milestone reopens this decision with new product requirements.

M35 changes no schema, runtime behavior, cache payloads, audit payloads, ledger
tables, or Summer/Genelet admin code.

## Rationale

The existing `pub` boundary already represents the actor that owns publisher
inventory and receives publisher-facing admin access. Creating a parallel `ssp`
role for the current `/pz` path would duplicate login, cookie, issuer,
templates, component permissions, and account lifecycle rules without a distinct
owner.

The existing schema and runtime also already key the important contracts by
publisher, site, and slot:

- direct tokens pack `pub_id`, `site_id`, `slot_id`, and `size_id`;
- ACL and channel matching already use publisher and site identity;
- the direct publisher cache `pubmap:by-id` is derived from existing publisher
  cache data;
- tracking and ledger inputs already carry publisher, site, slot, size, demand,
  and price facts;
- M30 audit metadata already separates the `/pz` source from ADX/OpenRTB
  traffic.

The missing concepts identified during M34 are taxonomy and reporting
dimensions, not a different account owner. They are better represented as
nullable or defaulted fields on the current publisher tables than as a second
account tree.

## Future Reconsideration Triggers

A later milestone should reconsider a separate SSP account or schema boundary
only if at least one of these becomes a concrete requirement:

- a legal or settlement owner differs from the publisher inventory owner;
- reseller, network, or intermediary accounts need to own many publishers
  without being the publishers themselves;
- SSP operators need materially different authentication, permissions, or
  lifecycle workflows from publisher users;
- compliance or reporting requires isolation that cannot be represented by
  additive fields on `pub`, `pub_site`, or `pub_slot`;
- third-party API partners need independent credentials, quotas, or billing that
  should not belong to publisher accounts.

Until one of those requirements exists, a separate account role is more
complexity than boundary.

## Consequences

Future SSP schema work should extend the current publisher tables and direct
publisher cache additively. Summer/Genelet changes should add controlled fields
to existing publisher/site/slot forms rather than creating a new role surface.

This keeps current direct SSP serving stable, avoids duplicating account
infrastructure, and preserves current reporting joins. The tradeoff is that any
future SSP-intermediary business model will need a new ADR or schema milestone
before implementation.
