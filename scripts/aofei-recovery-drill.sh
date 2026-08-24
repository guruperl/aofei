#!/usr/bin/env bash
set -euo pipefail

# O02/O03 clean-room recovery drill. This script creates only uniquely named,
# disposable containers and an owner-only temporary directory. It never reads
# the configured local/production containers and is not a production backup
# implementation; production dumps must be encrypted before leaving MySQL.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
DRILL_ID="${RANDOM}-$$"
SOURCE_CONTAINER="aofei-o02-source-$DRILL_ID"
RESTORE_CONTAINER="aofei-o02-restore-$DRILL_ID"
REDIS_CONTAINER="aofei-o02-redis-$DRILL_ID"
DRILL_DIR=$(mktemp -d /tmp/aofei-o02-recovery.XXXXXX)
MYSQL_PASSWORD=o02_drill_password
STARTED_AT=$(date +%s)

cleanup_recovery_drill() {
	docker stop "$SOURCE_CONTAINER" "$RESTORE_CONTAINER" "$REDIS_CONTAINER" >/dev/null 2>&1 || true
	if [[ "$DRILL_DIR" == /tmp/aofei-o02-recovery.* && -d "$DRILL_DIR" ]]; then
		find "$DRILL_DIR" -depth -mindepth 1 -delete
		rmdir "$DRILL_DIR"
	fi
}
trap cleanup_recovery_drill EXIT
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
	docker run --rm -d --name "$container_name" \
		-p 127.0.0.1::3306 \
		-e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" \
		-e MYSQL_DATABASE=aofei \
		mysql:8.0.41 >/dev/null
	wait_for_mysql "$container_name"
}

mysql_exec() {
	local container_name=$1
	local statement=$2
	docker exec "$container_name" mysql -N -uroot -p"$MYSQL_PASSWORD" aofei -e "$statement"
}

cd "$ROOT"
start_mysql "$SOURCE_CONTAINER"
docker exec -i "$SOURCE_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/step4_init.sql
docker exec -i "$SOURCE_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/demand.sql

mysql_exec "$SOURCE_CONTAINER" "
INSERT INTO acct_statement
  (request_key,party_type,party_id,cadence,period_start,period_end,currency,source_amount,adjustment_amount,total_amount,status,created_by,created_at,updated_at)
VALUES ('o02-drill-statement','advertiser',1,'daily','2026-08-01','2026-08-01','USD',1.000000,0.100000,1.100000,'Draft','o02-drill',UTC_TIMESTAMP(),UTC_TIMESTAMP());
INSERT INTO acct_adjustment (statement_id,amount,reason,created_by,created_at)
VALUES (LAST_INSERT_ID(),0.100000,'O02 recovery drill','o02-drill',UTC_TIMESTAMP());
INSERT INTO acct_audit (statement_id,actor,event,status_to,reason,created_at)
SELECT statement_id,'o02-drill','created','Draft','O02 recovery drill',UTC_TIMESTAMP()
FROM acct_statement WHERE request_key='o02-drill-statement';
INSERT INTO measurement_action
  (adv_id,campaign_id,item_id,creative_id,pub_id,site_id,slot_id,lineage_hash,token_hash,event_fingerprint,event_id,event_type,occurred_at,currency,attribution_type,late,privacy_mode,privacy_reason,action_pseudonym,expires_at)
VALUES (1,1,1,1,2,2,2,UNHEX(REPEAT('11',32)),UNHEX(REPEAT('22',32)),UNHEX(REPEAT('33',32)),'o02-drill-action','conversion',UTC_TIMESTAMP(6),'USD','unattributed',0,'contextual','o02_recovery_drill',REPEAT('a',64),DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 DAY));
INSERT INTO ledger_log (timely,spend,imps,clis,created)
VALUES ('2026-08-01 12:00:00',1.100000,1,1,UTC_TIMESTAMP());
INSERT INTO daily_log (daily,spend,imps,clis,created)
VALUES ('2026-08-01',1.100000,1,1,UTC_TIMESTAMP());
INSERT INTO report_delivery
  (timely,demand_source,adv_id,campaign_id,item_id,creative_id,pub_id,site_id,slot_id,
   country_id,state_id,device_os,device_type,inventory_environment,integration_mode,
   media_intent,placement,render_context,refresh_mode,refresh_seconds,ad_density,traffic_quality,
   source_quality,management_control,seller_type,seller_id,dimension_hash,
   wins,imps,clis,spend_usd,revenue_usd)
