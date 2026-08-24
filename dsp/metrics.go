package dsp

import (
	"context"
	"encoding/json"
	"expvar"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mediocregopher/radix/v4"
)

var (
	localCacheLoadedAtUnix  atomic.Int64
	localCacheMaxAgeSeconds atomic.Int64
)

func publishExpvarFunc(name string, fn func() any) expvar.Func {
	v := expvar.Func(fn)
	expvar.Publish(name, v)
	return v
}

var (
	metricBidRequests                    = expvar.NewInt("aofei_bid_requests_total")
	metricBidResponses                   = expvar.NewInt("aofei_bid_responses_total")
	metricBidNoBids                      = expvar.NewInt("aofei_bid_no_bids_total")
	metricECPMErrors                     = expvar.NewInt("aofei_ecpm_errors_total")
	metricCreativeRejections             = expvar.NewInt("aofei_creative_rejections_total")
	metricAuditEnqueued                  = expvar.NewInt("aofei_audit_enqueued_total")
	metricAuditDropped                   = expvar.NewInt("aofei_audit_dropped_total")
	metricAuditPublishErrors             = expvar.NewInt("aofei_audit_publish_errors_total")
	metricAuditQueueDepth                = expvar.NewInt("aofei_audit_queue_depth")
	metricMiddlemanCallbackSetupFailures = expvar.NewInt("aofei_middleman_callback_setup_failures_total")
	metricMiddlemanForwardOK             = expvar.NewInt("aofei_middleman_forward_ok_total")
	metricMiddlemanForwardErrors         = expvar.NewInt("aofei_middleman_forward_errors_total")
	metricMiddlemanRouteCacheHits        = expvar.NewInt("aofei_middleman_route_cache_hits_total")
	metricMiddlemanRouteCacheMisses      = expvar.NewInt("aofei_middleman_route_cache_misses_total")
	metricMiddlemanRouteCacheRefreshes   = expvar.NewInt("aofei_middleman_route_cache_refreshes_total")
	metricMiddlemanRouteCacheErrors      = expvar.NewInt("aofei_middleman_route_cache_refresh_errors_total")
	metricMiddlemanBidderOutcomes        = expvar.NewMap("aofei_middleman_bidder_outcomes_total")
	metricMiddlemanBidRejections         = expvar.NewMap("aofei_middleman_bid_rejections_total")
	metricMiddlemanCandidates            = expvar.NewMap("aofei_middleman_candidates_total")
	metricLocalCacheReloads              = expvar.NewInt("aofei_local_cache_reloads_total")
	metricLocalCacheReloadErrors         = expvar.NewInt("aofei_local_cache_reload_errors_total")
	metricLocalCacheReloadMillis         = expvar.NewInt("aofei_local_cache_reload_last_ms")
	metricLocalCacheReloadEntries        = expvar.NewInt("aofei_local_cache_reload_last_entries")
	metricSSPRequests                    = expvar.NewInt("aofei_ssp_requests_total")
	metricSSPMalformed                   = expvar.NewInt("aofei_ssp_malformed_total")
	metricSSPValidationErrors            = expvar.NewInt("aofei_ssp_validation_errors_total")
	metricSSPPolicyRejections            = expvar.NewInt("aofei_ssp_policy_rejections_total")
	metricSSPFilledAdUnits               = expvar.NewInt("aofei_ssp_filled_ad_units_total")
	metricSSPNoFillAdUnits               = expvar.NewInt("aofei_ssp_no_fill_ad_units_total")
	metricSSPInventoryTokenOutcomes      = expvar.NewMap("aofei_ssp_inventory_token_outcomes_total")
	metricSSPPublisherAuthOutcomes       = expvar.NewMap("aofei_ssp_publisher_auth_outcomes_total")
	metricSSPPublisherAuthRefreshes      = expvar.NewInt("aofei_ssp_publisher_auth_snapshot_refreshes_total")
	metricSSPPublisherAuthRefreshErrors  = expvar.NewInt("aofei_ssp_publisher_auth_snapshot_refresh_errors_total")
	metricSSPPublisherAuthLoadedAt       = expvar.NewInt("aofei_ssp_publisher_auth_snapshot_loaded_at_unix")
	metricPrivacyDecisions               = expvar.NewMap("aofei_privacy_decisions_total")
	metricPrivacyInvalidSignals          = expvar.NewInt("aofei_privacy_invalid_signals_total")
	metricPrivacyMiddlemanBlocked        = expvar.NewInt("aofei_privacy_middleman_blocked_total")
	metricQualityRefreshes               = expvar.NewInt("aofei_quality_enforcement_refresh_total")
	metricQualityRefreshErrors           = expvar.NewInt("aofei_quality_enforcement_refresh_error_total")
	metricQualityEvaluationErrors        = expvar.NewInt("aofei_quality_enforcement_evaluation_error_total")
	metricQualityThrottle                = expvar.NewInt("aofei_quality_enforcement_throttle_total")
	metricQualityReject                  = expvar.NewInt("aofei_quality_enforcement_reject_total")
	metricQualityQuarantine              = expvar.NewInt("aofei_quality_enforcement_quarantine_total")
	metricTrackingReplaySuppressed       = expvar.NewInt("aofei_tracking_replay_suppressed_total")
	metricTrackingReplayFailOpen         = expvar.NewInt("aofei_tracking_replay_fail_open_total")
	metricTrackingReplayRedisErrors      = expvar.NewInt("aofei_tracking_replay_redis_errors_total")
	metricTrackingReplayUnkeyed          = expvar.NewInt("aofei_tracking_replay_unkeyed_total")
	metricTrackingCapUpdateFailOpen      = expvar.NewInt("aofei_tracking_cap_update_fail_open_total")
	metricTrackingRetryablePublishErrors = expvar.NewInt("aofei_tracking_retryable_publish_errors_total")
	metricTrackingClaimReleases          = expvar.NewInt("aofei_tracking_claim_releases_total")
	metricTrackingClaimReleaseErrors     = expvar.NewInt("aofei_tracking_claim_release_errors_total")
	metricMiddlemanCallbackOutcomes      = expvar.NewMap("aofei_middleman_callback_outcomes_total")
	metricActionRequests                 = expvar.NewInt("aofei_action_requests_total")
	metricActionAccepted                 = expvar.NewInt("aofei_action_accepted_total")
	metricActionDuplicates               = expvar.NewInt("aofei_action_duplicates_total")
	metricActionRejections               = expvar.NewMap("aofei_action_rejections_total")
	metricActionAttributions             = expvar.NewMap("aofei_action_attributions_total")
	metricActionTouches                  = expvar.NewMap("aofei_action_touches_total")
	metricActionTouchErrors              = expvar.NewInt("aofei_action_touch_errors_total")
	metricDeliveryScheduleRejected       = expvar.NewInt("aofei_delivery_schedule_rejected_total")
	metricDeliveryCacheStaleRejected     = expvar.NewInt("aofei_delivery_cache_stale_rejected_total")
	metricDeliveryWindowRejected         = expvar.NewInt("aofei_delivery_window_rejected_total")
	metricDeliveryCachedBudgetRejected   = expvar.NewInt("aofei_delivery_cached_budget_rejected_total")
	metricDeliveryPolicyErrors           = expvar.NewInt("aofei_delivery_policy_errors_total")
	metricDeliveryReservationAttempts    = expvar.NewInt("aofei_delivery_reservation_attempts_total")
	metricDeliveryReservations           = expvar.NewInt("aofei_delivery_reservations_total")
	metricDeliveryReservationRejected    = expvar.NewInt("aofei_delivery_reservation_rejected_total")
	metricDeliveryReservationErrors      = expvar.NewInt("aofei_delivery_reservation_errors_total")
	metricDeliveryReleases               = expvar.NewInt("aofei_delivery_releases_total")
	metricDeliveryReleaseErrors          = expvar.NewInt("aofei_delivery_release_errors_total")
	metricDeliveryFinalized              = expvar.NewInt("aofei_delivery_finalized_total")
	metricDeliveryFinalizeErrors         = expvar.NewInt("aofei_delivery_finalize_errors_total")
	metricDeliveryClicks                 = expvar.NewInt("aofei_delivery_clicks_total")
	metricDeliveryClickErrors            = expvar.NewInt("aofei_delivery_click_errors_total")
	metricTrafficRequests                = expvar.NewMap("aofei_traffic_requests_total")
	metricTrafficResponses               = expvar.NewMap("aofei_traffic_responses_total")
	metricTrafficRejections              = expvar.NewMap("aofei_traffic_rejections_total")
	metricTrafficInFlight                = expvar.NewMap("aofei_traffic_in_flight")
	metricBidPathLatency                 = expvar.NewMap("aofei_bid_path_latency_ms")
	metricDependencyUp                   = expvar.NewMap("aofei_dependency_up")
	metricDependencyCheckLastMS          = expvar.NewMap("aofei_dependency_check_last_ms")
	metricDependencyCheckErrors          = expvar.NewMap("aofei_dependency_check_errors_total")
	metricDBPool                         = expvar.NewMap("aofei_db_pool")
)

