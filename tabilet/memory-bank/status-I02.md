# Status I02 - Android And iOS Publisher SDKs

State: `[ ]` Planned

## Goal

Provide maintained native publisher libraries around the stable `/pz` contract
when customer demand justifies their long-term support cost.

## Dependencies

- P01 direct SSP commercial readiness.
- P03 direct SSP request authenticity.
- I01 interoperability and error semantics.
- S01 mobile identity, consent, and disclosure rules.
- S05 native rendering and runtime trust-boundary hardening.
- A03 exact monetary source and compatibility contracts.
- M46 cross-lane correctness and operational-safety remediation.
- A named Android or iOS integration with supported OS/version requirements.

## Tasks

| Item | State | Notes |
|---|---:|---|
| SDK contract | `[ ]` | Freeze supported `/pz` request/response, timeout, retry, lifecycle, impression, click, error, and server-owned monetary-projection behavior for native clients. |
| Android library | `[ ]` | Implement a versioned Android package with tests, sample app, release artifacts, and upgrade guidance. |
| iOS library | `[ ]` | Implement a versioned Swift package with tests, sample app, release artifacts, and upgrade guidance. |
| Privacy integration | `[ ]` | Propagate approved consent/regulatory and opt-out signals under S01. Send IFA/app/device/user fields only for an approved purpose, never use the browser cookie or IP/UA fallback, expect exact coordinates to be removed, and stay contextual for unsupported GPP mappings. |
| Rendering safety | `[ ]` | Define banner, video, and native rendering ownership, web-view policy, click handling, and hostile-markup isolation. |
| Support lifecycle | `[ ]` | Publish compatibility, semantic versioning, deprecation, telemetry, troubleshooting, and vulnerability-response policies. |

## Acceptance Criteria

- SDKs are thin, documented clients of the public `/pz` contract rather than a
  second auction implementation.
- SDKs treat request floors and response prices as protocol projections only;
  they cannot create, relabel, cache, or mutate authoritative accounting facts.
- Sample apps demonstrate fill, no-fill, timeout, impression, click, consent,
  and invalid configuration.
- Release and security ownership is explicit for both ecosystems.

## Verification

- Platform unit/integration tests, sample application builds, contract tests
  against `cmd/unify`, privacy checks, release reproducibility, and Aofei
  compatibility gates.

## Reconciliation From S01

- The named integration must identify its CMP/consent source, jurisdictions,
  vendor/purpose contract, mobile limit-tracking behavior, identifier lifecycle,
  and deletion/support owner before I02 is triggered. Example apps default to
  identifier-free contextual requests and cannot embed fabricated consent.

## Reconciliation From S04

- Rendering safety starts from S04's separation: dashboards and diagnostics
  show creative as escaped source, while SDK ad rendering is a distinct,
  sandboxed delivery surface. Web views must disable unsafe bridges/navigation,
  isolate origins/storage, and test hostile markup, event handlers, redirects,
  and URL schemes.
- SDK samples must not rely on remote executable assets or reinterpret a
  server-delivered string as generally trusted application HTML.

## Reconciliation From P01

- Native libraries must use active `App` inventory and the existing SDK/API
  sample contract; they cannot send browser inventory, use the browser cookie,
  or weaken exact app identity, packed token, unique safe code, single-media,
  dimension, or configured-floor validation. The request may raise but never
  lower the cache-owned USD CPM floor.
- Contract/sample tests must cover fill, ordinary no-fill, invalid inventory,
  tracker ownership, D01 reservation release/finalization, and A01
  `CPM / 1000` reconciliation. SDK rendering remains separately sandboxed and
  D02-validated rather than inheriting browser `srcdoc` trust.

## Reconciliation From D02

- SDK renderers consume only D02-validated Banner URL-container markup, VAST
  Video, or requested-asset Native responses. They must still enforce platform
  navigation/web-view policies and cannot treat a validated bid payload as
  generally trusted application HTML.
- Contract tests include media/size/MIME/HTTPS mismatches, malformed VAST,
  hostile Banner navigation, strict Native asset IDs/types, and response-failure
  reservation release on the named integration's actual renderer.

## Reconciliation From O02

