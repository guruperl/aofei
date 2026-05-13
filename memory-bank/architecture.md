# Architecture

## Package Layout

| Path | Role |
|---|---|
| `cmd/unify` | Main combined service entrypoint for Summer/Genelet admin and DSP handlers. |
| `internal/jobs/cache`, `cmd/redis-cache` | Populates Redis or spread cache files from MySQL state. |
| `cmd/nats-client` | Local NATS client/log consumer command. |
| `cmd/spread` | Spread/cache command support. |
| `internal/jobs/ledger`, `cmd/ledger`, `cmd/winloss`, `cmd/maxmind` | Operational commands for ledger, win/loss, and geodata workflows. |
| `dsp/` | DSP config, controller, bid handling, and win/loss logic. |
| `match/` | Runtime matching models for advertisers, creatives, audience maps, caps, sizes, and Redis/spread serialization. |
| `acl/` | Access/control and publisher mapping helpers used by bid and cache paths. |
| `summer/` | Admin UI data models, filters, components, and OpenRTB-oriented admin entities, including advertiser-owned bidder endpoint modules. |
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
4. The cache job reads MySQL through `dsp.Config`, builds `PubMap`, `RAdv`,
   audience, and creative caches, discovers active creative size IDs from the
   schema, then replaces Redis cache entries or publishes spread/NATS reset and
   snapshot messages. It runs through standalone `cmd/redis-cache` or opt-in
   `cmd/unify` background flags. `cmd/spread` must be running when spread
   messages should become `.local/spread/` file snapshots; on startup it
   best-effort bootstraps those snapshots from Redis when Redis and MySQL are
   reachable.
5. `cmd/unify` reads `SUMMER` and `AOFEI`, wires Summer/Genelet admin routes,
   and serves DSP bid paths using the same MySQL/Redis/NATS config.
6. Bid/win/loss/log flows use Redis for mutable runtime state and NATS/spread/log
   paths for message and log transport. Local/spread bid mode loads static
   cache snapshots into memory at controller startup and through an explicit
   reload hook; request handlers read only the current in-memory snapshot.
   OpenRTB bid requests are matched per impression; response bids are grouped
   by campaign seat. DSP-generated `/imp` and `/clk` tracker URLs are HMAC
   signed over concrete query payloads, and click redirects plus cap mutations
   require valid signatures. `/win` and `/loss` remain analytics callbacks with
   signatures over immutable packed fields so exchange auction macros can still
   be resolved by the exchange. Native click links use `/clk` as a tracking
   redirect with a direct advertiser fallback; banner creatives opt into the
   same redirect through `{CLICK_URL}`.
7. `cmd/nats-client` consumes NATS log subjects into `.local/logs/log_*`
   interval files. The ledger job consumes `winloss.<stamp>` files into
   interval and daily ledger tables through standalone `cmd/ledger` or opt-in
   `cmd/unify` background flags; missing input remains retryable and non-fatal
   in embedded mode.
8. `cmd/maxmind` reads country and state IDs from Docker MySQL and atomically
   regenerates the configured MaxMind runtime JSON without loading the existing
   geodata file first.

Middleman AdX fallback is schema-defined but not yet active in the bid path.
Advertiser-owned OpenRTB endpoints live in `adv_bidder`, with optional
synthetic campaign, item, and creative IDs for existing ledger/report joins.
Summer/Genelet exposes advertiser-safe endpoint metadata forms and admin review
and approval forms. Approval creates a missing inactive synthetic chain or
validates an existing complete same-advertiser chain, then marks the bidder
credential active and the bidder active. Operators will later assign active
route groups to publisher/site/slot inventory through `mid_route_*` tables. The
synthetic item/campaign chain is also the planned eligibility surface for bidder
fanout: existing `ac`, `ch_ac`, `ch_belong`, `access_order`, `fl_sitetypes`, and
channel matching rules should decide whether a bidder may receive the original
publisher/site request before a downstream call is made. Runtime fanout, route
caching, callback proxying, and reporting integration remain future milestones.

Request, response, and attribute audit messages are best-effort analytics.
`dsp.Controller` enqueues them to a bounded in-process queue after writing the
HTTP bid response, and a background publisher sends them to core NATS without
request-path flushes.

## Admin Runtime Boundary

Summer/Genelet admin code uses the generated `SUMMER` config and Docker MySQL.
Admin tests that need a database read `etc/summer.local.json`; they must not use
the lower-case DSP `AOFEI` config because Genelet expects `ConnectArray`,
`Template`, and `UploadDir`.

## Active Configuration Boundary

- `etc/aofei.json` and `etc/summer.json` are checked-in examples.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated local files
  and must remain ignored.
- Summer/Genelet UI templates live in the sibling `../pzdesign/tmpls` tree, and
  static UI assets live under `../pzdesign/www`; generated local Summer config
  points `Template` and `DocumentRoot` at those paths.
- Production configs default to `/etc/aofei/aofei.json` and
  `/etc/aofei/summer.json`, passed through `AOFEI` and `SUMMER`.
- `etc/maxmind.json` is the active MaxMind config reference.
- `etc/maxmind.json` references an external GeoLite2 City `.mmdb` through
  `city_file`, currently `/media/GeoLite2-City.mmdb`.
- Real geodata payloads are external runtime/test assets. `etc/GeoLite2-City.mmdb`
  and `etc/qq-pz.dat` are ignored and must not be committed.
- The retired root config directory is no longer active and should not be
  recreated.
- Operational commands use the generated `AOFEI` config. `cmd/ledger`,
  `cmd/winloss`, and `cmd/maxmind` disable controller NATS and MaxMind startup
  explicitly when they only need database/config access. Redis cache refresh
  remains a singleton scheduled `cmd/redis-cache` job on one dedicated node.
  Ledger runs as a singleton scheduled `cmd/ledger` job on the node where
  `cmd/nats-client` aggregates win/loss log files.

## Cache Boundary

The multiple-cache split is documented in
[docs/multiple-cache.md](../docs/multiple-cache.md). Static publisher, slot,
audience, and creative data is local and snapshot-swapped in memory for
local/spread bid serving. Redis remains the shared mutable-state backend for
frequency caps, uploaded audience sets, and future counters. Frequency-cap
tracker updates keep the `bothcap:<user_id>` hash and binary `BothCap` payload,
but refresh through Redis optimistic transactions to avoid concurrent lost
updates.

## Database Boundary

`etc/step4_init.sql` is the source-of-truth baseline for local MySQL. When Docker
MySQL schema changes are intentionally made, export or otherwise update
`etc/step4_init.sql` in the same change. The baseline must not contain explicit
legacy definers or legacy named database auth references.

## Known Architecture Gaps

- The parent `go.work` still does not include this module path, so repository
  package commands require `GOWORK=off` unless that workspace is intentionally
  changed.
- Production deployment now has a Linux systemd-oriented runbook; local Docker
  remains the development workflow, not the production ownership model.
- Runtime config parsing needs one validation/defaulting boundary across DSP
  and Summer/Genelet so missing service blocks fail with actionable errors.
- Redis and spread cache payloads need typed/versioned contracts instead of
  direct struct serialization.
- Summer/Genelet admin SQL now has a central identifier/query-building seam for
  component metadata and request-driven filters; handwritten module SQL should
  continue to use narrow allowlists for any interpolated identifiers.
- Production auth hardening still needs a future migration from SHA1-era
  password compatibility to a modern password-hash contract.
