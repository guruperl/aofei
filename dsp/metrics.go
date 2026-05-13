package dsp

import "expvar"

var (
	metricBidRequests                    = expvar.NewInt("aofei_bid_requests_total")
	metricBidResponses                   = expvar.NewInt("aofei_bid_responses_total")
	metricBidNoBids                      = expvar.NewInt("aofei_bid_no_bids_total")
	metricECPMErrors                     = expvar.NewInt("aofei_ecpm_errors_total")
	metricAuditEnqueued                  = expvar.NewInt("aofei_audit_enqueued_total")
	metricAuditDropped                   = expvar.NewInt("aofei_audit_dropped_total")
	metricAuditPublishErrors             = expvar.NewInt("aofei_audit_publish_errors_total")
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