func recordSSPInventoryTokenOutcome(outcome string) {
	switch outcome {
	case "legacy_accepted", "v2_accepted", "legacy_disabled", "legacy_rejected", "v2_rejected", "mixed_rejected", "unknown_rejected":
	default:
		outcome = "unknown_rejected"
	}
	metricSSPInventoryTokenOutcomes.Add(outcome, 1)
}

func recordSSPPublisherAuthOutcome(outcome string) {
	switch outcome {
	case "compatibility", "accepted", "required_rejected", "invalid_rejected", "stale_rejected",
		"inventory_rejected", "scope_rejected", "policy_rejected", "replay_rejected", "dependency_error":
	default:
		outcome = "invalid_rejected"
	}
	metricSSPPublisherAuthOutcomes.Add(outcome, 1)
}

func recordMiddlemanCallbackOutcome(outcome string) {
	switch outcome {
	case "forward_inflight_duplicate", "forward_completed_duplicate", "publish_duplicate", "bill_duplicate",
		"local_publish_retryable", "claim_released", "claim_release_error", "claim_completion_error", "retry_queued", "retry_unavailable", "retry_enqueue_error":
	default:
		outcome = "other"
	}
	metricMiddlemanCallbackOutcomes.Add(outcome, 1)
}

func recordActionRejection(reason string) {
	switch reason {
	case "method", "content_type", "body", "payload", "token", "signature", "dependency", "conflict":
	default:
		reason = "other"
	}
	metricActionRejections.Add(reason, 1)
}

