# Status M43 - Repository CI Baseline

State: `[+]` Completed

## Tasks

- `[+]` Add Aofei push/pull-request verification workflow.
- `[+]` Add pzdesign push/pull-request verification workflow with sibling
  Aofei checkout.
- `[+]` Pin Go/staticcheck versions, read-only permissions, caches, and
  superseded-run cancellation.
- `[+]` Document CI ownership and local command parity.
- `[+]` Run workflows' commands locally and deep-review the YAML.
- `[+]` Pin pzdesign's sibling Aofei checkout and add its `cmd/unify` race
  gate.
- `[+]` Fetch full primary-repository history and check committed PR/push event
  ranges, including the initial-history empty-tree case.
- `[+]` Run actionlint and committed-whitespace range verification, then close
  the reopened milestone.

## Acceptance

- `[+]` Aofei CI runs tests, vet, staticcheck, scoped race, docs, and diff
  checks.
- `[+]` Pzdesign CI runs tests, vet, template checks, and staticcheck with only
  the documented legacy style exclusions.
- `[+]` Both workflows resolve their modules from clean sibling checkouts.
- `[+]` Diff hygiene fails on committed whitespace errors instead of checking
  only an empty worktree diff.

## Verification

- `[+]` Go 1.23.5 package tests and vet in both repositories.
- `[+]` Staticcheck v0.5.1 with each workflow's exact check set.
- `[+]` Aofei's scoped race command.
- `[+]` Both pzdesign template parsers.
- `[+]` Pzdesign `GOWORK=off go test -race ./cmd/unify`.
- `[+]` Actionlint v1.7.7 on both workflow files.
- `[+]` `./scripts/aofei-doc-check.sh`.
- `[+]` `git diff --check && git -C ../pzdesign diff --check`.
- `[+]` Actionlint v1.7.7 plus temporary-repository committed whitespace
  failures for normal and empty-tree ranges.

## Notes

- Finding: B5.
- Hosted CI intentionally excludes Docker smoke, database-backed admin tests,
  and schema drift; those remain explicit local/operator gates.
- Pzdesign's all-package staticcheck baseline still requires `ST1000`,
  `ST1003`, and `ST1006` exclusions for legacy generated-style code.
- The M39-M44 review's reproducibility finding was resolved by pinning
  pzdesign CI to Aofei revision `3348166a5405ac11417cc69c7cf453d7fa9fba13`;
  dependency adoption now requires an explicit workflow pin update.
- Reopened after the follow-up review found both clean-checkout
  `git diff --check` steps inspected no committed changes.
- Both workflows now fetch full primary history and select committed ranges from
  event SHAs; local uncommitted diff hygiene remains a closeout command.
- No `evolution/` entry was added because CI enforces existing contracts rather
  than changing them.
