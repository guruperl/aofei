# Operational Commands

This document covers the active local operational commands. All commands use the
Docker-generated DSP config:

```bash
./scripts/aofei-local.sh up
export AOFEI="$PWD/etc/aofei.local.json"
```

Do not use the retired root config directory or legacy MySQL credentials for
these commands.

The current implementation/activation state and matching lane contracts are in
the [documentation and milestone index](README.md). The convenience
`./scripts/aofei-local.sh install` installs the core runtime set; build the
restricted accounting, identity, experiment, traffic-quality, and hosted-
payment maintenance binaries explicitly on their authorized release hosts.

## Production Placement

Use these commands as separate process roles:

| Role | Command | Placement |
|---|---|---|
| Production configuration preflight | `cmd/config-preflight` | every distinct HTTP-node service environment before rollout; read-only and dependency-free |
| HTTP/UI/ADX | `../pzdesign/cmd/unify` | every HTTP node |
| NATS log writer | `cmd/nats-client` | separate systemd service on nodes that write/aggregate logs |
| Redis cache refresh | `cmd/redis-cache -cache=redis` | singleton cron/timer on one cache node |
| Middleman callback retry | `cmd/mid-callback-retry` | singleton cron/timer on one operations node |
| Ledger | `cmd/ledger` | singleton cron/timer on the log aggregation node |
| Manual accounting | `cmd/accounting` | authorized accounting operations node; never a public service |
| Action reconciliation/retention | `cmd/action-measurement` | singleton operations timer; export/delete only from an authorized privacy host |
| Experiment control/retention | `cmd/report-experiment` | authorized operations/privacy host; explicit audited transitions, bounded prune, and exact subject deletion |
| Traffic-quality aggregate/review maintenance | `cmd/traffic-quality` | restricted quality operations host; bounded aggregate ingest, per-rule health, and evidence retention only |
| Identity grants/recovery/retention | `../pzdesign/cmd/identity-admin` | restricted identity maintenance host; never a public service or ordinary HTTP-node timer |
| Spread snapshots | `cmd/spread` | nodes that need spread disk cache |
| MaxMind refresh | `cmd/maxmind` | manual or scheduled maintenance node |
| Win/loss simulator | `cmd/winloss` | manual smoke/CI only |

Do not run Redis cache refresh or ledger on every `unify` node. Ledger must run
where the complete `log_winloss/winloss.<stamp>` files are available.
Mutating `redis-cache`, `ledger`, `mid-callback-retry`, and `winloss` runs take
a Redis singleton lock by default; read-only modes skip the lock.
Every mutating `redis-cache` mode (`redis`, `spread`, `all`, and `routes`) uses
the same `aofei:redis-cache` lock because modes share live cache families,
shadow keys, or source data and must not overlap.
Singleton locks begin renewal at one-third of `-lock-ttl`. Transient Redis
errors retry with bounded backoff only inside the last confirmed lease window;
a token mismatch stops immediately, and dependency uncertainty cancels work no
later than that conservative deadline. Commands perform a bounded token-checked
release before returning failure, so they cannot remove a successor's lease.
Do not treat the lock as the durable correctness boundary: ledger interval/day
uniqueness, callback/action idempotency, and cache generation replacement remain
required through Redis partitions.

`cmd/unify` handles `SIGINT` and `SIGTERM`, drains in-flight HTTP requests for
up to 15 seconds, and closes the DSP controller afterward so queued audits and
owned service connections shut down in order.

## Common Prerequisites

- Docker MySQL, Redis, and NATS are started by `./scripts/aofei-local.sh up`.
- The database is loaded with `./scripts/aofei-local.sh reset-sample` when a
  command reads schema or sample data.
- Redis cache data is populated before bid-path or win/loss simulation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Production does not use the checked-in/local tracking secret. From each exact
canary service environment, before any restart, run:

```bash
GOWORK=off GOTOOLCHAIN=go1.23.5 \
  go run ./cmd/config-preflight -s "$AOFEI"
```

