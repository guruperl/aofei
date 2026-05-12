# Architecture

## Package Layout

| Path | Role |
|---|---|
| `cmd/unify` | Main combined service entrypoint for Summer/Genelet admin and DSP handlers. |
| `cmd/redis-cache` | Populates Redis or spread cache files from MySQL state. |
| `cmd/nats-client` | Local NATS client/log consumer command. |
| `cmd/spread` | Spread/cache command support. |
| `cmd/ledger`, `cmd/winloss`, `cmd/maxmind` | Operational commands for ledger, win/loss, and geodata workflows. |
| `dsp/` | DSP config, controller, bid handling, and win/loss logic. |
| `match/` | Runtime matching models for advertisers, creatives, audience maps, caps, sizes, and Redis/spread serialization. |
| `acl/` | Access/control and publisher mapping helpers used by bid and cache paths. |
| `summer/` | Admin UI data models, filters, components, and OpenRTB-oriented admin entities. |
| `genelet/` | Local web/admin framework helpers used by Summer. |
| `maxmind/` | Geo/IP lookup helpers and tests. |
| `etc/` | Active SQL baseline, sample configs, generated local configs, samples, and data-load helper code. |
| `scripts/` | Local Docker service helper scripts. |
| `backup/` | Historical files moved out of active paths. |
| `docs/` | Stable long-form references. |
| `memory-bank/` | Current product, architecture, tech stack, milestone, and status memory. |
| `evolution/` | Versioned history of direction changes. |

## Runtime Data Flow

1. `scripts/aofei-local.sh` starts Docker MySQL, Redis, and NATS, then writes
   local configs into `etc/aofei.local.json` and `etc/summer.local.json`.
2. `etc/step4_init.sql` initializes the active MySQL schema and baseline data.
3. `etc/demand.sql` plus `go run ./etc pub` can load sample local demand and
   publisher data.
4. `cmd/redis-cache` reads MySQL through `dsp.Config`, builds `PubMap`, `RAdv`,
   audience, and creative caches, discovers active creative size IDs from the
   schema, then writes Redis cache entries or publishes spread/NATS messages.
   `cmd/spread` must be running when spread messages should become
   `.local/spread/` file snapshots.
5. `cmd/unify` reads `SUMMER` and `AOFEI`, wires Summer/Genelet admin routes,
   and serves DSP bid paths using the same MySQL/Redis/NATS config.
6. Bid/win/loss/log flows use Redis for runtime state and NATS/spread/log paths
   for message and log transport.
7. `cmd/nats-client` consumes NATS log subjects into `.local/logs/log_*`
   interval files, and `cmd/ledger` consumes `winloss.<stamp>` files into
   interval and daily ledger tables.

## Admin Runtime Boundary

Summer/Genelet admin code uses the generated `SUMMER` config and Docker MySQL.
Admin tests that need a database read `etc/summer.local.json`; they must not use
the lower-case DSP `AOFEI` config because Genelet expects `ConnectArray`,
`Template`, and `UploadDir`.

## Active Configuration Boundary

- `etc/aofei.json` and `etc/summer.json` are checked-in examples.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated local files
  and must remain ignored.
- `etc/maxmind.json` is the active MaxMind config reference.
- `conf/` is no longer active and should not be recreated.
- Operational commands use the generated `AOFEI` config. `cmd/ledger`,
  `cmd/winloss`, and `cmd/maxmind` disable controller NATS and MaxMind startup
  explicitly when they only need database/config access.

## Database Boundary

`etc/step4_init.sql` is the source-of-truth baseline for local MySQL. When Docker
MySQL schema changes are intentionally made, export or otherwise update
`etc/step4_init.sql` in the same change. The baseline must not contain explicit
legacy definers or `eightran_*` auth references.

## Known Architecture Gaps

- Full `go test ./...` is not yet the canonical verification target because
  historical Go files moved into `backup/` still appear as a Go package.
- Production deployment notes are historical; the current supported workflow is
  local Docker development.
- Runtime config parsing needs one validation/defaulting boundary across DSP
  and Summer/Genelet so missing service blocks fail with actionable errors.
- Redis and spread cache payloads need typed/versioned contracts instead of
  direct struct serialization.
- Summer/Genelet admin SQL needs a whitelisted identifier/query-building seam
  before request or component metadata is interpolated into statements.
- Production hardening still needs explicit decisions for credentials, CORS,
  uploads, static file serving, and legacy password hashing.
