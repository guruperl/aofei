# Status O05 — Generic clean bootstrap

State: `[-]` In progress

## Goal

Make `bootstrap` a target-neutral first activation from an uninstalled systemd
state, without depending on a healthy legacy process or embedding a target-
specific migration path in Aofei.

## Tasks

| Item | State | Notes |
| --- | ---: | --- |
| Clean bootstrap state machine | `[+]` | Absent `current`, installed unit, and manager load state are mandatory; base configs are snapshotted before one immutable release is installed and started, and failure safely restores the uninstalled state. |
| Synthetic verification | `[+]` | Mutation-free state rejection, config projection, start/health/history failure, explicit and unconfirmed stop failure, exact cleanup, and ordinary deploy regression are covered. |
| Public contract and private handoff | `[-]` | Public operator docs and memory/evolution direction are updated. Exact W8M maintenance and bootstrap evidence remain to be recorded privately without moving host policy into Aofei. |

## Acceptance

- Bootstrap has no running-legacy-service, legacy PID, or legacy-health
  dependency and records `none` as its previous selection.
- Bootstrap refuses an existing current selection, installed unit, or loaded
  unit before effects.
- Failure before start restores both base configs, removes the selection and
  installed unit, reloads systemd, and records health as not applicable.
- Failure after start first stops the candidate; inability to stop leaves the
  selected candidate intact and records `rollback_failed` rather than mutating
  files under a possibly running process.
- Ordinary deploy continues to require and recover a verified healthy prior
  release.
- Full Aofei verification and a bounded ten-iteration review-fix gate pass.

## Review-Fix Gate

- Iteration 1: P1 confirmed. Bootstrap cleanup trusted a successful
  `systemctl stop` result without proving the unit was inactive and `MainPID`
  was zero, so a misleading manager response could permit file mutation under
  a still-running candidate. Cleanup now requires an inactive/failed unit and
  zero process identity before changing files; synthetic false-success and
  explicit stop-failure cases preserve the selected candidate and record
  `rollback_failed`.
- Iteration 2: P2 confirmed. The root capability summary still claimed that no
  remediation milestone remained while O05 was in progress, and the serial
  roadmap notation could be read as making O05 depend on unrelated S07. The
  summary now reports O05 directly, and the order makes O04 -> O05 explicit
  while describing S07 as an independent M46 successor.
- Iteration 3: clean for the public O05 implementation. Whole-diff and
  state-machine review found no remaining P1/P2. Full package tests, vet,
  pinned staticcheck, the scoped race suite including `internal/deployment`,
  the documentation guard, and diff hygiene pass. Private W8M realization and
  exact activation evidence remain pending, so O05 is not closed yet.