VALUES ('2026-08-01 12:00:00','Local',1,1,1,1,2,2,2,1,1,3,2,'Web','BrowserTag',
        'Banner','AboveFold','WebPage','None',0,'Standard','Reviewed','OwnedOperated',
        'Publisher','Publisher','seller-2',UNHEX(SHA2('o02-p02-drill',256)),1,1,1,1.100000,1.100000);
INSERT INTO report_experiment
  (owner_type,experiment_name,experiment_version,status,assignment_salt,primary_metric,
   guardrail_metric,starts_at,created_by_uid,created_at)
VALUES ('Operator','O02 recovery drill',1,'Running','00112233445566778899aabbccddeeff',
        'actions','spend','2026-08-01 00:00:00',1,UTC_TIMESTAMP(6));
SET @report_experiment_id=LAST_INSERT_ID();
INSERT INTO report_experiment_variant
  (experiment_id,experiment_version,variant_key,allocation_basis_points)
VALUES (@report_experiment_id,1,'control',5000),(@report_experiment_id,1,'treatment',5000);
INSERT INTO report_exposure
  (experiment_id,experiment_version,subject_hash,variant_key,exposed_at,expires_at)
VALUES (@report_experiment_id,1,UNHEX(REPEAT('44',32)),'control','2026-08-01 12:00:00.000000',DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 DAY));
SET @report_exposure_id=LAST_INSERT_ID();
INSERT INTO report_experiment_outcome
  (exposure_id,metric_name,metric_value,idempotency_key,occurred_at)
VALUES (@report_exposure_id,'actions',1.000000,UNHEX(REPEAT('55',32)),'2026-08-01 12:01:00.000000');
INSERT INTO report_exposure
  (experiment_id,experiment_version,subject_hash,variant_key,exposed_at,expires_at)
VALUES (@report_experiment_id,1,UNHEX(REPEAT('45',32)),'treatment','2026-08-01 12:00:00.000000',DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND));
INSERT INTO report_experiment_audit
  (experiment_id,experiment_version,actor_uid,event,reason,created_at)
VALUES (@report_experiment_id,1,1,'Created','O02 recovery drill',UTC_TIMESTAMP(6));
INSERT INTO adv_campaign
  (campaign_id,adv_id,campaign_name,foreign_id,access_order,active,created)
VALUES (3,1,'D03 synthetic reporting','d03-drill','Inherit','No',UTC_TIMESTAMP());
INSERT INTO adv_item
  (item_id,campaign_id,item_name,item_click,cost_type,cost,fl_sitetypes,access_order,channel_order,active,created)
VALUES (3,3,'D03 synthetic reporting','about:blank','CPM',0,'App,Web','Inherit','Black','No',UTC_TIMESTAMP());
INSERT INTO adv_creative
  (creative_id,creative_name,item_id,size_id,media_type,weight,active,created)
VALUES (3,'D03 synthetic reporting',3,4194368,'Banner',1,'No',UTC_TIMESTAMP());
INSERT INTO adv_bidder
  (bidder_id,adv_id,synthetic_campaign_id,synthetic_item_id,synthetic_creative_id,bidder_name,endpoint_url,openrtb_version,seat,credential_ref,credential_status,timeout_ms,active,created)
VALUES (1,1,3,3,3,'D03 fixture bidder','https://bidder.example/openrtb','2.5','fixture-seat','D03_DRILL_HEADERS','Active',100,'Yes',UTC_TIMESTAMP());
INSERT INTO mid_route_group
  (group_id,group_name,trigger_mode,total_timeout_ms,margin_pct,min_margin_cpm,active,created)
