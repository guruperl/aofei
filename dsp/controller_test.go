package dsp

import "testing"

func TestControllerOptionsCanDisableNATSAndMaxMindIndependently(t *testing.T) {
	defaults := applyControllerOptions()
	if !defaults.nats || !defaults.maxmind {
		t.Fatalf("defaults = %+v, want both optional services enabled", defaults)
	}

	withoutNATS := applyControllerOptions(WithoutNATS())
	if withoutNATS.nats || !withoutNATS.maxmind {
		t.Fatalf("WithoutNATS = %+v, want nats disabled and maxmind enabled", withoutNATS)
	}

	withoutMaxMind := applyControllerOptions(WithoutMaxMind())
	if !withoutMaxMind.nats || withoutMaxMind.maxmind {
		t.Fatalf("WithoutMaxMind = %+v, want nats enabled and maxmind disabled", withoutMaxMind)
	}

	withoutBoth := applyControllerOptions(WithoutNATS(), WithoutMaxMind())
	if withoutBoth.nats || withoutBoth.maxmind {
		t.Fatalf("without both = %+v, want both disabled", withoutBoth)
	}
}
