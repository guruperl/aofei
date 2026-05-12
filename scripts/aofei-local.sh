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

AOFEI_CONFIG="$ROOT/etc/aofei.local.json"
SUMMER_CONFIG="$ROOT/etc/summer.local.json"
DSN="${DB_USER}:${DB_PASS}@tcp(${HOST}:${PORT})/${DB_NAME}?parseTime=true"
REDIS_ADDR="${REDIS_HOST}:${REDIS_PORT}"
NATS_URL="nats://${NATS_HOST}:${NATS_PORT}"

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
	sed -E \
		-e 's%/\*!50017 DEFINER=`[^`]+`@`[^`]+`\*/ %%g' \
		-e 's%DEFINER=`[^`]+`@`[^`]+` %%g' \
		"$source_sql" | mysql_root --database="$DB_NAME"
}

generate_configs() {
	mkdir -p \
		"$ROOT/etc" \
		"$ROOT/.local/logs/log_request" \
		"$ROOT/.local/logs/log_response" \
		"$ROOT/.local/logs/log_attribute" \
		"$ROOT/.local/logs/log_winloss" \
		"$ROOT/.local/spread" \
		"$ROOT/.local/templates" \
		"$ROOT/.local/uploads" \
		"/tmp/aofei-www"

	cat >"$AOFEI_CONFIG" <<JSON
{
  "document_root": "/tmp/aofei-www",
  "server_url": "http://localhost:8080",
  "server_port": "8080",
  "ips": "$ROOT/etc/maxmind.json",
  "nats_url": "$NATS_URL",
  "redis": {"Addr": "$REDIS_ADDR", "User": "", "Pass": ""},
  "connect_array": ["mysql", "$DSN"],
  "spread": "$ROOT/.local/spread",
  "log_request": "$ROOT/.local/logs/log_request",
  "log_response": "$ROOT/.local/logs/log_response",
  "log_attribute": "$ROOT/.local/logs/log_attribute",
  "log_winloss": "$ROOT/.local/logs/log_winloss"
}
JSON

	cat >"$SUMMER_CONFIG" <<JSON
{
  "ProjectRoot": "$ROOT",
  "Script": "/goto",
  "ServerURL": "http://localhost:8080",
  "ServerPort": "8080",
  "DocumentRoot": "/tmp/aofei-www",
  "UploadURL": "http://localhost:8080/uploads",
  "UploadDir": "$ROOT/.local/uploads",
  "Template": "$ROOT/.local/templates",
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
	up
	local source_sql
	source_sql="$(baseline_sql)"
	if [ ! -f "$source_sql" ]; then
		echo "Baseline SQL not found: $source_sql" >&2
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
  AND a.domain = 'default'
  AND EXISTS (
    SELECT 1 FROM adv_campaign c
    WHERE c.adv_id = a.adv_id AND c.campaign_name = 'camp 001'
  )
  AND EXISTS (
    SELECT 1 FROM adv_item i
    JOIN adv_campaign c ON c.campaign_id = i.campaign_id
    WHERE c.adv_id = a.adv_id AND i.item_name IN ('default', 'defaultItem1')
  )
  AND EXISTS (
    SELECT 1 FROM adv_creative cr
    JOIN adv_item i ON i.item_id = cr.item_id
    JOIN adv_campaign c ON c.campaign_id = i.campaign_id
    WHERE c.adv_id = a.adv_id AND cr.creative_name = 'creative 001'
  );
" 2>/dev/null || printf '0')"
	if [ "$sample_count" = "0" ]; then
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
	redis_up
	docker exec "$REDIS_CONTAINER" redis-cli FLUSHALL
}

install_commands() {
	(
		cd "$ROOT"
		GOWORK=off go install \
			./cmd/ledger \
			./cmd/maxmind \
			./cmd/nats-client \
			./cmd/redis-cache \
			./cmd/unify \
			./cmd/winloss \
			./cmd/spread
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
