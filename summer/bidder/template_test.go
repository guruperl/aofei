package bidder

import (
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestBidderTemplatesRender(t *testing.T) {
	root := os.Getenv("PZDESIGN_TMPLS")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "pzdesign", "tmpls"))
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Skipf("pzdesign templates not found at %s", root)
	}

	tests := []struct {
		role   string
		action string
		lists  []map[string]interface{}
	}{
		{"adv", "topics", []map[string]interface{}{templateBidderRow()}},
		{"adv", "startnew", nil},
		{"adv", "edit", []map[string]interface{}{templateBidderRow()}},
		{"adv", "insert", nil},
		{"adv", "update", nil},
		{"admin", "topics", []map[string]interface{}{templateBidderRow()}},
		{"admin", "edit", []map[string]interface{}{templateBidderRow()}},
		{"admin", "update", nil},
		{"admin", "approve", []map[string]interface{}{templateBidderRow()}},
	}

	for _, tt := range tests {
		t.Run(tt.role+"_"+tt.action, func(t *testing.T) {
			tmplPath := filepath.Join(root, tt.role, "bidder", tt.action+".g")
			globPath := filepath.Join(root, tt.role, "*.g")
			tmpl := template.New(tt.action + ".g").Option("missingkey=zero")
			parsed, err := tmpl.ParseFiles(tmplPath)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err = parsed.ParseGlob(globPath)
			if err != nil {
				t.Fatal(err)
			}

			page := &genelet.Tmpl{
				Lists:   tt.lists,
				ARGS:    templateArgs(tt.role),
				Other:   map[string]interface{}{"Role": tt.role, "Action": tt.action, "Component": "bidder", "Tag": "g"},
				Success: true,
			}
			if _, err := page.Get_page(parsed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func templateArgs(role string) url.Values {
	args := url.Values{}
	switch role {
	case "adv":
		args.Set("a_email", "adv@example.test")
		args.Set("a_company", "Advertiser")
	case "admin":
		args.Set("admin_id", "101")
		args.Set("admin_login", "admin")
	}
	return args
}

func templateBidderRow() map[string]interface{} {
	return map[string]interface{}{
		"bidder_id":             "11",
		"adv_id":                "1",
		"adv_email":             "adv@example.test",
		"synthetic_campaign_id": "101",
		"synthetic_item_id":     "201",
		"synthetic_creative_id": "301",
		"bidder_name":           "Remote Bidder",
		"endpoint_url":          "https://bidder.example/openrtb",
		"openrtb_version":       "2.5",
		"seat":                  "seat-1",
		"credential_ref":        "secret/ref",
		"credential_status":     "Active",
		"timeout_ms":            "100",
		"active":                "Yes",
		"created":               "2026-05-12",
	}
}
