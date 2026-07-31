# Tech Stack

## Languages And Module

- Language: Go
- Main module: `github.com/guruperl/aofei`
- Sibling admin/design module: `github.com/guruperl/pzdesign`
- `go.mod`: Go 1.22 with toolchain 1.23.5

Use `GOWORK=off` for local commands from this repository. The parent `go.work`
currently does not include this module path, and plain `go list ./...` fails
from this checkout under that parent workspace.

## Core Dependencies

- MySQL driver: `github.com/go-sql-driver/mysql`
- Redis client: `github.com/mediocregopher/radix/v4`
- NATS client: `github.com/nats-io/nats.go`
- OpenRTB: `github.com/prebid/openrtb/v20`
- Logging: `go.uber.org/zap`
- Geo/IP and user agent helpers: local `maxmind/`, `github.com/mssola/user_agent`

## Local Services

The supported local runtime uses Docker:

| Service | Image | Default bind |
|---|---|---|
| MySQL | `mysql:8.0.41` | `127.0.0.1:3307` |
| Redis | `redis:7-alpine` | `127.0.0.1:6379` |
| NATS | `nats:2-alpine` | `127.0.0.1:4222` |

Main helper:

```bash
./scripts/aofei-local.sh up
./scripts/aofei-local.sh reset
./scripts/aofei-local.sh load
./scripts/aofei-local.sh sample
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh status
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh dump-schema
./scripts/aofei-local.sh diff-schema
./scripts/aofei-local.sh install
./scripts/aofei-local.sh down
```

## Runtime Config

Generated local configs:

```bash
etc/aofei.local.json
etc/summer.local.json
```

Environment variables:

```bash
AOFEI="$PWD/etc/aofei.local.json"
SUMMER="$PWD/etc/summer.local.json"
TRACKING_SECRET="..."
```

These files are local artifacts and are ignored by git.
`tracking_secret` in the DSP config signs generated `/imp`, `/clk`, `/win`,
`/loss`, and `/mid/*` callback URLs; when omitted, `TRACKING_SECRET` is used as
the fallback. `tracking_signature_ttl_seconds` bounds signed URL replay and
defaults to 86400. `cap_state_ttl_seconds` bounds idle `bothcap:<user_id>`
retention and defaults to 7776000 seconds (90 days); cap refresh does not
shorten a longer existing TTL.
`trusted_proxy_cidrs` is empty by default. `/pz` uses `RemoteAddr` for OpenRTB
`device.ip` unless the peer address matches an explicit proxy IP or CIDR in
that list, in which case it accepts `X-Forwarded-For` or `X-Real-IP`.
Middleman fallback is controlled by `middleman_enabled`,
`middleman_timeout_ms`, `middleman_max_bidders_per_imp`, and
`middleman_exchange_domain`. `middleman_route_cache_ttl_ms` defaults to 5000
and bounds each worker's immutable decoded route snapshot; concurrent misses
share one refresh and expired routes are not used after refresh failure. The
shared load uses an independent deadline derived from `middleman_timeout_ms`;
callers wait with their own contexts and cannot cancel the load for one another.
`trigger_mode='Always'` fanout also requires
`middleman_always_enabled`; the default is false. Middleman callback proxying is
controlled by `middleman_callback_ttl_seconds`, `middleman_callback_timeout_ms`,
and `middleman_callback_base_url`; it requires `tracking_secret` and Redis.
Bidder `credential_ref` values name environment variables containing JSON
outbound header maps for downstream OpenRTB calls.

Summer/Genelet admin tests must use `SUMMER`; the Genelet config format uses
upper-case keys such as `ConnectArray`, `Template`, and `UploadDir`.
The checked-in Summer config includes `admin`, `adv`, `pub`, and `agent` roles.
Middleman bidder endpoints use the existing `adv` role through the `adv_bidder`
module. Summer/Genelet code lives in the sibling `../pzdesign` checkout together
with HTML templates under `../pzdesign/tmpls` and static UI assets under
`../pzdesign/www`. Generated local Summer config points `ProjectRoot`,
`Template`, and `DocumentRoot` at that checkout.

