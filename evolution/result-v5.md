# Result V5

Resulting direction after M16:

- Downstream bidder endpoints are owned by existing advertiser accounts rather
  than a separate `dsp` role.
- The Summer `bidder` module stores endpoint metadata in `adv_bidder`.
- `adv_bidder` can reference synthetic campaign, item, and creative rows so
  existing advertiser ledger and reporting joins can be reused.
- Advertiser users may manage owned endpoint metadata; operators own credential
  references, synthetic reporting IDs, activation, route groups, inventory
  assignment, and margin settings.
- Middleman runtime fanout is intentionally deferred until route caching,
  downstream client behavior, callback proxying, and reporting are implemented
  in later milestones.
