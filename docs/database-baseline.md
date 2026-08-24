# Database Baseline

The active local database baseline is:

```text
etc/step4_init.sql
```

It contains the active schema and non-sensitive reference catalogs. Mutable
development accounts and bid-path data are loaded separately from the fully
synthetic `etc/demand.sql` fixture.

## Rules

- `etc/step4_init.sql` must recreate the active local schema.
- It should include tables, views, routines, triggers, and non-sensitive
  reference catalogs needed by the local package.
- It must not contain account, campaign, publisher, ledger, login, traffic,
  uploaded-media, production-derived, or personal records.
- It must not contain explicit legacy MySQL definers.
- It must not require or recreate legacy named MySQL users.
- Tables use `utf8mb4` with `utf8mb4_0900_ai_ci` collation for the MySQL 8
  local and production baseline.
- Local Docker auth is created by `scripts/aofei-local.sh`.

## Reset And Load

```bash
./scripts/aofei-local.sh reset
./scripts/aofei-local.sh load
./scripts/aofei-local.sh status
```

Sample data can be added after the baseline:

```bash
./scripts/aofei-local.sh sample
```

The sample loader checks the complete synthetic advertiser, publisher, admin,
campaign, item, creative, site, and slot fixture instead of using broad table
counts. It refuses a partial/conflicting fixture rather than overwriting local
rows.

Or together:

```bash
./scripts/aofei-local.sh reset-sample
```

`load` is intentionally replay-safe, not idempotent. It exits before import if
the target database already has schema objects. Use `reset && load` when
rebuilding the baseline.

## Schema Stewardship Commands

Guard the checked-in baseline against legacy dump metadata:

```bash
./scripts/aofei-local.sh check-sql
```

This fails on explicit `DEFINER=` clauses and legacy account-name references,
while allowing `SQL SECURITY DEFINER` syntax that has no user-bound definer.

Dump the current Docker MySQL schema:

```bash
./scripts/aofei-local.sh dump-schema
```

The normalized dump is written under ignored `.local/schema/`, with volatile
dump comments, dump definers, `AUTO_INCREMENT=N`, and temporary database names
removed.

Compare the current Docker schema to a fresh database rebuilt from
`etc/step4_init.sql`:

```bash
./scripts/aofei-local.sh diff-schema
```

The comparison includes base tables, views, stored routines, triggers, and
events. The temporary baseline-check database is dropped when the command exits.

## Baseline Inventory

After `reset && load`, the expected inventory is:

| Object/data | Count |
|---|---:|
| Base tables | 95 |
| Views | 0 |
| Stored routines | 6 |
| Triggers | 61 |
| Events | 0 |
| Advertisers | 0 |
| Publishers | 0 |
| Publisher request credentials | 0 |
| Advertiser bidder endpoints | 0 |
| Accounting statements/adjustments/audits | 0 |
| Action measurement touches/facts | 0 |
| Delivery report/experiment facts | 0 |
| Analysts/identity sessions/grants/security audits | 0 |
| Traffic-quality rules/decisions/evidence/cases/enforcements/billing/audits | 0 |
| Hosted-payment bindings/operations/objects/events/reconciliations/audits | 0 |

After `sample`, the expected local-only identities are `admin_local`,
`advertiser@example.test`, and `publisher@example.test`; their documented
development password is `local-demo-password`. Middleman AdX schema is present
but starts empty. `adv_bidder` stores
advertiser-owned OpenRTB endpoint metadata and optional synthetic
campaign/item/creative reporting IDs. `mid_route_*` tables store
operator-controlled fallback route configuration. `ledger_mid` and `daily_mid`
store middleman callback-derived reporting facts for advertiser pay-side reports
and admin charge/pay/margin settlement views. `mid_callback_retry` stores
auditable retry rows for retryable downstream middleman callback forwarding
failures without foreign keys so rows remain inspectable if route or bidder rows
are later removed. Retry workers set `claimed_at` while processing rows; stale
`Processing` rows whose claim age exceeds the retry worker stale threshold are
eligible for reclaim.

Publisher slots store `pub_slot.size_id` as the packed width/height used by
publisher-facing direct SSP tags. The default baseline value is `4194368`
(`64x64`) for historical rows and for insert paths that do not provide an
explicit size.

P02 extends the existing publisher boundary additively. `pub` stores public
seller id/type/ASI/name/domain plus an operator authorization flag; a change to
an already authorized seller tuple revokes authorization. `pub_site` stores the
controlled inventory environment, canonical identity, optional public review
URL, and integration mode. `pub_slot` stores controlled media intent,
placement, render context, refresh, density, traffic/source quality, and
management-control fields. Publisher/site/slot changes enqueue the ordinary
publisher-cache refresh path. These fields do not create another account or
settlement owner. Populated systems require a conservative backfill with
explicit `Unknown` values and operator reapproval; never infer authorization
from legacy free-form fields.

