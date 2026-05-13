package ledger

import (
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestMiddlemanLedgerTemplatesRender(t *testing.T) {
	root := os.Getenv("PZDESIGN_TMPLS")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "pzdesign", "tmpls"))
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Skipf("pzdesign templates not found at %s", root)
	}

	tests := []struct {
		role  string
		lists []map[string]interface{}
		other map[string]interface{}
	}{
		{
			role:  "adv",
			lists: []map[string]interface{}{templateAdvMiddlemanHour()},
			other: map[string]interface{}{
				"ledger_topicsMidTopBidders": []map[string]interface{}{templateAdvMiddlemanBidder()},
				"ledger_topicsMidTopSlots":   []map[string]interface{}{templateAdvMiddlemanSlot()},
			},
		},
		{
			role:  "admin",
			lists: []map[string]interface{}{templateAdminMiddlemanHour()},
			other: map[string]interface{}{
				"ledger_topicsMidTopBidders":    []map[string]interface{}{templateAdminMiddlemanBidder()},
				"ledger_topicsMidTopRoutes":     []map[string]interface{}{templateAdminMiddlemanRoute()},
				"ledger_topicsMidTopPublishers": []map[string]interface{}{templateAdminMiddlemanPublisher()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			tmplPath := filepath.Join(root, tt.role, "ledger", "topicsMid24Hours.g")
			globPath := filepath.Join(root, tt.role, "*.g")
			tmpl := template.New("topicsMid24Hours.g").Option("missingkey=zero")
			parsed, err := tmpl.ParseFiles(tmplPath)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err = parsed.ParseGlob(globPath)
			if err != nil {
				t.Fatal(err)
			}

			other := map[string]interface{}{
				"Role":      tt.role,
				"Action":    "topicsMid24Hours",
				"Component": "ledger",
				"Tag":       "g",
			}
			for k, v := range tt.other {
				other[k] = v
			}
			page := &genelet.Tmpl{
				Lists:   tt.lists,
				ARGS:    templateLedgerArgs(tt.role),
				Other:   other,
				Success: true,
			}
			if _, err := page.Get_page(parsed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func templateLedgerArgs(role string) url.Values {
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

func templateAdvMiddlemanHour() map[string]interface{} {
	return map[string]interface{}{"hours": "12:00", "imps": 10, "clis": 1, "spend": 0.95}
}

func templateAdvMiddlemanBidder() map[string]interface{} {
	return map[string]interface{}{
		"bidder_id":   7,
		"bidder_name": "Remote Bidder",
		"spend":       0.95,
		"imps":        10,
		"clis":        1,
		"cpm":         95.0,
		"cpc":         0.95,
		"ctr":         0.1,
	}
}

func templateAdvMiddlemanSlot() map[string]interface{} {
	return map[string]interface{}{
		"slot_id":   10,
		"slot_name": "Home Slot",
		"spend":     0.95,
		"imps":      10,
		"clis":      1,
		"cpm":       95.0,
		"cpc":       0.95,
		"ctr":       0.1,
	}
}

func templateAdminMiddlemanHour() map[string]interface{} {
	return map[string]interface{}{
		"hours":          "12:00",
		"wins":           9,
		"losses":         1,
		"imps":           8,
		"clis":           1,
		"charge_spend":   1.20,
		"pay_spend":      0.95,
		"margin_spend":   0.25,
		"margin_rate":    0.2083,
		"forward_errors": 0,
	}
}

func templateAdminMiddlemanBidder() map[string]interface{} {
	row := templateAdminMiddlemanHour()
	row["bidder_id"] = 7
	row["bidder_name"] = "Remote Bidder"
	return row
}

func templateAdminMiddlemanRoute() map[string]interface{} {
	row := templateAdminMiddlemanHour()
	row["group_id"] = 8
	row["group_name"] = "Fallback"
	row["route_bidder_id"] = 9
	row["target_id"] = 10
	return row
}

func templateAdminMiddlemanPublisher() map[string]interface{} {
	row := templateAdminMiddlemanHour()
	row["pub_id"] = 30
	row["pub_email"] = "pub@example.test"
	return row
}
