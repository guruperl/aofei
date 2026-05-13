# Product

## What This Project Is

`aofei` / `winter` is a Go DSP package for OpenRTB-style real-time bidding. It
combines bid request handling, advertiser/campaign matching, publisher and slot
mapping, audience and creative selection, frequency capping, admin integration,
and local service orchestration.

The package is currently runnable from a clean local Docker harness: MySQL is
the active relational backend, Redis stores mutable runtime state and Redis-mode
cache payloads, NATS is the local message bus for spread/log flows, and
local/spread bid mode can serve static cache reads from in-process snapshots.

## Primary Users

- Developers and agents maintaining the DSP package.
- Operators who need to reset and inspect a local database/cache/message-bus
  runtime.
- Admin UI users managing advertisers, publishers, campaigns, slots, creatives,
  ledger records, and targeting data.
- Bid path integrations that send OpenRTB requests and expect deterministic
  low-latency responses.

## Core Domain

- Advertisers (`adv`) own campaigns, items, creatives, balances, and targeting.
- Publishers (`pub`) expose sites, slots, publisher attributes, and traffic
  metadata.
- Advertisers (`adv`) can own middleman bidder endpoints for fallback exchange
  fanout. Existing advertiser auth and reporting remain the account boundary.
  Advertisers can create and edit safe endpoint metadata in Summer/Genelet,
  while operators retain route, credential, synthetic reporting row, and traffic
  activation control. Operators manage route groups, route bidders, and traffic
  targets through the admin Summer/Genelet `midroute` UI. The synthetic
  campaign/item chain reuses the existing ACL and channel matching model to
  decide which original publisher/site inventory may be forwarded to a bidder.
- Matching code turns database state into Redis and spread/static cache
  structures such as `PubMap`, `RAdv`, audience maps, and creative maps.
- DSP runtime code reads request data, config, cache entries, mutable Redis
  state, MaxMind lookup data, and logging paths to produce bid responses and
  win/loss records.
- Redis cache refresh and ledger aggregation remain singleton scheduled jobs,
  normally on dedicated cache and log aggregation nodes rather than every
  `cmd/unify` node.
- The sibling `github.com/guruperl/pzdesign` module provides Summer/Genelet
  admin-model plumbing over the same schema and imports this module's domain
  packages where needed.

## Current Product Direction

The prior hardening direction remains active, but the next feature direction is
to add middleman AdX fallback in staged milestones after establishing the
advertiser-owned endpoint and reporting schema boundary:

- Local runtime should be Docker-backed and free of historical production auth.
- The active schema should be represented by `etc/step4_init.sql`.
- Redis and NATS should be available locally through the same helper flow as
  MySQL.
- Static publisher, slot, audience, and creative data should be inspectable as
  Redis payloads, spread disk snapshots, and local in-process generations.
- Middleman fallback is enabled only by explicit DSP config after
  advertiser-owned endpoint approval, route cache population, synthetic
  reporting row validation, and ACL/channel eligibility checks.
- Middleman route operations expose cache freshness and health in Summer while
  keeping cache publication on the singleton `cmd/redis-cache` node.
- Retryable downstream middleman callback forwarding failures are queued
  durably after `/mid/*` callbacks and retried by a singleton operations
  command; `/bid` remains cache/Redis-only and does not write MySQL retry rows.
- `trigger_mode='Always'` middleman auction expansion is gated by
  `middleman_always_enabled`; when enabled, eligible marked-up middleman bids
  compete with local bids by effective CPM.
- Direct publisher SSP traffic is the next post-M26 product direction. The
  existing `pub` role remains the publisher account and inventory owner. The v1
  browser contract is `POST /pz` with packed `site` and `adUnits[].slot` tokens
  and a JSON array of HTML strings in ad-unit order. M28 serves valid requests
	  through the existing local Aofei bid path. M29 adds publisher slot tag
	  copy/download UI, stored slot sizes, external `ads.js` endpoint resolution,
	  and endpoint-limited permissive `/pz` CORS. M30 adds SSP audit-source
	  separation, browser-only best-effort cookies with IP+UA fallback, and
	  ledger compatibility checks. M31 adds exact cached-site-host
	  origin/referrer enforcement for browser traffic while keeping `/pz` CORS
	  credentialless and keeps `/pz` plus audit `source:"ssp"` as the current
	  direct SSP source boundary. M32 adds
	  mobile/API serving on the same `/pz` and `pub` boundary by accepting
	  SDK `app`, `device`, and `user` objects, honoring explicit
	  `responseFormat:"json"` and `"openrtb"` outputs, and preserving omitted or
	  `"html"` browser responses as the existing ordered HTML-string array.
	  Richer supply taxonomy remains an ADR milestone rather than runtime schema
	  work.
- Root documentation should be short, current, and operational.
- Detailed project memory should live in `memory-bank/`.

## Non-Goals

- Restoring old root config directory runtime patterns.
- Using or documenting legacy named database credentials as active auth.
- Building production deployment automation before local harness correctness is
  settled.
- Treating moved historical files in `backup/` as active source inputs.
- Introducing a separate `ssp` account role for the v1 direct publisher path.
