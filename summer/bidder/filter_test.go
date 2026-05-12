package bidder

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAdvPresetCannotSetOperatorFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/goto/adv/json/bidder?action=update", nil)
	req.Form = url.Values{
		"adv_id":                {"42"},
		"synthetic_campaign_id": {"7"},
		"synthetic_item_id":     {"8"},
		"synthetic_creative_id": {"9"},
		"credential_ref":        {"secret/ref"},
		"credential_status":     {"Active"},
		"active":                {"Yes"},
	}

	filter := &Filter{}
	filter.R = req
	filter.RoleValue = "adv"
	filter.Action = "update"

	if err := filter.Preset(); err != nil {
		t.Fatal(err)
	}
	for _, field := range operatorFields {
		if got := req.Form.Get(field); got != "" {
			t.Fatalf("%s = %q, want stripped", field, got)
		}
	}
	for _, field := range []string{"credential_status", "active"} {
		if got := req.Form.Get(field); got != "" {
			t.Fatalf("%s = %q, want read-only stripped", field, got)
		}
	}
}

func TestAdvInsertDefaultsInactiveMissingCredential(t *testing.T) {
	req := httptest.NewRequest("POST", "/goto/adv/json/bidder?action=insert", nil)
	req.Form = url.Values{"adv_id": {"42"}}

	filter := &Filter{}
	filter.R = req
	filter.RoleValue = "adv"
	filter.Action = "insert"

	model := &Model{}
	extra := url.Values{}
	if err := filter.Before(model, extra, url.Values{}); err != nil {
		t.Fatal(err)
	}

	if got := extra.Get("adv_id"); got != "42" {
		t.Fatalf("extra adv_id = %q, want 42", got)
	}
	if got := req.Form.Get("credential_status"); got != "Missing" {
		t.Fatalf("credential_status = %q, want Missing", got)
	}
	if got := req.Form.Get("active"); got != "No" {
		t.Fatalf("active = %q, want No", got)
	}
}

func TestAdvAfterHidesOperatorFields(t *testing.T) {
	lists := []map[string]interface{}{
		{
			"bidder_id":             "1",
			"bidder_name":           "remote",
			"synthetic_campaign_id": "7",
			"synthetic_item_id":     "8",
			"synthetic_creative_id": "9",
			"credential_ref":        "secret/ref",
			"credential_status":     "Active",
			"active":                "Yes",
		},
	}
	other := map[string]interface{}{}
	model := &Model{}
	model.LISTS = &lists
	model.OTHER = &other

	filter := &Filter{}
	filter.R = httptest.NewRequest("GET", "/goto/adv/json/bidder?action=topics", nil)
	filter.R.Form = url.Values{}
	filter.RoleValue = "adv"

	if err := filter.After(model); err != nil {
		t.Fatal(err)
	}
	for _, field := range operatorFields {
		if _, ok := lists[0][field]; ok {
			t.Fatalf("%s remained in adv response", field)
		}
	}
	if got := lists[0]["credential_status"]; got != "Active" {
		t.Fatalf("credential_status = %q, want visible Active", got)
	}
	if got := lists[0]["active"]; got != "Yes" {
		t.Fatalf("active = %q, want visible Yes", got)
	}
}

func TestValidateEndpointURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		ok   bool
	}{
		{"https", "https://bidder.example/openrtb", true},
		{"http", "http://127.0.0.1:8080/bid", true},
		{"relative", "/bid", false},
		{"ftp", "ftp://bidder.example/bid", false},
		{"userinfo", "https://user:pass@bidder.example/bid", false},
	}

	for _, tt := range tests {
		form := url.Values{"endpoint_url": {tt.url}}
		err := validateEndpointFields(form, "insert")
		if tt.ok && err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("%s: got nil error, want validation failure", tt.name)
		}
	}
}

func TestNormalizeTimeout(t *testing.T) {
	form := url.Values{"endpoint_url": {"https://bidder.example/openrtb"}}
	if err := validateEndpointFields(form, "insert"); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("timeout_ms"); got != "100" {
		t.Fatalf("default timeout_ms = %q, want 100", got)
	}

	form = url.Values{
		"endpoint_url": {"https://bidder.example/openrtb"},
		"timeout_ms":   {"250"},
	}
	if err := validateEndpointFields(form, "update"); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("timeout_ms"); got != "250" {
		t.Fatalf("timeout_ms = %q, want 250", got)
	}

	form.Set("timeout_ms", "0")
	if err := validateEndpointFields(form, "update"); err == nil {
		t.Fatal("timeout_ms=0 accepted, want validation failure")
	}

	form.Set("timeout_ms", "5001")
	if err := validateEndpointFields(form, "update"); err == nil {
		t.Fatal("timeout_ms=5001 accepted, want validation failure")
	}
}