VALUES (1,'D03 fallback fixture','Fallback',100,0.1000,0.0100,'Yes',UTC_TIMESTAMP());
INSERT INTO mid_route_bidder
  (route_bidder_id,group_id,bidder_id,priority,timeout_ms,active,created)
VALUES (1,1,1,100,100,'Yes',UTC_TIMESTAMP());
INSERT INTO mid_route_target
  (target_id,group_id,priority,active,created)
VALUES (1,1,100,'Yes',UTC_TIMESTAMP());
INSERT INTO mid_callback_retry
  (token,source,callback_url,attempts,next_attempt_at,status,claimed_at,created,updated)
VALUES ('o03-drill-token','win','https://callback.example/o03-recovery',1,
        DATE_SUB(UTC_TIMESTAMP(),INTERVAL 20 MINUTE),'Processing',
        DATE_SUB(UTC_TIMESTAMP(),INTERVAL 20 MINUTE),UTC_TIMESTAMP(),UTC_TIMESTAMP());
" >/dev/null

DUMP_FILE="$DRILL_DIR/aofei.sql"
CHECKSUM_FILE="$DRILL_DIR/aofei.sql.sha256"
docker exec "$SOURCE_CONTAINER" mysqldump -uroot -p"$MYSQL_PASSWORD" \
	--single-transaction --routines --triggers --events --hex-blob --skip-comments aofei > "$DUMP_FILE"