This is a configuration-only check: it connects to no dependency and emits no
secret. It applies bid-mode validation and rejects checked-in example values,
surrounding whitespace, and tracking secrets shorter than 32 bytes. Archive
only its fixed `production_config_preflight=passed` result with rollout
evidence.

This refresh also compiles `middleman:routes:v2` for M25 middleman routing and
the fallback-only legacy `middleman:routes` key for M24 rolling-deploy safety.
Full Redis refreshes build completeness-marked shadow keys. One Redis script
validates every staged hash marker and route key before atomically replacing all
static cache families and removing obsolete slot-size hashes. Failed builds,
evicted shadows, and partially recreated shadows leave the live generation
unchanged. The reusable implementation is `cache.PublishRedisGeneration`;
direct live cache sinks cannot reset a family, and generation sinks require an
explicit non-empty namespace.
Run it only from the dedicated cache-maintenance node; `cmd/unify` does not
refresh bidder routes itself. After operators edit route groups, route bidders,
or route targets in Summer, run this refresh before expecting HTTP workers to
use the new route state. Workers observe it after their configured
`middleman_route_cache_ttl_ms` interval, default five seconds.

Full refresh also compiles D01 campaign/ad-group windows, weekly schedules,
pacing, and reconciled budget facts into RAdv version 2. With the default
`delivery_cache_max_age_seconds=900`, schedule `-cache=redis` (or `-cache=all`
for spread/local mode) at least every 300 seconds. A stale delivery snapshot
fails closed; route-only refresh does not extend it.

Before a full publisher generation, run the DB-only readiness gate:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -validate-publishers
```

It takes no mutation lock and performs no Redis operation. It validates active
publisher/site/slot identity, Web/App type, dimensions, and finite non-negative
USD CPM floors, then prints the exact packed site/slot tokens. Resolve failures
before `-cache=redis|spread|all`. P01 HTTP workers require this additive
type/floor metadata; publish the compatible generation before rolling them.
The complete activation and rollback sequence is in
[publisher-activation.md](publisher-activation.md).

To refresh only the middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes
```

This Redis-only mode marshals the legacy fallback and v2 route payloads first,
then replaces both live keys together with one atomic Redis command.

To inspect only the route cache JSON and metadata:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes -read
```

To validate current MySQL routes, the published Redis v2 generation, config,
and environment-backed credential references without printing values or
changing state:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -validate-middleman \
  -activation-stage=preflight
```

Use `fallback` only with both disclosure/runtime gates enabled and `Always`
disabled; use `always` only after the separate gate and an active Always route
are approved. The exact staged workflow is in
[middleman-activation.md](middleman-activation.md).

- Generated log directories live under `.local/logs/` and are ignored by git:
  `.local/logs/log_request/`, `.local/logs/log_response/`,
  `.local/logs/log_attribute/`, and `.local/logs/log_winloss/`.

## `cmd/nats-client`

Purpose: subscribe to local NATS log subjects and write newline-delimited log
files into the generated log directories.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/nats-client -interval=10
```

Inputs:

- `AOFEI` config with `nats_url` and `log_*` paths.
- NATS subjects: `request`, `response`, `attribute`, and `winloss`.

Outputs:

- `request.<stamp>` under `.local/logs/log_request/`.
- `response.<stamp>` under `.local/logs/log_response/`.
- `attribute.<stamp>` under `.local/logs/log_attribute/`.
- `winloss.<stamp>` under `.local/logs/log_winloss/`.

Notes:

- File rotation is based on `-interval` minutes.
- At startup and each rotation, generated files older than
  `privacy_log_retention_hours` (default 168) are removed. Only the four known
  subject prefixes are eligible; unrelated files and symlinks are retained.
- Request and attribute data is privacy-scrubbed before NATS. Retention does
  not replace encrypted storage or a separate backup-expiry policy; see
  [privacy-data-governance.md](privacy-data-governance.md).
- Missing log directories are created at `0750`. Existing directories must
  already grant no permissions beyond `0750`; the command validates and never
  chmods them, preserving setgid/sticky ownership policy. Generated log files
  use `0640`. Ledger input files should never be world-readable or
  group/world-writable.
- Unknown subjects are logged as ignored.
- A full internal log queue is returned as an error instead of blocking the NATS
  callback.
- `SIGINT` and `SIGTERM` stop the service through a signal-aware context. On
  shutdown the command drains NATS, flushes queued log messages, and closes open
  file handles before exiting.

## `cmd/spread`

Purpose: subscribe to spread/cache NATS subjects and persist cache snapshots to
the generated spread directory.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/spread
```

