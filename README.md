# Aofei / Winter DSP

`github.com/guruperl/aofei` is the Go domain and runtime package for the W8M
advertising marketplace. It owns OpenRTB bidding, campaign and publisher
matching, direct publisher SSP traffic on `POST /pz`, config-gated external
DSP / AdX middleman bidding, measurement, cache compilers, operational jobs,
the active MySQL baseline, and the local Docker harness.

The sibling `../pzdesign` module owns `cmd/unify`, the Summer management UI,
templates, and public assets. It imports the sibling `../genelet` framework and
this module's domain packages. Run and verify the three modules separately;
the parent workspace does not include this module.

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
GOWORK=off go test -race ./hostedpayment ./trafficquality ./dsp ./match ./internal/cmdboot ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./internal/jobs/action ./cmd/spread ./cmd/nats-client ./cmd/action-measurement ./cmd/hosted-payment ./cmd/traffic-quality
./scripts/aofei-doc-check.sh
./scripts/aofei-public-data-check.sh
gitleaks git --redact .
git diff --check
```

Rehearse a clean MySQL restore and derived Redis-cache rebuild without touching
the configured local stack:

```bash
./scripts/aofei-recovery-drill.sh
```

The drill uses uniquely named disposable containers and destroys its
owner-only temporary dump on exit. It is not a production backup or SLO result.

Run the operational command package gate:

```bash
GOWORK=off go test ./cmd/accounting ./cmd/action-measurement \
  ./cmd/hosted-payment ./cmd/ledger ./cmd/maxmind \
  ./cmd/mid-callback-retry ./cmd/nats-client ./cmd/redis-cache \
  ./cmd/report-experiment ./cmd/spread ./cmd/traffic-quality ./cmd/winloss
```

See [docs/operational-commands.md](docs/operational-commands.md) for the local
contracts for `cmd/redis-cache`, `cmd/ledger`, `cmd/accounting`,
`cmd/action-measurement`, `cmd/report-experiment`, `cmd/hosted-payment`,
`cmd/traffic-quality`, `cmd/nats-client`, `cmd/winloss`, `cmd/spread`,
`cmd/maxmind`, and `cmd/mid-callback-retry`,
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
  go test ./...)
(cd ../genelet && GOWORK=off go test ./...)
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

The helper installs the core Aofei runtime commands and `../pzdesign/cmd/unify`.
Other feature-specific restricted maintenance commands are documented with their
placement and direct `go install` invocations in
[docs/operational-commands.md](docs/operational-commands.md).

## Capability And Activation Status

The original D/P/R/I/S/A/O baseline is implemented through A02. D04, D05, and
S06 are complete. P03 is in progress with its threat and compatibility
contract plus default-off versioned locator codec/runtime dual reader complete;
its remaining implementation precedes S05, O03, R03, and A03. I02 remains
separately demand-gated. Implementation does not imply production activation:

| Area | Current state |
|---|---|
| D01 delivery, D02 auction/creative safety | Implemented and part of the core runtime contract. |
| D03 external DSP / AdX middleman | Implemented; checked-in disclosure and traffic gates remain off until a named partner passes staged activation. |
| D04 delivery/tracking integrity | Implemented; confirmed ACL, callback, cap-time, bounded-input, and tracking semantic corrections are complete. |
| D05 post-D04 auction remediation | Implemented; legacy cap migration compatibility, invalid-cap isolation, optional OpenRTB dimensions, and bid hot-path cleanup are complete. |
| P01 direct SSP, P02 supply transparency | Implemented; each publisher still requires inventory, privacy, cache, reporting, and settlement acceptance. |
| P03 direct SSP authenticity | In progress; the contract and default-off v2 locator codec/dual reader are implemented, while SDK authentication, integration, enforcement, and rollout remain pending. |
| R01 attribution, R02 analytics/experiments | Implemented; experiments are observational and cannot change bids or budgets. |
| R03 experiment/report integrity | Planned; will version assignment privacy and strengthen analytical validation. |
| I01 OpenRTB interoperability | Implemented as a bounded OpenRTB 2.5 profile. |
| I02 Android/iOS publisher SDKs | Planned and demand-gated; `/pz` JSON/OpenRTB examples exist, but maintained native SDK packages do not. |
| I03 advertiser management API | Implemented but independently disabled by default. |
| S01 privacy, S04 rendering safety | Implemented core boundaries. |
| S02 identity/RBAC, S03 traffic quality | Implemented but independently disabled by default pending migration, keys, permissions, and rollout evidence. |
| S05 runtime trust boundaries | Planned; owns outbound-network, creative-consumer, principal, and quality-version hardening. |
| S06 public account abuse protection | Implemented and active on W8M with its scoped Cloudflare widget and Free-plan exact-path rate rule. |
| A01 manual accounting | Implemented and remains the financial authority and outage fallback. |
| A02 hosted funding/payout | Implemented but disabled by default; live provider use requires separate legal, finance, tax, risk, privacy, and support approval. |
| A03 exact monetary sources | Planned as a versioned schema/API/cache migration; existing historical float precision is not overstated. |
| O01 traffic controls, O02 single-region availability | Implemented operating contracts; no production 99.9% or provider-backed RPO/RTO claim is made without retained production evidence. |
| O03 job/cache/filesystem reliability | Planned; owns singleton, publication, spread, filesystem, and geodata hardening. |

See the [documentation and milestone index](docs/README.md) for the authoritative
guide for each lane and the matching status file. Historical M-lane evidence is
kept in `memory-bank/`; it is not a current deployment guide.

## Repository Map

- `dsp/`, `match/`, and `acl/`: request handling, matching, cache models, and
  publisher/access-control logic.
- `internal/jobs/` and `cmd/`: cache, measurement, reporting, accounting,
  quality, payment, log, and maintenance commands.
- `etc/`: active SQL baseline, safe example config, local-only generated config,
  and synthetic fixture data.
- `scripts/`: local Docker orchestration, cache/recovery smoke, benchmarks, and
  repository guards.
- [docs/README.md](docs/README.md): complete documentation index by audience and
  A/D/I/O/P/R/S lane.
- [memory-bank/](memory-bank/): current product, architecture, toolchain,
  milestone, and per-lane status source of truth.
- [AGENTS.md](AGENTS.md), [GOAL.md](GOAL.md), and [SECURITY.md](SECURITY.md):
  repository work protocol, reusable multi-milestone loop with bounded
  review/fix iterations, and private security reporting/data-handling rules.
- `../pzdesign` and `../genelet`: separately versioned service/UI and Web
  framework modules.
- [backup/README.md](backup/README.md): policy that keeps operational snapshots,
  database dumps, and third-party data outside Git.

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
- admin integration: sibling `../pzdesign` tests with `SUMMER`, plus the
  separately versioned `../genelet` suite;
- schema drift: `./scripts/aofei-local.sh check-sql` and `diff-schema`.

`.github/workflows/verify.yml` enforces package tests, vet, pinned staticcheck,
the scoped race suite, documentation, public-data and Gitleaks guards, and
event-aware committed-range diff hygiene on pushes and pull requests. Local
closeout still uses `git diff --check` for uncommitted changes. Docker smoke,
database-backed admin integration, recovery rehearsal, and schema drift remain
explicit local/operator checks.
