#!/usr/bin/env bash
set -euo pipefail

# R02 clean-room MySQL reporting benchmark. It uses one uniquely named,
# disposable container and synthetic interval facts; it never reads or changes
# the configured local or production database.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
BENCHMARK_ID="${RANDOM}-$$"
MYSQL_CONTAINER="aofei-r02-report-$BENCHMARK_ID"
MYSQL_PASSWORD=r02_benchmark_password
MYSQL_IMAGE=${MYSQL_IMAGE:-mysql:8.0.41}
ROWS=${REPORT_BENCHMARK_ROWS:-100000}
RUNS=${REPORT_BENCHMARK_RUNS:-5}

cleanup_reporting_benchmark() {
	docker stop "$MYSQL_CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup_reporting_benchmark EXIT

for command_name in docker date sort sha256sum uname; do
	command -v "$command_name" >/dev/null 2>&1 || {
		echo "missing required command: $command_name" >&2
		exit 1
	}
done
if [[ ! "$ROWS" =~ ^[0-9]+$ || "$ROWS" -lt 1000 || "$ROWS" -gt 1000000 ]]; then
	echo "REPORT_BENCHMARK_ROWS must be between 1000 and 1000000" >&2
	exit 1
fi
if [[ ! "$RUNS" =~ ^[0-9]+$ || "$RUNS" -lt 3 || "$RUNS" -gt 20 ]]; then
	echo "REPORT_BENCHMARK_RUNS must be between 3 and 20" >&2
	exit 1
fi

docker run --rm -d --name "$MYSQL_CONTAINER" \
	-e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" \
	-e MYSQL_DATABASE=aofei \
	"$MYSQL_IMAGE" >/dev/null
for attempt in $(seq 1 60); do
	if docker exec "$MYSQL_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e 'SELECT 1' >/dev/null 2>&1; then
		break
	fi
	if [[ "$attempt" -eq 60 ]]; then
		docker logs "$MYSQL_CONTAINER" | tail -50 >&2
		exit 1
	fi
	sleep 1
done

cd "$ROOT"
docker exec -i "$MYSQL_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei < etc/step4_init.sql

docker exec "$MYSQL_CONTAINER" mysql -uroot -p"$MYSQL_PASSWORD" aofei -e "
SET SESSION cte_max_recursion_depth=$((ROWS + 1));
INSERT INTO report_delivery
  (timely,demand_source,adv_id,campaign_id,item_id,creative_id,bidder_id,group_id,
   route_bidder_id,target_id,pub_id,site_id,slot_id,country_id,state_id,device_os,
   device_type,inventory_environment,integration_mode,media_intent,placement,
   render_context,refresh_mode,refresh_seconds,ad_density,traffic_quality,source_quality,
   management_control,seller_type,seller_id,dimension_hash,
   wins,losses,imps,clis,spend_usd,revenue_usd,cost_usd,margin_usd,
   downstream_cpm_sum,returned_cpm_sum,callback_errors)
WITH RECURSIVE seq(n) AS (
  SELECT 0 UNION ALL SELECT n+1 FROM seq WHERE n+1<$ROWS
)
SELECT TIMESTAMP('2026-07-30 00:00:00')+INTERVAL n SECOND,
       ELT((n MOD 4)+1,'Local','Fallback','Always','MiddlemanUnknown'),
       (n MOD 50)+1,(n MOD 200)+1,(n MOD 500)+1,(n MOD 1000)+1,
       n MOD 11,n MOD 7,n MOD 13,n MOD 17,(n MOD 30)+1,(n MOD 120)+1,
       (n MOD 300)+1,(n MOD 250)+1,n MOD 32,n MOD 12,n MOD 8,
       ELT((n MOD 3)+1,'Web','App','CTV'),ELT((n MOD 3)+1,'BrowserTag','SDK','ADX'),
       ELT((n MOD 3)+1,'Banner','Video','Native'),ELT((n MOD 3)+1,'AboveFold','InFeed','Interstitial'),
       ELT((n MOD 2)+1,'WebPage','InApp'),ELT((n MOD 2)+1,'None','Event'),0,
       ELT((n MOD 3)+1,'Low','Standard','High'),'Reviewed',
       ELT((n MOD 2)+1,'OwnedOperated','Partner'),ELT((n MOD 2)+1,'Publisher','Partner'),
       ELT((n MOD 2)+1,'Publisher','Intermediary'),CONCAT('seller-',n MOD 30),
       UNHEX(SHA2(CONCAT('p02-report-',n),256)),
       1,n MOD 2,1,n MOD 3,0.001000,0.001100,0.000800,0.000300,
       0.800000,1.100000,n MOD 2
FROM seq;
ANALYZE TABLE report_delivery;
" >/dev/null

SUPPLY_DIMENSIONS="inventory_environment,integration_mode,media_intent,placement,render_context,refresh_mode,refresh_seconds,ad_density,traffic_quality,source_quality,management_control,seller_type,seller_id"
ADVERTISER_QUERY="SELECT demand_source,campaign_id,item_id,creative_id,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS,SUM(imps),SUM(clis),SUM(spend_usd) FROM report_delivery WHERE adv_id=17 AND timely>=DATE_SUB('2026-07-31',INTERVAL 1 DAY) AND timely<DATE_ADD('2026-07-31',INTERVAL 1 DAY) GROUP BY demand_source,campaign_id,item_id,creative_id,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS ORDER BY SUM(spend_usd) DESC LIMIT 200"
PUBLISHER_QUERY="SELECT demand_source,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS,SUM(imps),SUM(clis),SUM(revenue_usd) FROM report_delivery WHERE pub_id=19 AND timely>=DATE_SUB('2026-07-31',INTERVAL 1 DAY) AND timely<DATE_ADD('2026-07-31',INTERVAL 1 DAY) GROUP BY demand_source,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS ORDER BY SUM(revenue_usd) DESC LIMIT 200"
OPERATOR_QUERY="SELECT demand_source,adv_id,campaign_id,item_id,creative_id,bidder_id,group_id,route_bidder_id,target_id,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS,SUM(wins),SUM(losses),SUM(imps),SUM(clis),SUM(spend_usd),SUM(revenue_usd),SUM(cost_usd),SUM(margin_usd),SUM(callback_errors) FROM report_delivery WHERE timely>=DATE_SUB('2026-07-31',INTERVAL 1 DAY) AND timely<DATE_ADD('2026-07-31',INTERVAL 1 DAY) GROUP BY demand_source,adv_id,campaign_id,item_id,creative_id,bidder_id,group_id,route_bidder_id,target_id,pub_id,site_id,slot_id,country_id,state_id,device_os,device_type,$SUPPLY_DIMENSIONS ORDER BY SUM(revenue_usd) DESC LIMIT 200"

measure_query_ms() {
	local query=$1
	local values=()
	local started_at finished_at
	for run in $(seq 1 "$RUNS"); do
		started_at=$(date +%s%N)
		docker exec "$MYSQL_CONTAINER" mysql -N -s -uroot -p"$MYSQL_PASSWORD" aofei -e "$query" >/dev/null 2>&1
		finished_at=$(date +%s%N)
		values+=("$(((finished_at - started_at) / 1000000))")
	done
	printf '%s\n' "${values[@]}" | sort -n | awk -v runs="$RUNS" 'NR==int((runs+1)/2){median=$1} END{print median ":" $1}'
}

ADVERTISER_LATENCY=$(measure_query_ms "$ADVERTISER_QUERY")
PUBLISHER_LATENCY=$(measure_query_ms "$PUBLISHER_QUERY")
OPERATOR_LATENCY=$(measure_query_ms "$OPERATOR_QUERY")
ADVERTISER_PLAN=$(docker exec "$MYSQL_CONTAINER" mysql -N -s -uroot -p"$MYSQL_PASSWORD" aofei -e "EXPLAIN FORMAT=JSON $ADVERTISER_QUERY" 2>/dev/null | sha256sum | awk '{print $1}')
PUBLISHER_PLAN=$(docker exec "$MYSQL_CONTAINER" mysql -N -s -uroot -p"$MYSQL_PASSWORD" aofei -e "EXPLAIN FORMAT=JSON $PUBLISHER_QUERY" 2>/dev/null | sha256sum | awk '{print $1}')
OPERATOR_PLAN=$(docker exec "$MYSQL_CONTAINER" mysql -N -s -uroot -p"$MYSQL_PASSWORD" aofei -e "EXPLAIN FORMAT=JSON $OPERATOR_QUERY" 2>/dev/null | sha256sum | awk '{print $1}')
MYSQL_VERSION=$(docker exec "$MYSQL_CONTAINER" mysql -N -s -uroot -p"$MYSQL_PASSWORD" -e 'SELECT VERSION()' 2>/dev/null)
CPU_COUNT=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo unknown)

echo "reporting_benchmark=passed rows=$ROWS runs=$RUNS range_days=2 result_limit=200 mysql=$MYSQL_VERSION image=$MYSQL_IMAGE architecture=$(uname -m) cpu_count=$CPU_COUNT"
echo "advertiser_latency_ms_median_max=$ADVERTISER_LATENCY plan_sha256=$ADVERTISER_PLAN"
echo "publisher_latency_ms_median_max=$PUBLISHER_LATENCY plan_sha256=$PUBLISHER_PLAN"
echo "operator_latency_ms_median_max=$OPERATOR_LATENCY plan_sha256=$OPERATOR_PLAN"