func recordActionAttribution(attribution string) {
	switch attribution {
	case "click", "view", "unattributed":
	default:
		attribution = "unattributed"
	}
	metricActionAttributions.Add(attribution, 1)
}

var (
	dependencyUpRedis = new(expvar.Int)
	dependencyUpMySQL = new(expvar.Int)
	dependencyUpNATS  = new(expvar.Int)
	dependencyMSRedis = new(expvar.Int)
	dependencyMSMySQL = new(expvar.Int)
	dbPoolMaxOpen     = new(expvar.Int)
	dbPoolOpen        = new(expvar.Int)
	dbPoolInUse       = new(expvar.Int)
	dbPoolIdle        = new(expvar.Int)
	dbPoolWaitCount   = new(expvar.Int)
	dbPoolWaitMS      = new(expvar.Int)
)

var bidPathLatencyHistograms = map[string]*latencyHistogram{}

var metricMiddlemanBidderLatency = newLatencyHistogram()

type latencyHistogram struct {
	mu      sync.Mutex
	count   uint64
	totalUS uint64
	bounds  []int64
	counts  []uint64
}

type latencyHistogramSnapshot struct {
	Count   uint64            `json:"count"`
	MeanMS  float64           `json:"mean_ms"`
	P50MS   int64             `json:"p50_ms"`
	P95MS   int64             `json:"p95_ms"`
	P99MS   int64             `json:"p99_ms"`
	Buckets map[string]uint64 `json:"buckets"`
}

func newLatencyHistogram() *latencyHistogram {
	bounds := []int64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000}
	return &latencyHistogram{bounds: bounds, counts: make([]uint64, len(bounds)+1)}
}

func (h *latencyHistogram) Observe(duration time.Duration) {
	if h == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	millis := duration.Milliseconds()
	h.mu.Lock()
	h.count++
	h.totalUS += uint64(duration.Microseconds())
	index := len(h.bounds)
	for i, bound := range h.bounds {
		if millis <= bound {
			index = i
			break
		}
	}
	h.counts[index]++
	h.mu.Unlock()
}

