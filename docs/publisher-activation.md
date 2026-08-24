# Publisher Activation And Rollback

This runbook is the operator acceptance gate for commercial direct publisher
traffic on `POST /pz`. It covers both `Web` browser tags and `App` SDK/API
integrations. It does not authorize a production launch by itself: D01, S01,
O01, and A01 must also be accepted, and the named publisher must have an
approved privacy and support owner.

P03 repository implementation and review are complete. Its
[direct SSP authenticity contract](direct-ssp-authenticity.md), default-off v2
token reader, SDK/server request-authentication boundary, independent
browser/App enforcement, shared portal/readiness issuer, evidence tooling, and
rollback operations are implemented. No production tag migration or legacy
withdrawal has occurred and checked-in SDK authentication remains off.
Therefore the checks below are the named-publisher activation gate; repository
completion alone does not establish production authenticity. Do not enable
either P03 gate, deny legacy reads, or approve a new App integration as
authenticated production traffic without the canary, support, monitoring, and
rollback evidence required below.

## 1. Inventory readiness

Confirm the MySQL source of truth before publishing any cache generation:

- the publisher is `active='Yes'` and has not exhausted its publisher
  impression limit;
- each approved `pub_site` row is `active='Yes'`, has one nonempty identity,
  and has type `Web` or `App`;
- each site has a controlled inventory environment, canonical identity,
  integration mode, and optional public review URL; `Web`/`BrowserTag` and
  `App`/`SDK` combinations must agree with the legacy site type;
- a Web identity is the exact production hostname expected in `Origin` or
  `Referer`; an App identity is the exact application id, bundle, or domain
  approved for the integration;
- each approved `pub_slot` row is `active='Yes'`, has a unique nonempty name,
  a valid packed width/height, and a finite non-negative `bidfloor` in USD CPM;
- the slot media intent, placement, render context, refresh mode, density,
  traffic/source quality, and management-control categories match the
  integration being accepted; timed refreshes are 15--3,600 seconds;
- proposed seller id, type, advertising-system domain (`ASI`), name, and domain
  are operator-reviewed. Only `seller_authorized='Yes'` can produce an
  outbound supply chain, and any publisher-side metadata change revokes that
  approval until a new operator review.

The publisher workspace exposes the configured floor as “最低竞价（USD CPM）”.
The request may raise this floor, but the serving path always uses the greater
of the configured and request values. A client can never lower it.

