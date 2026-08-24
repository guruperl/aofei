// Package opsmetrics owns fixed-cardinality operational outcome counters.
package opsmetrics

import (
	"expvar"
	"strings"
)

type fixedCounter struct {
	values  *expvar.Map
	allowed map[string]struct{}
}

func newFixedCounter(name string, outcomes ...string) *fixedCounter {
	values := expvar.NewMap(name)
	allowed := make(map[string]struct{}, len(outcomes)+1)
	for _, outcome := range append(outcomes, "other") {
		allowed[outcome] = struct{}{}
		values.Set(outcome, new(expvar.Int))
	}
	return &fixedCounter{values: values, allowed: allowed}
}

func (counter *fixedCounter) add(outcome string) {
	if counter == nil {
		return
	}
	if _, ok := counter.allowed[outcome]; !ok {
		outcome = "other"
	}
	counter.values.Add(outcome, 1)
}

var leaseOutcomes = newFixedCounter("aofei_singleton_lease_outcomes_total",
	"acquired", "held", "acquire_error", "renewed_after_error", "renewal_error",
	"ownership_lost", "uncertainty_expired", "released", "release_not_owner", "release_error")

var cacheOutcomes = newFixedCounter("aofei_cache_publication_outcomes_total",
	"redis_succeeded", "redis_failed", "spread_succeeded", "spread_failed",
	"all_succeeded", "all_failed", "routes_succeeded", "routes_failed")

var spreadOutcomes = newFixedCounter("aofei_spread_generation_outcomes_total",
	"bootstrap_succeeded", "bootstrap_failed", "generation_started", "generation_committed",
	"generation_incomplete", "generation_rejected", "generation_stale",
	"legacy_write_succeeded", "legacy_write_failed")

var filesystemOutcomes = newFixedCounter("aofei_filesystem_outcomes_total",
	"directory_succeeded", "directory_failed", "write_succeeded", "write_failed", "sync_failed")

var callbackRetryOutcomes = newFixedCounter("aofei_callback_retry_outcomes_total",
	"forward_succeeded", "forward_retryable", "forward_abandoned", "forward_rejected",
	"state_succeeded", "state_retrying", "state_abandoned",
	"state_error_after_forward", "state_error_before_forward")

var experimentOutcomes = newFixedCounter("aofei_experiment_operation_outcomes_total",
	"list_succeeded", "list_failed", "create_succeeded", "create_failed",
	"start_succeeded", "start_failed", "stop_succeeded", "stop_failed",
	"complete_succeeded", "complete_failed", "prune_succeeded", "prune_failed",
	"delete_subject_succeeded", "delete_subject_failed")

func RecordLease(outcome string) {
	leaseOutcomes.add(outcome)
}

func RecordCache(mode string, succeeded bool) {
	switch mode {
	case "redis", "spread", "all", "routes":
	default:
		mode = "other"
	}
	suffix := "_failed"
	if succeeded {
		suffix = "_succeeded"
	}
	cacheOutcomes.add(mode + suffix)
}

func RecordSpread(outcome string) {
	spreadOutcomes.add(outcome)
}

func RecordFilesystem(outcome string) {
	filesystemOutcomes.add(outcome)
}

func RecordCallbackRetry(outcome string) {
	callbackRetryOutcomes.add(outcome)
}

func RecordExperiment(operation string, succeeded bool) {
	switch operation {
	case "list", "create", "start", "stop", "complete", "prune", "delete_subject":
	default:
		operation = "other"
	}
	suffix := "_failed"
	if succeeded {
		suffix = "_succeeded"
	}
	experimentOutcomes.add(operation + suffix)
}

// KnownOutcome reports whether an already-published metric contains a fixed
// outcome. It is intended for source-level guards and tests, not mutation.
func KnownOutcome(metricName, outcome string) bool {
	variable := expvar.Get(metricName)
	values, ok := variable.(*expvar.Map)
	if !ok || strings.TrimSpace(outcome) != outcome {
		return false
	}
	return values.Get(outcome) != nil
}
