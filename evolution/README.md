# Evolution Log

Use this directory for material direction changes, not normal task progress.

Create the next `prompt-vN.md` and `result-vN.md` pair only when a change
materially alters at least one of these:

- Product direction or non-goals.
- Architecture boundaries or ownership.
- Milestone targets or acceptance criteria.
- Public/private contracts, including runtime config, schema, cache, operator,
  or API contracts.

Do not add a new evolution version for routine implementation, documentation
syncs, review-findings cleanup, or status updates that stay within the current
direction. In those cases, update the relevant docs and
`memory-bank/status-M*.md` file instead.
