# Result V25

S03 establishes W8M's optional explainable traffic-quality boundary.

Implemented direction:

- Eight fixed signals derive from strict bounded aggregate windows. Dependency,
  timeout, malformed partner, quota, credential, and other infrastructure or
  workflow outcomes cannot be supplied as IVT. Missing/partial evidence always
  reduces action and billing to Observe.
- Versioned rules progress through Draft, Observe, Canary, and Active.
  Canary-to-active blocking requires selected complete canary decisions,
  completed human review, and a false-positive rate within the named limit.
  Rules, decisions, case history, and audit evidence remain immutable or
  explicitly versioned.
- Advertisers and publishers see and appeal only their own disclosed cases;
  exact grants bound delegated reads. Recent-MFA administrators own rollout,
  resolution, enforcement, rollback, and separate billing
  recommendation/approval. A quality hold can move only an A01 Draft/Confirmed
  statement to Held and never rewrites measurement or accounting facts.
- Publisher, advertiser, and middleman-partner serving checks use a bounded
  immutable snapshot refreshed independently of requests. Refresh errors retain
  the prior snapshot only until maximum age; expired state fails open.
- Raw event/partner keys are HMAC-SHA-256 digested before storage. Short-lived
  evidence is bounded and pruned through a fresh-context dedicated connection;
  metrics use fixed cardinality and per-rule health remains access-controlled.

Contract consequences:

- The clean schema gains nine `quality_*` tables and ten triggers, reaching 88
  tables, 6 routines, and 43 triggers. Database checks preserve incomplete-
  evidence observe-only behavior and unambiguous rollout state.
- `traffic_quality` is an independent default-off configuration block whose
  digest key is supplied only through a named environment variable. Enabling it
  requires the S03 migration, S02 identity/permissions, observe/canary review,
  and a strict initial snapshot load.
- `cmd/traffic-quality` is the restricted aggregate/health/retention surface;
  Summer supplies scoped review and appeal. Neither surface accepts or exposes
  raw request identity, secrets, or automatic learned decisions.
