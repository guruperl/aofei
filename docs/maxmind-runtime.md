# MaxMind Runtime Assets

`etc/maxmind.json` is the active runtime config reference for the current
MaxMind-backed IP lookup path. DSP configs point their `ips` value at this JSON
file; generated local configs normally use an absolute path to it.

## Files

| Path | Git status | Purpose |
|---|---|---|
| `etc/maxmind.json` | checked in | Runtime JSON with the City database path plus country/state ID maps. |
| `/media/GeoLite2-City.mmdb` | external | Default City `.mmdb` source referenced by `city_file`. |
| `external/GeoLite2-City.mmdb` | external/local | Optional downloaded City `.mmdb` used by asset-backed tests when present. |
| `etc/GeoLite2-City.mmdb` | ignored | Optional local test copy for `maxmind` package lookup tests. |
| `etc/qq-pz.dat` | ignored | Optional legacy local test data for `maxmind/ipsearch`. |

Do not commit real `.mmdb`, `qq-pz.dat`, or production geodata payloads.

## Generate `etc/maxmind.json`

The generator reads country and state IDs from the MySQL schema in `AOFEI` and
writes the configured `ips` path atomically. It does not load the existing
MaxMind runtime JSON before writing the replacement.

```bash
./scripts/aofei-local.sh reset-sample

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/maxmind -city=/media/GeoLite2-City.mmdb
```

Use `-city` to set the external City `.mmdb` path written into
`city_file`. The command does not copy or validate the `.mmdb` payload.

## Tests

Compile and pure-unit tests must pass without local geodata assets:

```bash
GOWORK=off go test ./maxmind ./maxmind/ipsearch -run '^$'
GOWORK=off go test ./cmd/maxmind
```

Full package tests skip asset-backed lookups when local files are absent:

```bash
GOWORK=off go test ./maxmind ./maxmind/ipsearch
```

`maxmind` package asset-backed tests look for the City `.mmdb` in this order:

1. `AOFEI_GEOLITE_CITY_FILE`
2. `external/GeoLite2-City.mmdb`
3. `etc/GeoLite2-City.mmdb`

The test creates a temporary runtime JSON wrapper around the `.mmdb`, matching
the `LoadIPData` contract.

Run asset-backed checks explicitly when the optional files exist:

```bash
AOFEI_GEOLITE_CITY_FILE="$PWD/external/GeoLite2-City.mmdb" \
  GOWORK=off go test ./maxmind -run TestIpsearch

test -f etc/qq-pz.dat && GOWORK=off go test ./maxmind/ipsearch -run TestIpsearch
```