Inputs:

- `AOFEI` config with `nats_url` and `spread`.
- Cache subjects produced by `cmd/redis-cache -cache=spread` or
  `cmd/redis-cache -cache=all`.
- Optional Redis/MySQL access from the same `AOFEI` config for startup
  bootstrap: MySQL compiles the snapshot while Redis owns the shared mutation
  lease and monotonic sequence.

Outputs:

- `.local/spread/.aofei-current` (the selected sequence)
- `.local/spread/.aofei-generations/<sequence>/pubmap/`
- `.local/spread/.aofei-generations/<sequence>/audience/`
- `.local/spread/.aofei-generations/<sequence>/creative/`
- `.local/spread/.aofei-generations/<sequence>/slot/<size_id>/`

Middleman bidder routes are not spread snapshots. They remain Redis-only under
`middleman:routes:v2`, with `middleman:routes` kept as a fallback-only legacy
key.

Notes:

- Log subjects are intentionally ignored by this command.
- Cache subjects are received with a NATS tail wildcard, so publisher domains
  containing dots are valid subject payload keys.
- Full refreshes use the Redis `aofei:spread:generation` sequence. The receiver
  stages generation-tagged entries, verifies their count and SHA-256 manifest,
  then atomically replaces the current pointer; it retains the current and
  immediately previous generation.
- The configured spread root must be a specific directory rather than `.`,
  `/`, or a relative parent traversal. Missing directories are created at
  `0750`; an existing directory must already grant no permissions beyond
  `0750` and is never chmod'd. Snapshot and pointer files use `0640`. Each file and containing
  directory is synced around atomic replacement; no process-local flock is
  used as a publication guarantee.
- A reconnect gap or failed/incomplete commit leaves the prior generation
  selected. Duplicate messages are idempotent, and a lower overlapping
  sequence is ignored.
- Legacy `__reset__`, `DELETE`, and slot `cleanup` subjects remain receiver-first
  rollout compatibility only before the first generation pointer exists. Once
  generation publication is active, legacy direct mutations are ignored.
- `SIGINT` and `SIGTERM` stop the service through a signal-aware context and
  drain the NATS connection before exit.

## `cmd/ledger`

Purpose: aggregate win/loss log files into interval ledger tables, then aggregate
interval ledger rows into daily ledger tables.

Interval invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/ledger -interval=10
```

Specific interval stamp:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/ledger -interval=10 -timestamp=<stamp>
```

Daily invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/ledger -daily -timestamp=2026-05-12
```

Inputs:

- Docker MySQL from `AOFEI`.
- Win/loss source file `.local/logs/log_winloss/winloss.<stamp>`.
- Publisher slot and advertiser creative dimensions from MySQL.

Outputs:

- Interval rows in `ledger_log`, `ledger_pub`, `ledger_adv`, and
  `ledger_pub_adv`.
- Middleman interval rows in `ledger_mid` when win/loss records include
  `WinLoss.Middleman` metadata.
- Daily rows in `daily_log`, `daily_pub`, `daily_adv`, and `daily_pub_adv`.
- Middleman daily rows in `daily_mid`.
- Balance counters updated from the inserted ledger rows.

Notes:

- Missing `winloss.<stamp>` is a retryable missing-input error. The command does
  not create a zero ledger row for an absent source file.
- Interval and daily writes run inside transactions.
- Demand dimensions are counted by `creative_id`.
- Middleman advertiser reports use pay-side spend. Admin settlement reports use
  charge spend, pay spend, and margin from callback metadata.

Ledger should run only on the node where `cmd/nats-client` aggregates
`log_winloss/winloss.<stamp>` files. Do not run it on every `unify` node.
Redis cache population is also a singleton operational job; run
`cmd/redis-cache -cache=redis` from cron or a systemd timer on one dedicated
node.

## `cmd/accounting`

Purpose: create idempotent advertiser or publisher statements from completed
daily ledger facts; add immutable adjustments; enforce hold, confirmation,
maker-checker settlement, and correction transitions; reconcile a snapshot;
and export scoped CSV.

Example:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/accounting -action=create \
  -party=advertiser -party-id=7 -cadence=daily \
  -from=2026-08-01 -to=2026-08-01 \
  -request-key=invoice-20260801-adv-7 -reason='daily close'
```

