# Documentation And Milestone Index

This directory contains the active product and operating contracts for Aofei /
W8M. Use this index to choose a guide and to trace every zero-padded product
lane to its authoritative status file. The status files record implementation
completion; deployment and commercial activation still require the gates in
the linked runbooks.

Current implemented baseline: D01 through A02 in the original strict roadmap
order, D04, D05, S06, P03, S05, and O03 are complete. P03's authenticity gates
remain disabled by default until a separately authorized named-publisher
canary. The remaining review order continues through R03 and A03. I02
remains separately demand-gated; its P03/S05 prerequisites are complete, but a
named Android or iOS integration still has not defined supported platforms and
lifecycle requirements. The active MySQL baseline is 95 tables, 0 views, 6
routines, and 61 triggers;
planned milestones do not imply schema or runtime activation.

## Start By Audience

| Audience | Start here | Then use |
|---|---|---|
| Advertisers and DSP agents | [Chinese advertiser and agent manual](advertiser-dsp-agent-manual.zh-CN.md) | [Management API](advertiser-management-api.md), [conversion attribution](conversion-attribution.md), and [middleman bidding](middleman-adx.md) when those optional features are enabled. |
| Publishers | [Chinese publisher manual](publisher-manual.zh-CN.md) | [Publisher activation](publisher-activation.md), the public [`/pz` contract](ssp-direct-traffic.md), and the [P03 authenticity contract](direct-ssp-authenticity.md). |
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
| D04 delivery/tracking integrity | Completed | [status-D04.md](../memory-bank/status-D04.md) | Existing [delivery](delivery-guardrails.md), [auction](auction-pricing-creatives.md), and [measurement](openrtb-measurement.md) contracts |
| D05 post-D04 auction remediation | Completed | [status-D05.md](../memory-bank/status-D05.md) | Existing [DSP workflow](dsp-workflow.md), [auction](auction-pricing-creatives.md), and [measurement](openrtb-measurement.md) contracts |
| P01 direct SSP readiness | Completed; publisher activation-gated | [status-P01.md](../memory-bank/status-P01.md) | [Publisher activation](publisher-activation.md) and [`/pz` contract](ssp-direct-traffic.md) |
| P02 supply and seller transparency | Completed | [status-P02.md](../memory-bank/status-P02.md) | [Supply taxonomy ADR](adr/0001-richer-supply-taxonomy.md) and [ownership ADR](adr/0002-ssp-account-schema-boundary.md) |
| P03 direct SSP authenticity | Completed; disabled by default | [status-P03.md](../memory-bank/status-P03.md) | [Authenticity contract](direct-ssp-authenticity.md), existing [`/pz` contract](ssp-direct-traffic.md), [publisher activation](publisher-activation.md), and [identity boundary](identity-access-security.md) |
| R01 conversion and attribution | Completed | [status-R01.md](../memory-bank/status-R01.md) | [Conversion and attribution](conversion-attribution.md) |
| R02 analytics and experiments | Completed | [status-R02.md](../memory-bank/status-R02.md) | [Marketplace analytics and experiments](marketplace-analytics-experiments.md) |
| R03 experiment/report integrity | Planned | [status-R03.md](../memory-bank/status-R03.md) | Existing [analytics and experiment contract](marketplace-analytics-experiments.md) |
| I01 OpenRTB interoperability | Completed | [status-I01.md](../memory-bank/status-I01.md) | [DSP workflow](dsp-workflow.md), [middleman OpenRTB](middleman-adx.md), and [adoption review](prebid-openrtb-adoption.md) |
| I02 Android/iOS publisher SDKs | Planned; demand-gated | [status-I02.md](../memory-bank/status-I02.md) | Current server contract only: [`/pz`](ssp-direct-traffic.md). Maintained native SDK packages do not exist. |
| I03 advertiser management API | Completed; disabled by default | [status-I03.md](../memory-bank/status-I03.md) | [Management API guide](advertiser-management-api.md) and [OpenAPI 3.1](management-api-openapi.yaml) |
| S01 privacy and consent | Completed | [status-S01.md](../memory-bank/status-S01.md) | [Privacy, consent, and data governance](privacy-data-governance.md) |
| S02 identity, MFA, and RBAC | Completed; disabled by default | [status-S02.md](../memory-bank/status-S02.md) | [Identity and access security](identity-access-security.md) |
| S03 traffic quality | Completed; disabled by default | [status-S03.md](../memory-bank/status-S03.md) | [Traffic quality and anti-fraud](traffic-quality-anti-fraud.md) |
| S04 template/XSS safety | Completed | [status-S04.md](../memory-bank/status-S04.md) | [Template rendering security](template-rendering-security.md) and [pzdesign rendering inventory](../../pzdesign/docs/rendering-security.md) |
| S05 runtime trust boundaries | Completed | [status-S05.md](../memory-bank/status-S05.md) | [Creative consumer boundary](creative-rendering-boundary.md), [principal provenance](principal-provenance.md), existing [privacy](privacy-data-governance.md), [rendering](template-rendering-security.md), and [traffic-quality](traffic-quality-anti-fraud.md) contracts |
| S06 public account abuse protection | Completed; active on W8M | [status-S06.md](../memory-bank/status-S06.md) | [Public account abuse protection](public-account-abuse-protection.md) |
| A01 accounting and settlement | Completed | [status-A01.md](../memory-bank/status-A01.md) | [Accounting and manual settlement](accounting-settlement.md) |
| A02 hosted funding and payout | Completed; disabled by default | [status-A02.md](../memory-bank/status-A02.md) | [Hosted funding and payout](hosted-funding-payout.md) |
| A03 exact monetary sources | Planned | [status-A03.md](../memory-bank/status-A03.md) | Existing [accounting](accounting-settlement.md), [management API](advertiser-management-api.md), and [hosted payment](hosted-funding-payout.md) contracts |
| O01 production traffic controls | Completed | [status-O01.md](../memory-bank/status-O01.md) | [Production traffic and observability](production-traffic-observability.md) |
| O02 single-region availability | Completed; production claims evidence-gated | [status-O02.md](../memory-bank/status-O02.md) | [Availability, recovery, and SLO](single-region-availability.md) |
| O03 job/cache/filesystem reliability | In progress | [status-O03.md](../memory-bank/status-O03.md) | Existing [operational commands](operational-commands.md), [cache architecture](multiple-cache.md), and [recovery](single-region-availability.md) contracts |

