#!/usr/bin/env bash
set -euo pipefail

# Rehearse the M46 populated-schema margin migration in one uniquely named
# disposable MySQL container. Configured/local databases are never read.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
DRILL_ID="${RANDOM}-$$"
CONTAINER="aofei-m46-margin-$DRILL_ID"
MYSQL_PASSWORD=m46_margin_drill_password

cleanup_margin_drill() {
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup_margin_drill EXIT

for command_name in docker; do
	command -v "$command_name" >/dev/null 2>&1 || {
		echo "missing required command: $command_name" >&2
		exit 1
	}
done

docker run -d --name "$CONTAINER" \
	-e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" \
	-e MYSQL_DATABASE=aofei \
	mysql:8.0.41 >/dev/null
for attempt in $(seq 1 60); do
	if docker exec "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e 'SELECT 1' >/dev/null 2>&1; then
		break
	fi
	if [[ $attempt == 60 ]]; then
		docker logs "$CONTAINER" | tail -50 >&2
		exit 1
	fi
	sleep 1
done

cd "$ROOT"
docker exec -i "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/step4_init.sql
docker exec -i "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei <<'SQL'
ALTER TABLE mid_route_bidder DROP CHECK mid_route_bidder_margin_fraction_chk;
ALTER TABLE mid_route_group DROP CHECK mid_route_group_margin_fraction_chk;
INSERT INTO mid_route_group
  (group_name,trigger_mode,total_timeout_ms,margin_pct,min_margin_cpm,active,created)
VALUES ('invalid percent fixture','Fallback',100,15.0000,0.000000,'No',UTC_TIMESTAMP());
SQL

if docker exec -i "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/m46_middleman_margin_migration.sql >/dev/null 2>&1; then
	echo "M46 margin migration accepted invalid source data" >&2
	exit 1
fi
constraints=$(docker exec "$CONTAINER" mysql -N -B -uroot -p"$MYSQL_PASSWORD" aofei -e \
	"SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_schema='aofei' AND constraint_name IN ('mid_route_group_margin_fraction_chk','mid_route_bidder_margin_fraction_chk')")
if [[ $constraints != 0 ]]; then
	echo "failed M46 preflight partially installed $constraints constraints" >&2
	exit 1
fi

docker exec "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e \
	"DELETE FROM mid_route_group WHERE group_name='invalid percent fixture'"
docker exec -i "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/m46_middleman_margin_migration.sql

constraints=$(docker exec "$CONTAINER" mysql -N -B -uroot -p"$MYSQL_PASSWORD" aofei -e \
	"SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_schema='aofei' AND constraint_name IN ('mid_route_group_margin_fraction_chk','mid_route_bidder_margin_fraction_chk')")
if [[ $constraints != 2 ]]; then
	echo "M46 margin migration installed $constraints constraints, want 2" >&2
	exit 1
fi
if docker exec "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e \
	"INSERT INTO mid_route_group (group_name,margin_pct) VALUES ('rejected group',1.0001)" >/dev/null 2>&1; then
	echo "group margin constraint accepted 1.0001" >&2
	exit 1
fi
if docker exec "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e \
	"SET FOREIGN_KEY_CHECKS=0; INSERT INTO mid_route_bidder (group_id,bidder_id,margin_pct) VALUES (1,1,-0.0001)" >/dev/null 2>&1; then
	echo "route margin constraint accepted -0.0001" >&2
	exit 1
fi
if docker exec -i "$CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/m46_middleman_margin_migration.sql >/dev/null 2>&1; then
	echo "M46 margin migration accepted an already-migrated source" >&2
	exit 1
fi

echo "M46 middleman margin migration drill passed."
