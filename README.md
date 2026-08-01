# Aofei / Winter DSP

`github.com/guruperl/aofei` is a Go package for an OpenRTB-oriented DSP stack.
It contains the bid path, campaign and publisher matching logic, Summer/Genelet
integration, cache population commands, local Docker service helpers, and SQL
baseline data needed to run the package locally. The same bid engine also serves
direct publisher SSP traffic on `POST /pz` and can fall back to middleman AdX
bidders when middleman config is enabled. The Summer/Genelet source tree
now lives in the sibling `../pzdesign` module,
`github.com/guruperl/pzdesign`.

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

Use the broader verification split when changing runtime behavior, cache
contracts, command workflows, or shared libraries:

```bash
GOWORK=off go vet ./...
GOWORK=off staticcheck ./...
GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client
./scripts/aofei-doc-check.sh
./scripts/aofei-public-data-check.sh
gitleaks git --redact .
git diff --check
```

Review operational command prerequisites, invocations, outputs, and known
blockers:

```bash
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind ./cmd/mid-callback-retry
```

See [docs/operational-commands.md](docs/operational-commands.md) for the local
contracts for `cmd/redis-cache`, `cmd/ledger`, `cmd/nats-client`,
`cmd/winloss`, `cmd/spread`, `cmd/maxmind`, and `cmd/mid-callback-retry`,
including where each command should run in production.

See [docs/maxmind-runtime.md](docs/maxmind-runtime.md) for the active
`etc/maxmind.json` contract, expected external GeoLite2 City path, ignored
local test assets, and MaxMind verification commands.

Run the bid-path smoke after `reset-sample` and Redis cache population. It uses
`etc/samples/sample_bid.json`, the generated local DSP config, and Docker Redis
to exercise `dsp.Controller.ServeBid` through `httptest`.

Run the admin compatibility checks against Docker MySQL:

```bash
(cd ../pzdesign && GOWORK=off SUMMER="$PWD/../aofei/etc/summer.local.json" \
  go test ./genelet ./summer ./summer/pub ./summer/slot)
```

The helper starts:

- MySQL `mysql:8.0.41` on `127.0.0.1:3307`
- Redis `redis:7-alpine` on `127.0.0.1:6379`
- NATS `nats:2-alpine` on `127.0.0.1:4222`

`reset-sample` recreates the database, imports the schema/reference-only
`etc/step4_init.sql`, and loads the deterministic fixture in `etc/demand.sql`.
The local-only logins are `admin_local`, `advertiser@example.test`, and
`publisher@example.test`, all with password `local-demo-password`. Never copy
these public development credentials into production.

Stop the local services without deleting Docker volumes:

```bash
./scripts/aofei-local.sh down
```

Install the package command binaries:

```bash
./scripts/aofei-local.sh install
```

## Repository Map

- [docs/advertiser-dsp-agent-manual.zh-CN.md](docs/advertiser-dsp-agent-manual.zh-CN.md):
  面向广告主与代理商的中文投放手册，涵盖广告活动、广告组、广告素材、
  定向、计量和外部 DSP / ADX 需求方竞价端点。
- [docs/publisher-manual.zh-CN.md](docs/publisher-manual.zh-CN.md):
  面向流量方（发布商）的中文接入手册，涵盖网站/App、广告位、网页标签、
  SDK/API、来源校验、报表和排障。
- [docs/operations-maintenance-manual.zh-CN.md](docs/operations-maintenance-manual.zh-CN.md):
  面向系统运维与维护人员的中文手册，涵盖部署、缓存、作业、监控、
  故障处理、备份恢复和变更验证。
- [AGENTS.md](AGENTS.md): bootstrap guide for agents working in this repo.
- [SECURITY.md](SECURITY.md): private vulnerability reporting and repository
  data-handling rules.
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
- [docs/ssp-direct-traffic.md](docs/ssp-direct-traffic.md): direct publisher
  `POST /pz` contract, packed token validation, browser/SDK policy, response
  formats, and audit boundary.
- [docs/middleman-adx.md](docs/middleman-adx.md): advertiser-owned bidder
  endpoints, route cache, fallback and `Always` runtime, callback proxy, and
  charge/pay/margin reporting.
- [docs/operational-commands.md](docs/operational-commands.md): local
  operational command contracts for logs, ledger, spread, win/loss simulation,
  MaxMind inventory, and middleman callback retry processing.
- [docs/maxmind-runtime.md](docs/maxmind-runtime.md): MaxMind config,
  external geodata assets, generation, and test behavior.
- [docs/performance-roadmap.md](docs/performance-roadmap.md): advisory
  performance roadmap for measurement, same-stack scaling, and conditional
  technology swaps.
- [docs/prebid-openrtb-adoption.md](docs/prebid-openrtb-adoption.md):
  documentation-only review of Prebid Server OpenRTB patterns to consider for
  later `aofei` validation, matching, middleman, performance, privacy, and
  observability milestones.
- [docs/adr/0001-richer-supply-taxonomy.md](docs/adr/0001-richer-supply-taxonomy.md):
  future additive supply taxonomy direction for publisher tables, cache, and
  audits.
- [docs/adr/0002-ssp-account-schema-boundary.md](docs/adr/0002-ssp-account-schema-boundary.md):
  decision to keep `pub`, `pub_site`, and `pub_slot` as the direct SSP account
  and inventory ownership boundary.
- [../pzdesign/docs/genelet-manual.md](../pzdesign/docs/genelet-manual.md):
  Genelet config, routes, auth, component, CRUD, upload, CORS, and error
  contracts.
- [../pzdesign/docs/summer-ui-structure.md](../pzdesign/docs/summer-ui-structure.md):
  Summer module layout, component conventions, registry, UI options, and cache
  side effects.
- [docs/dsp-architecture.zh.md](docs/dsp-architecture.zh.md): historical DSP
  architecture note in Chinese.
- [docs/legacy-operations.md](docs/legacy-operations.md): historical manual
  deployment notes retained for reference.
- [backup/README.md](backup/README.md): policy prohibiting runtime snapshots,
  database dumps, and third-party data in the repository.

## Development Notes

Use `GOWORK=off` for package commands from this repository. The parent
workspace's `go.work` does not include this module path, so plain
`go list ./...` fails from this checkout unless the parent workspace is changed.

Do not use legacy MySQL users in local development. The Docker helper creates
and uses the `aofei` database user.

`AOFEI` points at the lower-case DSP JSON config. `SUMMER` points at the
upper-case Summer/Genelet JSON config.

The test taxonomy is:

- package gate: `GOWORK=off go test ./...`;
- runtime hardening: `go vet`, `staticcheck`, and the scoped `-race` command
  above;
- Docker smoke: `./scripts/aofei-cache-smoke.sh` and
  `AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'`;
- admin integration: sibling `../pzdesign` Summer/Genelet tests with `SUMMER`;
- schema drift: `./scripts/aofei-local.sh check-sql` and `diff-schema`.

`.github/workflows/verify.yml` enforces the package, vet, staticcheck, scoped
race, documentation, and committed-range diff-hygiene gates on pushes and pull
requests. Local closeout still uses `git diff --check` for uncommitted changes.
Docker smoke, database-backed admin integration, and schema drift remain
explicit local/operator checks.
