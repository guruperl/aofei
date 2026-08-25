#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

CONTAINER="${AOFEI_MYSQL_CONTAINER:-aofei-mysql}"
VOLUME="${AOFEI_MYSQL_VOLUME:-aofei-mysql-data}"
IMAGE="${AOFEI_MYSQL_IMAGE:-mysql:8.0.41}"
HOST="${AOFEI_MYSQL_HOST:-127.0.0.1}"
PORT="${AOFEI_MYSQL_PORT:-3307}"
ROOT_PASSWORD="${AOFEI_MYSQL_ROOT_PASSWORD:-root_pass}"
DB_NAME="${AOFEI_MYSQL_DATABASE:-aofei}"
DB_USER="${AOFEI_MYSQL_USER:-aofei}"
DB_PASS="${AOFEI_MYSQL_PASSWORD:-aofei_pass}"
REDIS_CONTAINER="${AOFEI_REDIS_CONTAINER:-aofei-redis}"
REDIS_VOLUME="${AOFEI_REDIS_VOLUME:-aofei-redis-data}"
REDIS_IMAGE="${AOFEI_REDIS_IMAGE:-redis:7-alpine}"
REDIS_HOST="${AOFEI_REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${AOFEI_REDIS_PORT:-6379}"
NATS_CONTAINER="${AOFEI_NATS_CONTAINER:-aofei-nats}"
NATS_IMAGE="${AOFEI_NATS_IMAGE:-nats:2-alpine}"
NATS_HOST="${AOFEI_NATS_HOST:-127.0.0.1}"
NATS_PORT="${AOFEI_NATS_PORT:-4222}"
PZDESIGN_ROOT="$(cd "${AOFEI_PZDESIGN_ROOT:-$ROOT/../pzdesign}" 2>/dev/null && pwd -P || printf '%s' "${AOFEI_PZDESIGN_ROOT:-$ROOT/../pzdesign}")"

LOCAL_STATE_DIR="${AOFEI_LOCAL_STATE_DIR:-$ROOT/.local}"
AOFEI_CONFIG="${AOFEI_CONFIG_PATH:-$ROOT/etc/aofei.local.json}"
SUMMER_CONFIG="${AOFEI_SUMMER_CONFIG_PATH:-$ROOT/etc/summer.local.json}"
SPREAD_DIR="${AOFEI_SPREAD_DIR:-$LOCAL_STATE_DIR/spread}"
LOG_DIR="${AOFEI_LOG_DIR:-$LOCAL_STATE_DIR/logs}"
UPLOAD_DIR="${AOFEI_UPLOAD_DIR:-$LOCAL_STATE_DIR/uploads}"
SCHEMA_DIR="$ROOT/.local/schema"
CURRENT_SCHEMA="$SCHEMA_DIR/aofei.schema.sql"
DSN="${DB_USER}:${DB_PASS}@tcp(${HOST}:${PORT})/${DB_NAME}?parseTime=true"
REDIS_ADDR="${REDIS_HOST}:${REDIS_PORT}"
NATS_URL="nats://${NATS_HOST}:${NATS_PORT}"
DIFF_SCHEMA_DB=""

