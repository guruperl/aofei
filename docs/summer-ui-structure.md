# Summer UI Structure

Summer is the admin UI/model layer on top of Genelet. It provides module models,
filters, component JSON, option dictionaries, upload handling, and cache update
hooks for advertiser, publisher, campaign, item, slot, creative, balance,
payment, ledger, and access-control workflows.

## Module Layout

Most modules use the same shape:

```text
summer/{module}/component.json
summer/{module}/model.go
summer/{module}/filter.go
```

The root `summer` package provides shared model/filter behavior, dictionaries,
OpenRTB option packing helpers, upload helpers, and shared tests. The
`summer/registry` package declares every component-backed module once and is
used by `cmd/unify` to build model, storage-model, and filter maps.

Add a new component-backed module by adding the module directory, model/filter
types, `component.json`, and one `summer/registry.Entry`. The registry test
checks that every `component.json` has a matching entry and every entry has a
component file.

## Component JSON Conventions

Components define:

- `actions`: action names, allowed role groups, validation fields, and options.
- `fks`: role-specific foreign-key signature fields.
- `current_table`, `current_key`, `current_keys`, `current_id_auto`: primary
  CRUD metadata.
- `insert_pars`, `update_pars`, `edit_pars`, `topics_pars`: allowed fields for
  each CRUD action.
- `current_tables`: joined table metadata for topic/list queries.
- `topics_hash`: SQL select expression to output-label mapping.
- `nextpages`: dependent model calls used to populate related UI data.

Table, alias, key, and CRUD field values must be SQL identifiers. Joined table
conditions are validated by Genelet before use. Keep arbitrary SQL expressions
limited to component-owned select expressions, not request input.

## Actions And Roles

Genelet chooses the action from the HTTP method defaults or request action
parameter. Summer filters then shape request args and side effects for each
role:

- `admin`: full maintenance flows and cache publication actions.
- `adv`: advertiser-facing campaign, item, creative, target, payment, and
  account flows.
- `pub`: publisher-facing site, slot, access-control, and take-down flows.
- `agent`: delegated admin flows.

Filters should add query restrictions through `extra url.Values`, not by
concatenating request strings into SQL. Models should use Genelet CRUD helpers
unless a hand-written query is necessary and all identifiers come from a narrow
allowlist.

## UI Options

Shared option data lives in `summer.LARGES` and dictionary helpers. Request
state must use `summer.LargeOptions(name)` or `Filter.AfterItemSet`; both clone
the shared option rows before setting `selected` or translated labels. Do not
write request-specific fields directly into `LARGES`.

Slot and item filters pack/unpack OpenRTB option masks for language, MIME,
device, position, expandable, creative, and channel fields. Size helpers convert
between packed `size_id` and width/height values.

## Cache Side Effects

Summer filter `After` methods publish cache updates for selected admin actions:

- item/campaign/creative insert/update refreshes creative cache entries.
- targetname insert and admin item update can refresh audience or advertiser
  cache entries.
- publisher take-down updates publisher cache entries.
- advertiser attrname upload can feed uploaded audience values into Redis.

Storage adapters are supplied by `cmd/unify` through `model.Storage`:

```text
Redis  -> radix.Client
Nc     -> *nats.Conn
Spread -> string spread root
```

Summer accesses those through typed helper functions so a missing adapter is a
no-op and a wrong adapter type is an explicit error.

## Local Usage

Start local services and load sample data:

```bash
./scripts/aofei-local.sh reset-sample
```

Run focused admin checks:

```bash
GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  go test ./summer ./summer/pub ./summer/slot ./summer/weight
```

Run the unified service with generated configs:

```bash
GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/unify
```

## Maintenance Workflow

- Keep `component.json`, model field lists, filters, and the active MySQL schema
  aligned.
- Use `summer/registry` for component-backed modules instead of adding local
  maps in `cmd/unify`.
- Keep request-derived SQL inputs in `url.Values` and let Genelet validate and
  parameterize them.
- Update `docs/genelet-manual.md` when framework route, auth, CRUD, CORS,
  upload, or error behavior changes.
- Update this document when Summer module layout, registry, action behavior,
  UI option conventions, or cache side effects change.
