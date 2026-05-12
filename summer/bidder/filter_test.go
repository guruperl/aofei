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
}