Reconcile charge, pay, and margin for a completed middleman period:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/accounting -action=reconcile-middleman \
  -from=2026-08-01 -to=2026-08-31
```

This command does not use a Redis singleton lock: MySQL request-key uniqueness,
row locks, state preconditions, and transactions serialize its mutations. Run
it only after the relevant daily ledger is complete. It never updates ledger
facts, `adv_balance` limits, or Redis delivery floors. Restrict the config and
database role to distinct non-shared Unix operator accounts; the audited actor
is derived from the effective Unix UID and cannot be overridden by a flag or
environment variable. Keep CSV outside Git in encrypted storage, and never
place card or bank credentials in reason or reference fields. Full formulas,
actions, approval separation, evidence references, reconciliation, and the v1
to v2 migration are in [accounting-settlement.md](accounting-settlement.md).

## `cmd/action-measurement`

Purpose: reconcile late/restored same-lineage touches into facts that are still
unattributed, prune expired R01 rows, and perform authorized pseudonym-scoped
export/deletion. It does not read or mutate D01 reservations or A01 accounting
tables.

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/action-measurement -action=reconcile -limit=1000

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/action-measurement -action=prune -limit=10000
```

`reconcile` locks a bounded batch with `SKIP LOCKED`, applies click precedence
and configured windows, and updates only `unattributed` facts. `prune` deletes
rows whose schema-assigned `expires_at` has passed. Run both as singleton
timers, alert on nonzero exits, and repeat batches until the reported count is
zero when clearing a backlog.

After verified privacy authorization, `-action=export|delete
-pseudonym=<64-lowercase-hex>` operates on one R01 pseudonym. Export writes CSV
to stdout without token hashes, auction ids, or publisher ids. Redirect output
to encrypted controlled storage; avoid shell history and shared logs. Deletion
also removes same-lineage touches only when no other action references remain.
See [conversion-attribution.md](conversion-attribution.md).

## `cmd/report-experiment`

Purpose: create, list, start, stop, or complete an R02 controlled experiment,
prune expired pseudonymous facts, or delete one exact verified subject through
an explicit operator/privacy boundary. It never modifies bids, budgets,
schedules, reservations, or accounting facts.

Invocation examples:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/report-experiment -action=list

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/report-experiment \
  -action=create -owner=operator -name=reviewed-copy -version=1 \
  -primary-metric=actions -guardrail-metric=spend \
  -starts-at=2026-08-03T00:00:00Z \
  -retention-hours=2160 \
  -variants=control=5000,treatment=5000 \
  -reason='reviewed local experiment'
