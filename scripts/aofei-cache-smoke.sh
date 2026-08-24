#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
LOCAL_STATE_DIR="${AOFEI_LOCAL_STATE_DIR:-$ROOT/.local}"
AOFEI_CONFIG="${AOFEI_CONFIG_PATH:-$ROOT/etc/aofei.local.json}"
SPREAD_DIR="${AOFEI_SPREAD_DIR:-$LOCAL_STATE_DIR/spread}"
REDIS_CONTAINER="${AOFEI_REDIS_CONTAINER:-aofei-redis}"
SPREAD_PID=""
CACHE_READ_OUTPUT=""
SPREAD_LOG="$LOCAL_STATE_DIR/spread-smoke.log"

cleanup() {
	if [ -n "$SPREAD_PID" ] && kill -0 "$SPREAD_PID" >/dev/null 2>&1; then
		kill "$SPREAD_PID" >/dev/null 2>&1 || true
		wait "$SPREAD_PID" >/dev/null 2>&1 || true
	fi
	if [ -n "$CACHE_READ_OUTPUT" ]; then
		rm -f "$CACHE_READ_OUTPUT"
	fi
	rm -f "$SPREAD_LOG"
}
trap cleanup EXIT

run_cache() {
	(cd "$ROOT" && GOWORK=off AOFEI="$AOFEI_CONFIG" go run ./cmd/redis-cache "$@")
}

require_redis_key() {
	local pattern="$1"
	local count
	count="$(docker exec "$REDIS_CONTAINER" redis-cli --scan --pattern "$pattern" | wc -l | tr -d ' ')"
	if [ "${count:-0}" -eq 0 ]; then
		echo "Missing Redis key pattern: $pattern" >&2
		return 1
	fi
}

require_spread_dir() {
	local dir="$1"
	local root="$SPREAD_DIR"
	local sequence
	if [ -f "$SPREAD_DIR/.aofei-current" ]; then
		IFS= read -r sequence <"$SPREAD_DIR/.aofei-current"
		case "$sequence" in
			""|*[!0-9]*)
				echo "Invalid spread generation pointer: $sequence" >&2
				return 1
				;;
		esac
		root="$SPREAD_DIR/.aofei-generations/$sequence"
	fi
	if [ ! -d "$root/$dir" ]; then
		echo "Missing spread directory: $root/$dir" >&2
		return 1
	fi
}

echo "Resetting sample runtime..."
"$ROOT/scripts/aofei-local.sh" reset-sample
"$ROOT/scripts/aofei-local.sh" redis-flush
case "$SPREAD_DIR" in
	""|/|.|"$ROOT"|"$(dirname "$ROOT")")
		echo "Refusing unsafe spread smoke directory: $SPREAD_DIR" >&2
		exit 1
		;;
esac
rm -rf "$SPREAD_DIR"
mkdir -p "$SPREAD_DIR"

echo "Populating Redis cache..."
run_cache -cache=redis
CACHE_READ_OUTPUT="$(mktemp)"
run_cache -cache=redis -read >"$CACHE_READ_OUTPUT"

require_redis_key "pubmap"
require_redis_key "pubmap:by-id"
require_redis_key "audience"
require_redis_key "creative"
require_redis_key "slot:*"

grep -q "pubmap" "$CACHE_READ_OUTPUT"
grep -q "Audiences" "$CACHE_READ_OUTPUT"
grep -q "Creatives" "$CACHE_READ_OUTPUT"

echo "Starting spread receiver..."
(cd "$ROOT" && GOWORK=off AOFEI="$AOFEI_CONFIG" go run ./cmd/spread >"$SPREAD_LOG" 2>&1) &
SPREAD_PID="$!"
for _ in $(seq 1 20); do
	if grep -q "Listening on" "$SPREAD_LOG" 2>/dev/null; then
		break
	fi
	if ! kill -0 "$SPREAD_PID" >/dev/null 2>&1; then
		echo "Spread receiver exited early:" >&2
		cat "$SPREAD_LOG" >&2
		exit 1
	fi
	sleep 1
done
if ! grep -q "Listening on" "$SPREAD_LOG" 2>/dev/null; then
	echo "Spread receiver did not become ready:" >&2
	cat "$SPREAD_LOG" >&2
	exit 1
fi

echo "Populating spread cache..."
run_cache -cache=spread
sleep 2
require_spread_dir "pubmap"
require_spread_dir "audience"
require_spread_dir "creative"
require_spread_dir "slot"

echo "Populating combined cache mode..."
run_cache -cache=all
sleep 2
require_spread_dir "pubmap"
require_spread_dir "audience"
require_spread_dir "creative"
require_spread_dir "slot"
require_redis_key "pubmap"
require_redis_key "pubmap:by-id"
require_redis_key "audience"
require_redis_key "creative"
require_redis_key "slot:*"

echo "Cache smoke passed."
