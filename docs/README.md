# Documentation And Milestone Index

This directory contains the active product and operating contracts for Aofei /
W8M. Use this index to choose a guide and to trace every zero-padded product
lane to its authoritative status file. The status files record implementation
completion; deployment and commercial activation still require the gates in
the linked runbooks.

Current baseline: D01 through A02 in the strict roadmap order are complete.
I02 is the only planned lane and remains blocked until a named Android or iOS
integration defines supported platforms and lifecycle requirements. The active
MySQL baseline contains 94 tables, 0 views, 6 routines, and 55 triggers.

## Start By Audience

| Audience | Start here | Then use |
|---|---|---|
| Advertisers and DSP agents | [Chinese advertiser and agent manual](advertiser-dsp-agent-manual.zh-CN.md) | [Management API](advertiser-management-api.md), [conversion attribution](conversion-attribution.md), and [middleman bidding](middleman-adx.md) when those optional features are enabled. |
| Publishers | [Chinese publisher manual](publisher-manual.zh-CN.md) | [Publisher activation](publisher-activation.md) and the public [`/pz` contract](ssp-direct-traffic.md). |
| System operators and maintainers | [Chinese operations manual](operations-maintenance-manual.zh-CN.md) or [production runbook](production-runbook.md) | [Operational commands](operational-commands.md), [database baseline](database-baseline.md), and [single-region recovery](single-region-availability.md). |
| Developers and reviewers | Root [README](../README.md), [architecture](../memory-bank/architecture.md), and [tech stack](../memory-bank/tech-stack.md) | The lane contract below, then its status file and tests. |

## Activation Vocabulary

- **Implemented** means the reviewed code, schema, tests, and documentation are
  present.
- **Disabled by default** means operators must apply the documented migration,
  keys, permissions, canary, and rollback process before enabling it.
- **Activation-gated** means code exists but no real partner, publisher, or
  provider traffic is implied by the repository state.
- **Planned** means the product is not implemented and must not be advertised
  as available.

## Current Lane Map

| Lane | State | Authoritative status | Primary contract |
|---|---|---|---|
| D01 delivery guardrails | Completed | [status-D01.md](../memory-bank/status-D01.md) | [Campaign delivery guardrails](delivery-guardrails.md) |
| D02 auction, pricing, creatives | Completed | [status-D02.md](../memory-bank/status-D02.md) | [Auction, pricing, and creative contract](auction-pricing-creatives.md) |
| D03 external DSP / AdX middleman | Completed; activation-gated | [status-D03.md](../memory-bank/status-D03.md) | [Middleman activation](middleman-activation.md) and [runtime contract](middleman-adx.md) |
| P01 direct SSP readiness | Completed; publisher activation-gated | [status-P01.md](../memory-bank/status-P01.md) | [Publisher activation](publisher-activation.md) and [`/pz` contract](ssp-direct-traffic.md) |
| P02 supply and seller transparency | Completed | [status-P02.md](../memory-bank/status-P02.md) | [Supply taxonomy ADR](adr/0001-richer-supply-taxonomy.md) and [ownership ADR](adr/0002-ssp-account-schema-boundary.md) |
| R01 conversion and attribution | Completed | [status-R01.md](../memory-bank/status-R01.md) | [Conversion and attribution](conversion-attribution.md) |
| R02 analytics and experiments | Completed | [status-R02.md](../memory-bank/status-R02.md) | [Marketplace analytics and experiments](marketplace-analytics-experiments.md) |
| I01 OpenRTB interoperability | Completed | [status-I01.md](../memory-bank/status-I01.md) | [DSP workflow](dsp-workflow.md), [middleman OpenRTB](middleman-adx.md), and [adoption review](prebid-openrtb-adoption.md) |
| I02 Android/iOS publisher SDKs | Planned; demand-gated | [status-I02.md](../memory-bank/status-I02.md) | Current server contract only: [`/pz`](ssp-direct-traffic.md). Maintained native SDK packages do not exist. |
| I03 advertiser management API | Completed; disabled by default | [status-I03.md](../memory-bank/status-I03.md) | [Management API guide](advertiser-management-api.md) and [OpenAPI 3.1](management-api-openapi.yaml) |
| S01 privacy and consent | Completed | [status-S01.md](../memory-bank/status-S01.md) | [Privacy, consent, and data governance](privacy-data-governance.md) |
| S02 identity, MFA, and RBAC | Completed; disabled by default | [status-S02.md](../memory-bank/status-S02.md) | [Identity and access security](identity-access-security.md) |
| S03 traffic quality | Completed; disabled by default | [status-S03.md](../memory-bank/status-S03.md) | [Traffic quality and anti-fraud](traffic-quality-anti-fraud.md) |
| S04 template/XSS safety | Completed | [status-S04.md](../memory-bank/status-S04.md) | [Template rendering security](template-rendering-security.md) and [pzdesign rendering inventory](../../pzdesign/docs/rendering-security.md) |
| A01 accounting and settlement | Completed | [status-A01.md](../memory-bank/status-A01.md) | [Accounting and manual settlement](accounting-settlement.md) |
| A02 hosted funding and payout | Completed; disabled by default | [status-A02.md](../memory-bank/status-A02.md) | [Hosted funding and payout](hosted-funding-payout.md) |
| O01 production traffic controls | Completed | [status-O01.md](../memory-bank/status-O01.md) | [Production traffic and observability](production-traffic-observability.md) |
| O02 single-region availability | Completed; production claims evidence-gated | [status-O02.md](../memory-bank/status-O02.md) | [Availability, recovery, and SLO](single-region-availability.md) |

## Supporting Runtime References

| Subject | Documents |
|---|---|
| Runtime and matching | [DSP workflow](dsp-workflow.md), [audience matching](audience-matching.md), [multiple-cache architecture](multiple-cache.md), and [OpenRTB measurement](openrtb-measurement.md) |
| Local development | [Local Docker runtime](local-docker-runtime.md), [database baseline](database-baseline.md), and [MaxMind assets](maxmind-runtime.md) |
| Production operations | [Production runbook](production-runbook.md), [operational commands](operational-commands.md), [Chinese operations manual](operations-maintenance-manual.zh-CN.md), and [performance roadmap](performance-roadmap.md) |
| Product boundaries | [Deferred investments](defer.md), [Prebid/OpenRTB review](prebid-openrtb-adoption.md), and [publisher ownership ADRs](adr/) |
| Moved source-owned docs | [Genelet manual pointer](genelet-manual.md) and [Summer UI pointer](summer-ui-structure.md) |
| Historical reference only | [Historical DSP architecture note](dsp-architecture.zh.md) and [legacy operations notes](legacy-operations.md) |

The root [milestone index](../memory-bank/milestone.md) contains the strict
dependency order and complete M-lane history. Historical documents retain the
facts and measurements true at their closeout date; this index and the current
lane files govern present behavior.
