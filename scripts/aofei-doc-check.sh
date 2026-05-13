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
	rg -n -- "$pattern" "$@"
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
			docs/*.md \
			memory-bank/product.md \
			memory-bank/architecture.md \
			memory-bank/tech-stack.md \
			memory-bank/milestone.md 2>/dev/null
		find docs -maxdepth 1 -name '*.md' -print
		find evolution -maxdepth 1 -name '*.md' -print
		find memory-bank -maxdepth 1 -name 'status-M*.md' -print
	} | sort -u | grep -v '^docs/legacy-operations\.md$'
)

config_examples=(
	etc/aofei.json
	etc/summer.example.json
	etc/maxmind.json
)

if [ -e memory-bank/status.md ]; then
	fail "memory-bank/status.md must not be recreated; use memory-bank/status-M*.md files."
fi

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