Production defaults are `/etc/aofei/aofei.json` and
`/etc/aofei/summer.json`, passed through `AOFEI` and `SUMMER`. The checked-in
Summer example is `etc/summer.example.json`. The production
runbook is [docs/production-runbook.md](../docs/production-runbook.md).
Summer/Genelet CORS allows the exact `ServerURL` origin plus exact entries in
`CORSOrigins`.
Genelet framework contracts are documented in
`../pzdesign/docs/genelet-manual.md`; Summer admin module and cache-side-effect
conventions are documented in `../pzdesign/docs/summer-ui-structure.md`.

## Schema Baseline Commands

`etc/step4_init.sql` is the active schema and baseline-data contract. The local
helper keeps schema stewardship commands under the same Docker workflow:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh dump-schema
./scripts/aofei-local.sh diff-schema
```

`check-sql` rejects explicit `DEFINER=` clauses and legacy account-name auth
references in `etc/step4_init.sql`. `dump-schema` writes a normalized current
Docker schema to ignored `.local/schema/aofei.schema.sql`. `diff-schema`
rebuilds a temporary database from `etc/step4_init.sql`, normalizes both dumps,
diffs baseline against the current Docker schema, and drops the temporary
database on exit.

When schema changes are intentional, update `etc/step4_init.sql`, rebuild with
`reset && load`, run `check-sql` and `diff-schema`, and update
`docs/database-baseline.md` plus the memory bank if the inventory or workflow
changed.

## Cache Commands

Run the one-command Docker cache smoke:

```bash
./scripts/aofei-cache-smoke.sh
```

Populate Redis from MySQL:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Run the bid-path smoke after `reset-sample` and Redis cache population:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go test ./dsp -run 'Test.*Smoke'
```

Read Redis cache content:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis -read
```

Populate spread/NATS cache messages:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=spread
```

Persist spread messages to `.local/spread/` by running the receiver in another
terminal before `-cache=spread` or `-cache=all`:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/spread
```

Populate spread/NATS and Redis together:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=all
```

Run Redis cache refresh from one dedicated node only, normally through cron or a
systemd timer:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Refresh only the Redis middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes
```

Read only the Redis middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes -read
```

Do not run a Redis cache refresher from every `unify` node. When spread mode is
used, keep `cmd/spread` running on nodes whose local disk snapshots should be
updated from NATS messages.