Run the read-only inventory gate from the cache-maintenance node:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/redis-cache -validate-publishers
```

The command reads MySQL only. It does not connect to Redis, take the cache
mutation lock, or change inventory. It fails if active commercial metadata is
incomplete and otherwise prints a deterministic manifest containing publisher,
site, slot, type, dimensions, floor, normalized supply categories, seller
approval state, the exact `site_token` and `slot_token`, their version, and
display-safe token/authentication rotation metadata. In an enabled v2
configuration the command loads the deployment token key solely to mint the
same current-epoch public locators as `unify`; it fails closed if that key is
unavailable. It never prints the key environment name/value, a publisher
credential id, or a private signing value. Store the output as ordinary
deployment evidence, not as a credential or consent record.

## 2. Cache publication order

P01 adds site type and configured floor to the existing `pubmap` payload and
the derived `pubmap:by-id` record. The Gob change is additive: older readers can
decode the new fields. New P01 HTTP workers deliberately reject pre-P01 entries
that have no type/floor metadata.

P02 adds seller, site, and slot supply metadata to those same Gob records.
Older readers ignore these additive fields. New readers normalize absent P02
fields from an older compatible generation to explicit `Unknown` categories;
they never infer seller authorization. The stricter P01 type/floor readiness
rule remains unchanged.

Use this rolling order:

1. Install the P01 `redis-cache` binary on the singleton cache-maintenance node.
2. Run `-validate-publishers` and resolve every failure.
3. Publish one complete shadow generation and atomically swap it live:

   ```bash
   GOWORK=off AOFEI=/etc/aofei/aofei.json \
     /opt/aofei/bin/redis-cache -cache=redis
   ```

   Use `-cache=all` instead when HTTP nodes serve local/spread snapshots.
4. Inspect both publisher views and the other static families:

   ```bash
   GOWORK=off AOFEI=/etc/aofei/aofei.json \
     /opt/aofei/bin/redis-cache -cache=redis -read
   ```

   Verify `pubmap`, `pubmap:by-id`, `slot:*`, `audience`, `creative`, and
   RAdv v2 delivery policy. In spread mode, verify the complete publisher
   snapshot and local reload metrics as well.
5. Only then roll P01 `unify` binaries through the HTTP canary and remaining
   nodes.

Failed generation builds leave the prior live Redis generation unchanged.
Local/spread workers reload complete immutable snapshots on their configured
interval; the standard D01 bound is five minutes. A stale RAdv delivery policy
still fails closed under `delivery_cache_max_age_seconds`.

To disable inventory, set the publisher, site, or slot inactive in the
control plane and publish another complete generation. Do not expect a MySQL
edit to affect an already loaded HTTP worker immediately.

For rollback, keep the P01 cache generation while rolling HTTP workers back,
or finish rolling all P01 HTTP workers back before restoring a pre-P01 cache.
Never run a P01 HTTP worker against deliberately restored pre-P01 publisher
payloads; direct SSP will correctly fail closed.

### P03 cache-first authenticity migration

P03 activation is a cache-first, reversible migration; it is not implied by a
binary deployment. Before changing either default-off gate:

1. Name the publisher, site/App, slot, publisher contact, operator, privacy
   approval, support window, and rollback owner. Confirm the O01 `ssp` QPS,
   burst, concurrency, timeout, compressed-body, and decompressed-body profile
   from a measured capacity baseline. Cloudflare's S06 UI-only rule must remain
   exact-path scoped and must not include `/pz`.
2. Deploy the dual-reader binary and one complete publisher cache generation to
   every accepting node while v2 remains disabled. Prove the manifest, cache
   timestamp/age, and current v1 request path before provisioning a key.
3. Provision a new owner-only 32-byte key on every accepting node. Set
   `direct_ssp_tokens.enabled:true`, keep `legacy_read_mode:"allow"`, and use
   one current key id/epoch everywhere. Restart/read back every node before
   distributing any v2 tag. The manifest must now say `token_version=v2` and
   its emitted tuple must succeed on every canary node; tampered, mixed-version,
   and inactive-cache tuples must fail before side effects.
4. Regenerate only the named publisher's Web tag and observe a bounded support
   window. `aofei_ssp_inventory_token_outcomes_total` must show expected v2
   acceptance without rising v2/mixed/unknown rejection; legacy acceptance
   remains the rollback path. Do not deny legacy reads until all named tags are
   replaced and legacy traffic is zero for the approved compatibility window.
5. For an App canary, enable `direct_ssp_auth` consistently on every accepting
   node before issuing its credential. Complete the section 4 matrix and watch
   `aofei_ssp_publisher_auth_outcomes_total`, snapshot age/errors, O01
   admission, Redis, cache freshness, and audit backlog. A compatibility count
   after the gate is expected to be zero.

For a browser-locator rotation, deploy the old key as `previous` and the new
key/epoch as `current` on all readers before emitting the new value. Remove the
old selector only after replacement and the approved overlap. To roll back,
stop tag distribution and restore the last known-good current/previous issuer
configuration while legacy remains allowed; do not delete cache data. For an
SDK-auth incident, do not make an authenticated App silently credentialless:
first deactivate that App inventory, publish and verify a complete cache
generation, then roll back the auth configuration/binary if necessary. Redis
replay keys expire naturally within the bounded request-proof window and must
not be scan-deleted.

## 3. Web browser acceptance

Use only a `Web` site. Its slot page produces a browser tag and does not produce
an App API sample.

1. Load the generated tag from a page on the exact configured hostname.
2. Verify the request uses `platform:"browser"`, the current packed tokens,
   exactly one supported media type, and dimensions matching the slot token.
3. Prove a filled request, an ordinary no-fill, malformed/invalid token
   rejection, and an exact-host `Origin`/`Referer` rejection.
4. Verify `ads.js` reports deterministic container states (`filled`,
   `no-fill`, or `error`). Filled markup is placed in an opaque-origin
   `srcdoc` iframe without `allow-same-origin`; no-fill clears the container.
5. Trigger one signed impression and click and reconcile the publisher/site/
   slot/size identities through audit, ledger, and A01 statement evidence.
6. For approved seller metadata, inspect the external bidder request and prove
   that `source.schain` matches the server-owned cache rather than any browser
   claim. Repeat once without approval and prove that no chain is emitted.

The iframe is a delivery boundary, not a substitute for D02 creative and URL
validation. Management/review pages continue to display stored creative as
escaped source only.

Apply the S01 privacy matrix. Browser cookies are allowed only for an approved
personalization grant; contextual, denied, opt-out, COPPA, invalid, and unmapped
signals neither read nor set `aofei_pz_uid`, and IP/User-Agent are never joined
as a fallback identity.

## 4. App SDK/API acceptance

Use only an `App` site. Its slot page produces an SDK/API request and does not
produce a browser tag.

1. After P03 authorizes the gate, enable `direct_ssp_auth` consistently on all
   accepting nodes. Through the S02-protected **Publisher request credentials**
   page, use the exact publisher scope and approved App site to issue a
   credential after recent MFA. Copy its private value once into the
   integration's approved secret manager; it cannot be recovered later and
   must never be embedded in a public sample or mobile source tree.
2. Send `platform:"sdk"` with `responseFormat:"json"` and repeat with
   `responseFormat:"openrtb"`.
3. Sign the exact decompressed JSON bytes and canonical `POST /pz` context with
   a fresh timestamp and one-use nonce. Supply all four `X-W8M-PZ-*` headers.
   Prove missing, stale, replayed, body-mismatched, revoked, and cross-App
   requests fail before auction or audit side effects, including a concurrent
   replay with one winner.
4. Use the current packed tokens and the exact approved App identity. Supplied
   `app.id`, `app.bundle`, or `app.domain` must match; a mismatch returns `400`
   before auction side effects.
5. Verify fill, no-fill, timeout/error handling, impression, click, and tracker
   ownership. Do not retry ordinary no-fill as a network error.
6. Confirm the SDK request never reads or sets the browser cookie and never
   relies on IP/User-Agent fallback identity.
7. Prove the approved consent/regulatory matrix and identifier-free contextual
   default. Exact coordinates are removed; no sample or fixture value is
   consent evidence.
8. Exercise immediate rotation, an explicitly bounded overlap when required,
   and revocation across every HTTP node. A stale verifier snapshot or
   unavailable Redis replay dependency must return generic `503`, not admit the
   request. Retain only redacted audit/metric evidence.

Native platform libraries remain demand-gated under I02. Until a named mobile
integration exists, this is an HTTP contract acceptance test, not a promise of
maintained Android or iOS packages.

## 5. Seller transparency and supply-chain acceptance

Seller transparency describes who sells the approved inventory; it is not a
payment instruction. The existing `pub` account remains the publisher revenue
and A01 settlement owner. Seller metadata cannot redirect pay, create a second
settlement party, or authorize a revenue-share formula.

- A publisher may propose seller id, `Publisher` or `Intermediary` type, ASI,
  public name, and public domain. The operator reviews and authorizes the exact
  stored values. A later publisher edit automatically revokes authorization.
- An authorized `Publisher` produces an OpenRTB supply chain with
  `complete=1`. An authorized `Intermediary` produces `complete=0` because W8M
  does not invent an unrecorded upstream owner. An unauthorized or incomplete
  seller produces no chain.
- Browser and SDK request bodies cannot set or replace seller identity or
  `source.schain`; `/pz` derives both only from the approved cache.
- A middleman request may retain an already approved standard chain, but only
  after bounded validation. Invalid chains reject that partner request;
  uncontrolled node extensions and `source.pchain` are removed.

The activation evidence must cover an owned-and-operated publisher, an
intermediary/resale case when commercially supported, an unauthorized seller,
and a hostile client-supplied source claim. Confirm the received external
request contains only the expected public seller identifiers.

## 6. Canary, monitoring, and commercial reconciliation

Start with an explicitly named publisher/site/slot and the O01 `ssp` admission
profile. Record the capacity-baseline scenario, concurrency, payload shape,
fill/no-fill mix, and dependency profile. Watch fixed-cardinality request,
rejection, latency, Redis, cache freshness, NATS audit queue, and middleman
metrics. Also alert on D01 reservation acquisition, rejection, release,
finalization, expiry, and reconciliation anomalies.

Before paid launch, reconcile one accepted impression end to end:

- OpenRTB and `/pz` prices remain USD CPM;
- the reservation and ledger fact use one-impression USD (`CPM / 1000`) under
  `usd-cpm-impression-v2`;
- `daily_pub` agrees with the A01 publisher statement at six-decimal source
  precision;
- correction and settlement evidence is immutable and contains no full payment
  card or bank credentials.

Rollback when invalid traffic reaches auction side effects, old cache metadata
is observed, tracker or reservation ownership diverges, the publisher statement
does not reconcile, or O01 canary thresholds are breached. Disable the affected
inventory, publish a complete generation, verify bounded propagation, and then
roll back the binary/cache pair in the safe order above.

## 7. Verification evidence

Attach the following to the activation record:

- read-only publisher manifest and cache-generation timestamp;
- seller approval plus owned/intermediary/unauthorized supply-chain evidence;
- browser and/or SDK contract cases, including negative cases;
- fill/no-fill, impression, click, audit, reservation, and statement evidence;
- fixed-label O01 capacity/latency snapshot and alert state;
- named operator, publisher contact, privacy approval, rollback owner, and
  support window.

Never attach raw consent strings, full request bodies, credentials, or full
payment details.

The repository-only abuse proof is repeatable without production services:

```bash
./scripts/aofei-p03-proof.sh
```

It covers legacy/v2/tamper/mixed outcomes, fixed authentication outcomes,
32-way concurrent nonce replay with exactly one winner, body/freshness/scope
failures, bounded credential rotation/revocation, and the O01 `ssp` rate and
concurrency gates. It does not replace the named live canary above.