- A named SDK defines bounded timeout/retry behavior against the regional edge;
  no automatic retry may duplicate action or other non-idempotent writes.
  Release tests cover readiness drain, one-node loss, no-fill, and shared
  dependency outcomes while preserving the existing `/pz` contract.
- Mobile availability/support claims require an edge-inclusive O02 measurement
  window and supported OS/network matrix. A local two-node test or origin-only
  benchmark is not SDK availability evidence.

## Reconciliation From D03

- External DSP fallback remains a server-side implementation detail behind the
  stable `/pz` fill/no-fill/error contract. An SDK never calls a downstream
  bidder endpoint, resolves a credential, selects a route, proxies a partner
  callback, or exposes partner metadata to the host application.
- Named-integration tests must show the same rendering and tracker ownership for
  valid local and middleman fills and ordinary no-fill on partner failure. D03
  privacy/activation gates may stop external fanout without requiring an SDK
  release or changing the public response contract.

## Reconciliation From R02

- Marketplace analytics and controlled assignment remain server-side behind
  `/pz`. An SDK neither receives route/commercial report dimensions nor computes
  experiment buckets unless the named integration adds a separately reviewed
  S01 purpose, pseudonym lifecycle, deletion owner, exposure/outcome contract,
  and compatibility policy.
- SDK telemetry may carry only documented idempotent impression/click events
  needed by existing measurement. It cannot invent experiment outcomes, embed
  assignment salts, expose subject hashes, or use R02 results to change floors,
  retries, refresh, or rendering automatically.

## Reconciliation From P02

- A named mobile integration uses operator-approved `App` inventory,
  canonical app identity, `SDK` integration mode, slot media/render/refresh
  categories, and the current packed tokens. The SDK cannot submit trusted
  seller metadata, `source.schain`, quality approval, or management ownership;
  the server derives disclosure from the publisher cache.
- Sample and contract tests cover incompatible Web/BrowserTag metadata,
  unauthorized seller state, approved owned/intermediary chains behind
  middleman fanout, and timed-refresh bounds without exposing seller approval
  controls or report dimensions to the host application.

## Reconciliation From S02

- Native runtime requests never carry Summer role cookies, S02 database
  sessions, TOTP codes, recovery codes, analyst grants, or management API
  credentials. Publisher portal actions that register/rotate an SDK app token
  require the publisher's exact account scope, a named S02 permission, recent
  MFA, and redacted audit; the application receives only the separately scoped
  public runtime identifier/token defined by the named integration.
- SDK configuration and support tools cannot expose another publisher's app,
  slot, consent evidence, quality review, reports, or credential metadata.
  Cross-account denial occurs before token generation, configuration download,
  or telemetry queries, and hostile values stay escaped in every portal and
  generated sample surface.

## Reconciliation From I03

- Mobile ad requests remain thin publisher clients of `/pz`; they never call
  advertiser `/api/v1`, embed an I03 service bearer, reuse its idempotency key,
  poll campaign publication operations, or expose advertiser/report resources
  to the host application. Publisher runtime identity and advertiser management
  identity remain separate contracts.
- If a named mobile customer later needs publisher-side automation, it requires
  a separately reviewed publisher API with publisher-bound scopes, S02
  lifecycle controls, account isolation, idempotency, audit, quotas, and cache
  state. It cannot broaden I03 advertiser scopes or make the SDK a credential
  issuance/secret-storage surface.

## Reconciliation From S03

- A mobile SDK may send only the bounded existing measurement and reviewed
  App/origin fields required by `/pz`; it cannot classify traffic as IVT,
  create rules/cases, choose enforcement, resolve appeals, or hold billing.
  Automation and invalid-origin signals are derived and reviewed server-side
  under a named S03 rule version.
- SDK telemetry and diagnostics never carry quality event/partner digests, raw
  evidence, rule thresholds, cross-account case data, or enforcement ids.
  Missing telemetry and device/network/dependency failures remain incomplete
  evidence or availability outcomes and therefore cannot trigger blocking or
  billing changes.
- A publisher reviews and appeals its own disclosed quality case through the
  S02-authenticated portal, not through an SDK runtime token. Named-integration
  tests must prove that a stale/failed quality snapshot preserves the documented
  fail-open serving behavior and that reviewed publisher enforcement yields the
  ordinary `/pz` no-fill contract without disclosing the reason to the app.

