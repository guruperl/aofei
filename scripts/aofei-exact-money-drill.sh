#!/usr/bin/env bash
set -euo pipefail

# A03 frozen-backup migration and rollback rehearsal. It creates only uniquely
# named disposable MySQL containers and an owner-only temporary directory. It
# never reads configured or production databases, and its synthetic backup is
# destroyed on exit.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
DRILL_ID="${RANDOM}-$$"
LEGACY_CONTAINER="aofei-a03-legacy-$DRILL_ID"
ROLLBACK_CONTAINER="aofei-a03-rollback-$DRILL_ID"
MIGRATED_CONTAINER="aofei-a03-migrated-$DRILL_ID"
DRILL_DIR=$(mktemp -d /var/tmp/aofei-a03-money.XXXXXX)
MYSQL_PASSWORD=a03_drill_password

cleanup_exact_money_drill() {
	docker rm -f "$LEGACY_CONTAINER" "$ROLLBACK_CONTAINER" "$MIGRATED_CONTAINER" >/dev/null 2>&1 || true
	if [[ "$DRILL_DIR" == /var/tmp/aofei-a03-money.* && -d "$DRILL_DIR" ]]; then
		# MySQL changes restored data ownership to its container uid. Remove only
		# this validated disposable mount as container root, then remove the now
		# empty owner directory on the host.
		docker run --rm --entrypoint find -v "$DRILL_DIR:/cleanup" mysql:8.0.41 \
			/cleanup -depth -mindepth 1 -delete >/dev/null 2>&1 || true
		rmdir "$DRILL_DIR"
	fi
}
trap cleanup_exact_money_drill EXIT
umask 077

for command_name in docker sha256sum go; do
	command -v "$command_name" >/dev/null 2>&1 || {
		echo "missing required command: $command_name" >&2
		exit 1
	}
done

wait_for_mysql() {
	local container_name=$1
	for attempt in $(seq 1 60); do
		if docker exec "$container_name" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e 'SELECT 1' >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done
	docker logs "$container_name" | tail -50 >&2
	return 1
}

start_mysql() {
	local container_name=$1
	local data_dir=${2:-}
	if [[ -n "$data_dir" ]]; then
		docker run -d --name "$container_name" \
			-v "$data_dir:/var/lib/mysql" \
			-e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" \
			mysql:8.0.41 >/dev/null
	else
		docker run -d --name "$container_name" \
			-e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" \
			-e MYSQL_DATABASE=aofei \
			mysql:8.0.41 >/dev/null
	fi
	wait_for_mysql "$container_name"
}

mysql_exec() {
	local container_name=$1
	local statement=$2
	docker exec "$container_name" mysql -N -B -uroot -p"$MYSQL_PASSWORD" aofei -e "$statement"
}

