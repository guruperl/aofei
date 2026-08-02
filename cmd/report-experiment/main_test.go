package main

import (
	"testing"
	"time"
)

func TestParseVariantsNormalizesDeterministicAllocation(t *testing.T) {
	got, err := parseVariants("treatment=5000,control=5000")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "control" || got[0].AllocationBasisPts != 5000 || got[1].Key != "treatment" {
		t.Fatalf("variants = %#v", got)
	}
	if _, err := parseVariants("control"); err == nil {
		t.Fatal("malformed variant allocation was accepted")
	}
}

func TestExperimentFromFlagsKeepsOperatorAndAdvertiserBoundaries(t *testing.T) {
	oldOwner, oldAdvID, oldName, oldVersion := owner, advID, name, version
	oldPrimary, oldGuardrail, oldStart, oldEnd, oldVariants := primaryMetric, guardrailMetric, startsAt, endsAt, variants
	oldRetention := retentionHours
	defer func() {
		owner, advID, name, version = oldOwner, oldAdvID, oldName, oldVersion
		primaryMetric, guardrailMetric, startsAt, endsAt, variants = oldPrimary, oldGuardrail, oldStart, oldEnd, oldVariants
		retentionHours = oldRetention
	}()
	owner, advID, name, version = "advertiser", 17, "copy-test", 3
	primaryMetric, guardrailMetric = "actions", "spend"
	retentionHours = 2160
	startsAt, endsAt = "2026-08-01T00:00:00Z", "2026-08-08T00:00:00Z"
	variants = "control=5000,treatment=5000"
	experiment, err := experimentFromFlags()
	if err != nil {
		t.Fatal(err)
	}
	if experiment.OwnerType != "Advertiser" || experiment.AdvID == nil || *experiment.AdvID != 17 || experiment.RetentionHours != 2160 || experiment.StartsAt.Location() != time.UTC {
		t.Fatalf("experiment = %#v", experiment)
	}
	owner, advID = "advertiser", 0
	if _, err := experimentFromFlags(); err == nil {
		t.Fatal("advertiser experiment without owner id was accepted")
	}
}