func (h *latencyHistogram) String() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	snapshot := latencyHistogramSnapshot{
		Count:   h.count,
		Buckets: make(map[string]uint64, len(h.counts)),
	}
	if h.count != 0 {
		snapshot.MeanMS = float64(h.totalUS) / float64(h.count) / 1000
		snapshot.P50MS = h.percentile(50)
		snapshot.P95MS = h.percentile(95)
		snapshot.P99MS = h.percentile(99)
	}
	for i, count := range h.counts {
		label := "+Inf"
		if i < len(h.bounds) {
			label = time.Duration(h.bounds[i] * int64(time.Millisecond)).String()
		}
		snapshot.Buckets[label] = count
	}
	data, _ := json.Marshal(snapshot)
	return string(data)
}

func (h *latencyHistogram) percentile(percent uint64) int64 {
	if h.count == 0 {
		return 0
	}
	want := (h.count*percent + 99) / 100
	var seen uint64
	for i, count := range h.counts {
		seen += count
		if seen >= want {
			if i < len(h.bounds) {
				return h.bounds[i]
			}
			return h.bounds[len(h.bounds)-1]
		}
	}
	return h.bounds[len(h.bounds)-1]
}

func init() {
	expvar.Publish("aofei_middleman_bidder_latency_ms", metricMiddlemanBidderLatency)
	publishExpvarFunc("aofei_local_cache_loaded_at_unix", func() any { return localCacheLoadedAtUnix.Load() })
	publishExpvarFunc("aofei_local_cache_age_seconds", func() any { return localCacheAgeSecondsAt(time.Now()) })
	publishExpvarFunc("aofei_local_cache_stale", func() any { return localCacheStaleAt(time.Now()) })
	for _, shape := range []string{"adx", "ssp", "local", "middleman", "cap", "audience", "compressed", "fill", "no_fill", "rejection", "overload"} {
		histogram := newLatencyHistogram()
		bidPathLatencyHistograms[shape] = histogram
		metricBidPathLatency.Set(shape, histogram)
	}
	metricDependencyUp.Set("redis", dependencyUpRedis)
	metricDependencyUp.Set("mysql", dependencyUpMySQL)
	metricDependencyUp.Set("nats", dependencyUpNATS)
	metricDependencyCheckLastMS.Set("redis", dependencyMSRedis)
	metricDependencyCheckLastMS.Set("mysql", dependencyMSMySQL)
	metricDBPool.Set("max_open", dbPoolMaxOpen)
	metricDBPool.Set("open", dbPoolOpen)
	metricDBPool.Set("in_use", dbPoolInUse)
	metricDBPool.Set("idle", dbPoolIdle)
	metricDBPool.Set("wait_count", dbPoolWaitCount)
	metricDBPool.Set("wait_ms", dbPoolWaitMS)
}

func recordBidPathLatency(shape string, duration time.Duration) {
	if histogram := bidPathLatencyHistograms[shape]; histogram != nil {
		histogram.Observe(duration)
	}
}

func normalizeTrafficSurface(surface string) string {
	switch surface {
	case "adx", "ssp":
		return surface
	default:
		return "unknown"
	}
}

func recordTrafficRequest(surface string) {
	metricTrafficRequests.Add(normalizeTrafficSurface(surface), 1)
}

func recordTrafficInFlight(surface string, delta int64) {
	metricTrafficInFlight.Add(normalizeTrafficSurface(surface), delta)
}

func recordTrafficRejection(surface, reason string) {
	surface = normalizeTrafficSurface(surface)
	switch reason {
	case "body", "encoding", "qps", "concurrency", "timeout":
	default:
		reason = "other"
	}
	metricTrafficRejections.Add(surface+":"+reason, 1)
}

func recordTrafficResponse(surface string, status int, duration time.Duration) {
	surface = normalizeTrafficSurface(surface)
	outcome := "error"
	shape := "rejection"
	switch {
	case status >= 200 && status < 204:
		outcome = "fill"
		shape = "fill"
	case status == 204:
		outcome = "no_fill"
		shape = "no_fill"
	case status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable:
		outcome = "overload"
		shape = "overload"
	case status >= 400 && status < 500:
		outcome = "rejected"
	case status >= 500:
		outcome = "dependency_error"
	}
	metricTrafficResponses.Add(surface+":"+outcome, 1)
	recordBidPathLatency(surface, duration)
	recordBidPathLatency(shape, duration)
}