```

Mutations require a nonempty bounded audit reason. The command records the
effective OS UID as actor; run it only as an approved service/operations
principal. Advertiser ownership additionally requires `-owner=advertiser` and
an active `-adv-id`. Allocation must contain 2–20 valid variant keys totaling
exactly 10,000 basis points. `start`, `stop`, and `complete` require
`-experiment-id` and `-reason`. Every mutation acquires the fixed
`aofei:report-experiment` Redis lease and runs with its renewable lease-owned
context; `-lock-ttl` defaults to five minutes. A lost or uncertain lease
cancels work by its last confirmed expiry. `list` remains DB-only and
read-only.

Run `-action=prune -limit=1000` as a singleton timer. It deletes expired
outcomes then exposures in one bounded transaction. A verified erasure uses
`-action=delete-subject -experiment-id=<id> -version=<version>
-subject-hash=<64-hex> -reason=<non-identifying-reason>` on the privacy host.
Keep the hash out of shared shell history and logs; the immutable audit records
actor/reason but not the hash.

Assignment/exposure/outcome recording is an application integration through
the `reporting` package, not a command flag or public endpoint. The operator
list and Summer aggregate HTML/JSON never expose salts, subject hashes,
idempotency digests, stop/audit reasons, or per-subject rows. See
[marketplace-analytics-experiments.md](marketplace-analytics-experiments.md).

## `../pzdesign/cmd/identity-admin`

Purpose: create a read-only analyst, grant/revoke one exact permission/resource,
reset one account's TOTP after external identity verification, or prune bounded
expired security evidence. It requires Summer `Identity.Enabled=true`, the
configured 32-byte environment key, an explicit effective-UID-to-administrator
mapping in `Identity.MaintenanceActors`, and a bounded reason. New analyst passwords are accepted only through
`IDENTITY_NEW_PASSWORD`; no password or key flag exists.

Build and run it from the sibling UI/service repository:

```bash
(cd ../pzdesign && GOWORK=off go install ./cmd/identity-admin)

SUMMER=/etc/aofei/summer.json /opt/aofei/bin/identity-admin \
  -action=reset-totp \
  -subject-role=pub -subject-id=123 \
  -reason='verified recovery case'
```

The command derives its launcher from the effective Unix UID, maps it to the
reviewed administrator id in the restricted config, and prefixes that launcher
in every audit reason. Restrict binary/config/key write access to named
operators and apply maker/checker review outside the command. Exact create/grant/revoke/prune
examples, retention, rollout, and key-loss behavior are in
[identity-access-security.md](identity-access-security.md).

## `cmd/traffic-quality`

Purpose: ingest one trusted bounded aggregate window, inspect fixed per-rule
health, or remove expired short-lived evidence. It is not an HTTP service and
does not accept raw requests, identity evidence, IP addresses, cookies, device
identifiers, auction ids, or credentials.

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality -action=assess-window <aggregate-window.json

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality -action=health \
  -since-hours=24

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality -action=prune-evidence \
  -limit=1000 \
  -reason='scheduled evidence retention'
```

`assess-window` accepts at most 64 KiB of strict JSON, rejects unknown fields,
and derives only the eight closed S03 signals. Counters are individually
bounded at one billion, and an observation more than one minute in the future
is rejected before database work. Output omits event and partner digests.

`health` permits a 1–9600-hour lookback and returns rule ids/versions, fixed
counters, false-positive basis points, limits, and rollback recommendation.
`prune-evidence` permits batches of 1–10000 and uses a dedicated connection
whose retention flag is cleared with a fresh bounded context; cleanup failure
discards the connection. Both actions derive a non-spoofable
`admin:unix-uid:<effective-uid>` launcher with only the selected read/retention
permission and no MFA claim. Restrict execution/config access and retain
external change approval. See
[traffic-quality-anti-fraud.md](traffic-quality-anti-fraud.md).

## `cmd/mid-callback-retry`

Purpose: retry failed downstream middleman callback forwards from
`mid_callback_retry`.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/mid-callback-retry -limit=100 -max-attempts=5 -timeout=2s
```

Read/dry-run mode:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/mid-callback-retry -read
```

