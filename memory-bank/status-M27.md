# Status M27 - SSP Contract And Cache Lookup Foundation

## Goal

Define the direct publisher SSP v1 contract and make packed direct tag supply
tokens resolvable from cache before runtime `/pz` serving is added.

## Tasks

- `[+]` Document the v1 `/pz` browser contract.
  - Added `docs/ssp-direct-traffic.md`.
  - Request fields: `id`, `platform`, `site`, `adUnits[].code`,
    `adUnits[].slot`, `adUnits[].mediaTypes`, and optional `floor`.
  - `site` remains packed `(pub_id, site_id)`.
  - `adUnits[].slot` remains packed `(slot_id, size_id)`.
  - `adUnits[].code` is a DOM element id only for the v1 contract.

- `[+]` Add direct publisher cache lookup by `pub_id`.
  - Added `acl.DirectPub`, `acl.DirectPubMap`, and Redis hash
    `pubmap:by-id`.
  - The by-id value carries the publisher domain, active publisher object, and
    reverse `site_id`/`slot_id` metadata for future ACL matching.
  - Redis static cache reset deletes the additive by-id hash.
  - Local/static cache derives the same lookup in memory from `pubmap`.

- `[+]` Add direct token parser and supply validation foundation.
  - Added historical SSP base32 no-padding little-endian token pack/unpack
    helpers.
  - Added `dsp.ParseSSPRequest` and `SSPRequest.ValidateSupply`.
  - Validation unpacks `site`, looks up/matches `pub_id`, unpacks each slot
    token, validates site/slot pairs, and reconstructs site/slot strings.

- `[+]` Add parser/validation tests for current and historical samples.
  - Current Pzdesign-style sample covers `site` string, `slot`, `code`, and
    nested `mediaTypes`.
  - Historical Holiday-style sample covers object `site.ID`, legacy code-as-slot
    parsing, and top-level banner/video/native media fields.
  - V1 supply validation rejects legacy code-as-slot requests because `code` is
    not trusted as supply identity.

- `[+]` Resolve post-review cache/parser/API findings.
  - `acl.Pub.ToRedis` now updates or deletes additive `pubmap:by-id` entries
    alongside `pubmap` for admin incremental publisher cache updates.
  - Direct token unpacking rejects decoded payloads that are not exactly two
    `uint32` values.
  - The legacy code-as-slot parser helper is unexported so future runtime
    serving code uses validated `slot` tokens.

- `[+]` Resolve M27-M29 deep-review direct cache finding.
  - Direct publisher cache payloads now include `pub_slot.size_id` metadata for
    each cached slot.
  - `/pz` supply validation rejects forged slot tokens whose packed `size_id`
    does not match the configured cached slot size.

## Acceptance

- `[+]` No `/pz` runtime serving is wired in M27.
- `[+]` Direct tag tokens parse and validate against cached publisher/site/slot
  data without MySQL on the request path.
- `[+]` Existing `/bid/{domain}` publisher lookup remains domain-keyed through
  `pubmap`; the new `pubmap:by-id` hash is additive.

## Verification

- `[+]` `GOWORK=off go test ./acl ./dsp ./internal/jobs/cache`
- `[+]` `GOWORK=off go test ./dsp ./acl ./match ./adminapi`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-cache-smoke.sh`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `bash -n scripts/aofei-cache-smoke.sh`
- `[+]` `git diff --check`
