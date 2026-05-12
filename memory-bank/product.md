# Product

## What This Project Is

`aofei` / `winter` is a Go DSP package for OpenRTB-style real-time bidding. It
combines bid request handling, advertiser/campaign matching, publisher and slot
mapping, audience and creative selection, frequency capping, admin data models,
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
- Advertisers (`adv`) can own middleman bidder endpoints for future fallback
  exchange fanout. Existing advertiser auth and reporting remain the account
  boundary, while operators retain route, credential, synthetic reporting row,
  and traffic activation control.
- Matching code turns database state into Redis and spread/static cache
  structures such as `PubMap`, `RAdv`, audience maps, and creative maps.
- DSP runtime code reads request data, config, cache entries, mutable Redis
  state, MaxMind lookup data, and logging paths to produce bid responses and
  win/loss records.
- Summer/Genelet code provides admin-model plumbing over the same schema.

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
- Middleman fallback should be implemented only after advertiser-owned endpoint,
  route configuration, synthetic reporting row, and accounting contracts are
  explicit.
- Root documentation should be short, current, and operational.
- Detailed project memory should live in `memory-bank/`.

## Non-Goals

- Restoring old root config directory runtime patterns.
- Using or documenting legacy named database credentials as active auth.
- Building production deployment automation before local harness correctness is
  settled.
- Treating moved historical files in `backup/` as active source inputs.
