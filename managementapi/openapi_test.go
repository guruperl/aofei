package managementapi

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIAndGeneratedClientCoverVersionedRoutes(t *testing.T) {
	contract, err := os.ReadFile("../docs/management-api-openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(contract)
	for _, required := range []string{
		"openapi: 3.1.0", "bearerAuth", "Idempotency-Key", "If-Match",
		"/api/v1/advertiser:", "/api/v1/campaigns:",
		"/api/v1/campaigns/{campaign_id}:", "/api/v1/campaigns/{campaign_id}/items:",
		"/api/v1/items/{item_id}:", "/api/v1/items/{item_id}/creatives:",
		"/api/v1/creatives/{creative_id}:", "/api/v1/items/{item_id}/targeting:",
		"/api/v1/operations/{operation_id}:", "/api/v1/reports/delivery:",
		"x-w8m-scope: api.campaign.write", "x-w8m-scope: api.report.read",
		"NativeCreative:", "Campaign:", "DeliveryReport:", "Operation:",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("OpenAPI contract is missing %q", required)
		}
	}
	client, err := os.ReadFile("client/client.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(client), "// Code generated from docs/management-api-openapi.yaml; DO NOT EDIT.") {
		t.Fatal("generated client provenance marker is missing")
	}
	for _, method := range []string{"Advertiser(", "ListCampaigns(", "CreateCampaign(", "UpdateCampaign(", "ListItems(", "CreateItem(", "ListCreatives(", "CreateCreative(", "GetTargeting(", "UpdateTargeting(", "GetOperation(", "DeliveryReports("} {
		if !strings.Contains(string(client), method) {
			t.Errorf("generated client is missing %s", method)
		}
	}
}