usage() {
	cat <<'USAGE'
Usage: scripts/aofei-local.sh <command>

Commands:
  up            Start Docker MySQL, Redis, and NATS, then generate local configs.
  reset         Recreate the local database and app user.
  load          Import the baseline SQL into the database.
  sample        Import etc/demand.sql and run the default pub generator.
  reset-sample  Run reset, load, and sample.
  status        Show MySQL/Redis/NATS status and database object counts.
  check-sql     Check etc/step4_init.sql for dump definers and legacy auth.
  dump-schema   Dump normalized current Docker MySQL schema to .local/schema/.
  diff-schema   Diff current Docker MySQL schema against etc/step4_init.sql.
  install       Install local Go command binaries.
  down          Stop MySQL, Redis, and NATS without deleting data.
  mysql-up      Start only the Docker MySQL container.
  mysql-down    Stop only the Docker MySQL container.
  redis-up      Start only the Docker Redis container.
  redis-down    Stop only the Docker Redis container.
  redis-flush   Delete all keys from Docker Redis.
  redis-status  Show Docker Redis status and key count.
  nats-up       Start only the Docker NATS container.
  nats-down     Stop only the Docker NATS container.
  nats-status   Show Docker NATS status.

Environment overrides:
  AOFEI_MYSQL_CONTAINER, AOFEI_MYSQL_VOLUME, AOFEI_MYSQL_IMAGE
  AOFEI_MYSQL_HOST, AOFEI_MYSQL_PORT
  AOFEI_MYSQL_ROOT_PASSWORD, AOFEI_MYSQL_DATABASE
  AOFEI_MYSQL_USER, AOFEI_MYSQL_PASSWORD
  AOFEI_MYSQL_BASELINE_SQL
  AOFEI_REDIS_CONTAINER, AOFEI_REDIS_VOLUME, AOFEI_REDIS_IMAGE
  AOFEI_REDIS_HOST, AOFEI_REDIS_PORT
  AOFEI_NATS_CONTAINER, AOFEI_NATS_IMAGE, AOFEI_NATS_HOST, AOFEI_NATS_PORT
  AOFEI_LOCAL_STATE_DIR, AOFEI_CONFIG_PATH, AOFEI_SUMMER_CONFIG_PATH
  AOFEI_SPREAD_DIR, AOFEI_LOG_DIR, AOFEI_UPLOAD_DIR
USAGE
}

container_exists() {
	docker container inspect "$1" >/dev/null 2>&1
}

container_running() {
	[ "$(docker container inspect -f '{{.State.Running}}' "$1" 2>/dev/null || true)" = "true" ]
}

wait_mysql() {
	for _ in $(seq 1 60); do
		if docker exec -e MYSQL_PWD="$ROOT_PASSWORD" "$CONTAINER" mysql -uroot -e "SELECT 1" >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done

	echo "MySQL did not become ready in time." >&2
	return 1
}

wait_redis() {
	for _ in $(seq 1 30); do
		if docker exec "$REDIS_CONTAINER" redis-cli PING >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done

	echo "Redis did not become ready in time." >&2
	return 1
}

wait_nats() {
	for _ in $(seq 1 30); do
		if docker exec "$NATS_CONTAINER" wget -q -O - "http://127.0.0.1:8222/varz" >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done

	echo "NATS did not become ready in time." >&2
	return 1
}

mysql_root() {
	docker exec -e MYSQL_PWD="$ROOT_PASSWORD" -i "$CONTAINER" mysql -uroot "$@"
}

mysql_dump_schema() {
	local schema_name="$1"
	docker exec -e MYSQL_PWD="$ROOT_PASSWORD" "$CONTAINER" \
		mysqldump -uroot \
		--no-data \
		--routines \
		--triggers \
		--events \
		--single-transaction \
		--skip-comments \
		--set-gtid-purged=OFF \
		--column-statistics=0 \
		"$schema_name"
}

baseline_sql() {
	if [ -n "${AOFEI_MYSQL_BASELINE_SQL:-}" ]; then
		printf '%s\n' "$AOFEI_MYSQL_BASELINE_SQL"
	else
		printf '%s\n' "$ROOT/etc/step4_init.sql"
	fi
}

drop_legacy_users() {
	mysql_root <<SQL
DROP USER IF EXISTS 'eightran_goto'@'%';
DROP USER IF EXISTS 'eightran'@'%';
FLUSH PRIVILEGES;
SQL
}

import_sql_without_definers() {
	local source_sql="$1"
	local target_db="${2:-$DB_NAME}"
	sed -E \
		-e 's%/\*!50017 DEFINER=`[^`]+`@`[^`]+`\*/ %%g' \
		-e 's%DEFINER=`[^`]+`@`[^`]+` %%g' \
		"$source_sql" | mysql_root --database="$target_db"
}

