# Result V29

S05 establishes W8M's runtime trust boundaries across Aofei, Genelet, and
pzdesign.

Implemented direction:

- Configurable bidder and callback requests use one URL and dial policy that
  denies special/future address space, mixed DNS answers, rebinding, proxies,
  custom network dial paths, unsafe TLS, HTTPS downgrade, unsafe redirects,
  ambient cookies, and cross-authority credentials. Injected redirect hooks
  cannot rewrite the immutable history used by mandatory checks.
- The only maintained executable creative sink is pzdesign's opaque-origin
  `ads.js` iframe. Hostile fixtures lock its sandbox, referrer, permissions,
  and host-page isolation; repository guards reject another raw DOM sink or an
  undeclared WebView/native consumer across current and platform source
  languages. Valid advertising scripts and comma-bearing `srcset` URLs remain
  supported under the contained-markup contract.
- Genelet binds authorized role/account, exact component/action/permission/
  resource, MFA state, and reauthentication deadline in a typed request
  capability whose provenance and context keys are private. Summer consumes
  that capability rather than `_g*` strings. Restricted maintenance commands
  derive effective-Unix principals or a reviewed UID-to-admin mapping and
  cannot synthesize wildcard/recent-MFA authority.
- Traffic-quality assessment selects the highest immutable version per stable
  rule key and rollout mode, preserving Active, Canary, and Observe evidence in
  deterministic authority order. Partial/missing evidence still reduces every
  selected mode to Observe.
- Two narrow triggers protect enforcement and quality-billing identity while
  permitting only reviewed rollback and independent billing transitions. The
  clean baseline is 95 tables, 6 routines, and 57 triggers.

Contract consequences:

- Private management URLs remain source-only; any future preview/fetch reopens
  outbound review. A maintained Android/iOS renderer remains conditional I02
  work and will trip the consumer-inventory guard until it has a named owner,
  platform contract, and hostile tests.
- Production secrets, schema migration, deployment, traffic activation, and
  external services were not mutated. Default-off P03, S02, S03, I03, and A02
  activation gates remain independent of repository completion.
