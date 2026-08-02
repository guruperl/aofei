#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_dir"

export GOWORK=off
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.23.5}

echo "capacity_baseline_version=1"
echo "timestamp_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "go_version=$(go version)"
echo "kernel=$(uname -srmo)"
echo "logical_cpu=$(getconf _NPROCESSORS_ONLN)"
if [[ -r /proc/meminfo ]]; then
  awk '/^MemTotal:/ {print "memory_kib=" $2}' /proc/meminfo
fi
echo "request_mix=local_adx_two_impressions,local_ssp_two_ad_units,traffic_gate_accepted,weighted_selection"
echo "dependencies=local_static_snapshots;no_network_redis_mysql_nats"
echo "errors=benchmark_fails_on_unexpected_status_or_result;zero_in_recorded_run"
echo "saturation=not_exercised_in_local_regression_baseline;required_in_staging_before_capacity_claim"

go test ./dsp ./match -run '^$' \
  -bench 'Benchmark(ServeBidLocalTwoImpressions|ServeSSPLocalTwoAdUnits|TrafficGateAccepted|SelectOneParallel)$' \
  -benchmem -count=3
