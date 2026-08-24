package opsmetrics

import (
	"expvar"
	"strings"
	"testing"
)

func TestOperationalMetricsHaveClosedOutcomeSets(t *testing.T) {
	tests := []struct {
		name     string
		known    string
		recorder func(string)
	}{
		{"aofei_singleton_lease_outcomes_total", "acquired", RecordLease},
		{"aofei_spread_generation_outcomes_total", "generation_committed", RecordSpread},
		{"aofei_filesystem_outcomes_total", "write_succeeded", RecordFilesystem},
		{"aofei_callback_retry_outcomes_total", "state_error_after_forward", RecordCallbackRetry},
	}
	for _, test := range tests {
		if !KnownOutcome(test.name, test.known) || !KnownOutcome(test.name, "other") {
			t.Fatalf("%s fixed outcomes are not initialized", test.name)
		}
		unknown := "https://secret.example/callback?id=123"
		test.recorder(unknown)
		values := expvar.Get(test.name).(*expvar.Map)
		if values.Get(unknown) != nil {
			t.Fatalf("%s admitted an unbounded outcome", test.name)
		}
		if !strings.Contains(values.Get("other").String(), "1") {
			t.Fatalf("%s other counter was not incremented", test.name)
		}
	}

	RecordCache("customer-supplied-mode", false)
	cache := expvar.Get("aofei_cache_publication_outcomes_total").(*expvar.Map)
	if cache.Get("customer-supplied-mode_failed") != nil {
		t.Fatal("cache metric admitted an unbounded mode")
	}
	if cache.Get("other").String() != "1" {
		t.Fatalf("cache other = %s, want 1", cache.Get("other"))
	}

	RecordExperiment("subject-or-account-id", false)
	experiments := expvar.Get("aofei_experiment_operation_outcomes_total").(*expvar.Map)
	if experiments.Get("subject-or-account-id_failed") != nil {
		t.Fatal("experiment metric admitted an unbounded operation")
	}
	if experiments.Get("other").String() != "1" {
		t.Fatalf("experiment other = %s, want 1", experiments.Get("other"))
	}
}
