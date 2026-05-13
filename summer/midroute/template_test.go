package midroute

import (
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestMidrouteTemplatesRender(t *testing.T) {
	root := os.Getenv("PZDESIGN_TMPLS")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "pzdesign", "tmpls"))
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Skipf("pzdesign templates not found at %s", root)
	}

	tests := []struct {
		action string
		lists  []map[string]interface{}
		other  map[string]interface{}
	}{
		{"topics", []map[string]interface{}{templateGroupRow()}, nil},
		{"health", []map[string]interface{}{templateHealthRow()}, templateCacheOther()},
		{"startnew", []map[string]interface{}{templateGroupFormRow()}, nil},
		{"insert", []map[string]interface{}{{"group_id": int64(11)}}, nil},
		{"edit", []map[string]interface{}{templateGroupRow()}, nil},
		{"update", []map[string]interface{}{templateGroupRow()}, nil},
		{"delete", []map[string]interface{}{{"group_id": int64(11)}}, nil},
		{"bidders", []map[string]interface{}{templateBidderRouteRow()}, templateRouteOther()},
		{"startnewBidder", []map[string]interface{}{templateNewBidderRouteRow()}, templateRouteOther()},
		{"insertBidder", []map[string]interface{}{templateBidderRouteRow()}, nil},
		{"editBidder", []map[string]interface{}{templateBidderRouteRow()}, templateRouteOther()},
		{"updateBidder", []map[string]interface{}{templateBidderRouteRow()}, nil},
		{"deleteBidder", []map[string]interface{}{{"route_bidder_id": int64(21), "group_id": int64(11)}}, nil},
		{"targets", []map[string]interface{}{templateTargetRouteRow()}, templateRouteOther()},
		{"startnewTarget", []map[string]interface{}{templateNewTargetRouteRow()}, templateRouteOther()},
		{"insertTarget", []map[string]interface{}{templateTargetRouteRow()}, nil},
		{"editTarget", []map[string]interface{}{templateTargetRouteRow()}, templateRouteOther()},
		{"updateTarget", []map[string]interface{}{templateTargetRouteRow()}, nil},
		{"deleteTarget", []map[string]interface{}{{"target_id": int64(31), "group_id": int64(11)}}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			tmplPath := filepath.Join(root, "admin", "midroute", tt.action+".g")
			globPath := filepath.Join(root, "admin", "*.g")
			tmpl := template.New(tt.action + ".g").Option("missingkey=zero")
			parsed, err := tmpl.ParseFiles(tmplPath)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err = parsed.ParseGlob(globPath)
			if err != nil {
				t.Fatal(err)
			}

			other := map[string]interface{}{
				"Role":      "admin",
				"Action":    tt.action,
				"Component": "midroute",
				"Tag":       "g",
			}
			for k, v := range tt.other {
				other[k] = v
			}
			page := &genelet.Tmpl{
				Lists:   tt.lists,
				ARGS:    templateArgs(),
				Other:   other,
				Success: true,
			}
			if _, err := page.Get_page(parsed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func templateArgs() url.Values {
	args := url.Values{}
	args.Set("admin_id", "101")
	args.Set("admin_login", "admin")
	return args
}

func templateRouteOther() map[string]interface{} {
	return map[string]interface{}{
		"midroute_group":        templateGroupRow(),
		"midroute_bidders":      []map[string]interface{}{templateBidderOption()},
		"midroute_sizes":        []map[string]interface{}{templateSizeOption()},
		"midroute_cache_status": templateCacheStatus(),
	}
}

func templateCacheOther() map[string]interface{} {
	return map[string]interface{}{
		"midroute_cache_status": templateCacheStatus(),
	}
}

func templateCacheStatus() map[string]interface{} {
	return map[string]interface{}{
		"cache_key":              "middleman:routes:v2",
		"cache_status":           "fresh",
		"cache_generated_at":     "2026-05-13T00:00:00Z",
		"cache_entry_count":      int64(1),
		"cache_route_high_water": "2026-05-13T00:00:00Z",
		"db_route_high_water":    "2026-05-13T00:00:00Z",
		"cache_source":           "mysql",
		"cache_checksum":         "abc123",
	}
}

func templateGroupRow() map[string]interface{} {
	return map[string]interface{}{
		"group_id":         int64(11),
		"group_name":       "Fallback Buyers",
		"trigger_mode":     "Fallback",
		"total_timeout_ms": int64(100),
		"margin_pct":       "0.1000",
		"min_margin_cpm":   "0.0500",
		"active":           "Yes",
		"bidder_count":     int64(1),
		"target_count":     int64(1),
	}
}

func templateGroupFormRow() map[string]interface{} {
	row := templateGroupRow()
	row["group_id"] = int64(0)
	return row
}

func templateBidderOption() map[string]interface{} {
	return map[string]interface{}{
		"bidder_id":         int64(7),
		"bidder_name":       "Remote Bidder",
		"adv_id":            int64(2),
		"adv_email":         "adv@example.test",
		"credential_status": "Active",
		"active":            "Yes",
	}
}

func templateBidderRouteRow() map[string]interface{} {
	return map[string]interface{}{
		"route_bidder_id":          int64(21),
		"group_id":                 int64(11),
		"bidder_id":                int64(7),
		"bidder_name":              "Remote Bidder",
		"adv_id":                   int64(2),
		"adv_email":                "adv@example.test",
		"priority":                 int64(100),
		"timeout_ms":               int64(90),
		"margin_pct":               "0.0500",
		"min_margin_cpm":           "0.0100",
		"bidder_credential_status": "Active",
		"bidder_active":            "Yes",
		"active":                   "Yes",
	}
}

func templateNewBidderRouteRow() map[string]interface{} {
	return map[string]interface{}{
		"group_id":       int64(11),
		"bidder_id":      int64(0),
		"priority":       int64(100),
		"timeout_ms":     "",
		"margin_pct":     "",
		"min_margin_cpm": "",
		"active":         "Yes",
	}
}

func templateSizeOption() map[string]interface{} {
	return map[string]interface{}{
		"size_id":   int64(16),
		"size_name": "Medium Rectangle",
		"width":     int64(300),
		"height":    int64(250),
	}
}

func templateTargetRouteRow() map[string]interface{} {
	return map[string]interface{}{
		"target_id":       int64(31),
		"group_id":        int64(11),
		"entitytype_id":   int64(31),
		"entitytype_site": true,
		"entity_id":       int64(3),
		"entity_name":     "defaultWeb",
		"size_id":         int64(16),
		"size_name":       "Medium Rectangle 300x250",
		"priority":        int64(100),
		"active":          "Yes",
	}
}

func templateNewTargetRouteRow() map[string]interface{} {
	return map[string]interface{}{
		"group_id":          int64(11),
		"entitytype_global": true,
		"entity_id":         "",
		"size_id":           int64(0),
		"priority":          int64(100),
		"active":            "Yes",
	}
}

func templateHealthRow() map[string]interface{} {
	return map[string]interface{}{
		"severity":          "error",
		"issue_type":        "missing_credential_ref",
		"group_id":          int64(11),
		"group_name":        "Fallback Buyers",
		"bidder_id":         int64(7),
		"bidder_name":       "Remote Bidder",
		"credential_ref":    "MID_BIDDER_HEADERS",
		"credential_status": "Active",
		"bidder_active":     "Yes",
		"detail":            "credential_ref is empty",
	}
}