D01 delivery enforcement adds `adv_balance.current_day` for UTC daily-ledger
baselines; `adv_campaign.delivery_timezone`, `weekly_schedule`, and
`pacing_mode`; and matching weekly schedule/pacing fields on `adv_item`.
Start/end timestamps remain the existing campaign/item columns and are treated
as UTC by the runtime. These fields are part of the RAdv version-2 cache
compiler contract. Existing populated databases require a deployment-managed
migration before new cache compiler or Summer binaries are enabled; the
baseline file is not a production migration script.

D02 changes the `adv_item.cost_type` default from the historical `CPC` value to
`CPM` while retaining all four enum labels for readable populated-schema
compatibility. Runtime and cache compilation accept only reviewed positive USD
CPM rows; they do not reinterpret legacy CPC/CPA/ROI values. Creative cache
records add media type and first-media MIME without changing the payload
version, and active rows must satisfy the source/URL/size/weight contract before
publication. The creative compiler does not reinterpret the campaign's
`foreign_id` external business identifier as a Native fallback URL. Populated
systems must run the audit, item-specific migration, and
cache-first rollout in
[auction-pricing-creatives.md](auction-pricing-creatives.md); replaying this
baseline or changing the enum default alone does not migrate stored demand.

Publisher `pub_id` values are database identities, not credentials or secrets.
The legacy helper that creates a publisher plus default inventory uses a
cryptographically generated nonzero 32-bit value and retries a detected primary
key collision; another unique-key violation remains an error. Authorization
must continue to use the authenticated role/account context rather than relying
on an unguessable numeric ID.

A01 adds the singleton `acct_contract` row with
`usd-cpm-impression-v2`, plus empty `acct_statement`, `acct_adjustment`, and
`acct_audit` tables. Adjustment and audit triggers reject update/delete. The
inactive `pay_*` compatibility tables contain only non-identity reference and
status metadata: full card/bank fields, sender identity, and payment IP fields
are absent. The legacy `view_payment` and balance-crediting `trig_payment` are
also absent, and no Summer module routes to these tables. A populated v1 system
must follow the reviewed conversion and sensitive-field retirement procedure in
[accounting-settlement.md](accounting-settlement.md); replaying the baseline is
not a migration.

R01 adds empty `measurement_touch` and `measurement_action` tables. Signed
impression/click lineage is separated from advertiser action idempotency,
attribution, and a distinct domain-separated pseudonym. Both tables have
explicit expiry and no foreign keys to mutable campaign/account rows so an
analytical historical fact never blocks account cleanup. They contain no D01
reservation identity and are not referenced by A01 statement or settlement
queries. Populated systems require a reviewed online migration and backfill
policy; never replay the baseline. See
[conversion-attribution.md](conversion-attribution.md).

R02/P02 add empty `report_delivery`, `report_experiment`,
`report_experiment_variant`, `report_exposure`,
`report_experiment_outcome`, and `report_experiment_audit` tables. Delivery
facts retain explicit demand, route, coarse geo/device, supply taxonomy, and
authorized seller dimensions. Their unique identity is UTC interval plus a
deterministic SHA-256 `dimension_hash` over the complete dimension tuple; the
generated two-byte `device_key` remains a lookup compatibility field.
Exposure and outcome rows reject updates and remain append-only until their
bounded expiry or an exact authorized subject deletion; audit rows are
immutable. Outcome values are idempotent, six-decimal observations linked to
an existing pseudonymous exposure and are not auction, delivery, or accounting
authorities. Apply the populated-system
migration before deploying ledger/report binaries; never replay the baseline.
See [marketplace-analytics-experiments.md](marketplace-analytics-experiments.md).

R03 adds `report_experiment.assignment_algorithm_version`. Its compatibility
default is v1 for pre-R03 rows; trusted new creates always store v2 and generate
their own salt. Four triggers freeze the complete experiment-version contract,
enforce forward state transitions, and prevent variant changes after Draft, so
the active baseline is 95 tables, 6 routines, and 61 triggers.

O02 makes `ledger_log.timely` and `daily_log.daily` unique. The operational
Redis lease is renewable but cannot prove exclusive ownership during every
partition; these constraints are the durable backstop that rejects a second
source row for the same interval or UTC day. Audit populated systems for
duplicates and resolve them through an approved accounting reconciliation
before adding either constraint. Do not delete conflicting evidence merely to
make the migration pass. The clean-room drill in
[single-region-availability.md](single-region-availability.md) verifies both
identities after restore.