Expected Redis cache families are `pubmap`, additive direct-SSP
`pubmap:by-id`, `audience`, `creative`, `middleman:routes:v2`,
fallback-only legacy `middleman:routes`, and
`slot:<size_id>` hashes keyed by slot id. Expected spread directories are
`.local/spread/pubmap/`, `.local/spread/audience/`,
`.local/spread/creative/`, and `.local/spread/slot/<size_id>/`.
Middleman route caches are Redis-only and are populated by the singleton
`cmd/redis-cache` job, not by `cmd/unify`.
Full Redis refreshes build internal `:next` shadows and install all live
families with one transaction; live key names and payloads are unchanged.
Direct SSP local/static serving derives its by-publisher-id lookup in memory
from `.local/spread/pubmap/`; it does not add a separate spread directory. The
direct cache includes slot-size metadata, and `/pz` rejects slot tokens whose
packed size does not match the configured slot size. Inactive publisher sites
and slots are omitted from the direct SSP cache.
`POST /pz` is served by `dsp.Controller.ServeSSP` through
`../pzdesign/cmd/unify`; valid omitted or `responseFormat:"html"` requests
return `200 application/json` arrays of HTML strings. `responseFormat:"json"`
returns ordered fill/no-fill objects with markup, tracker URLs, price/currency,
ids, dimensions, and parsed native payloads when applicable.
`responseFormat:"openrtb"` returns an OpenRTB `BidResponse`, including `200`
with an empty `seatbid` on all-no-fill. Malformed JSON, unsupported response
formats, invalid direct tokens, missing slots, unsupported media, and cache
validation failures return HTTP errors.
When `middleman_enabled` is true, validated `/pz` auctions use the same
middleman route/runtime controls as ADX after local matching. Local no-fills can
fan out through `Fallback` routes, and local fills can fan out through `Always`
routes only when `middleman_always_enabled` is also true. Middleman SSP fanout
uses the synthesized internal OpenRTB request. Middleman JSON responses omit
local-only `impressionUrl` and `clickUrl`; OpenRTB responses group bids by the
final winner seat.
Duplicate non-empty `adUnits[].code` values are invalid and return `400` before
cookies, bidding, middleman fanout, or audit publishing. Empty or omitted codes
remain valid and receive generated `ssp-<index>` OpenRTB impression ids.
`../pzdesign/cmd/unify` also handles `OPTIONS /pz` and applies permissive CORS
headers only on `/pz`: origin `*`, methods `POST, OPTIONS`, and header
`Content-Type`. Publisher slot pages load `../pzdesign/www/js/ads.js`; the
script posts to the origin it was loaded from plus `/pz` unless
`pzLoadAds(payload, {endpoint: "..."})` is used.
`/pz` remains credentialless at the CORS layer. The DSP sets a best-effort
`aofei_pz_uid` cookie only for browser-cookie traffic, meaning empty or omitted
`platform` and `platform:"browser"` requests. A returned valid browser cookie
becomes OpenRTB `user.id`/`buyeruid`, while missing cookies continue through
IP+UA fallback. `platform:"sdk"` requests are cookie-free: they do not read,
set, rotate, or propagate `aofei_pz_uid`.
After packed token and cache validation, `POST /pz` enforces browser
origin/referrer policy. Browser requests, including missing or empty `platform`,
must include a valid `Origin` or `Referer` whose host exactly matches the cached
site host, and any present `Origin` or `Referer` must match. `platform:"sdk"`
may omit both headers, but supplied headers must still match. Rejections return
`403` before cookies, bidding, or audit publishing and increment
`aofei_ssp_policy_rejections_total`.
SDK/in-app `/pz` requests may include body `app`, `device`, and `user` objects.
They synthesize `BidRequest.App` and leave `BidRequest.Site` nil. The validated
cached site string becomes authoritative app id/bundle/domain, and mismatched
supplied app identity fields return `400`. Body `device` identity fields feed
the existing attribute identity precedence, with request headers as IP/UA
fallback. Body `user.id` and `buyeruid` are honored for SDK only; browser
traffic keeps browser-only `aofei_pz_uid`.
SSP request/response audit logs are JSON envelopes with `source:"ssp"` and
`contract:"pz-v1"`. ADX request/response logs remain raw OpenRTB JSON, and
attribute logs include additive `source`/`contract` fields.
`cmd/unify -local` is an explicit override: when omitted, the Aofei config's
`is_local` value is preserved; when enabled, local static snapshots are loaded
before serving requests. `local_cache_max_age_seconds` is an alert-only
freshness threshold for local/static mode. Loaded local snapshots publish
`aofei_local_cache_loaded_at_unix`; scrape-time
`aofei_local_cache_age_seconds` and `aofei_local_cache_stale` continue to update
while traffic is idle, but stale snapshots do not fail closed by age alone.

## Operational Commands

