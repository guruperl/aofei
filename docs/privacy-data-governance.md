# Privacy, Consent, And Data Governance

This document is the technical privacy contract for W8M bid traffic. It
describes what the software enforces; it is not legal advice and does not by
itself establish compliance. Before production personalized advertising, the
operator must have counsel approve the markets, purposes, vendor registration,
publisher notice, partner terms, and data-subject workflow represented by the
configuration.

The implementation follows the minimization, purpose limitation, storage
limitation, integrity, and accountability direction described by the
[European Commission's GDPR principles](https://commission.europa.eu/law/law-topic/data-protection/information-business-and-organisations/principles-gdpr_en).
Its signal vocabulary follows OpenRTB 2.6 `regs` fields and the
[IAB Tech Lab GPP](https://iabtechlab.com/gpp/) and
[OpenRTB](https://iabtechlab.com/standards/openrtb/) specifications. COPPA and
California opt-out review must use the current
[FTC COPPA guidance](https://www.ftc.gov/business-guidance/resources/complying-coppa-frequently-asked-questions)
and [California Attorney General CCPA guidance](https://oag.ca.gov/privacy/ccpa).
TCF configuration must follow the current
[IAB Europe TCF policies](https://iabeurope.eu/iab-europe-transparency-consent-framework-policies/).

## Enforced Decision Contract

Every valid `/bid` and `/pz` request receives exactly one internal decision:

| Input | Runtime mode | Identity/cookie | External bidder |
|---|---|---|---|
| `regs.coppa=1` | `restricted` | Never | Never |
| `Sec-GPC: 1`, `DNT: 1`, `device.dnt=1`, or `device.lmt=1` | `contextual` | Never | Contextual-only when explicitly enabled |
| Valid US Privacy string with sale or LSPA opt-out | `contextual` | Never | Contextual-only when explicitly enabled |
| Valid GPP envelope plus applicable section ids | `contextual` | Never | Contextual-only when explicitly enabled |
| Malformed, contradictory, or unsupported signal | `contextual` | Never | Contextual-only when explicitly enabled |
| `regs.gdpr=1` without an accepted TCF grant | `contextual` | Never | Contextual-only when explicitly enabled |
| Current service-specific TCF v2 grant for every configured purpose and the configured, disclosed W8M vendor | `personalized` | Allowed | Still receives an independently contextualized request |
| `regs.gdpr=0` or no signal | `contextual` | Never | Contextual-only when explicitly enabled |

The most restrictive applicable signal wins. Invalid signals never become an
implicit grant. GPP jurisdiction sections are carried in a scrubbed external
request when relevant, but W8M does not yet interpret a GPP section as an
affirmative personalization grant. This avoids pretending that one generic
parser has resolved market-specific sale, sharing, sensitive-data, and purpose
rules. A reviewed mapping can be added later without weakening today's default.

`privacy_tcf_vendor_id=0` disables personalized processing. A nonzero vendor
id, an available `tracking_secret`, a policy version at or above
`privacy_tcf_min_policy_version`, valid CMP/version metadata, all
`privacy_tcf_purpose_ids`, vendor consent, compatible publisher restrictions,
and the TCF v2.3 disclosed-vendors segment are all required. The defaults are
purposes 1, 3, and 4 and minimum policy version 5; these defaults are technical
fixtures, not a legal conclusion for a particular deployment.

## Data Inventory

| Data | Source | Allowed purpose | Processing and recipient | Retention |
|---|---|---|---|---|
| IP address and raw user agent | ADX request or direct publisher HTTP/SDK request | Transient device, coarse geo, security, and eligible audience derivation | Local process only for an accepted personalized request; removed from contextual matching, audits, and bidder fanout | Request memory only |
| Browser `aofei_pz_uid` | Random value created by `/pz` | Local frequency control and measurement identity | Created/read only for an accepted personalized browser request; HMAC-pseudonymized before cap/tracking state | Cookie default 30 days; configured by `privacy_browser_id_ttl_seconds` |
| IFA, buyer id, user id, deprecated device ids | OpenRTB/SDK body | Approved local audience matching and frequency control | Raw value may be used transiently only in personalized local matching; HMAC domain-separated before cap keys or tracking payloads; never sent to middleman or audits | Request memory; pseudonymous cap state default 90 idle days |
| Exact geo and demographics | OpenRTB/SDK body or IP lookup | Approved local audience matching | Latitude, longitude, accuracy, and fix age are removed in every mode; other detailed geo and demographics are removed in contextual/restricted mode and from audits/bidder fanout | Request memory only |
| Coarse country/state, language, device/OS class, publisher/site/slot, content categories | Request and publisher cache | Contextual matching, aggregate reporting, routing | Local matcher; scrubbed contextual bidder view; privacy-safe attribute audit | Audit default 168 hours; aggregate ledger by its accounting policy |
| Approved supply taxonomy and seller transparency | Operator-approved MySQL publisher/site/slot rows and additive publisher cache | Inventory review, aggregate reporting, and standard OpenRTB seller disclosure | Public allowlisted environment/integration/placement/quality categories and authorized seller id/type/ASI/name/domain may enter privacy-safe attribute audits, derived reports, and a validated `source.schain`; raw client claims, private contracts, endpoints, credentials, and uncontrolled extensions are excluded | Publisher account/inventory lifecycle; derived audit/report retention follows its own policy |
| Consent and regulatory strings | OpenRTB/SDK body and HTTP headers | Select and evidence the processing mode | Parsed in memory; raw consent, GPP, and US Privacy strings are removed from audits and metrics; scrubbed regulatory signals may reach a contextual bidder | Request memory only |
| Uploaded audience identifiers | Advertiser upload | Advertiser-scoped audience membership check | Raw Redis set `upload:<adv_id>:<marker>`; no listing in responses or metrics; never used outside personalized matching | Default 30 days since the last write, configured by `privacy_audience_ttl_seconds` |
| Pseudonymous cap state | HMAC of accepted identity | Impression/click frequency control | Redis `bothcap:<pseudonym>`; no reversible identity in the key | `cap_state_ttl_seconds`, default 90 idle days |
| Signed tracking and replay state | Served local bid | Impression/click/win/loss measurement and idempotency | Signed URLs and short Redis markers; valid contextual events have no user cap identity | Exact signature deadline, default about 24 hours plus accepted future skew |
| Action lineage, touches, and facts | Served local creative plus advertiser backend action | Analytical conversion/action attribution | HMAC-protected delivery lineage, MySQL touch/action rows, fixed contextual reason, and a distinct HMAC pseudonym of a random token id; no raw identity, consent, reservation token, or billing mutation | `action_retention_hours`, default 2160 hours and required to cover maximum accepted action age; bounded prune plus authorized pseudonym export/deletion |
| Experiment exposure and outcome | Approved application integration | Controlled observational A/B analysis | Per-experiment SHA-256 subject hash, variant, declared metric value, and caller-supplied idempotency digest; no raw account/cookie/email/device/event identifier and no automatic serving mutation | Per-experiment `retention_hours`, 24–9600 hours and default 2160; bounded prune plus exact audited experiment/version subject-hash deletion |
| Traffic-quality evidence and decisions | Trusted bounded aggregate worker plus reviewed rule version | Explainable invalid-traffic review, appeal, and scoped serving/billing recommendation | Raw internal event/partner keys exist in memory only; MySQL stores keyed HMAC-SHA-256 digests, fixed signals/actions, numeric scope, bounded safe summary, immutable decisions/history, and aggregate counters; no IP, cookie, device id, auction id, token, URL, or credential | Rule evidence 1–720 hours and hidden immediately after expiry; aggregate outcome 365–2555 days; bounded dedicated-connection prune leaves decisions, case history, counters, and audit intact |
| Middleman callback context | Selected external bid | Proxy downstream callbacks and reconcile charge/pay facts | Redis token context; contains transaction and price metadata, not raw user identity | `middleman_callback_ttl_seconds`, default 24 hours plus the accepted five-minute future-clock-skew window (86,700 seconds) |
| Request/response/attribute/winloss files | Privacy-scrubbed NATS subjects | Operations, ledger input, aggregate diagnostics | Owner/group-restricted files; request/user identity and precise derived identity are removed before NATS, including the user suffix embedded in local auction bid IDs | `privacy_log_retention_hours`, default 168 hours; pruned at consumer startup and rotation |
| Public account human-verification data | Registration/recovery browser, direct peer, and Turnstile response | Prevent automated account creation and account-email abuse | Cloudflare's widget receives its security telemetry under the operator's Cloudflare contract; W8M sends the one-time token, derived client IP, and fixed action to Siteverify, retains no token, and stores only expiring deployment-keyed HMAC email/IP quota keys in Redis | Token and raw client address are request-memory only in W8M; quota windows are at most 24 hours; Cloudflare retention is an external processor-contract/notice gate before enablement |
| Account email, billing, and settlement records | Authenticated advertiser/publisher/operator workflows | Account service and accounting | MySQL and hosted-provider contracts, outside the bid identity path | Product/account policy and statutory accounting schedule |
| Hosted funding/payout state | S02-authorized A01 workflow and signed Stripe events | Advertiser funding, publisher payout, refund/dispute handling, and reconciliation | Exact USD amounts, account/statement ids, opaque provider object ids, two-letter publisher onboarding country, payload hashes, bounded non-credential states/reasons, and redacted audit; full card/bank data, identity documents, API/webhook secrets, signatures, raw bodies, and hosted URLs are excluded | Provider events default 400 days when unreferenced; immutable operations, object mappings, reconciliation evidence, and audit follow the approved financial/statutory schedule |
| TOTP, recovery, and authenticated session state | Enrolled commercial/operator accounts | Strong authentication, recovery, revocation, and reauthentication | AES-256-GCM TOTP ciphertext plus keyed recovery/session digests in MySQL; plaintext setup/recovery material is shown only to the account and excluded from logs/audits | Pending/enabled account lifecycle; sessions use absolute/idle limits; used/revoked operational state 30 days |
| Permission grants and account-security evidence | Named administrator/account actions | Least privilege, incident review, and non-repudiation | Exact role/resource grants and immutable redacted audit rows; failed login identifiers are keyed digests and secrets/request bodies are excluded | Grants until explicit revocation; audit 365–2555 days, example 400, via bounded retention command |

Unknown JSON fields and every uncontrolled `ext` object are discarded at the
middleman boundary. Each bidder receives a new typed request containing only
the impression ids assigned to that bidder. The only extension data W8M adds
back is controlled `request_domain` and cooperative click notification
metadata. A standard `source.schain` is either generated from operator-approved
seller state or preserved only after strict validation; node extensions and
`source.pchain` are removed. Client-supplied `/pz` seller/source claims never
become server approval. `privacy_contextual_middleman_enabled` is a separate
explicit gate in addition to `middleman_enabled`; it is false by default.

Legacy creative macros that would expose raw IP, user agent, city, carrier,
device version/model/brand, IFA/GAID/device ids, or MAC-derived ids now expand
to an empty string. Coarse contextual macros such as country, region, OS class,
device type, language, app bundle, publisher, slot, and campaign ids remain.
Advertisers must use aggregate reporting rather than rebuilding user profiles
inside tracker URLs.

## Storage, Encryption, And Key Ownership

The repository does not implement disk, database, Redis, NATS, or backup
encryption. Production operation therefore requires all of the following:

- TLS-authenticated private links for MySQL, Redis, NATS, and operator access;
- encrypted service volumes and encrypted, access-controlled backup storage;
- a managed secret store for `tracking_secret`, database credentials, Redis
  credentials, NATS credentials, SMTP credentials, the S02 identity encryption
  key, Stripe API/webhook secrets, and bidder headers;
- least-privilege service identities and recorded access to audience/log data;
- a named key owner, rotation schedule, restore test, and incident revocation
  procedure.

`tracking_secret` is both a URL-signing key and the root for domain-separated
HMAC privacy pseudonyms. Rotating it invalidates outstanding tracking URLs and
changes new cap identities. The current runtime has no dual-key validation
window, so rotation requires a documented maintenance window, acceptance of
temporary cap discontinuity, and monitoring of signature failures. Never put
old keys in Git or logs.

The separate Genelet identity key encrypts TOTP secrets and derives recovery,
session, and failed-login digests. It must be a stable 32-byte key shared by all
HTTP nodes and the restricted identity maintenance host. There is no dual-key
read window. Losing it makes TOTP enrollment data unrecoverable; replacing it
invalidates existing session/recovery digests and requires coordinated TOTP
reset/re-enrollment. See
[identity-access-security.md](identity-access-security.md).

Traffic-quality event and partner digests use another deployment key named by
`traffic_quality.digest_key_env`. The value must decode to at least 32 bytes
and belongs only in the secret environment. It must not reuse the tracking or
identity key, and it never belongs in JSON, metrics, review pages, or command
output. Rotation changes future digests; retain the old key only in approved
offline evidence handling until the corresponding bounded evidence expires.

S06 public account quotas derive email/IP pseudonyms from the existing Summer
deployment secret under distinct HMAC namespaces. Raw values and Turnstile
tokens are not retained. Before enabling the widget, the operator must approve
Cloudflare as the security processor for the intended markets, update the
public account/privacy notice and vendor records as required, restrict widget
hostnames, and document Cloudflare-side retention/deletion and incident terms.
Turning on the code or edge rule is not a substitute for that review.

Backups may outlive online TTLs. The backup policy must either exclude transient
Redis/log data or expire backup generations on an approved schedule. Deletion
requests must be recorded against backup generations, and restored data must be
re-deleted before a restored service is exposed.

## Access, Export, Correction, And Deletion

There is deliberately no public endpoint that searches raw identifiers. A
verified request is handled by a named privacy operator using the account,
advertiser id, marker, source system, and time range supplied by the requester.

1. Verify identity and authority outside application logs; never paste a raw
   identifier into tickets, chat, command history, or metrics.
2. Ask the originating publisher/advertiser to export or correct its source
   record. W8M privacy-safe audits cannot reconstruct raw identity.
3. For an uploaded audience, use `uploaded.DeleteAudienceIdentifier` for one
   authorized `(adv_id, marker, identifier)` tuple, or
   `uploaded.DeleteAudience` to retire that audience source. These operations
   return only whether a member/key was removed and never return set contents.
4. For pseudonymous cap state, derive the current domain-separated pseudonym in
   an approved offline tool and delete only the exact
   `bothcap:<pseudonym>` key. Old-key generations require the corresponding key
   owner; otherwise document expiry as the only available erasure mechanism.
5. Stop or correct the upstream source so the next upload/request cannot
   recreate the data. Record affected backups and verify Redis membership/key
   absence without exporting neighboring records.
6. Preserve only the minimal case/audit record required by the approved policy.

For an authorized R01 action pseudonym, `cmd/action-measurement
-action=export|delete -pseudonym=<value>` provides a scoped workflow. Export
omits token hashes and auction/publisher identity; deletion removes matching
action facts and now-orphaned touches transactionally. The pseudonym is derived
from a random action-token id, not a raw person identifier, and must still be
handled as controlled case data.

Audience set TTL is installed in the same Redis script as `SADD`; a persistent
or shorter-lived set gains the configured TTL, while a longer TTL is not
shortened. Log pruning recognizes only the four generated subject prefixes and
does not delete unrelated files or symlinks.

For an authorized R02 experiment erasure, derive the stored per-experiment
subject hash only on the privacy host, then run `cmd/report-experiment
-action=delete-subject` with the exact experiment id/version and a
non-identifying reason. Outcomes and the one exposure are deleted
transactionally; the immutable `SubjectErased` audit stores actor/reason but not
the hash. Scheduled `-action=prune` independently removes bounded expired
outcomes/exposures without enumerating adjacent subjects.

## Operator Configuration And Evidence

Relevant DSP JSON fields:

```json
{
  "privacy_tcf_vendor_id": 0,
  "privacy_tcf_min_policy_version": 5,
  "privacy_tcf_purpose_ids": [1, 3, 4],
  "privacy_contextual_middleman_enabled": false,
  "privacy_browser_id_ttl_seconds": 2592000,
  "privacy_log_retention_hours": 168,
  "privacy_audience_ttl_seconds": 2592000
}
```

R01 action facts use the separate `action_retention_hours` because their
reviewed lifecycle can exceed short operational log retention. It defaults to
2160 hours and may not be configured shorter than `action_max_age_hours`.

S03 uses rule-specific `evidence_retention_hours` from 1 through 720 hours and
`aggregate_retention_days` from 365 through 2555 days. Run the bounded
`cmd/traffic-quality -action=prune-evidence` only from an S02-controlled
maintenance workflow; its actor id is audit attribution, not authentication.

`/debug/vars` exposes fixed-cardinality evidence only:

- `aofei_privacy_decisions_total`, keyed by mode and a bounded reason;
- `aofei_privacy_invalid_signals_total`;
- `aofei_privacy_middleman_blocked_total`.

Metrics contain no identifiers, consent strings, IP addresses, user agents, or
URLs. Attribute audits add `privacy_mode` and `privacy_reason`; request audits
strip raw identity, precise device/location fields, consent strings, and
uncontrolled extensions. The reason taxonomy is an operational explanation,
not a legal determination.

P02 attribute audits may retain only the normalized public supply taxonomy and
authorized seller metadata loaded from the publisher cache. R02 interval facts
use the same closed categories and report an unauthorized or missing seller as
an empty id plus an explicit unknown category. They never retain request-body
seller/source objects, partner credentials, route-private metadata, or seller
approval history.

Before enabling personalized traffic or contextual middleman fanout:

1. obtain legal/policy approval and record the applicable markets and purposes;
2. publish an approved privacy notice and publisher integration instructions;
3. verify the configured TCF vendor id and purposes against current policy;
4. contractually restrict every external bidder to the disclosed contextual
   fields and approved retention;
5. run the consent matrix, sanitation, bidder-isolation, audit-redaction,
   audience TTL/deletion, log-pruning, race, full test, vet, and static analysis
   gates;
6. canary with privacy counters and no raw-body debug capture;
7. disable both privacy/middleman gates and revoke credentials during an
   incident, then preserve only approved privacy-safe evidence.
