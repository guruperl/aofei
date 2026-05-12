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
- It must not require or recreate `eightran_*` users.
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

## Updating The Baseline

When an intentional schema change is made inside Docker MySQL:

1. Export the updated schema and required baseline data.
2. Strip explicit definers and legacy auth references.
3. Replace or patch `etc/step4_init.sql`.
4. Recreate the database from `etc/step4_init.sql`.
5. Verify object counts and relevant schema differences.
6. Update `etc/step5.notes`, this document, and the memory bank if the workflow
   changed.

The detailed drift-check command is a milestone target under
`memory-bank/milestone.md`.