S02 adds the `analyst`, `auth_mfa`, `auth_recovery_code`, `auth_session`,
`auth_permission_grant`, and `auth_security_audit` tables. TOTP secrets are
AES-256-GCM ciphertext, while recovery codes and opaque sessions are stored as
fixed-length keyed digests. Analysts require an exact active permission and
resource grant in addition to their role permission. The two
`auth_security_audit` triggers reject update/delete except for the bounded
retention command's connection-local maintenance flag. Production grants must
also deny `UPDATE` and `DELETE` to the HTTP application principal and give the
retention command a separate maintenance principal/configuration. A populated
system must apply a reviewed online migration before enabling Summer
`Identity`; replaying the baseline is not a production migration. See
[identity-access-security.md](identity-access-security.md).

P03 adds `pub_request_credential` for SDK/server request authentication. Each
row belongs to one existing publisher and one approved App site, stores a
32-byte Ed25519 public verifier plus expiry/rotation/revocation metadata, and
never stores the private seed, raw signature, request body, or replay nonce.
The composite site/publisher foreign key prevents a cross-account scope; the
existing immutable `auth_security_audit` table records issue, rotation, and
revocation with a domain-separated object hash. A populated deployment must
apply the reviewed additive table, composite `pub_site` owner key, and exact
S02 permission grants before enabling `direct_ssp_auth`; never replay the clean
baseline as a production migration. See
[direct-ssp-authenticity.md](direct-ssp-authenticity.md).

I03 adds `api_credential`, `api_idempotency`, `api_operation`, and `api_audit`,
plus an `api_version` column and before-update version trigger on
`adv_campaign`, `adv_item`, and `adv_creative`. Bearer and idempotency values
are deployment-keyed digests; completed write responses are bounded to the
24-hour retry window. Each operation has an opaque publication-generation token
so only mutations visible before a cache build can be marked active afterward.
API audit is immutable except for its separated bounded retention gate. A
populated migration must backfill every version to one,
create the four tables and five triggers online, and finish before API
enablement. Older portal binaries tolerate the additive columns/triggers, but
the trigger must remain while API clients depend on optimistic conflicts. See
[advertiser-management-api.md](advertiser-management-api.md).

S03 adds empty `quality_rule`, `quality_decision`, `quality_evidence`,
`quality_case`, `quality_case_event`, `quality_enforcement`, `quality_billing`,
`quality_counter`, and `quality_audit` tables. Twelve triggers prevent rule
behavior rewrites, decision mutation, evidence deletion outside the bounded
retention connection, case-event mutation, enforcement/billing identity
rewrites, and audit mutation. The two S05 protected-update triggers allow only
the documented enforcement rollback and billing review transitions while
keeping their decision, scope, statement, digest, disposition, and creation
facts fixed. Database checks also guarantee that incomplete evidence remains
observe-only, decision billing matches the applied action, and serving canary
state is unambiguous. Populated systems that already have S03 must add the two
S05 triggers while traffic-quality writes are paused; new systems apply the
reviewed nine-table/twelve-trigger migration before enabling
`traffic_quality`. Never replay the baseline as a production migration. See
[traffic-quality-anti-fraud.md](traffic-quality-anti-fraud.md).

A02 adds empty `hosted_binding`, `hosted_operation`,
`hosted_provider_object`, `hosted_event`, `hosted_reconciliation`, and
`hosted_audit` tables. Twelve triggers preserve binding/operation financial
identity, provider-object ownership, signed event evidence, reconciliation
facts, and audit history. The bounded event-retention gate cannot delete an
event still referenced by reconciliation evidence. Tables contain opaque
provider identifiers, hashes, an immutable operation-to-binding selection, and
a two-letter publisher onboarding country only—never API keys, webhook secrets,
signatures, raw bodies, card data, bank data, or identity documents. A
populated deployment must apply a reviewed additive migration while the
feature remains disabled; never replay the baseline. See
[hosted-funding-payout.md](hosted-funding-payout.md).

## Updating The Baseline

When an intentional schema change is made inside Docker MySQL:

1. Make the schema change in Docker MySQL.
2. Export the updated schema and required baseline data.
3. Strip explicit definers and legacy auth references.
4. Replace or patch `etc/step4_init.sql`.
5. Run `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`.
6. Run `./scripts/aofei-local.sh check-sql`.
7. Run `./scripts/aofei-local.sh diff-schema` and review any drift.
8. Update `etc/step5.notes`, this document, and the memory bank if the workflow
   or inventory changed.