## Supporting Runtime References

| Subject | Documents |
|---|---|
| Runtime and matching | [DSP workflow](dsp-workflow.md), [audience matching](audience-matching.md), [multiple-cache architecture](multiple-cache.md), and [OpenRTB measurement](openrtb-measurement.md) |
| Local development | [Local Docker runtime](local-docker-runtime.md), [database baseline](database-baseline.md), and [MaxMind assets](maxmind-runtime.md) |
| Production operations | [Production runbook](production-runbook.md), [operational commands](operational-commands.md), [Chinese operations manual](operations-maintenance-manual.zh-CN.md), and [performance roadmap](performance-roadmap.md) |
| Product boundaries | [Creative consumers](creative-rendering-boundary.md), [principal provenance](principal-provenance.md), [deferred investments](defer.md), [Prebid/OpenRTB review](prebid-openrtb-adoption.md), [Cloudflare W8M Free-plan boundary](cloudflare-w8m.md), and [publisher ownership ADRs](adr/) |
| Moved source-owned docs | [Genelet manual pointer](genelet-manual.md) and [Summer UI pointer](summer-ui-structure.md) |
| Historical reference only | [Historical DSP architecture note](dsp-architecture.zh.md) and [legacy operations notes](legacy-operations.md) |

The root [milestone index](../memory-bank/milestone.md) contains the strict
dependency order and complete M-lane history. Historical documents retain the
facts and measurements true at their closeout date; this index and the current
lane files govern present behavior.