Active local operational command contracts are documented in
`docs/operational-commands.md`.

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/nats-client -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/spread
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -daily -timestamp=YYYY-MM-DD
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/mid-callback-retry -limit=100 -max-attempts=5 -timeout=2s
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/mid-callback-retry -read -json
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/winloss --bid=/bid/exchange.example.test win
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/maxmind -city=/path/to/GeoLite2-City.mmdb
```

Run ledger from the node where `cmd/nats-client` aggregates log files. Do not
run ledger on every `unify` node. When middleman callback metadata is present,
ledger also fills `ledger_mid` and `daily_mid`; advertiser reports use pay-side
spend, while admin reports expose charge, pay, and margin.
Run `cmd/mid-callback-retry` as a singleton operations job for retryable
downstream middleman callback forwarding failures. It forwards downstream only
and does not republish win/loss records. Its output includes due and stale
processing backlog counts for operator alerting; use `-json` for stable
automation fields.

Generated log directories are `.local/logs/log_request/`,
`.local/logs/log_response/`, `.local/logs/log_attribute/`, and
`.local/logs/log_winloss/`. `cmd/nats-client` creates or tightens these
directories to `0750` and generated interval files to `0640`; ledger input logs
should not be world-readable or group/world-writable. `cmd/maxmind` reads MySQL
country/state tables and atomically writes the configured MaxMind JSON path,
normally `etc/maxmind.json`.

## MaxMind Assets

`etc/maxmind.json` is the active geodata config reference. Its `city_file`
currently points to the external GeoLite2 City database at
`/media/GeoLite2-City.mmdb`.

Ignored optional local assets:

```bash
external/GeoLite2-City.mmdb
etc/GeoLite2-City.mmdb
etc/qq-pz.dat
```

Compile and pure-unit tests must pass without those files. Full lookup tests in
`maxmind` and `maxmind/ipsearch` skip with explicit messages when local assets
are absent. `AOFEI_GEOLITE_CITY_FILE` can point `maxmind` tests at a downloaded
City `.mmdb`; otherwise they fall back to `external/GeoLite2-City.mmdb` and then
`etc/GeoLite2-City.mmdb`. Details live in `docs/maxmind-runtime.md`.

## Verification

Package gate:

```bash
GOWORK=off go test ./...
```

`GOWORK=off go list ./...` should not include `github.com/guruperl/aofei/backup`;
historical Go helpers in `backup/` carry the `ignore` build tag.

Runtime hardening checks:

```bash
GOWORK=off go vet ./...
GOWORK=off staticcheck ./...
GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client
GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem
./scripts/aofei-doc-check.sh
git diff --check
```

Integration and smoke checks are explicit command families rather than hidden
package-test requirements:

```bash
./scripts/aofei-cache-smoke.sh
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
(cd ../pzdesign && GOWORK=off go test ./cmd/unify)
(cd ../pzdesign && GOWORK=off staticcheck ./cmd/unify)
```

Admin compatibility verification:

```bash
./scripts/aofei-local.sh reset-sample
(cd ../pzdesign && GOWORK=off SUMMER="$PWD/../aofei/etc/summer.local.json" go test ./genelet ./summer ./summer/pub ./summer/slot)
```

Schema baseline verification:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
```

GitHub Actions runs the package, vet, staticcheck, scoped race, documentation,
and committed-range diff-hygiene gates on pushes and pull requests. Pull
requests check merge-base-to-head and pushes check event `before`-to-`after`,
with an empty-tree fallback for initial history. The local closeout command
remains `git diff --check` so uncommitted whitespace is also covered. The
sibling pzdesign
workflow checks out public Aofei beside its own checkout to satisfy
`replace github.com/guruperl/aofei => ../aofei`. That checkout is pinned to an
explicit reviewed Aofei commit and must be bumped intentionally when pzdesign
adopts a new revision. The workflow then runs tests, the `cmd/unify` race test,
vet, staticcheck with the documented legacy style exclusions, and both
template parsers. Both workflows use Go 1.23.5 and staticcheck v0.5.1. Docker smoke,
database-backed admin tests, and schema checks remain explicit local/operator
gates because hosted CI does not start the local service stack.

## External Requirements

- Docker CLI and a working Docker daemon.
- Internet access only when pulling Docker images or Go modules.
- No production credentials are required for local development.
