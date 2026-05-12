# Operational Commands

This document covers the active local operational commands for M6. All commands
use the Docker-generated DSP config:

```bash
./scripts/aofei-local.sh up
export AOFEI="$PWD/etc/aofei.local.json"
```

Do not use `conf/` or legacy MySQL credentials for these commands.

## Common Prerequisites

- Docker MySQL, Redis, and NATS are started by `./scripts/aofei-local.sh up`.
- The database is loaded with `./scripts/aofei-local.sh reset-sample` when a
  command reads schema or sample data.
- Redis cache data is populated before bid-path or win/loss simulation:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
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

Outputs:

- `.local/spread/pubmap/`
- `.local/spread/audience/`
- `.local/spread/creative/`
- `.local/spread/slot/<size_id>/`

Notes:

- Log subjects are intentionally ignored by this command.
- `DELETE` payloads remove snapshots.
- `cleanup` subjects clear the relevant size directory before writing the next
  snapshot.

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
- Daily rows in `daily_log`, `daily_pub`, `daily_adv`, and `daily_pub_adv`.
- Balance counters updated from the inserted ledger rows.

Notes:

- Missing `winloss.<stamp>` is a retryable missing-input error. The command does
  not create a zero ledger row for an absent source file.
- Interval and daily writes run inside transactions.
- Demand dimensions are counted by `creative_id`.

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
- `clk`: call only the click tracker.
- omitted mode: randomly choose win/impression/click or loss.

Inputs:

- Running local DSP HTTP server at `server_url` from `AOFEI`.
- Redis cache populated for the sample bid request.
- Bid response containing at least one seat bid, one bid, native markup, one
  impression tracker, and one click tracker.

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
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
GOWORK=off go test ./dsp -run 'Controller|Win|Loss|^$'
```

Local runtime checks:

```bash
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh nats-status
find .local/logs -maxdepth 2 -type d | sort
```
