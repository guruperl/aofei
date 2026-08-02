#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"

failed=0

fail() {
	printf 'doc-check: %s\n' "$*" >&2
	failed=1
}

run_rg() {
	local pattern="$1"
	shift

	set +e
	if command -v rg >/dev/null 2>&1; then
		rg -n -- "$pattern" "$@"
	else
		grep -En -- "$pattern" "$@"
	fi
	local status=$?
	set -e

	if [ "$status" -eq 0 ]; then
		return 0
	fi
	if [ "$status" -eq 1 ]; then
		return 1
	fi
	exit "$status"
}

mapfile -t active_docs < <(
	{
		git ls-files \
			README.md \
			AGENTS.md \
			GOAL.md \
			docs/*.md \
			memory-bank/product.md \
			memory-bank/architecture.md \
			memory-bank/tech-stack.md \
			memory-bank/milestone.md 2>/dev/null
		find docs -maxdepth 1 -name '*.md' -print
		find . -maxdepth 1 -name 'GOAL.md' -print | sed 's#^\./##'
		find evolution -maxdepth 1 -name '*.md' -print
		find memory-bank -maxdepth 1 -name 'status-*.md' -print
	} | sort -u | grep -v '^docs/legacy-operations\.md$'
)

config_examples=(
	etc/aofei.json
	etc/summer.example.json
	etc/maxmind.json
)

if [ -e memory-bank/status.md ]; then
	fail "memory-bank/status.md must not be recreated; use lane status files."
fi

for required_o02_path in \
	docs/single-region-availability.md \
	scripts/aofei-recovery-drill.sh
do
	if [ ! -f "$required_o02_path" ]; then
		fail "$required_o02_path is required by the O02 operating contract."
	fi
done
if [ -f scripts/aofei-recovery-drill.sh ] && [ ! -x scripts/aofei-recovery-drill.sh ]; then
	fail "scripts/aofei-recovery-drill.sh must remain executable."
fi

mapfile -t status_files < <(
	find memory-bank -maxdepth 1 -type f -name 'status-*.md' -print | sort -V
)

for status_file in "${status_files[@]}"; do
	status_basename="${status_file##*/}"
	if [[ "$status_basename" =~ ^status-M[0-9]{2,}\.md$ ]]; then
		:
	elif [[ "$status_basename" =~ ^status-[DPRISAO][0-9]{2,}\.md$ ]]; then
		:
	else
		fail "$status_basename must use a zero-padded M/D/P/R/I/S/A/O lane ID."
	fi

	if ! grep -Fq "]($status_basename)" memory-bank/milestone.md; then
		fail "$status_basename must be indexed from memory-bank/milestone.md."
	fi
done

if [ "${#active_docs[@]}" -gt 0 ] &&
	run_rg '\]\([^)]*(memory-bank/)?status\.md\)' "${active_docs[@]}"; then
	fail "active docs must not link to the removed aggregate memory-bank/status.md."
fi

if [ "${#active_docs[@]}" -gt 0 ] &&
	run_rg '(^|[^[:alnum:]_.-])conf/' "${active_docs[@]}"; then
	fail "active docs must not reference retired root config-directory workflows."
fi

if [ "${#active_docs[@]}" -gt 0 ] &&
	run_rg 'ref\.sql|cmd/nats/|cmd/redis/' "${active_docs[@]}"; then
	fail "active docs contain stale baseline or retired command path references."
fi

if run_rg 'eightran|12pass34' "${active_docs[@]}" "${config_examples[@]}"; then
	fail "active docs/config examples contain legacy credential tokens."
fi

if run_rg '(^|/)conf/|cmd/nats/|cmd/redis/' .gitignore; then
	fail ".gitignore contains retired config or command paths."
fi

for ignored_binary in \
	cmd/nats-client/nats-client \
	cmd/redis-cache/redis-cache \
	cmd/spread/spread \
	cmd/maxmind/maxmind
do
	if ! grep -Fxq "$ignored_binary" .gitignore; then
		fail ".gitignore must ignore $ignored_binary."
	fi
done

if ! grep -Fq 'aofei:aofei_pass@tcp(127.0.0.1:3307)/aofei' etc/aofei.json; then
	fail "etc/aofei.json must use the Docker local MySQL example DSN."
fi

if ! grep -Fq 'aofei:aofei_pass@tcp(127.0.0.1:3307)/aofei' etc/summer.example.json; then
	fail "etc/summer.example.json must use the Docker local MySQL example DSN."
fi

if run_rg 'local-dev-secret|smtp_pass|-(local-secret|local-coding)' etc/summer.example.json; then
	fail "etc/summer.example.json must use placeholder secrets, not local runtime secrets."
fi

for config in etc/aofei.json etc/summer.example.json; do
	if ! grep -Fq 'aofei:aofei_pass@tcp(127.0.0.1:3307)/aofei' "$config"; then
		fail "$config must use the Docker local MySQL example DSN."
	fi
done

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "Documentation guard passed."