schema_object_count() {
	mysql_root -N -B -e "
SELECT
  (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${DB_NAME}') +
  (SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema = '${DB_NAME}') +
  (SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = '${DB_NAME}') +
  (SELECT COUNT(*) FROM information_schema.events WHERE event_schema = '${DB_NAME}');
"
}

check_sql() {
	local source_sql="$ROOT/etc/step4_init.sql"
	local failed=0

	if [ ! -f "$source_sql" ]; then
		echo "Baseline SQL not found: $source_sql" >&2
		return 1
	fi

	if grep -Eni 'DEFINER[[:space:]]*=' "$source_sql"; then
		echo "Baseline SQL contains explicit dump definers. Remove DEFINER= clauses from $source_sql." >&2
		failed=1
	fi

	if grep -Eni 'eightran' "$source_sql"; then
		echo "Baseline SQL contains legacy eightran auth references. Remove them from $source_sql." >&2
		failed=1
	fi

	if [ "$failed" -ne 0 ]; then
		return 1
	fi

	echo "Baseline SQL guard passed: $source_sql"
}

normalize_schema_dump() {
	local source_dump="$1"
	local target_dump="$2"
	local source_schema="$3"

	sed -E \
		-e '/^--/d' \
		-e '/^\/\*!999999\\- enable the sandbox mode \*\//d' \
		-e 's/ AUTO_INCREMENT=[0-9]+//g' \
		-e 's%/\*![0-9]+ DEFINER=`[^`]+`@`[^`]+` SQL SECURITY DEFINER \*/%/*!50013 SQL SECURITY DEFINER */%g' \
		-e 's%/\*![0-9]+ DEFINER=`[^`]+`@`[^`]+`\*/ %%g' \
		-e 's%DEFINER=`[^`]+`@`[^`]+` %%g' \
		-e "s/\`${source_schema}\`/\`${DB_NAME}\`/g" \
		"$source_dump" >"$target_dump"
}

write_schema_dump() {
	local schema_name="$1"
	local target_dump="$2"
	local raw_dump

	mkdir -p "$SCHEMA_DIR"
	raw_dump="$(mktemp)"
	if mysql_dump_schema "$schema_name" >"$raw_dump"; then
		normalize_schema_dump "$raw_dump" "$target_dump" "$schema_name"
		rm -f "$raw_dump"
	else
		rm -f "$raw_dump"
		return 1
	fi
}

dump_schema() {
	mysql_up
	write_schema_dump "$DB_NAME" "$CURRENT_SCHEMA"
	echo "Wrote normalized schema dump: $CURRENT_SCHEMA"
}

cleanup_diff_schema() {
	if [ -n "$DIFF_SCHEMA_DB" ]; then
		mysql_root -e "DROP DATABASE IF EXISTS \`${DIFF_SCHEMA_DB}\`;" >/dev/null 2>&1 || true
	fi
}

require_local_destructive_target() {
	local command_name="$1"
	case "$CONTAINER:$VOLUME:$REDIS_CONTAINER:$REDIS_VOLUME:$DB_NAME" in
		aofei-mysql:aofei-mysql-data:aofei-redis:aofei-redis-data:aofei) ;;
		*)
			if [ "${AOFEI_ALLOW_CUSTOM_DESTRUCTIVE:-}" != "1" ]; then
				echo "$command_name refuses custom Docker/database targets without AOFEI_ALLOW_CUSTOM_DESTRUCTIVE=1." >&2
				echo "Targets: mysql=$CONTAINER volume=$VOLUME redis=$REDIS_CONTAINER redis_volume=$REDIS_VOLUME db=$DB_NAME" >&2
				return 1
			fi
			;;
	esac
	for value in "$CONTAINER" "$VOLUME" "$REDIS_CONTAINER" "$REDIS_VOLUME" "$DB_NAME"; do
		if [ -z "$value" ] || [ "$value" = "/" ] || [ "$value" = "." ]; then
			echo "$command_name refuses empty or unsafe target value: '$value'." >&2
			return 1
		fi
	done
}

