package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/guruperl/aofei/managementapi"
)

func TestGeneratedClientSendsVersionedSafetyHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/campaigns/7" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("Idempotency-Key") != "retry-key" || r.Header.Get("If-Match") != `"v3"` || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %#v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"resource": map[string]any{"id": 7, "advertiser_id": 2, "name": "campaign", "status": "New", "version": 4, "delivery": map[string]any{"pacing": "Fast", "total_limits": map[string]any{}, "daily_limits": map[string]any{}}, "created_at": "2026-08-01T00:00:00Z"}, "operation": map[string]any{"id": "00112233445566778899aabbccddeeff", "resource_type": "campaign", "resource_id": 7, "accepted_version": 4, "configuration_state": "Accepted", "activation_state": "Pending", "accepted_at": "2026-08-01T00:00:00Z", "activation_deadline": "2026-08-01T00:15:00Z"}}})
	}))
	defer server.Close()
	client, err := New(server.URL, "token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.UpdateCampaign(context.Background(), 7, 3, "retry-key", api.CampaignInput{Name: "campaign", Delivery: api.DeliveryPolicy{Pacing: "Fast"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data.Resource.Version != 4 || response.Data.Operation.ActivationState != "Pending" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGeneratedClientDecodesStableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"version_conflict","message":"conflict","request_id":"req"}}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "token", server.Client())
	_, err := client.GetCampaign(context.Background(), 1)
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Code != "version_conflict" || apiErr.RequestID != "req" {
		t.Fatalf("error = %#v", err)
	}
}

func TestGeneratedClientRejectsAmbiguousOrCredentialedBaseURL(t *testing.T) {
	for _, raw := range []string{
		"ftp://api.example.invalid", "https://user:pass@api.example.invalid",
		"https://api.example.invalid?tenant=other", "https://api.example.invalid#fragment",
	} {
		if _, err := New(raw, "token", nil); err == nil {
			t.Errorf("New(%q) accepted an unsafe base URL", raw)
		}
	}
}