comparison_digest() {
	local container_name=$1
	docker exec -i "$container_name" mysql -N -B -uroot -p"$MYSQL_PASSWORD" aofei <<'SQL'
SET SESSION group_concat_max_len=1048576;
SELECT SHA2(GROUP_CONCAT(CONCAT(source_table,'/',source_pk,'/',source_column,'=',source_value)
  ORDER BY source_table,source_pk,source_column SEPARATOR '|'),256)
FROM (
  SELECT 'adv_item' source_table,CAST(item_id AS CHAR) source_pk,'cost' source_column,
         COALESCE(CAST(CAST(cost AS DECIMAL(12,6)) AS CHAR),'<NULL>') source_value FROM adv_item
  UNION ALL SELECT 'pub_slot',CAST(slot_id AS CHAR),'bidfloor',COALESCE(CAST(CAST(bidfloor AS DECIMAL(12,6)) AS CHAR),'<NULL>') FROM pub_slot
  UNION ALL SELECT 'adv_balance',CAST(balance_id AS CHAR),'limit_spend',COALESCE(CAST(CAST(limit_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM adv_balance
  UNION ALL SELECT 'adv_balance',CAST(balance_id AS CHAR),'current_spend',COALESCE(CAST(CAST(current_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM adv_balance
  UNION ALL SELECT 'his_balance',CAST(his_balance_id AS CHAR),'budget_old',CAST(CAST(budget_old AS DECIMAL(20,9)) AS CHAR) FROM his_balance
  UNION ALL SELECT 'his_balance',CAST(his_balance_id AS CHAR),'budget_add',CAST(CAST(budget_add AS DECIMAL(20,9)) AS CHAR) FROM his_balance
  UNION ALL SELECT 'his_balance',CAST(his_balance_id AS CHAR),'budget_new',CAST(CAST(budget_new AS DECIMAL(20,9)) AS CHAR) FROM his_balance
  UNION ALL SELECT 'ledger_log',CAST(log_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_log
  UNION ALL SELECT 'ledger_adv',CAST(la_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_adv
  UNION ALL SELECT 'ledger_pub',CAST(lp_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_pub
  UNION ALL SELECT 'ledger_pub_adv',CAST(lpa_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_pub_adv
  UNION ALL SELECT 'ledger_mid',CAST(lm_id AS CHAR),'charge_spend',COALESCE(CAST(CAST(charge_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_mid
  UNION ALL SELECT 'ledger_mid',CAST(lm_id AS CHAR),'pay_spend',COALESCE(CAST(CAST(pay_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_mid
  UNION ALL SELECT 'ledger_mid',CAST(lm_id AS CHAR),'margin_spend',COALESCE(CAST(CAST(margin_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM ledger_mid
  UNION ALL SELECT 'daily_log',CAST(log_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_log
  UNION ALL SELECT 'daily_adv',CAST(la_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_adv
  UNION ALL SELECT 'daily_pub',CAST(lp_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_pub
  UNION ALL SELECT 'daily_pub_adv',CAST(lpa_id AS CHAR),'spend',COALESCE(CAST(CAST(spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_pub_adv
  UNION ALL SELECT 'daily_mid',CAST(lm_id AS CHAR),'charge_spend',COALESCE(CAST(CAST(charge_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_mid
  UNION ALL SELECT 'daily_mid',CAST(lm_id AS CHAR),'pay_spend',COALESCE(CAST(CAST(pay_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_mid
  UNION ALL SELECT 'daily_mid',CAST(lm_id AS CHAR),'margin_spend',COALESCE(CAST(CAST(margin_spend AS DECIMAL(20,9)) AS CHAR),'<NULL>') FROM daily_mid
  UNION ALL SELECT 'mid_route_group',CAST(group_id AS CHAR),'min_margin_cpm',CAST(CAST(min_margin_cpm AS DECIMAL(12,6)) AS CHAR) FROM mid_route_group
  UNION ALL SELECT 'mid_route_bidder',CAST(route_bidder_id AS CHAR),'min_margin_cpm',COALESCE(CAST(CAST(min_margin_cpm AS DECIMAL(12,6)) AS CHAR),'<NULL>') FROM mid_route_bidder
) monetary_sources;
SQL
}

cd "$ROOT"
start_mysql "$LEGACY_CONTAINER"
docker exec -i "$LEGACY_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/step4_init.sql
docker exec -i "$LEGACY_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/demand.sql

# Recreate a populated pre-A03 source from the current clean baseline. This is
# a deterministic compatibility fixture, not an inverse production migration.
docker exec -i "$LEGACY_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei <<'SQL'
DROP TRIGGER IF EXISTS money_migration_evidence_immutable_update;
DROP TRIGGER IF EXISTS money_migration_evidence_immutable_delete;
DROP TABLE money_migration_evidence;
UPDATE acct_contract SET unit_version='usd-cpm-impression-v2',effective_at='2026-08-01 00:00:00',
 notes='A03 synthetic legacy source' WHERE contract_id=1;
ALTER TABLE adv_item MODIFY cost DOUBLE NULL;
ALTER TABLE pub_slot MODIFY bidfloor FLOAT NULL DEFAULT 0;
ALTER TABLE adv_balance MODIFY limit_spend FLOAT NULL,MODIFY current_spend FLOAT NULL DEFAULT 0;
ALTER TABLE his_balance MODIFY budget_old FLOAT NOT NULL,MODIFY budget_add FLOAT NOT NULL,MODIFY budget_new FLOAT NOT NULL;
ALTER TABLE ledger_log MODIFY spend FLOAT NULL;
ALTER TABLE ledger_adv MODIFY spend FLOAT NULL;
ALTER TABLE ledger_pub MODIFY spend FLOAT NULL;
ALTER TABLE ledger_pub_adv MODIFY spend FLOAT NULL;
ALTER TABLE ledger_mid MODIFY charge_spend FLOAT NULL DEFAULT 0,MODIFY pay_spend FLOAT NULL DEFAULT 0,MODIFY margin_spend FLOAT NULL DEFAULT 0;
ALTER TABLE daily_log MODIFY spend FLOAT NULL;
ALTER TABLE daily_adv MODIFY spend FLOAT NULL;
ALTER TABLE daily_pub MODIFY spend FLOAT NULL;
ALTER TABLE daily_pub_adv MODIFY spend FLOAT NULL;
ALTER TABLE daily_mid MODIFY charge_spend FLOAT NULL DEFAULT 0,MODIFY pay_spend FLOAT NULL DEFAULT 0,MODIFY margin_spend FLOAT NULL DEFAULT 0;
ALTER TABLE mid_route_group MODIFY min_margin_cpm DECIMAL(10,4) NOT NULL DEFAULT 0.0000;
ALTER TABLE mid_route_bidder MODIFY min_margin_cpm DECIMAL(10,4) NULL;
ALTER TABLE report_delivery ALTER accounting_version SET DEFAULT 'usd-cpm-impression-v2';

UPDATE adv_item SET cost=1.234567 WHERE item_id=1;
UPDATE pub_slot SET bidfloor=0.123456 WHERE slot_id=1;
INSERT INTO adv_balance (limit_spend,current_spend,current_day,created)
VALUES (123.456789123,23.456789123,'2026-08-24',UTC_TIMESTAMP());
SET @balance_id=LAST_INSERT_ID();
INSERT INTO his_balance (balance_id,budget_old,budget_add,budget_new,created)
VALUES (@balance_id,100.000000001,23.456789122,123.456789123,UTC_TIMESTAMP());
INSERT INTO adv_bidder (adv_id,bidder_name,endpoint_url,credential_status,active,created)
VALUES (1,'A03 comparison bidder','https://bidder.example.test/openrtb','Active','No',UTC_TIMESTAMP());
SET @bidder_id=LAST_INSERT_ID();
INSERT INTO mid_route_group (group_name,trigger_mode,total_timeout_ms,margin_pct,min_margin_cpm,active,created)
VALUES ('A03 comparison route','Fallback',100,0.1000,0.0123,'No',UTC_TIMESTAMP());
SET @group_id=LAST_INSERT_ID();
INSERT INTO mid_route_bidder (group_id,bidder_id,priority,min_margin_cpm,active,created)
VALUES (@group_id,@bidder_id,100,0.0068,'No',UTC_TIMESTAMP());
SET @route_bidder_id=LAST_INSERT_ID();

INSERT INTO ledger_log (timely,spend,imps,clis,created)
VALUES ('2026-08-24 12:00:00',0.003703701,3,1,UTC_TIMESTAMP());
SET @ledger_log_id=LAST_INSERT_ID();
INSERT INTO ledger_adv (log_id,creative_id,item_id,campaign_id,adv_id,spend,imps,clis)
VALUES (@ledger_log_id,1,1,1,1,0.003703701,3,1);
SET @ledger_adv_id=LAST_INSERT_ID();
INSERT INTO ledger_pub (log_id,slot_id,site_id,pub_id,spend,imps,clis)
VALUES (@ledger_log_id,2,2,2,0.003703701,3,1);
SET @ledger_pub_id=LAST_INSERT_ID();
INSERT INTO ledger_pub_adv (lp_id,la_id,spend,imps,clis)
VALUES (@ledger_pub_id,@ledger_adv_id,0.003703701,3,1);
INSERT INTO ledger_mid
 (log_id,bidder_id,group_id,route_bidder_id,target_id,adv_id,campaign_id,item_id,creative_id,
  pub_id,site_id,slot_id,wins,imps,charge_spend,pay_spend,margin_spend)
VALUES (@ledger_log_id,@bidder_id,@group_id,@route_bidder_id,0,1,1,1,1,2,2,2,1,1,
        0.001234567,0.001000001,0.000234566);

INSERT INTO daily_log (daily,spend,imps,clis,created)
VALUES ('2026-08-24',0.003703701,3,1,UTC_TIMESTAMP());
SET @daily_log_id=LAST_INSERT_ID();
INSERT INTO daily_adv (log_id,creative_id,item_id,campaign_id,adv_id,spend,imps,clis)
VALUES (@daily_log_id,1,1,1,1,0.003703701,3,1);
SET @daily_adv_id=LAST_INSERT_ID();
INSERT INTO daily_pub (log_id,slot_id,site_id,pub_id,spend,imps,clis)
VALUES (@daily_log_id,2,2,2,0.003703701,3,1);
SET @daily_pub_id=LAST_INSERT_ID();
INSERT INTO daily_pub_adv (lp_id,la_id,spend,imps,clis)
VALUES (@daily_pub_id,@daily_adv_id,0.003703701,3,1);
INSERT INTO daily_mid
 (log_id,bidder_id,group_id,route_bidder_id,target_id,adv_id,campaign_id,item_id,creative_id,
  pub_id,site_id,slot_id,wins,imps,charge_spend,pay_spend,margin_spend)
VALUES (@daily_log_id,@bidder_id,@group_id,@route_bidder_id,0,1,1,1,1,2,2,2,1,1,
        0.001234567,0.001000001,0.000234566);
INSERT INTO acct_statement
 (request_key,party_type,party_id,cadence,period_start,period_end,currency,source_amount,
  adjustment_amount,total_amount,status,created_by,created_at,updated_at)
VALUES ('a03-double-proof','advertiser',1,'daily','2026-08-24','2026-08-24','USD',
        0.003704,0.000000,0.003704,'Draft','a03-drill',UTC_TIMESTAMP(),UTC_TIMESTAMP());
SQL

LEGACY_DIGEST=$(comparison_digest "$LEGACY_CONTAINER")
LEGACY_DEBUG=$(mysql_exec "$LEGACY_CONTAINER" "SELECT CONCAT('item=',CAST(cost AS DECIMAL(12,6)),' floor=',CAST(bidfloor AS DECIMAL(12,6))) FROM adv_item CROSS JOIN pub_slot WHERE item_id=1 AND slot_id=1; SELECT CONCAT('balance=',CAST(limit_spend AS DECIMAL(20,9)),'/',CAST(current_spend AS DECIMAL(20,9))) FROM adv_balance ORDER BY balance_id DESC LIMIT 1; SELECT CONCAT('history=',CAST(budget_old AS DECIMAL(20,9)),'/',CAST(budget_add AS DECIMAL(20,9)),'/',CAST(budget_new AS DECIMAL(20,9))) FROM his_balance ORDER BY his_balance_id DESC LIMIT 1; SELECT CONCAT('ledger=',CAST(spend AS DECIMAL(20,9))) FROM ledger_log ORDER BY log_id DESC LIMIT 1; SELECT CONCAT('daily=',CAST(spend AS DECIMAL(20,9))) FROM daily_log ORDER BY log_id DESC LIMIT 1;")
DUMP_FILE="$DRILL_DIR/aofei-v2-physical.tar"
CHECKSUM_FILE="$DRILL_DIR/aofei-v2-physical.tar.sha256"
SOURCE_DATA="$DRILL_DIR/source-data"
ROLLBACK_DATA="$DRILL_DIR/rollback-data"
MIGRATED_DATA="$DRILL_DIR/migrated-data"
docker stop "$LEGACY_CONTAINER" >/dev/null
mkdir -p "$SOURCE_DATA" "$ROLLBACK_DATA" "$MIGRATED_DATA"
docker cp "$LEGACY_CONTAINER:/var/lib/mysql/." "$SOURCE_DATA"
tar -C "$SOURCE_DATA" -cf "$DUMP_FILE" .
test -s "$DUMP_FILE"
sha256sum "$DUMP_FILE" > "$CHECKSUM_FILE"
(cd "$DRILL_DIR" && sha256sum --check "$(basename "$CHECKSUM_FILE")") >/dev/null

tar -C "$ROLLBACK_DATA" -xf "$DUMP_FILE"
tar -C "$MIGRATED_DATA" -xf "$DUMP_FILE"
start_mysql "$ROLLBACK_CONTAINER" "$ROLLBACK_DATA"
if [[ $(comparison_digest "$ROLLBACK_CONTAINER") != "$LEGACY_DIGEST" ]]; then
	echo "frozen-backup rollback changed legacy monetary sources: source=$LEGACY_DIGEST restore=$(comparison_digest "$ROLLBACK_CONTAINER")" >&2
	echo "legacy_money_debug source=$LEGACY_DEBUG" >&2
	echo "legacy_money_debug restore=$(mysql_exec "$ROLLBACK_CONTAINER" "SELECT CONCAT('item=',CAST(cost AS DECIMAL(12,6)),' floor=',CAST(bidfloor AS DECIMAL(12,6))) FROM adv_item CROSS JOIN pub_slot WHERE item_id=1 AND slot_id=1; SELECT CONCAT('balance=',CAST(limit_spend AS DECIMAL(20,9)),'/',CAST(current_spend AS DECIMAL(20,9))) FROM adv_balance ORDER BY balance_id DESC LIMIT 1; SELECT CONCAT('history=',CAST(budget_old AS DECIMAL(20,9)),'/',CAST(budget_add AS DECIMAL(20,9)),'/',CAST(budget_new AS DECIMAL(20,9))) FROM his_balance ORDER BY his_balance_id DESC LIMIT 1; SELECT CONCAT('ledger=',CAST(spend AS DECIMAL(20,9))) FROM ledger_log ORDER BY log_id DESC LIMIT 1; SELECT CONCAT('daily=',CAST(spend AS DECIMAL(20,9))) FROM daily_log ORDER BY log_id DESC LIMIT 1;")" >&2
	exit 1
fi
if [[ $(mysql_exec "$ROLLBACK_CONTAINER" "SELECT unit_version FROM acct_contract WHERE contract_id=1") != "usd-cpm-impression-v2" ]]; then
	echo "rollback restore did not preserve the legacy contract" >&2
	exit 1
fi

start_mysql "$MIGRATED_CONTAINER" "$MIGRATED_DATA"
docker exec -i "$MIGRATED_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/a03_exact_money_migration.sql
MIGRATED_DIGEST=$(comparison_digest "$MIGRATED_CONTAINER")
if [[ "$MIGRATED_DIGEST" != "$LEGACY_DIGEST" ]]; then
	echo "v3 target-scale values differ from the frozen legacy comparison" >&2
	exit 1
fi

SCHEMA_RESULT=$(mysql_exec "$MIGRATED_CONTAINER" "
SELECT CONCAT(
 (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='aofei' AND data_type='decimal' AND
   (table_name,column_name) IN (('adv_item','cost'),('pub_slot','bidfloor'),
   ('adv_balance','limit_spend'),('adv_balance','current_spend'),
   ('his_balance','budget_old'),('his_balance','budget_add'),('his_balance','budget_new'),
   ('ledger_log','spend'),('ledger_adv','spend'),('ledger_pub','spend'),('ledger_pub_adv','spend'),
   ('ledger_mid','charge_spend'),('ledger_mid','pay_spend'),('ledger_mid','margin_spend'),
   ('daily_log','spend'),('daily_adv','spend'),('daily_pub','spend'),('daily_pub_adv','spend'),
   ('daily_mid','charge_spend'),('daily_mid','pay_spend'),('daily_mid','margin_spend'),
   ('mid_route_group','min_margin_cpm'),('mid_route_bidder','min_margin_cpm'))),':',
 (SELECT unit_version FROM acct_contract WHERE contract_id=1),':',
 (SELECT column_default FROM information_schema.columns WHERE table_schema='aofei' AND table_name='report_delivery' AND column_name='accounting_version'));" )
if [[ "$SCHEMA_RESULT" != "23:usd-cpm-impression-v3:usd-cpm-impression-v3" ]]; then
	echo "migrated exact schema/contract mismatch: $SCHEMA_RESULT" >&2
	exit 1
fi

EVIDENCE_RESULT=$(mysql_exec "$MIGRATED_CONTAINER" "
SELECT CONCAT(COUNT(*),':',SUM(conversion_rule='Quarantined'),':',
 COALESCE(MAX(CASE WHEN conversion_rule='LegacyRenderedHalfAway' AND
   ((source_table='adv_item' AND source_column='cost') OR
    (source_table='pub_slot' AND source_column='bidfloor'))
   THEN ABS(discrepancy) ELSE 0 END),0),':',
 COALESCE(MAX(CASE WHEN conversion_rule='LegacyRenderedHalfAway' AND NOT
   ((source_table='adv_item' AND source_column='cost') OR
    (source_table='pub_slot' AND source_column='bidfloor'))
   THEN ABS(discrepancy) ELSE 0 END),0),':',
 SUM(conversion_rule<>'LegacyRenderedHalfAway' AND COALESCE(discrepancy,0)<>0),':',
 SUM(source_table IN ('mid_route_group','mid_route_bidder') AND source_column='min_margin_cpm' AND conversion_rule='AlreadyExact'))
FROM money_migration_evidence;" )
IFS=: read -r evidence_count quarantined max_legacy_cpm_discrepancy max_legacy_amount_discrepancy nonlegacy_discrepancy exact_route_evidence <<<"$EVIDENCE_RESULT"
if [[ "$evidence_count" != 26 || "$quarantined" != 0 || "$nonlegacy_discrepancy" != 0 || "$exact_route_evidence" != 2 ]]; then
	echo "migration evidence mismatch: $EVIDENCE_RESULT" >&2
	exit 1
fi
if [[ $(mysql_exec "$MIGRATED_CONTAINER" "SELECT $max_legacy_cpm_discrepancy <= 0.000000500") != 1 ]]; then
	echo "legacy CPM discrepancy exceeded half a micro-USD CPM: $max_legacy_cpm_discrepancy" >&2
	exit 1
fi
if [[ $(mysql_exec "$MIGRATED_CONTAINER" "SELECT $max_legacy_amount_discrepancy <= 0.000000001") != 1 ]]; then
	echo "legacy amount discrepancy exceeded one nano-USD target unit: $max_legacy_amount_discrepancy" >&2
	exit 1
fi
if docker exec -i "$MIGRATED_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/a03_exact_money_migration.sql >/dev/null 2>&1; then
	echo "v3 database accepted a second migration run" >&2
	exit 1
fi
if [[ $(comparison_digest "$MIGRATED_CONTAINER") != "$MIGRATED_DIGEST" ]]; then
	echo "failed migration preflight changed exact monetary sources" >&2
	exit 1
fi
if mysql_exec "$MIGRATED_CONTAINER" "INSERT INTO ledger_log (timely,created) VALUES ('2026-08-24 12:00:00',UTC_TIMESTAMP())" >/dev/null 2>&1; then
	echo "interval ledger duplicate was accepted" >&2
	exit 1
fi
if mysql_exec "$MIGRATED_CONTAINER" "INSERT INTO daily_log (daily,created) VALUES ('2026-08-24',UTC_TIMESTAMP())" >/dev/null 2>&1; then
	echo "daily ledger duplicate was accepted" >&2
	exit 1
fi
if mysql_exec "$MIGRATED_CONTAINER" "INSERT INTO acct_statement (request_key,party_type,party_id,cadence,period_start,period_end,currency,source_amount,adjustment_amount,total_amount,status,created_by,created_at,updated_at) VALUES ('a03-double-proof','advertiser',1,'daily','2026-08-24','2026-08-24','USD',0,0,0,'Draft','a03-drill',UTC_TIMESTAMP(),UTC_TIMESTAMP())" >/dev/null 2>&1; then
	echo "statement idempotency key accepted a duplicate" >&2
	exit 1
fi

GOWORK=off go test ./dsp -run 'TestDeliveryReservationBoundsConcurrentSpendAndImpressions|TestDeliveryFinalizationKeepsSpendReservedAndClickIsIdempotent|TestTrackingPublishRetryDoesNotDoubleReserveDelivery'
GOWORK=off go test ./internal/jobs/ledger -run 'TestStatisticsAggregatesMinimumCPMWithoutFloatRounding|TestAdvertiserBalanceReconciliation'
GOWORK=off go test ./accounting -run 'TestCreateStatementIdempotentRetryDoesNotDuplicateAudit|TestNanoAggregateOverflowAndStatementRounding'
GOWORK=off go test ./hostedpayment -run 'TestDuplicateWebhookDoesNotRepeatSideEffects|TestProviderRetriesOnlySanitizedRetryableErrorsWithSameKey'

echo "exact_money_drill=passed source_digest=$LEGACY_DIGEST evidence_rows=$evidence_count quarantined=$quarantined max_legacy_cpm_discrepancy=$max_legacy_cpm_discrepancy max_legacy_amount_discrepancy=$max_legacy_amount_discrepancy contract=usd-cpm-impression-v3 backup_sha256=$(cut -d' ' -f1 "$CHECKSUM_FILE")"
