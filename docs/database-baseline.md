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
| Base tables | 57 |
| Views | 1 |
| Stored routines | 6 |
| Triggers | 18 |
| Events | 0 |
| Advertisers | 0 |
| Publishers | 0 |
| Advertiser bidder endpoints | 0 |

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
