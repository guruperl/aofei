# Aofei / Winter DSP

`github.com/genelet/winter` is a Go package for an OpenRTB-oriented DSP stack.
It contains the bid path, campaign and publisher matching logic, Summer/Genelet
admin models, cache population commands, local Docker service helpers, and SQL
baseline data needed to run the package locally.

Current local development uses Docker MySQL, Docker Redis, and Docker NATS. The
active database baseline is [etc/step4_init.sql](etc/step4_init.sql); generated
local configs live at `etc/aofei.local.json` and `etc/summer.local.json`.

## Quick Start

```bash
./scripts/aofei-local.sh up
./scripts/aofei-local.sh reset-sample

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go test ./dsp -run 'Test.*Smoke'

./scripts/aofei-local.sh status
```

Run the cache pipeline smoke, including Redis, NATS, and spread artifacts:

```bash
./scripts/aofei-cache-smoke.sh
```

Run the bid-path smoke after `reset-sample` and Redis cache population. It uses
`etc/samples/sample_bid.json`, the generated local DSP config, and Docker Redis
to exercise `dsp.Controller.ServeBid` through `httptest`.

Run the admin compatibility checks against Docker MySQL:

```bash
GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  go test ./summer ./summer/pub ./summer/slot

GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  go test ./genelet
```

The helper starts:

- MySQL `mysql:8.0.41` on `127.0.0.1:3307`
- Redis `redis:7-alpine` on `127.0.0.1:6379`
- NATS `nats:2-alpine` on `127.0.0.1:4222`

`reset-sample` recreates the database, imports `etc/step4_init.sql`, and makes
the sample publisher/demand state present. The sample demand import is
idempotent against the current baseline.

Stop the local services without deleting Docker volumes:

```bash
./scripts/aofei-local.sh down
```

Install the package command binaries:

```bash
./scripts/aofei-local.sh install
```

## Repository Map

- [AGENTS.md](AGENTS.md): bootstrap guide for agents working in this repo.
- [memory-bank/](memory-bank/): active project source of truth.
- [docs/local-docker-runtime.md](docs/local-docker-runtime.md): local Docker
  runtime commands and generated config notes.
- [docs/database-baseline.md](docs/database-baseline.md): schema baseline and
  drift rules.
- [docs/dsp-architecture.zh.md](docs/dsp-architecture.zh.md): historical DSP
  architecture note in Chinese.
- [docs/legacy-operations.md](docs/legacy-operations.md): historical manual
  deployment notes retained for reference.
- [backup/](backup/): historical files moved out of active runtime paths.

## Development Notes

Use `GOWORK=off` for package commands from this repository. The parent
workspace's `go.work` does not include this module path.

Do not use legacy `eightran_*` MySQL users in local development. The Docker
helper creates and uses the `aofei` database user.

`AOFEI` points at the lower-case DSP JSON config. `SUMMER` points at the
upper-case Summer/Genelet JSON config.
