# Architecture

## Package Layout

| Path | Role |
|---|---|
| `internal/jobs/cache`, `cmd/redis-cache` | Populates Redis or spread cache files from MySQL state. |
| `cmd/nats-client` | Local NATS client/log consumer command. |
| `cmd/spread` | Spread/cache command support. |
| `internal/jobs/ledger`, `cmd/ledger`, `cmd/winloss`, `cmd/maxmind` | Operational commands for ledger, win/loss, and geodata workflows. |
| `dsp/` | DSP config, controller, bid handling, and win/loss logic. |
| `match/` | Runtime matching models for advertisers, creatives, audience maps, caps, sizes, and Redis/spread serialization. |
| `acl/` | Access/control and publisher mapping helpers used by bid and cache paths. |
| `maxmind/` | Geo/IP lookup helpers and tests. |
| `etc/` | Active SQL baseline, sample configs, generated local configs, samples, and data-load helper code. |
| `scripts/` | Local Docker service helper scripts. |
| `backup/` | Historical files moved out of active paths. |
| `docs/` | Stable long-form references. |
| `memory-bank/` | Current product, architecture, tech stack, milestone, and status memory. |
| `evolution/` | Versioned history of direction changes. |

The sibling `../pzdesign` checkout is the Go module
`github.com/guruperl/pzdesign`. It owns `cmd/unify`, `genelet/`, `summer/`,
`tmpls/`, `www/`, and Summer/Genelet docs under `docs/`; its command and Summer
packages consume Aofei domain packages such as `dsp/`, `acl/`, `match/`, and
`uploaded/`.

## Runtime Data Flow

1. `scripts/aofei-local.sh` starts Docker MySQL, Redis, and NATS, then writes
   local configs into `etc/aofei.local.json` and `etc/summer.local.json`.
2. `etc/step4_init.sql` initializes the active MySQL schema and baseline data.
3. `etc/demand.sql` plus `go run ./etc pub` can load sample local demand and
   publisher data.
4. The cache job reads MySQL through `dsp.Config`, builds `PubMap`, the derived
   direct-SSP publisher-by-id lookup, `RAdv`, audience, creative, and
   Redis-only middleman route caches, discovers active creative size IDs from
   the schema, then replaces Redis cache entries or publishes spread/NATS reset
   and snapshot messages. It runs through standalone `cmd/redis-cache`;
   `cmd/unify` does not run cache refreshers. Route-only middleman cache
   publication is available through
   `cmd/redis-cache -cache=routes`. `cmd/spread` must be running when spread
   messages should become `.local/spread/` file snapshots; on startup it
   best-effort bootstraps those snapshots from Redis when Redis and MySQL are
   reachable.
5. `../pzdesign/cmd/unify` reads `SUMMER` and `AOFEI`, wires Summer/Genelet
   admin routes, and serves DSP bid paths using the same MySQL/Redis/NATS
   config.
6. Bid/win/loss/log flows use Redis for mutable runtime state and NATS/spread/log
   paths for message and log transport. Local/spread bid mode loads static
   cache snapshots into memory at controller startup and through an explicit
   reload hook; request handlers read only the current in-memory snapshot.
   OpenRTB bid requests are matched per impression; response bids are grouped
   by campaign seat. DSP-generated `/imp` and `/clk` tracker URLs are HMAC
   signed over concrete query payloads, and click redirects plus cap mutations
   require valid signatures. `/win` and `/loss` remain analytics callbacks with
   signatures over immutable packed fields so exchange auction macros can still
   be resolved by the exchange. Middleman `/mid/*` callback proxy URLs are
   signed by token and store selected-bid context in Redis. Native click links
   use `/clk` as a tracking redirect with a direct advertiser fallback; banner
   creatives opt into the same redirect through `{CLICK_URL}`.
   Direct publisher SSP traffic is a separate `POST /pz` entrypoint. The
   browser contract uses `site` packed as `(pub_id, site_id)` and
   `adUnits[].slot` packed as `(slot_id, size_id)`; the browser DOM `code` is
   not trusted as supply identity. The `/pz` adapter validates these tokens
   against the direct publisher cache, including the configured slot size,
   synthesizes internal OpenRTB browser impressions from headers and
   cache-derived site/slot strings, reuses the local Aofei bid path, and returns
   a JSON HTML-string array in ad-unit order with `""` for no-fill units. M28
   does not invoke middleman fallback for SSP traffic.
   M29 publisher tags are generated from the `pub` slot UI using configured
   `ServerURL`, stored `pub_slot.size_id`, DOM ids of the form
	   `pz-slot-<slot_id>`, and banner `mediaTypes` samples. `www/js/ads.js`
	   derives its default `/pz` endpoint from the loaded script origin and can be
	   overridden per call. `cmd/unify` applies permissive CORS headers only to
	   `POST/OPTIONS /pz`.
	   M30 identifies SSP traffic in audits with `source:"ssp"` and
	   `contract:"pz-v1"`, uses a valid browser-only `aofei_pz_uid` cookie as
	   user identity when present, and leaves missing-cookie requests on the
	   existing IP+UA fallback path. M31 keeps `/pz` CORS credentialless and
	   enforces validated `POST /pz` policy instead: browser requests must send a
	   valid `Origin` or `Referer` whose host exactly matches the cached site
	   string, any present `Origin` or `Referer` must match, and only
	   `platform:"sdk"` may omit both headers. SDK/in-app requests do not read,
	   set, or propagate `aofei_pz_uid`; until a richer mobile contract exists,
	   their identity remains the existing device-ID or UA+IP attribute fallback.