test -s "$DUMP_FILE"
sha256sum "$DUMP_FILE" > "$CHECKSUM_FILE"
(cd "$DRILL_DIR" && sha256sum --check "$(basename "$CHECKSUM_FILE")") >/dev/null
SOURCE_FACTS=$(mysql_exec "$SOURCE_CONTAINER" "
SELECT CONCAT(
 (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='aofei' AND table_type='BASE TABLE'),':',
 (SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema='aofei'),':',
 (SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema='aofei'),':',
 (SELECT COUNT(*) FROM adv),':',(SELECT COUNT(*) FROM pub),':',
 (SELECT COUNT(*) FROM acct_statement),':',(SELECT COUNT(*) FROM acct_adjustment),':',(SELECT COUNT(*) FROM acct_audit),':',
 (SELECT COUNT(*) FROM measurement_action),':',(SELECT COUNT(*) FROM ledger_log),':',(SELECT COUNT(*) FROM daily_log),':',
 (SELECT COUNT(*) FROM report_delivery),':',(SELECT COUNT(*) FROM report_experiment),':',
 (SELECT COUNT(*) FROM report_exposure),':',(SELECT COUNT(*) FROM report_experiment_outcome),':',
 (SELECT COUNT(*) FROM adv_bidder),':',(SELECT COUNT(*) FROM mid_route_group),':',(SELECT COUNT(*) FROM mid_route_bidder),':',(SELECT COUNT(*) FROM mid_route_target),':',
 (SELECT CONCAT(COUNT(*),'-',SUM(status='Processing')) FROM mid_callback_retry),':',
 (SELECT unit_version FROM acct_contract WHERE contract_id=1));")

docker stop "$SOURCE_CONTAINER" >/dev/null
start_mysql "$RESTORE_CONTAINER"
docker exec -i "$RESTORE_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < "$DUMP_FILE"
RESTORED_FACTS=$(mysql_exec "$RESTORE_CONTAINER" "
SELECT CONCAT(
 (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='aofei' AND table_type='BASE TABLE'),':',
 (SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema='aofei'),':',
 (SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema='aofei'),':',
 (SELECT COUNT(*) FROM adv),':',(SELECT COUNT(*) FROM pub),':',
 (SELECT COUNT(*) FROM acct_statement),':',(SELECT COUNT(*) FROM acct_adjustment),':',(SELECT COUNT(*) FROM acct_audit),':',
 (SELECT COUNT(*) FROM measurement_action),':',(SELECT COUNT(*) FROM ledger_log),':',(SELECT COUNT(*) FROM daily_log),':',
 (SELECT COUNT(*) FROM report_delivery),':',(SELECT COUNT(*) FROM report_experiment),':',
 (SELECT COUNT(*) FROM report_exposure),':',(SELECT COUNT(*) FROM report_experiment_outcome),':',
 (SELECT COUNT(*) FROM adv_bidder),':',(SELECT COUNT(*) FROM mid_route_group),':',(SELECT COUNT(*) FROM mid_route_bidder),':',(SELECT COUNT(*) FROM mid_route_target),':',
 (SELECT CONCAT(COUNT(*),'-',SUM(status='Processing')) FROM mid_callback_retry),':',
 (SELECT unit_version FROM acct_contract WHERE contract_id=1));")
if [[ "$RESTORED_FACTS" != "$SOURCE_FACTS" ]]; then
	echo "restored inventory mismatch: source=$SOURCE_FACTS restored=$RESTORED_FACTS" >&2
	exit 1
fi
if mysql_exec "$RESTORE_CONTAINER" "UPDATE acct_audit SET reason='must fail' WHERE actor='o02-drill';" >/dev/null 2>&1; then
	echo "restored accounting immutability trigger did not reject mutation" >&2
	exit 1
fi
if mysql_exec "$RESTORE_CONTAINER" "INSERT INTO ledger_log (timely,created) VALUES ('2026-08-01 12:00:00',UTC_TIMESTAMP());" >/dev/null 2>&1; then
	echo "restored interval-ledger identity did not reject a duplicate" >&2
	exit 1
fi
if mysql_exec "$RESTORE_CONTAINER" "INSERT INTO daily_log (daily,created) VALUES ('2026-08-01',UTC_TIMESTAMP());" >/dev/null 2>&1; then
	echo "restored daily-ledger identity did not reject a duplicate" >&2
	exit 1
fi
if mysql_exec "$RESTORE_CONTAINER" "UPDATE report_exposure SET variant_key='treatment' WHERE exposure_id=1;" >/dev/null 2>&1; then
	echo "restored experiment exposure immutability trigger did not reject mutation" >&2
	exit 1
fi
if mysql_exec "$RESTORE_CONTAINER" "UPDATE report_experiment_outcome SET metric_value=2.000000 WHERE outcome_id=1;" >/dev/null 2>&1; then
	echo "restored experiment outcome immutability trigger did not reject mutation" >&2
	exit 1
fi
REPORT_RESULT=$(mysql_exec "$RESTORE_CONTAINER" "
SELECT CONCAT(
  (SELECT COUNT(*) FROM report_experiment_variant v INNER JOIN report_experiment e USING (experiment_id,experiment_version) WHERE e.experiment_name='O02 recovery drill'),':',
  (SELECT SUM(allocation_basis_points) FROM report_experiment_variant v INNER JOIN report_experiment e USING (experiment_id,experiment_version) WHERE e.experiment_name='O02 recovery drill'),':',
  (SELECT COUNT(*) FROM report_exposure x INNER JOIN report_experiment e USING (experiment_id,experiment_version) WHERE e.experiment_name='O02 recovery drill'),':',
  (SELECT COUNT(*) FROM report_experiment_outcome),':',
  (SELECT CAST(COALESCE(SUM(metric_value),0) AS DECIMAL(20,6)) FROM report_experiment_outcome));")
if [[ "$REPORT_RESULT" != "2:10000:2:1:1.000000" ]]; then
	echo "restored experiment report mismatch: $REPORT_RESULT" >&2
	exit 1
fi

docker run --rm -d --name "$REDIS_CONTAINER" -p 127.0.0.1::6379 redis:7-alpine >/dev/null
for attempt in $(seq 1 30); do
	if docker exec "$REDIS_CONTAINER" redis-cli PING 2>/dev/null | grep -qx PONG; then
		break
	fi
	if [[ "$attempt" -eq 30 ]]; then
		echo "disposable Redis did not become ready" >&2
		exit 1
	fi
	sleep 1
done
MYSQL_PORT=$(docker port "$RESTORE_CONTAINER" 3306/tcp | awk -F: 'NR==1 {print $NF}')
REDIS_PORT=$(docker port "$REDIS_CONTAINER" 6379/tcp | awk -F: 'NR==1 {print $NF}')
DRILL_CONFIG="$DRILL_DIR/aofei.json"
cat > "$DRILL_CONFIG" <<JSON
{
  "document_root": "/tmp/aofei-www",
  "server_url": "http://127.0.0.1:8080",
  "server_port": "8080",
  "tracking_secret": "o02-disposable-drill-secret",
  "tracking_signature_ttl_seconds": 86400,
  "cap_state_ttl_seconds": 7776000,
  "delivery_cache_max_age_seconds": 900,
  "delivery_reservation_ttl_seconds": 86700,
  "delivery_state_ttl_seconds": 172800,
  "middleman_callback_ttl_seconds": 86700,
  "middleman_callback_timeout_ms": 1000,
  "middleman_route_cache_ttl_ms": 5000,
  "middleman_enabled": true,
  "middleman_always_enabled": false,
  "middleman_timeout_ms": 100,
  "middleman_max_bidders_per_imp": 5,
  "middleman_exchange_domain": "exchange.example",
  "middleman_callback_base_url": "https://exchange.example",
  "privacy_contextual_middleman_enabled": true,
  "redis": {"Network":"tcp","Addr":"127.0.0.1:$REDIS_PORT"},
  "connect_array": ["mysql", "root:$MYSQL_PASSWORD@tcp(127.0.0.1:$MYSQL_PORT)/aofei?parseTime=true"]
}
JSON
PRUNE_RESULT=$(GOWORK=off GOTOOLCHAIN=go1.23.5 AOFEI="$DRILL_CONFIG" \
	go run ./cmd/report-experiment -action=prune -limit=100)
if [[ "$PRUNE_RESULT" != "expired_exposures_deleted=1" ]]; then
	echo "restored experiment retention prune returned unexpected output: $PRUNE_RESULT" >&2
	exit 1
fi
if [[ $(mysql_exec "$RESTORE_CONTAINER" "SELECT CONCAT((SELECT COUNT(*) FROM report_exposure),':',(SELECT COUNT(*) FROM report_experiment_outcome));") != "1:1" ]]; then
	echo "restored experiment retention prune removed the wrong facts" >&2
	exit 1
fi
GOWORK=off GOTOOLCHAIN=go1.23.5 AOFEI="$DRILL_CONFIG" go run ./cmd/redis-cache -cache=redis >/dev/null
D03_MANIFEST=$(D03_DRILL_HEADERS='{"X-W8M-Drill":"fixture"}' \
	GOWORK=off GOTOOLCHAIN=go1.23.5 AOFEI="$DRILL_CONFIG" \
	go run ./cmd/redis-cache -validate-middleman -activation-stage=fallback)
if [[ "$D03_MANIFEST" != middleman_activation_ready\ stage=fallback* ]]; then
	echo "restored middleman activation preflight returned unexpected output" >&2
	exit 1
fi
CACHE_KEYS=$(docker exec "$REDIS_CONTAINER" redis-cli --scan | wc -l | tr -d ' ')
if [[ "$CACHE_KEYS" -lt 1 ]]; then
	echo "restored MySQL did not rebuild Redis cache" >&2
	exit 1
fi
CALLBACK_RECOVERY=$(GOWORK=off GOTOOLCHAIN=go1.23.5 AOFEI="$DRILL_CONFIG" \
	go run ./cmd/mid-callback-retry -read -json)
EXPECTED_CALLBACK_RECOVERY='{"due":0,"stale_processing":1,"selected":1,"forwarded":0,"succeeded":0,"retrying":0,"abandoned":0,"state_errors":0}'
if [[ "$CALLBACK_RECOVERY" != "$EXPECTED_CALLBACK_RECOVERY" ]]; then
	echo "restored callback retry evidence returned unexpected summary: $CALLBACK_RECOVERY" >&2
	exit 1
fi

FINISHED_AT=$(date +%s)
ELAPSED=$((FINISHED_AT - STARTED_AT))
echo "recovery_drill=passed elapsed_seconds=$ELAPSED inventory=$RESTORED_FACTS redis_keys=$CACHE_KEYS middleman_preflight=fallback callback_stale_processing=1 dump_sha256=$(cut -d' ' -f1 "$CHECKSUM_FILE")"
