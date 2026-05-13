# Operational Commands

This document covers the active local operational commands. All commands use the
Docker-generated DSP config:

```bash
./scripts/aofei-local.sh up
export AOFEI="$PWD/etc/aofei.local.json"
```

Do not use the retired root config directory or legacy MySQL credentials for
these commands.

## Production Placement

Use these commands as separate process roles:

| Role | Command | Placement |
|---|---|---|
| HTTP/UI/ADX | `cmd/unify` | every HTTP node |
| NATS log writer | `cmd/nats-client` | separate systemd service on nodes that write/aggregate logs |
| Redis cache refresh | `cmd/redis-cache -cache=redis` | singleton cron/timer on one cache node |
| Middleman callback retry | `cmd/mid-callback-retry` | singleton cron/timer on one operations node |
| Ledger | `cmd/ledger` | singleton cron/timer on the log aggregation node |
| Spread snapshots | `cmd/spread` | nodes that need spread disk cache |
| MaxMind refresh | `cmd/maxmind` | manual or scheduled maintenance node |
| Win/loss simulator | `cmd/winloss` | manual smoke/CI only |

Do not run Redis cache refresh or ledger on every `unify` node. Ledger must run
where the complete `log_winloss/winloss.<stamp>` files are available.

## Common Prerequisites

- Docker MySQL, Redis, and NATS are started by `./scripts/aofei-local.sh up`.
- The database is loaded with `./scripts/aofei-local.sh reset-sample` when a
  command reads schema or sample data.
- Redis cache data is populated before bid-path or win/loss simulation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

This refresh also compiles `middleman:routes` for enabled middleman fallback.
Run it only from the dedicated cache-maintenance node; `cmd/unify` does not
refresh bidder routes itself. After operators edit route groups, route bidders,
or route targets in Summer, run this refresh before expecting HTTP workers to
use the new route state.

To refresh only the middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes
```

To inspect only the route cache JSON and metadata:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes -read
```

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
- Unknown subjects are logged as ignored.
- A full internal log queue is returned as an error instead of blocking the NATS
  callback.

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
  bootstrap from current Redis static cache state.

Outputs:

- `.local/spread/pubmap/`
- `.local/spread/audience/`
- `.local/spread/creative/`
- `.local/spread/slot/<size_id>/`

Middleman bidder routes are not spread snapshots. They remain Redis-only under
`middleman:routes`.

Notes:

- Log subjects are intentionally ignored by this command.
- Cache subjects are received with a NATS tail wildcard, so publisher domains
  containing dots are valid subject payload keys.
- File snapshots are written by temp-file plus atomic rename.
- `__reset__` subjects clear a whole cache family before a full refresh.
- `DELETE` payloads remove individual snapshots.
- `cleanup` suffixes are slot-only and clear the relevant size directory before
  writing the next snapshot.

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

Inputs:

- Docker MySQL from `AOFEI`.
- Due rows in `mid_callback_retry` with status `Pending` or `Retrying`.

Outputs:

- Downstream HTTP GET calls to the already-expanded callback URL.
- Retry rows marked `Succeeded`, `Retrying`, or `Abandoned`.

Notes:

- `cmd/unify` enqueues only post-auction `/mid/win`, `/mid/loss`, and
  `/mid/bill` forwarding failures that are retryable: network/request errors,
  HTTP 429, and HTTP 5xx.
- Missing URLs, invalid URLs, duplicate callbacks, and HTTP 4xx responses other
  than 429 are not queued.
- The retry command forwards downstream only. It does not republish delivery,
  win, loss, or billable records, so ledger counts remain idempotent.
- Run this command as a singleton cron or systemd timer. Do not run it on every
  HTTP worker.

## `cmd/winloss`

Purpose: simulate an exchange calling the local DSP bid endpoint, then fire win,
loss, impression, or click URLs from the response.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/winloss --bid=/bid/exchange.example.test win
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

## `cmd/maxmind`

Purpose: inventory/build the MaxMind country and state map config from MySQL.

Invocation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/maxmind -city=/path/to/GeoLite2-City.mmdb
```

Inputs:

- Docker MySQL from `AOFEI`.
- External City `.mmdb` path supplied with `-city`.

Outputs:

- JSON written atomically to the configured `ips` path, normally
  `etc/maxmind.json`.

Notes:

- The command reads database country/state tables without loading the existing
  MaxMind runtime JSON first.
- The `-city` value is recorded as `city_file`; the `.mmdb` payload remains an
  external asset and is not copied into the repository.
- See [maxmind-runtime.md](maxmind-runtime.md) for asset and test details.

## Verification

Build and focused package tests:

```bash
GOWORK=off go test ./internal/jobs/cache ./internal/jobs/ledger ./cmd/redis-cache ./cmd/ledger ./cmd/unify
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
GOWORK=off go test ./dsp -run 'Controller|Win|Loss|^$'
```

Local runtime checks:

```bash
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh nats-status
find .local/logs -maxdepth 2 -type d | sort
```