Stable JSON summary for alerting:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/mid-callback-retry -read -json
```

Inputs:

- Docker MySQL from `AOFEI`.
- Due rows in `mid_callback_retry` with status `Pending` or `Retrying`, plus
  stale `Processing` rows whose `claimed_at` is older than the worker stale
  threshold; process mode claims rows as `Processing` before downstream
  forwarding.

Outputs:

- Downstream HTTP GET calls to the already-expanded callback URL.
- Loopback, private, link-local, unspecified, multicast, and DNS-rebinding
  callback targets are rejected before forwarding.
- Retry rows marked `Succeeded`, `Retrying`, or `Abandoned`.
- Retry-row `last_error` uses only a closed forward-outcome vocabulary. Raw
  URL-validation, DNS, dial, redirect, response, and injected-client errors are
  not persisted; the numeric HTTP response remains in `last_http_status`.
- One summary line with `due`, `stale_processing`, `selected`, `forwarded`,
  `succeeded`, `retrying`, `abandoned`, and `state_errors` counts. The summary
  is written even when a post-forward state transition fails and the command
  exits non-zero. Alert immediately on `state_errors`, when
  `stale_processing` remains non-zero across runs, or when `due` grows faster
  than the singleton job drains. Use `-json` for automation; the default text
  output is for humans.

Notes:

- `cmd/unify` enqueues only post-auction `/mid/win`, `/mid/loss`, and
  `/mid/bill` forwarding failures that are retryable: network/request errors,
  HTTP 429, and HTTP 5xx.
- Missing URLs, invalid URLs, duplicate callbacks, and HTTP 4xx responses other
  than 429 are not queued.
- The retry command forwards downstream only. It does not republish delivery,
  win, loss, or billable records, so ledger counts remain idempotent.
- Every forward still uses the guarded callback client: request-time and
  dial-time address checks, proxy prohibition, reviewed TLS settings, redirect
  credential stripping, and bounded response draining cannot be bypassed by an
  injected client.
- `forwarded=1 state_errors=1` means the guarded downstream attempt completed
  but the exact one-row `Processing` transition was not durably confirmed.
  Delivery is therefore uncertain under the at-least-once contract. The row
  remains or becomes stale `Processing` and may be sent again; the downstream
  endpoint must be idempotent for its callback identity.
- On a state error, stop repeated timer invocations, preserve only the fixed
  summary and dependency logs, repair MySQL, and use `-read -json` until the
  stale row is visible. Do not paste callback URLs/tokens into tickets and do
  not manually mark a row `Succeeded` without independently retained partner
  acknowledgement. Resume the singleton and reconcile the downstream result;
  a resend is expected and must not republish Aofei ledger events.
- Run this command as a singleton cron or systemd timer. Do not run it on every
  HTTP worker.

## `cmd/hosted-payment`

Purpose: inspect aggregate A02 funding/payout health and perform bounded
retention of unreferenced provider-event envelopes. It cannot create, approve,
execute, cancel, refund, reconcile, or resolve money operations.

Health:

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/hosted-payment -action=health
```

The JSON reports counts only: Held/approved operations, stale `Submitting`,
`Canceling`, and `Submitted` work, unresolved exceptions and age-policy breaches, oldest
unresolved age, and webhook volume over 24 hours. It never prints provider
tokens, account/payment details, secrets, or raw events.

Retention, from a restricted maintenance host and database principal:

```bash
AOFEI=/etc/aofei/aofei-maintenance.json \
  /opt/aofei/bin/hosted-payment -action=prune-events \
  -limit=1000 \
  -reason='approved provider-event retention schedule'
```

The command accepts a batch of 1–10000 and derives an audited
`admin:unix-uid:<effective-uid>` identity with the dedicated retention
permission. That principal is rejected by every money/reconciliation method.
A connection-local trigger gate permits deletion only for events
older than `event_retention_days` that are not linked to reconciliation
evidence. Cleanup runs on a fresh bounded context; an uncertain connection is
discarded. See [hosted-funding-payout.md](hosted-funding-payout.md) for the
webhook, outage, secret-rotation, reconciliation, and live-governance contract.

## `cmd/winloss`

