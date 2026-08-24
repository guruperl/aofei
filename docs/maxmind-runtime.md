# MaxMind Runtime Assets

`etc/maxmind.json` is the active runtime config reference for the current
MaxMind-backed IP lookup path. DSP configs point their `ips` value at this JSON
file; generated local configs normally use an absolute path to it.

## Files

| Path | Git status | Purpose |
|---|---|---|
| `etc/maxmind.json` | checked in | Runtime JSON with the City database path plus country/state ID maps. |
| `GeoLite2-City.mmdb` (`city_file`) | checked-in reference | Relative value resolved against the runtime JSON directory. |
| `.<config>.generations/<sha256>/GeoLite2-City.mmdb` | generated/ignored | Validated content-addressed City generation selected by generated JSON. |
| `.<config>.lock` | generated/ignored | Stable local lock serializing publication and retention. |
| `external/GeoLite2-City.mmdb` | external/local | Optional downloaded City `.mmdb` used by asset-backed tests when present. |
| `etc/GeoLite2-City.mmdb` | ignored | Optional local test copy for `maxmind` package lookup tests. |
| `etc/qq-pz.dat` | ignored | Optional legacy local test data for `maxmind/ipsearch`. |

Do not commit real `.mmdb`, `qq-pz.dat`, or production geodata payloads.

## Generate `etc/maxmind.json`

The generator reads country and state IDs from the MySQL schema in `AOFEI`,
copies the supplied City database into a content-addressed sibling generation,
validates its MMDB metadata, and atomically writes the configured `ips` JSON as
the final selection pointer. It reads the old JSON only to retain its selected
City generation for rollback.

```bash
./scripts/aofei-local.sh reset-sample

GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/maxmind -city=GeoLite2-City.mmdb
```

Use `-city` to set the external City `.mmdb` path written into `city_file`, or
set `AOFEI_GEOLITE_CITY_FILE`. A relative value is resolved against the
generated JSON's directory; production may instead supply an explicit absolute
source path. There is no host-specific default. The source remains an external
operator-owned asset and is never committed.

Publication is serialized on the stable sibling lock. The command hashes and
atomically copies the source at mode `0640`, verifies that it did not change
during the copy, parses the staged MMDB, retains the selected and immediately
prior generations, and only then replaces the JSON. Copy/validation/cleanup
failure therefore leaves the prior JSON selected. Files and parent-directory
renames are synced before success.

The supported City reader prefers the JSON-plus-MMDB path. An explicitly
configured `ips` filename ending in `.dat` remains a compatibility fallback and
is parsed by the strict legacy reader; malformed offsets, ranges, prefixes, or
records return errors rather than panicking. Format errors do not silently fall
through from JSON/MMDB to `.dat`. Both readers use Go-managed memory and retain
neither open files nor mmap handles.

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
3. `etc/GeoLite2-City.mmdb` (the checked-in relative runtime reference)

The test creates a temporary runtime JSON wrapper around the `.mmdb`, matching
the `LoadIPData` contract.

Pure tests also generate a minimal legacy `.dat`, exercise index zero and
malformed offset/range cases, and seed the parser fuzz target without requiring
the optional asset.

Run asset-backed checks explicitly when the optional files exist:

```bash
AOFEI_GEOLITE_CITY_FILE="$PWD/external/GeoLite2-City.mmdb" \
  GOWORK=off go test ./maxmind -run TestIpsearch

test -f etc/qq-pz.dat && GOWORK=off go test ./maxmind/ipsearch -run TestIpsearch
```
