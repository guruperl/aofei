# Status M8 - Full Repository Test Hygiene

Milestone status: `[ ]` Pending

Goal: Move from scoped smoke checks to a clean repository-level verification
target.

## Tasks

- `[ ]` Reproduce current package discovery state.
  - Files: repository root, `backup/`, `go.mod`.
  - Command:
    ```bash
    GOWORK=off go list ./...
    ```
  - Acceptance: discovered packages and any unwanted historical packages are
    recorded.

- `[ ]` Resolve `backup/` Go package discovery.
  - Files: `backup/`, possibly `backup/README.md` or subdirectories under
    `backup/`.
  - Acceptance: historical Go files are preserved but no unwanted `backup`
    package appears in active `go list ./...`.

- `[ ]` Run full package compile tests.
  - Files: all Go packages.
  - Command:
    ```bash
    GOWORK=off go test ./... -run '^$'
    ```
  - Acceptance: all active packages compile or failing packages are listed with
    exact blockers.

- `[ ]` Run full tests where safe.
  - Files: all Go packages.
  - Command:
    ```bash
    GOWORK=off go test ./...
    ```
  - Acceptance: command passes or canonical exclusions/skips are justified and
    documented.

- `[ ]` Decide canonical verification command.
  - Files: `README.md`, `AGENTS.md`, `memory-bank/tech-stack.md`.
  - Acceptance: all three files name the same command or explain why multiple
    stages exist.

- `[ ]` Add a repository verification script if command sequencing remains
  complex.
  - Files: `scripts/aofei-verify.sh`, `README.md`, `AGENTS.md`.
  - Acceptance: one command runs local smoke checks, Go compile checks, and
    whitespace checks.

- `[ ]` Add schema-contract coverage for SQL embedded outside the baseline
  file.
  - Files: `acl/`, `match/`, `summer/`, operational commands.
  - Acceptance: repository verification catches embedded SQL drift against the
    active Docker schema, or clearly documents scoped exclusions.

- `[ ]` Verify parent workspace behavior.
  - Files: `go.mod`, parent `go.work` if intentionally changed.
  - Command:
    ```bash
    go list ./...
    GOWORK=off go list ./...
    ```
  - Acceptance: docs explain when `GOWORK=off` is required and whether parent
    workspace changes are needed.

- `[ ]` Run M8 verification.
  - Command:
    ```bash
    GOWORK=off go list ./...
    GOWORK=off go test ./... -run '^$'
    git diff --check
    ```
  - Acceptance: canonical verification is clean or explicitly narrowed with
    documented reasons.

## Review Findings

- `[ ]` Record current test baseline: `GOWORK=off go test ./... -run '^$'`
  compiles, but full `GOWORK=off go test ./...` fails across multiple packages.

- `[ ]` Remove historical code from active package discovery. `go list ./...`
  still discovers a `backup` package.

- `[ ]` Resolve stale `advice` enum expectations. Tests expect `Unknown`, while
  current stringers return `All` for zero values.

- `[ ]` Fix stale and non-hermetic test fixtures in `dsp`, `match`, `maxmind`,
  `genelet`, and `summer`. Failures include missing sample files, missing
  geodata, legacy DB DSNs, cwd-dependent paths, and unconditional test errors.

- `[ ]` Add defensive DB-test setup. Some Genelet/Summer tests continue after
  failed config or DB setup and then panic on nil or empty data structures.

### Second Review Pass - 2026-05-12

- `[ ]` Add static correctness checks to the hygiene backlog. Current
  `GOWORK=off staticcheck -checks=SA* ./...` reports real issues including an
  invalid JSON tag, unused append result, unchecked `os.Open` before `defer`,
  ineffective assignments, and ignored test errors.

- `[ ]` Keep callback concurrency under test, not only compile/race smoke.
  Race compile-only checks pass for DSP, match, NATS client, and spread
  packages, but they do not exercise NATS callback write/rotation behavior.

- `[ ]` Replace legacy DSN test fixtures with Docker-aware setup or explicit
  skips. Genelet tests still hard-code legacy MySQL DSNs and some continue
  after setup failures.