## Reconciliation From P03 And S05

- P03 repository implementation is complete, but both authenticity gates stay
  disabled by default until a named publisher canary is separately authorized.
  S05 repository implementation and review are also complete. Those
  completions satisfy I02's server-auth and renderer-boundary prerequisites,
  but no named Android/iOS integration, OS/version matrix, consent contract, or
  lifecycle owner exists, so I02 is not triggered by the current goal.
- A native SDK must use the P03 publisher/App-scoped request credential,
  exact-decompressed-body signature, freshness proof, one-use replay rules,
  rotation, and revocation contract. A retry signs the exact new request with a
  fresh timestamp and nonce. The private seed belongs in platform-approved
  secure storage, never source, samples, telemetry, logs, or release artifacts.
  Packed or HMAC-protected inventory locators are public identifiers and are
  never used as the application's authentication secret.
- SDK WebViews/renderers must implement S05's tested origin, navigation,
  storage, bridge, redirect, and URL isolation for every Banner, Video, and
  Native consumer. Adding Java/Kotlin/Swift/Objective-C or cross-platform
  WebView code will trip S05's repository inventory guard; the guard may be
  updated only with the named I02 implementation, hostile fixtures, and a
  reviewed consumer boundary. A named integration cannot start until the P03
  legacy compatibility window and S05 rendering requirements are explicit.

## Reconciliation From R03

- Assignment algorithm v1/v2 selection, random salts, salt-bound assignment
  proofs, exposure/outcome validation, and experiment state remain server-only.
  An SDK never computes buckets, receives or persists a salt/proof/subject hash,
  chooses a variant, forges a declared metric, or reinterprets a legacy
  assignment.
- Native telemetry can feed only the separately reviewed idempotent server
  integration. It must preserve the exact experiment/version/owner/window and
  bounded retention contract; repeated-event ratios may exceed one but NaN,
  infinities, negative values outside the registry domain, and raw event or
  identity values are rejected before storage.
- Mobile diagnostics/export surfaces remain aggregate-only and cannot expose
  subject rows, salts, hashes, idempotency keys, stop/audit reasons, or
  cross-account results. Exact erasure and expiry pruning remain authorized
  server operations under the renewable lease, not SDK functions.
- No named Android/iOS integration, platform matrix, consent contract, or
  lifecycle owner has appeared, so R03 completion does not trigger I02.

## Reconciliation From A03

- Native clients never own monetary provenance. A request floor is an
  untrusted auction constraint that may raise but never lower the cache-owned
  configured floor; the server selects the billable exact USD CPM, owns its v3
  marker, reservation, signed tracker, ledger, statement, and provider
  reconciliation, and exposes protocol prices only as server-derived
  projections.
- An SDK cannot create or reserialize publisher/demand caches, claim an
  accounting version, reconstruct exact CPM from a response float, calculate
  middleman charge/pay/margin authority, mutate a budget/reservation, or emit a
  ledger, statement, settlement, or hosted-payment fact. Retries and telemetry
  preserve server-issued signed tracker identity and never feed a rounded
  response price back as authoritative input.
- Named-integration contract tests must prove that float projection does not
  become accounting authority, that malformed/unknown money markers fail
  closed, and that legacy drain compatibility stays a server rollout concern
  rather than an SDK write capability.

## Conditional Trigger Evaluation

- 2026-08-24 after S05 closeout: no named Android or iOS integration supplied
  supported OS/version, consent, renderer, release, or support-lifecycle
  requirements. I02 remains planned and is skipped, not completed or
  cancelled, when this goal reaches its conditional step.
- 2026-08-25 after A03 closeout: no named Android or iOS integration, platform
  matrix, consent contract, renderer owner, release owner, or lifecycle owner
  has appeared. A03 adds the server-owned monetary projection boundary above
  but does not trigger SDK implementation. I02 remains planned and is skipped,
  not completed or cancelled, for this goal.
- 2026-08-25 after M46 closeout: the remaining repository prerequisite is
  complete, including explicit demand-cost authority and fail-closed cache and
  operational boundaries. No named Android or iOS integration or lifecycle
  owner has appeared, so I02 remains planned and is skipped, not completed or
  cancelled.
