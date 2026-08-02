# Prompt V25

Add explainable traffic-quality and anti-fraud controls without making an
opaque model or an infrastructure failure an authority over serving or money.

- define a closed, bounded signal/action taxonomy with immutable rule versions,
  observe/canary rollout, deterministic selection, and false-positive rollback;
- keep raw traffic identity transient, persist only keyed digests and bounded
  summaries, expire evidence independently from aggregate outcomes, and enforce
  exact account/resource access;
- separate human review, appeal, serving enforcement, and billing
  recommendation, with recent MFA and maker/checker approval at every sensitive
  boundary;
- make serving consume a detached immutable snapshot that retains a last-known
  good value only while fresh and otherwise fails open; and
- integrate only with existing S01 privacy, S02 identity, A01 accounting,
  publisher, advertiser, and middleman boundaries while deferring learned fraud
  scoring.