7. `cmd/nats-client` consumes NATS log subjects into `.local/logs/log_*`
   interval files. The ledger job consumes `winloss.<stamp>` files into
   interval and daily ledger tables through standalone `cmd/ledger`; missing
   input remains retryable command input. Middleman callback metadata also
   populates `ledger_mid` and `daily_mid` for advertiser pay-side reports and
   admin settlement views.
8. `cmd/maxmind` reads country and state IDs from Docker MySQL and atomically
   regenerates the configured MaxMind runtime JSON without loading the existing
   geodata file first.

Middleman AdX fallback is active behind `middleman_enabled`. Advertiser-owned
OpenRTB endpoints live in `adv_bidder`, with synthetic campaign, item, and
creative IDs for existing ledger/report joins. Summer/Genelet exposes
advertiser-safe endpoint metadata forms and admin review and approval forms.
Approval creates a missing inactive synthetic chain or validates an existing
complete same-advertiser chain, then marks the bidder credential active and the
bidder active. Operators assign active route groups to publisher/site/slot
inventory through the admin `midroute` Summer module, which writes the
`mid_route_*` tables. The Redis `middleman:routes:v2` cache contains active
route/bidder entries, trigger mode, and synthetic item ACL payloads; the legacy
`middleman:routes` key is kept fallback-only for M24 rolling-deploy safety.
`Fallback` routes apply only to local no-bid impressions. `Always` routes apply
only when both `middleman_enabled` and `middleman_always_enabled` are true, and
then marked-up middleman bids compete with local bids on effective CPM. Route
edits do not refresh the cache from `cmd/unify`; the singleton
`cmd/redis-cache -cache=redis|all` job remains the cache publication path, with
`-cache=routes` available for route-only refresh.
The cache JSON includes additive freshness metadata when generated by M24+
cache jobs, and the admin `midroute` topics/health views show Redis freshness
and route health without running refreshes. Selected middleman winners create
Redis callback context under `middleman:cb:<token>` and return signed
`/mid/win`, `/mid/loss`,
and optional `/mid/bill` URLs. `burl` is the preferred billable event and win is
the billable fallback only when no `burl` exists. Downstream callbacks receive
net payable auction prices; Aofei logs charge-side prices through the synthetic
chain and middleman-specific charge/pay/margin facts through `ledger_mid` and
`daily_mid`. Retryable downstream callback forwarding failures are stored in
MySQL `mid_callback_retry` by `/mid/*` handlers only, never by `/bid`; the
singleton `cmd/mid-callback-retry` job claims rows as `Processing` and retries
downstream forwards without republishing ledger events.

Request, response, and attribute audit messages are best-effort analytics.
`dsp.Controller` enqueues them to a bounded in-process queue after writing the
HTTP bid response, and a background publisher sends them to core NATS without
request-path flushes.

## Admin Runtime Boundary

Summer/Genelet admin code lives in the sibling `../pzdesign` module and uses
the generated `SUMMER` config and Docker MySQL. Admin tests that need a database
read `../aofei/etc/summer.local.json` when run from `../pzdesign`; they must
not use the lower-case DSP `AOFEI` config because Genelet expects
`ConnectArray`, `Template`, and `UploadDir`.

## Active Configuration Boundary

- `etc/aofei.json` and `etc/summer.example.json` are checked-in examples.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated local files
  and must remain ignored.
- Summer/Genelet code, UI templates, and static UI assets live in the sibling
  `../pzdesign` module. Generated local Summer config points `ProjectRoot` at
  that checkout, `Template` at `../pzdesign/tmpls`, and `DocumentRoot` at
  `../pzdesign/www`.
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
- Mutating operations commands acquire Redis singleton locks, and the unified
  HTTP service exposes stdlib expvar metrics at `/debug/vars`.
- Middleman callback proxying uses Redis TTL keys for selected-bid context,
  cooperative click mapping, and billable-event idempotency. These keys are
  runtime state owned by `cmd/unify`, not cache data populated by
  `cmd/redis-cache`.

## Cache Boundary

The multiple-cache split is documented in
[docs/multiple-cache.md](../docs/multiple-cache.md). Static publisher, slot,
audience, and creative data is local and snapshot-swapped in memory for
local/spread bid serving. Redis remains the shared mutable-state backend for
frequency caps, uploaded audience sets, and future counters. Frequency-cap
tracker updates keep the `bothcap:<user_id>` hash and binary `BothCap` payload,
but refresh through Redis optimistic transactions to avoid concurrent lost
updates.
Direct SSP uses an additive `pubmap:by-id` Redis hash derived from `pubmap`.
The value includes publisher domain, the active publisher object, reverse
site/slot metadata, and slot-size metadata so `/pz` can validate packed direct
tag tokens and reconstruct site and slot strings for ACL matching without a
MySQL read.
Local/static mode derives the same lookup from the loaded `pubmap` snapshot in
memory; `/bid/{domain}` continues to read the existing domain-keyed publisher
cache.

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
- Redis and spread campaign cache payloads use typed version envelopes for
  RAdvs, audience, and creative data while retaining legacy decode support.
  The middleman route Redis payload is versioned JSON.
- Direct SSP advanced API/mobile/native response contracts and richer supply
  taxonomy remain future product work; M31 keeps the v1 JSON HTML-string array
  response and identifies direct supply by `/pz` plus audit `source:"ssp"`.
- Summer/Genelet admin SQL now has a central identifier/query-building seam for
  component metadata and request-driven filters; handwritten module SQL should
  continue to use narrow allowlists for any interpolated identifiers.
- Summer/Genelet admin auth verifies stored bcrypt password hashes through the
  `Password_hash` issuer field. SHA1-era credentials are legacy data that must
  be reset before production use.
