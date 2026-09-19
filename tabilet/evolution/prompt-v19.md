# Prompt V19

Implement M36 runtime safety and test/observability hardening after the
post-M35 whole-repo review.

Focus on meaningful confirmed risks rather than broad refactoring:

- make `cmd/spread` service shutdown signal-aware and testable;
- harden middleman bidder fanout so nil-client paths cannot use unsafe default
  HTTP behavior;
- add observability for cap contention, audit queue pressure, and middleman
  callback retry backlog/staleness;
- define an explicit verification taxonomy for package, race, staticcheck,
  Docker smoke, admin integration, and schema checks;
- decide the local/spread cache staleness runtime policy;
- triage small adjacent cleanup items without changing schema/cache payload
  shape or `/pz` response contracts.
