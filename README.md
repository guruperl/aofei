# Aofei / Winter DSP

`github.com/genelet/winter` is a Go package for an OpenRTB-oriented DSP stack.
It contains the bid path, campaign and publisher matching logic, Summer/Genelet
admin models, cache population commands, local Docker service helpers, and SQL
baseline data needed to run the package locally.

Current local development uses Docker MySQL, Docker Redis, and Docker NATS. The
active database baseline is [etc/step4_init.sql](etc/step4_init.sql); generated
local configs live at `etc/aofei.local.json` and `etc/summer.local.json`.
MaxMind runtime config lives at `etc/maxmind.json`; real `.mmdb` and legacy
geodata payloads are external ignored assets.

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

Run the canonical package verification gate from this repository:

```bash
GOWORK=off go test ./...
```

Review operational command prerequisites, invocations, outputs, and known
blockers:

```bash
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
```

See [docs/operational-commands.md](docs/operational-commands.md) for the local
contracts for `cmd/redis-cache`, `cmd/ledger`, `cmd/nats-client`,
`cmd/winloss`, `cmd/spread`, and `cmd/maxmind`, including where each command
should run in production.

See [docs/maxmind-runtime.md](docs/maxmind-runtime.md) for the active
`etc/maxmind.json` contract, expected external GeoLite2 City path, ignored
local test assets, and MaxMind verification commands.

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
- [docs/production-runbook.md](docs/production-runbook.md): current Linux
  systemd-oriented production runbook.
- [docs/database-baseline.md](docs/database-baseline.md): schema baseline and
  drift rules.
- [docs/multiple-cache.md](docs/multiple-cache.md): Redis, NATS/spread,
  disk-snapshot, and in-process static-cache roles plus likely bottlenecks.
- [docs/dsp-workflow.md](docs/dsp-workflow.md): current OpenRTB bid workflow,
  static/mutable cache reads, response construction, and click redirect flow.
- [docs/openrtb-measurement.md](docs/openrtb-measurement.md): win/loss,
  impression, click, NATS log, and ledger measurement behavior.
- [docs/audience-matching.md](docs/audience-matching.md): attribute extraction,
  audience predicates, cache contracts, and matching order.
- [docs/operational-commands.md](docs/operational-commands.md): local
  operational command contracts for logs, ledger, spread, win/loss simulation,
  and MaxMind inventory.
- [docs/maxmind-runtime.md](docs/maxmind-runtime.md): MaxMind config,
  external geodata assets, generation, and test behavior.
- [docs/genelet-manual.md](docs/genelet-manual.md): Genelet config, routes,
  auth, component, CRUD, upload, CORS, and error contracts.
- [docs/summer-ui-structure.md](docs/summer-ui-structure.md): Summer module
  layout, component conventions, registry, UI options, and cache side effects.
- [docs/dsp-architecture.zh.md](docs/dsp-architecture.zh.md): historical DSP
  architecture note in Chinese.
- [docs/legacy-operations.md](docs/legacy-operations.md): historical manual
  deployment notes retained for reference.
- [backup/](backup/): historical files moved out of active runtime paths.

## Development Notes

Use `GOWORK=off` for package commands from this repository. The parent
workspace's `go.work` does not include this module path, so plain
`go list ./...` fails from this checkout unless the parent workspace is changed.

Do not use legacy MySQL users in local development. The Docker helper creates
and uses the `aofei` database user.

`AOFEI` points at the lower-case DSP JSON config. `SUMMER` points at the
upper-case Summer/Genelet JSON config.

Docker smoke checks, admin DB checks, schema drift checks, and staticcheck are
useful local follow-ups, but the package gate is `GOWORK=off go test ./...`.