Purpose: simulate an exchange calling the local DSP bid endpoint, then fire win,
loss, impression, or click URLs from the response.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/winloss --bid=/bid/default win
```

Modes:

- `win`: call the win URL, then the impression tracker.
- `loss`: call the loss URL.
- `imp`: call only the impression tracker.
- `clk`: call only the native click URL. Current native markup uses the DSP
  `/clk` redirect URL as `link.url`; the simulator falls back to legacy native
  `clicktrackers` only when `link.url` is absent.
- omitted mode: randomly choose win/impression/click or loss.

Inputs:

- Running local DSP HTTP server at `server_url` from `AOFEI`.
- Redis cache populated for the sample bid request.
- Bid response containing at least one seat bid, one bid, native markup, one
  impression tracker, and one native click URL.

Outputs:

- HTTP calls to the selected win/loss/tracker URLs.
- DSP win/loss events are published only when the server-side controller has
  NATS enabled.

Notes:

- No-bid and malformed bid responses fail with clear errors instead of panics.
- The simulator takes a singleton lock by default. Use `-allow-concurrent` only
  for intentional load testing.

## `cmd/maxmind`

Purpose: inventory/build the MaxMind country and state map config from MySQL.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/maxmind -city=GeoLite2-City.mmdb
```

Inputs:

- Docker MySQL from `AOFEI`.
- External City `.mmdb` path supplied with `-city`.

Outputs:

- A validated content-addressed City MMDB beneath
  `.<ips-base>.generations/<sha256>/`.
- JSON written atomically to the configured `ips` path, normally
  `etc/maxmind.json`, selecting that staged generation by relative path.

Notes:

- The command reads database country/state tables and reads the existing JSON
  only to retain the currently selected City generation for rollback.
- The `-city` value (or `AOFEI_GEOLITE_CITY_FILE`) identifies the external
  source asset. Relative values resolve against the generated JSON directory;
  use an explicit absolute source value for an asset stored elsewhere.
- A stable `0640` sibling file lock serializes publishers and retains an asset
  across each complete runtime JSON/MMDB load. The source is hashed, copied
  atomically, checked for change during copy, and parsed as a supported City
  MMDB before JSON selection. Failure leaves the prior JSON selected; the
  selected and immediately prior City generations are retained. A static
  read-only config directory remains loadable before its first publication.
- The configured `ips` target rejects root/current-directory/parent-traversal
  values. JSON replacement uses mode `0640` and syncs both the file and its
  containing directory before success.
- Runtime loading prefers JSON plus MMDB. An `ips` path ending in `.dat` invokes
  the strict legacy compatibility reader; malformed files fail without panic
  and are never used as an implicit fallback for malformed JSON/MMDB.
- See [maxmind-runtime.md](maxmind-runtime.md) for asset and test details.

## Verification

The O02 clean-room recovery rehearsal is deliberately isolated from the
configured local stack:

```bash
./scripts/aofei-recovery-drill.sh
```

It creates uniquely named disposable MySQL source/restore and Redis containers,
loads a synthetic fixture, checksums a logical dump, restores routines and
triggers, proves A01 immutability plus interval/day uniqueness, rebuilds
derived Redis cache, and confirms that a restored stale callback claim appears
in the read-only retry summary without forwarding it. The owner-only temporary
directory and unencrypted local dump are destroyed on exit. Production backups
must instead be encrypted and stored off Git under the retention/RPO/RTO contract in
[single-region-availability.md](single-region-availability.md).

Build and focused package tests:

```bash
GOWORK=off go test ./internal/jobs/cache ./internal/jobs/ledger ./cmd/redis-cache ./cmd/ledger
(cd ../pzdesign && GOWORK=off go test ./cmd/unify)
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
GOWORK=off go test ./dsp -run 'Controller|Win|Loss|^$'
```

Local runtime checks:

```bash
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh nats-status
find .local/logs -maxdepth 2 -type d | sort
```

GitHub Actions performs committed-range whitespace checks. Pull requests diff
the merge base of the event base/head SHAs through the head SHA; pushes diff the
event `before` and `after` SHAs, using the empty tree for an initial history.
Keep `git diff --check` as the local closeout check for uncommitted changes.