diff_schema() {
	mysql_up
	check_sql

	local source_sql="$ROOT/etc/step4_init.sql"
	local baseline_schema="$SCHEMA_DIR/aofei.baseline.schema.sql"
	local current_schema="$SCHEMA_DIR/aofei.current.schema.sql"

	DIFF_SCHEMA_DB="${DB_NAME}_baseline_check_$$"
	trap cleanup_diff_schema EXIT

	mysql_root <<SQL
DROP DATABASE IF EXISTS \`${DIFF_SCHEMA_DB}\`;
CREATE DATABASE \`${DIFF_SCHEMA_DB}\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
SQL
	import_sql_without_definers "$source_sql" "$DIFF_SCHEMA_DB"

	write_schema_dump "$DIFF_SCHEMA_DB" "$baseline_schema"
	write_schema_dump "$DB_NAME" "$current_schema"

	if diff -u "$baseline_schema" "$current_schema"; then
		echo "Schemas match: $source_sql and Docker database ${DB_NAME}."
	fi
}

generate_configs() {
	# Runtime-owned directories must be born no broader than 0750. A scoped
	# umask affects only newly created components and never chmods an existing
	# operator-owned override.
	(umask 0027; mkdir -p \
		"$ROOT/etc" \
		"$(dirname "$AOFEI_CONFIG")" \
		"$(dirname "$SUMMER_CONFIG")" \
		"$LOG_DIR/log_request" \
		"$LOG_DIR/log_response" \
		"$LOG_DIR/log_attribute" \
		"$LOG_DIR/log_winloss" \
		"$SPREAD_DIR" \
		"$UPLOAD_DIR" \
		"/tmp/aofei-www")

	cat >"$AOFEI_CONFIG" <<JSON
{
  "document_root": "/tmp/aofei-www",
  "server_url": "http://localhost:8080",
  "server_port": "8080",
  "ips": "$ROOT/etc/maxmind.json",
  "nats_url": "$NATS_URL",
  "tracking_secret": "local-dev-tracking-secret",
  "tracking_signature_ttl_seconds": 86400,
  "cap_state_ttl_seconds": 7776000,
  "delivery_cache_max_age_seconds": 900,
  "delivery_reservation_ttl_seconds": 86700,
  "delivery_state_ttl_seconds": 172800,
  "middleman_enabled": false,
  "middleman_always_enabled": false,
  "middleman_timeout_ms": 100,
  "middleman_max_bidders_per_imp": 5,
  "middleman_exchange_domain": "localhost",
  "middleman_callback_ttl_seconds": 86700,
  "middleman_callback_timeout_ms": 1000,
  "middleman_callback_base_url": "http://localhost:8080",
  "openrtb_debug_enabled": false,
  "openrtb_debug_sample_rate": 0,
  "action_token_ttl_seconds": 2592000,
  "action_click_window_hours": 720,
  "action_view_window_hours": 168,
  "action_max_age_hours": 2160,
  "action_request_skew_seconds": 300,
  "action_retention_hours": 2160,
  "privacy_tcf_vendor_id": 0,
  "privacy_tcf_min_policy_version": 5,
  "privacy_tcf_purpose_ids": [1, 3, 4],
  "privacy_contextual_middleman_enabled": false,
  "privacy_browser_id_ttl_seconds": 2592000,
  "privacy_log_retention_hours": 168,
  "privacy_audience_ttl_seconds": 2592000,
  "trusted_proxy_cidrs": [],
  "metrics_allowed_cidrs": ["127.0.0.1/32", "::1/128"],
  "traffic_default": {
    "qps": 2000,
    "burst": 500,
    "max_concurrency": 256,
    "timeout_ms": 1000,
    "max_body_bytes": 1048576,
    "max_decompressed_body_bytes": 1048576
  },
  "traffic_partners": {},
  "redis": {"Addr": "$REDIS_ADDR", "User": "", "Pass": ""},
  "connect_array": ["mysql", "$DSN"],
  "db_max_open_conns": 32,
  "db_max_idle_conns": 8,
  "db_conn_max_lifetime_seconds": 1800,
  "db_conn_max_idle_time_seconds": 300,
  "spread": "$SPREAD_DIR",
  "log_request": "$LOG_DIR/log_request",
  "log_response": "$LOG_DIR/log_response",
  "log_attribute": "$LOG_DIR/log_attribute",
  "log_winloss": "$LOG_DIR/log_winloss"
}
JSON

	cat >"$SUMMER_CONFIG" <<JSON
{
  "ProjectRoot": "$PZDESIGN_ROOT",
  "Script": "/goto",
  "ServerURL": "http://localhost:8080",
  "CORSOrigins": [],
  "ServerPort": "8080",
  "DocumentRoot": "$PZDESIGN_ROOT/www",
  "UploadURL": "http://localhost:8080/uploads",
  "UploadDir": "$UPLOAD_DIR",
  "Template": "$PZDESIGN_ROOT/tmpls",
  "ConnectArray": ["mysql", "$DSN"],
  "Pubrole": "web",
  "Secret": "local-dev-secret"
}
JSON
}

mysql_up() {
	docker volume create "$VOLUME" >/dev/null

	if container_running "$CONTAINER"; then
		wait_mysql
	elif container_exists "$CONTAINER"; then
		docker start "$CONTAINER" >/dev/null
		wait_mysql
	else
		docker run -d \
			--name "$CONTAINER" \
			-e MYSQL_ROOT_PASSWORD="$ROOT_PASSWORD" \
			-p "${HOST}:${PORT}:3306" \
			-v "${VOLUME}:/var/lib/mysql" \
			"$IMAGE" \
			--character-set-server=utf8mb4 \
			--collation-server=utf8mb4_0900_ai_ci >/dev/null
		wait_mysql
	fi
}

redis_up() {
	docker volume create "$REDIS_VOLUME" >/dev/null

	if container_running "$REDIS_CONTAINER"; then
		wait_redis
	elif container_exists "$REDIS_CONTAINER"; then
		docker start "$REDIS_CONTAINER" >/dev/null
		wait_redis
	else
		docker run -d \
			--name "$REDIS_CONTAINER" \
			-p "${REDIS_HOST}:${REDIS_PORT}:6379" \
			-v "${REDIS_VOLUME}:/data" \
			"$REDIS_IMAGE" \
			redis-server --appendonly yes >/dev/null
		wait_redis
	fi
}

nats_up() {
	if container_running "$NATS_CONTAINER"; then
		wait_nats
	elif container_exists "$NATS_CONTAINER"; then
		docker start "$NATS_CONTAINER" >/dev/null
		wait_nats
	else
		docker run -d \
			--name "$NATS_CONTAINER" \
			-p "${NATS_HOST}:${NATS_PORT}:4222" \
			"$NATS_IMAGE" \
			-m 8222 >/dev/null
		wait_nats
	fi
}

up() {
	mysql_up
	redis_up
	nats_up
	generate_configs
}

reset_db() {
	require_local_destructive_target reset
	up
	mysql_root <<SQL
DROP DATABASE IF EXISTS \`${DB_NAME}\`;
CREATE DATABASE \`${DB_NAME}\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASS}';
ALTER USER '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';
FLUSH PRIVILEGES;
SQL
	drop_legacy_users
}

load_baseline() {
	require_local_destructive_target load
	up
	local source_sql
	source_sql="$(baseline_sql)"
	if [ ! -f "$source_sql" ]; then
		echo "Baseline SQL not found: $source_sql" >&2
		return 1
	fi
	local object_count
	object_count="$(schema_object_count)"
	if [ "${object_count:-0}" != "0" ]; then
		echo "Database ${DB_NAME} already has ${object_count} schema objects; run './scripts/aofei-local.sh reset' before load." >&2
		return 1
	fi
	import_sql_without_definers "$source_sql"
	drop_legacy_users
}

load_sample() {
	up
	local sample_count
	sample_count="$(mysql_root --database="$DB_NAME" -N -B -e "
SELECT COUNT(*)
FROM adv a
WHERE a.adv_id = 1
  AND a.email = 'advertiser@example.test'
  AND a.domain = 'advertiser.example.test'
  AND EXISTS (
    SELECT 1 FROM adv_campaign c
    WHERE c.adv_id = a.adv_id AND c.campaign_name = 'Example App Campaign'
  )
  AND EXISTS (
    SELECT 1 FROM adv_item i
    JOIN adv_campaign c ON c.campaign_id = i.campaign_id
    WHERE c.adv_id = a.adv_id AND i.item_name = 'Example App Ad Group'
  )
  AND EXISTS (
    SELECT 1 FROM adv_creative cr
    JOIN adv_item i ON i.item_id = cr.item_id
    JOIN adv_campaign c ON c.campaign_id = i.campaign_id
    WHERE c.adv_id = a.adv_id AND cr.creative_name = 'Example App Creative'
  )
  AND EXISTS (
    SELECT 1 FROM admin
    WHERE admin_id = 1 AND login = 'admin_local'
  )
  AND EXISTS (
    SELECT 1 FROM pub p
    JOIN pub_site s ON s.pub_id = p.pub_id
    JOIN pub_slot t ON t.site_id = s.site_id
    WHERE p.pub_id = 2
      AND p.email = 'publisher@example.test'
      AND p.domain = 'default'
      AND s.foreign_id = 'com.example.publisher'
      AND t.slot_name = 'defaultSlot'
  );
" 2>/dev/null || printf '0')"
	if [ "$sample_count" = "0" ]; then
		local sample_conflicts
		sample_conflicts="$(mysql_root --database="$DB_NAME" -N -B -e "
SELECT
  (SELECT COUNT(*) FROM admin WHERE admin_id = 1 OR login = 'admin_local') +
  (SELECT COUNT(*) FROM adv WHERE adv_id = 1 OR email = 'advertiser@example.test') +
  (SELECT COUNT(*) FROM pub WHERE pub_id = 2 OR email = 'publisher@example.test') +
  (SELECT COUNT(*) FROM adv_campaign WHERE campaign_id IN (1, 2)) +
  (SELECT COUNT(*) FROM adv_item WHERE item_id IN (1, 2)) +
  (SELECT COUNT(*) FROM adv_creative WHERE creative_id IN (1, 2)) +
  (SELECT COUNT(*) FROM pub_site WHERE site_id IN (1, 2)) +
  (SELECT COUNT(*) FROM pub_slot WHERE slot_id IN (1, 2));
" 2>/dev/null || printf '0')"
		if [ "$sample_conflicts" != "0" ]; then
			echo "The synthetic sample fixture is incomplete or conflicts with existing local rows." >&2
			echo "Use './scripts/aofei-local.sh reset-sample' on a disposable local target instead of overwriting data." >&2
			return 1
		fi
		mysql_root --database="$DB_NAME" <"$ROOT/etc/demand.sql"
	else
		echo "Skipping etc/demand.sql because the default sample demand is already present."
	fi
	(cd "$ROOT" && GOWORK=off AOFEI="$AOFEI_CONFIG" go run ./etc pub)
}

status() {
	if container_exists "$CONTAINER"; then
		docker ps -a --filter "name=^/${CONTAINER}$"
	else
		echo "Container $CONTAINER does not exist."
	fi

	if container_running "$CONTAINER"; then
		mysql_root --database="$DB_NAME" <<SQL
SELECT COUNT(*) AS base_tables
FROM information_schema.tables
WHERE table_schema = '${DB_NAME}' AND table_type = 'BASE TABLE';
SELECT COUNT(*) AS views_count
FROM information_schema.views
WHERE table_schema = '${DB_NAME}';
SELECT COUNT(*) AS routines_count
FROM information_schema.routines
WHERE routine_schema = '${DB_NAME}';
SELECT COUNT(*) AS triggers_count
FROM information_schema.triggers
WHERE trigger_schema = '${DB_NAME}';
SELECT COUNT(*) AS events_count
FROM information_schema.events
WHERE event_schema = '${DB_NAME}';
SELECT COUNT(*) AS adv_count FROM adv;
SELECT COUNT(*) AS pub_count FROM pub;
SQL
	fi

	redis_status
	nats_status
}

redis_status() {
	if ! container_exists "$REDIS_CONTAINER"; then
		echo "Container $REDIS_CONTAINER does not exist."
		return 0
	fi

	docker ps -a --filter "name=^/${REDIS_CONTAINER}$"

	if container_running "$REDIS_CONTAINER"; then
		docker exec "$REDIS_CONTAINER" redis-cli INFO keyspace | sed '/^#/d;/^$/d'
		docker exec "$REDIS_CONTAINER" redis-cli DBSIZE
	fi
}

nats_status() {
	if ! container_exists "$NATS_CONTAINER"; then
		echo "Container $NATS_CONTAINER does not exist."
		return 0
	fi

	docker ps -a --filter "name=^/${NATS_CONTAINER}$"

	if container_running "$NATS_CONTAINER"; then
		docker exec "$NATS_CONTAINER" wget -q -O - "http://127.0.0.1:8222/varz" | sed -n '1,20p'
	fi
}

mysql_down() {
	if container_running "$CONTAINER"; then
		docker stop "$CONTAINER" >/dev/null
	fi
}

redis_down() {
	if container_running "$REDIS_CONTAINER"; then
		docker stop "$REDIS_CONTAINER" >/dev/null
	fi
}

nats_down() {
	if container_running "$NATS_CONTAINER"; then
		docker stop "$NATS_CONTAINER" >/dev/null
	fi
}

redis_flush() {
	require_local_destructive_target redis-flush
	redis_up
	docker exec "$REDIS_CONTAINER" redis-cli FLUSHALL
}

install_commands() {
	(
		cd "$ROOT"
		GOWORK=off go install \
			./cmd/accounting \
			./cmd/action-measurement \
			./cmd/config-preflight \
			./cmd/ledger \
			./cmd/maxmind \
			./cmd/mid-callback-retry \
			./cmd/nats-client \
			./cmd/redis-cache \
			./cmd/winloss \
			./cmd/spread
	)
	(
		cd "$PZDESIGN_ROOT"
		GOWORK=off go install ./cmd/unify
	)
}

down() {
	mysql_down
	redis_down
	nats_down
}

case "${1:-}" in
	up)
		up
		;;
	mysql-up)
		mysql_up
		generate_configs
		;;
	mysql-down)
		mysql_down
		;;
	redis-up)
		redis_up
		generate_configs
		;;
	redis-down)
		redis_down
		;;
	redis-flush)
		redis_flush
		;;
	redis-status)
		redis_status
		;;
	nats-up)
		nats_up
		generate_configs
		;;
	nats-down)
		nats_down
		;;
	nats-status)
		nats_status
		;;
	reset)
		reset_db
		;;
	load)
		load_baseline
		;;
	sample)
		load_sample
		;;
	reset-sample)
		reset_db
		load_baseline
		load_sample
		;;
	status)
		status
		;;
	check-sql)
		check_sql
		;;
	dump-schema)
		dump_schema
		;;
	diff-schema)
		diff_schema
		;;
	install)
		install_commands
		;;
	down)
		down
		;;
	-h|--help|help|"")
		usage
		;;
	*)
		echo "Unknown command: $1" >&2
		usage >&2
		exit 2
		;;
esac
