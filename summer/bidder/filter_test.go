package bidder

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDSPPresetCannotSetOperatorFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/goto/dsp/json/bidder?action=update", nil)
	req.Form = url.Values{
		"dsp_id":            {"42"},
		"credential_ref":    {"secret/ref"},
		"credential_status": {"Active"},
		"active":            {"Yes"},
	}

	filter := &Filter{}
	filter.R = req
	filter.RoleValue = "dsp"
	filter.Action = "update"

	if err := filter.Preset(); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"credential_ref", "credential_status", "active"} {
		if got := req.Form.Get(field); got != "" {
			t.Fatalf("%s = %q, want stripped", field, got)
		}
	}
}

func TestDSPInsertDefaultsInactiveMissingCredential(t *testing.T) {
	req := httptest.NewRequest("POST", "/goto/dsp/json/bidder?action=insert", nil)
	req.Form = url.Values{"dsp_id": {"42"}}

	filter := &Filter{}
	filter.R = req
	filter.RoleValue = "dsp"
	filter.Action = "insert"

	model := &Model{}
	extra := url.Values{}
	if err := filter.Before(model, extra, url.Values{}); err != nil {
		t.Fatal(err)
	}

	if got := extra.Get("dsp_id"); got != "42" {
		t.Fatalf("extra dsp_id = %q, want 42", got)
	}
	if got := req.Form.Get("credential_status"); got != "Missing" {
		t.Fatalf("credential_status = %q, want Missing", got)
	}
	if got := req.Form.Get("active"); got != "No" {
		t.Fatalf("active = %q, want No", got)
	}
}
