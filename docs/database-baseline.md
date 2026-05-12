# Database Baseline

The active local database baseline is:

```text
etc/step4_init.sql
```

It was updated from the live database snapshot and then normalized for local
Docker MySQL use.

## Rules

- `etc/step4_init.sql` must recreate the active local schema.
- It should include tables, views, routines, triggers, and baseline data needed
  by the local package.
- It must not contain explicit legacy MySQL definers.
- It must not require or recreate legacy named MySQL users.
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

The sample loader checks for the default sample advertiser/campaign/item/creative
instead of using a broad advertiser row count. If those sample records are
already present, `etc/demand.sql` is skipped and the default publisher helper is
still run.

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
| Base tables | 56 |
| Views | 1 |
| Stored routines | 6 |
| Triggers | 18 |
| Events | 0 |
| Advertisers | 1 |
| Publishers | 14 |
| Downstream DSPs | 0 |

Middleman AdX schema is present but starts empty. `mid_dsp` stores downstream
DSP partner accounts for the Summer `dsp` role, `mid_bidder` stores their
OpenRTB endpoint metadata, `mid_route_*` tables store operator-controlled
fallback route configuration, and `daily_mid_bidder` is reserved for aggregated
middleman reporting.

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
