package dsp

import (
	"expvar"
	"sync/atomic"
	"time"
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
	metricAuditEnqueued                  = expvar.NewInt("aofei_audit_enqueued_total")
	metricAuditDropped                   = expvar.NewInt("aofei_audit_dropped_total")
	metricAuditPublishErrors             = expvar.NewInt("aofei_audit_publish_errors_total")
	metricAuditQueueDepth                = expvar.NewInt("aofei_audit_queue_depth")
	metricMiddlemanCallbackSetupFailures = expvar.NewInt("aofei_middleman_callback_setup_failures_total")
	metricMiddlemanForwardOK             = expvar.NewInt("aofei_middleman_forward_ok_total")
	metricMiddlemanForwardErrors         = expvar.NewInt("aofei_middleman_forward_errors_total")
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
)

func init() {
	publishExpvarFunc("aofei_local_cache_loaded_at_unix", func() any { return localCacheLoadedAtUnix.Load() })
	publishExpvarFunc("aofei_local_cache_age_seconds", func() any { return localCacheAgeSecondsAt(time.Now()) })
	publishExpvarFunc("aofei_local_cache_stale", func() any { return localCacheStaleAt(time.Now()) })
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