func recordMiddlemanBidderOutcome(outcome string) {
	switch outcome {
	case "fill", "no_bid", "invalid_response", "dependency_error", "timeout", "overload", "configuration_error":
	default:
		outcome = "configuration_error"
	}
	metricMiddlemanBidderOutcomes.Add(outcome, 1)
}

func recordMiddlemanBidRejection(reason string) {
	switch reason {
	case "profile", "endpoint", "credential", "timeout", "status", "content_type", "body",
		"envelope", "request_id", "currency", "seat", "impression", "identity", "price",
		"floor", "media", "size", "callback", "markup", "late":
	default:
		reason = "other"
	}
	metricMiddlemanBidRejections.Add(reason, 1)
}

func recordMiddlemanCandidate(stage string, count int) {
	switch stage {
	case "considered", "eligible", "assigned", "returned", "accepted":
	default:
		return
	}
	if count > 0 {
		metricMiddlemanCandidates.Add(stage, int64(count))
	}
}

func (self *Controller) refreshDependencyMetrics(ctx context.Context) {
	if self == nil {
		return
	}
	if self.Redis == nil {
		dependencyUpRedis.Set(0)
		dependencyMSRedis.Set(0)
	} else {
		started := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		err := self.Redis.Do(checkCtx, radix.Cmd(nil, "PING"))
		cancel()
		dependencyMSRedis.Set(time.Since(started).Milliseconds())
		if err != nil {
			dependencyUpRedis.Set(0)
			metricDependencyCheckErrors.Add("redis", 1)
		} else {
			dependencyUpRedis.Set(1)
		}
	}

	if self.DB == nil {
		dependencyUpMySQL.Set(0)
		dependencyMSMySQL.Set(0)
		dbPoolMaxOpen.Set(0)
		dbPoolOpen.Set(0)
		dbPoolInUse.Set(0)
		dbPoolIdle.Set(0)
		dbPoolWaitCount.Set(0)
		dbPoolWaitMS.Set(0)
	} else {
		stats := self.DB.Stats()
		dbPoolMaxOpen.Set(int64(stats.MaxOpenConnections))
		dbPoolOpen.Set(int64(stats.OpenConnections))
		dbPoolInUse.Set(int64(stats.InUse))
		dbPoolIdle.Set(int64(stats.Idle))
		dbPoolWaitCount.Set(stats.WaitCount)
		dbPoolWaitMS.Set(stats.WaitDuration.Milliseconds())
		started := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		err := self.DB.PingContext(checkCtx)
		cancel()
		dependencyMSMySQL.Set(time.Since(started).Milliseconds())
		if err != nil {
			dependencyUpMySQL.Set(0)
			metricDependencyCheckErrors.Add("mysql", 1)
		} else {
			dependencyUpMySQL.Set(1)
		}
	}

	if self.Nc != nil && self.Nc.IsConnected() {
		dependencyUpNATS.Set(1)
	} else {
		dependencyUpNATS.Set(0)
	}
}

func recordPrivacyDecision(decision privacyDecision) {
	decision = decision.normalized()
	metricPrivacyDecisions.Add(privacyDecisionLabel(decision), 1)
	if decision.InvalidSignal {
		metricPrivacyInvalidSignals.Add(1)
	}
}

func setLocalCacheFreshnessMetrics(loadedAt time.Time, maxAgeSeconds int) {
	if loadedAt.IsZero() {
		localCacheLoadedAtUnix.Store(0)
	} else {
		localCacheLoadedAtUnix.Store(loadedAt.Unix())
	}
	localCacheMaxAgeSeconds.Store(int64(maxAgeSeconds))
}

func localCacheAgeSecondsAt(now time.Time) int64 {
	loadedAtUnix := localCacheLoadedAtUnix.Load()
	if loadedAtUnix <= 0 {
		return 0
	}
	age := now.Unix() - loadedAtUnix
	if age < 0 {
		return 0
	}
	return age
}

func localCacheStaleAt(now time.Time) int64 {
	maxAgeSeconds := localCacheMaxAgeSeconds.Load()
	if maxAgeSeconds <= 0 {
		return 0
	}
	if localCacheAgeSecondsAt(now) > maxAgeSeconds {
		return 1
	}
	return 0
}
