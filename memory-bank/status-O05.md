# Status O05 — Generic clean bootstrap

State: `[+]` Completed

## Goal

Make `bootstrap` a target-neutral first activation from an uninstalled systemd
state, without depending on a healthy legacy process or embedding a target-
specific migration path in Aofei.

## Tasks

| Item | State | Notes |
| --- | ---: | --- |
| Clean bootstrap state machine | `[+]` | Absent `current`, installed unit, and manager load state are mandatory; base configs are snapshotted before one immutable release is installed and started, and failure safely restores the uninstalled state. |
| Synthetic verification | `[+]` | Mutation-free state rejection, config projection, start/health/history failure, explicit and unconfirmed stop failure, exact cleanup, and ordinary deploy regression are covered. |
| Public contract and private handoff | `[+]` | Public operator docs and V32 record the generic direction; private W8M D03 owns and has proved the installed-target transition and exact clean bootstrap without moving host policy into Aofei. |

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

## Exact realization evidence

- Private W8M D03 built the corrected release from clean published Aofei
  `8af4a63`, Pzdesign `ff48ff6`, and Genelet `d7ae11c`, and passed the exact
  healthy-current preflight before mutation.
- The target-private bridge moved the stopped zero-PID service through absent
  selection/unit and manager `LoadState=not-found`; generic bootstrap recorded
  `previous_release=none`, `old_pid=0`, installed the immutable release, and
  started PID `571974`.
- Loaded executable, working directory, inline config paths, environment-file
  set, direct health/readiness, and both configured public probes pass. The
  immediately prior release remains immutable and verified; no schema, cache,
  feature, dependency, front/provider, credential, browser, or retention
  operation ran.

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
- Iteration 4: P1 confirmed by the first exact target attempt. Clean bootstrap
  checked only the immediate unit directory; a peer-writable ancestor could
  replace that directory after validation, defeating the claimed no-peer-write
  boundary. Validate the complete absolute directory chain, accepting only the
  operator/root ownership and sticky shared ancestors, before installing the
  unit. The exact attempt itself failed before generic history/activation and
  the private bridge restored the healthy prior release.
- Iteration 5: clean after the directory-chain fix. Whole-milestone review
  found no remaining P1/P2. A synthetic peer-writable ancestor now fails before
  selection, snapshot, history, or service mutation; full package tests, vet,
  pinned staticcheck, the scoped race suite, documentation guard, and diff
  hygiene pass. Corrected exact W8M activation remains pending.
- Iteration 6: clean after the corrected exact activation. Whole-milestone and
  cross-repository review found no remaining P1/P2. The published bundle passed
  exact preflight and clean bootstrap; selected/prior verification, loaded
  manager state, direct/public health, owner-only snapshot/history metadata,
  private conformance/JSON/diff checks, and a complete private worktree
  gitleaks scan pass. The private history record is published and exclusions
  remained untouched.
