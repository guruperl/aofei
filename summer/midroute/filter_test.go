package midroute

import (
	"net/url"
	"testing"
)

func TestNormalizeGroupFields(t *testing.T) {
	form := url.Values{"group_name": {"fallback"}}
	if err := normalizeActionFields(form, "insert"); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("trigger_mode"); got != "Fallback" {
		t.Fatalf("trigger_mode=%q", got)
	}
	if got := form.Get("total_timeout_ms"); got != "100" {
		t.Fatalf("total_timeout_ms=%q", got)
	}
	if got := form.Get("active"); got != "Yes" {
		t.Fatalf("active=%q", got)
	}
}

func TestNormalizeUpdateFieldsDoesNotDefaultMissingValues(t *testing.T) {
	group := url.Values{"group_id": {"1"}, "margin_pct": {"0.25"}}
	if err := normalizeActionFields(group, "update"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"trigger_mode", "active", "total_timeout_ms", "min_margin_cpm"} {
		if group.Has(name) {
			t.Fatalf("group update defaulted %s=%q", name, group.Get(name))
		}
	}

	bidder := url.Values{"route_bidder_id": {"2"}, "timeout_ms": {""}}
	if err := normalizeActionFields(bidder, "updateBidder"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bidder_id", "active", "priority", "margin_pct", "min_margin_cpm"} {
		if bidder.Has(name) {
			t.Fatalf("bidder update defaulted %s=%q", name, bidder.Get(name))
		}
	}

	target := url.Values{"target_id": {"3"}, "priority": {"7"}}
	if err := normalizeActionFields(target, "updateTarget"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"entitytype_id", "entity_id", "size_id", "active"} {
		if target.Has(name) {
			t.Fatalf("target update defaulted %s=%q", name, target.Get(name))
		}
	}
}

func TestNormalizeUpdateFieldsRejectsBlankNonNullableValues(t *testing.T) {
	tests := []struct {
		action string
		form   url.Values
	}{
		{"update", url.Values{"group_id": {"1"}, "total_timeout_ms": {""}}},
		{"update", url.Values{"group_id": {"1"}, "margin_pct": {""}}},
		{"updateBidder", url.Values{"route_bidder_id": {"2"}, "priority": {""}}},
		{"updateTarget", url.Values{"target_id": {"3"}, "priority": {""}}},
	}
	for _, tt := range tests {
		if err := normalizeActionFields(tt.form, tt.action); err == nil {
			t.Fatalf("expected error for %s %#v", tt.action, tt.form)
		}
	}
}

func TestNormalizeGroupFieldsRejectsInvalidValues(t *testing.T) {
	tests := []url.Values{
		{"group_name": {"fallback"}, "trigger_mode": {"Sometimes"}},
		{"group_name": {"fallback"}, "total_timeout_ms": {"0"}},
		{"group_name": {"fallback"}, "total_timeout_ms": {"5001"}},
		{"group_name": {"fallback"}, "margin_pct": {"1.5"}},
		{"group_name": {"fallback"}, "min_margin_cpm": {"-0.01"}},
	}
	for _, form := range tests {
		if err := normalizeActionFields(form, "insert"); err == nil {
			t.Fatalf("expected error for %#v", form)
		}
	}
}

func TestNormalizeRouteBidderFieldsKeepsBlankOverrides(t *testing.T) {
	form := url.Values{"group_id": {"1"}, "bidder_id": {"2"}}
	if err := normalizeActionFields(form, "insertBidder"); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("priority"); got != "100" {
		t.Fatalf("priority=%q", got)
	}
	if got := form.Get("timeout_ms"); got != "" {
		t.Fatalf("timeout_ms=%q", got)
	}
}

func TestNormalizeRouteTargetFields(t *testing.T) {
	global := url.Values{"group_id": {"1"}}
	if err := normalizeActionFields(global, "insertTarget"); err != nil {
		t.Fatal(err)
	}
	if got := global.Get("priority"); got != "100" {
		t.Fatalf("priority=%q", got)
	}

	scoped := url.Values{"group_id": {"1"}, "entitytype_id": {"31"}, "entity_id": {"7"}, "size_id": {"16"}}
	if err := normalizeActionFields(scoped, "insertTarget"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeRouteTargetFieldsRejectsInvalidScope(t *testing.T) {
	tests := []url.Values{
		{"group_id": {"1"}, "entitytype_id": {"31"}},
		{"group_id": {"1"}, "entity_id": {"7"}},
		{"group_id": {"1"}, "entitytype_id": {"4"}, "entity_id": {"7"}},
	}
	for _, form := range tests {
		if err := normalizeActionFields(form, "insertTarget"); err == nil {
			t.Fatalf("expected error for %#v", form)
		}
	}
}

func TestMergeUpdateValuesPreservesAbsentAndClearsSubmittedNullable(t *testing.T) {
	current := map[string]interface{}{
		"group_name":     "Fallback",
		"timeout_ms":     int64(90),
		"margin_pct":     "0.0500",
		"min_margin_cpm": nil,
	}
	args := url.Values{"timeout_ms": {""}, "margin_pct": {"0.1000"}}

	if got := mergedValue(args, current, "group_name"); got != "Fallback" {
		t.Fatalf("absent group_name merged to %#v", got)
	}
	if got := mergedNullableValue(args, current, "timeout_ms"); got != nil {
		t.Fatalf("blank timeout_ms merged to %#v, want nil", got)
	}
	if got := mergedNullableValue(args, current, "margin_pct"); got != "0.1000" {
		t.Fatalf("submitted margin_pct merged to %#v", got)
	}
	if got := mergedNullableValue(args, current, "min_margin_cpm"); got != nil {
		t.Fatalf("absent nil min_margin_cpm merged to %#v", got)
	}
}
