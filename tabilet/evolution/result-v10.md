# Result V10

Resulting direction after M23:

- Summer/Genelet has an admin-only `midroute` module for route groups, route
  bidder membership, and traffic targets.
- Route edits are validated and written to the existing `mid_route_*` schema;
  no schema change is required.
- `cmd/unify` does not refresh route cache data after UI edits.
- `cmd/redis-cache -cache=redis|all` remains the singleton publisher of
  Redis-only `middleman:routes`.
- `trigger_mode='Always'`, spread route snapshots, durable callback retries, and
  real settlement execution remain future work. Arbitrary downstream markup
  rewriting remains closed unless a future reporting requirement makes
  cooperative click notify insufficient.
