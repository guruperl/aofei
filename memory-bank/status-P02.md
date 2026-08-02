# Status P02 - Supply Metadata And Seller Transparency

State: `[+] Complete`

## Goal

Implement richer supply identity and seller transparency on the existing
publisher ownership boundary.

## Dependencies

- P01 direct SSP readiness.
- S01 privacy and disclosure policy.
- ADR 0001 and ADR 0002 remain the starting decisions.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Taxonomy contract | `[+]` | Closed site/app environment, canonical identity, integration, media, placement, render, refresh, density, traffic/source quality, management, and seller vocabularies are validated in Aofei and Summer. |
| Schema and UI | `[+]` | Defaulted `pub`/`pub_site`/`pub_slot` fields, publisher forms, operator review pages, hostile-output tests, and a seller-approval revocation trigger retain the existing publisher role. |
| Cache and audits | `[+]` | Both publisher Gob views carry additive metadata; old/new readers are tested both ways, missing values normalize to `Unknown`, and runtime attributes clear unapproved seller data before privacy-safe audit. |
| Seller identity | `[+]` | Publisher-proposed id/type/ASI/name/domain require exact operator authorization; edits revoke approval and transparency never changes the A01 settlement owner. |
| Supply chain | `[+]` | Direct `/pz` derives owned/intermediary chains only from approved cache state; middleman fanout validates bounded standard chains, requires `hp=1`, and strips `pchain`/extensions. |
| Reporting | `[+]` | Signed measurement lineage and `report_delivery` retain every approved supply category, refresh seconds, and authorized seller id/type under a complete SHA-256 dimension identity. |

## Acceptance Criteria

- Existing `/pz` clients and cache readers remain compatible during rollout.
- Seller and supply-chain claims are derived from approved data, not trusted
  directly from browser input.
- No separate account role is added unless a new ADR documents a concrete
  legal, settlement, intermediary, or permission requirement.

## Verification

- Schema reset/load/check/diff, Summer CRUD, cache compatibility, audit,
  OpenRTB `schain`, privacy, and full repository closeout gates.

## Reconciliation From S01

- Seller transparency is public commercial metadata, but source input still
  passes through S01 minimization. `schain` and seller claims must be derived
  from approved server state and included in partner-specific sanitation tests.

## Reconciliation From S04

- Seller names, domains, supply-chain identifiers, placement labels, and public
  transparency metadata remain ordinary escaped strings in publisher/operator
  templates. A seller URL is not browser-fetched or made executable merely
  because it is approved commercial metadata.
- Schema/UI acceptance includes hostile HTML, attribute, query, and report
  fixtures for every new displayed supply field.

## Reconciliation From A01

- Seller/reseller metadata must map to the existing publisher party used by
  A01 statements, or a new ADR must define a genuine legal/settlement party.
  `schain` seller identity cannot silently redirect publisher pay or create a
  second settlement owner.
- New revenue-share or reseller-fee formulas require a versioned A01 accounting
  contract, immutable source facts, and migration; transparency metadata alone
  is not payment authority.

## Reconciliation From D02

- New placement, environment, refresh, quality, or supply-chain fields may
  further restrict eligible creative media and markup but cannot weaken D02's
  exact size/MIME/secure/source validation or trust browser-supplied seller
  claims as creative approval.
- Cache evolution must preserve the D02 cache-first media-metadata generation
  and RAdv v2 compatibility order while adding publisher fields additively.

## Reconciliation From O02

- The populated-schema audit, additive cache publication, canary, and rollback
  must use lifecycle readiness and preserve N-1 capacity. Missing, stale, or
  future local generations keep a node out of service; a failed publication
  leaves the prior compatible generation available.
- Backup/restore inventories include the approved seller and supply-chain
  metadata, while derived Redis/spread caches are rebuilt after authoritative
  MySQL restore. Public transparency data does not include backup identifiers,
  credentials, or recovery evidence.

## Reconciliation From D03

- Supply-chain and seller fields disclosed to an external bidder must be
  rebuilt from server-approved P02 state inside D03's contextual,
  per-impression request. Browser/app input, unknown `source.ext`, and private
  publisher or partner metadata cannot pass through merely because a route
  matched.
- The fallback canary must cover each supported ownership/reseller chain and
  partner policy before disclosure. Route targets narrow eligible supply; they
  do not authorize an unapproved seller claim or a different settlement owner.

## Reconciliation From R02

- Approved P02 taxonomy fields extend the R02 supply dimensions additively and
  only after schema/cache/report readers are compatible. Public seller,
  ownership/reseller, placement, integration, and quality categories may be
  reported; private contracts, endpoints, credentials, raw supply-chain input,
  or account identity never become advertiser dimensions.
- New dimension values preserve `report_delivery` UTC interval identity,
  advertiser/publisher scope, source freshness, and the measured query/retention
  review. Update the R02 benchmark before adding indexes, summaries, or OLAP;
  missing pre-migration values remain explicit unknown/default categories, not
  silently attributed supply.

## Verification Results

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. The
  documented Aofei scoped race suite and full pzdesign race suite passed.
- Pinned staticcheck v0.5.1 passed for Aofei and for pzdesign with its documented
  `ST1000`/`ST1003`/`ST1006` legacy exclusions.
- The SQL guard and a clean-room MySQL 8.0.41 import passed with 69 base tables,
  6 routines, and 26 triggers. The disposable dump/restore drill preserved
  accounting, action, report/supply, experiment, cache, and middleman evidence.
  No configured local or production container was reset or modified.
- Real disposable-MySQL ledger writes and all Summer advertiser, publisher, and
  operator report queries passed with the expanded supply dimensions. The
  100,000-row/five-run benchmark measured advertiser 100/119 ms, publisher
  105/118 ms, and operator 1684/1830 ms median/max on x86-64 with eight visible
  CPUs; these are local measurements, not production p95/p99.
- Publisher and direct-publisher Gob compatibility, seller authorization,
  hostile metadata, direct/client trust, owned/intermediary `schain`, audit,
  signed reporting, and refresh-identity tests passed. Both template surfaces
  (263 templates), public-copy/data guards, Aofei docs, and all three repository
  diff-hygiene checks passed.

## Review Closeout

- Deep review extended the read-only publisher manifest with exact seller and
  supply state, made operator site/slot inspection readable, prevented pending
  seller proposals from entering runtime audits, required standard chain
  `hp=1`, rejected seller-domain IP literals/control characters, and carried
  refresh seconds through signed measurement, storage, hashes, SQL, benchmarks,
  and UI.
- S02, I03, S03, A02, and I02 now carry explicit P02 reconciliation. P02
  implements the already accepted ADR 0001/0002 direction without changing the
  product ownership boundary, so no new evolution version is warranted.
- No commit was created because the active goal's commit policy is `none`.
