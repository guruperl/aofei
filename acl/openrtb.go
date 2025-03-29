package acl

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// NewOpenRTBACL returns the acl object from the bid request.
func NewOpenRTBACL(bidRequest *openrtb2.BidRequest, pubStr string) *ACL {
	if pubStr == "" && bidRequest.Ext != nil {
		hash := make(map[string]interface{})
		if err := json.Unmarshal(bidRequest.Ext, &hash); err == nil {
			if domain, ok := hash["request_domain"].(string); ok {
				pubStr = domain
			}
		}
	}
	if pubStr == "" {
		pubStr = PUBDefault
	}
	siteStr := SITEDefaultWeb
	if bidRequest.App != nil {
		siteStr = SITEDefaultApp
	}
	slotStr := SLOTDefault

	a := &ACL{
		BAdv:  bidRequest.BAdv,
		BApp:  bidRequest.BApp,
		White: bidRequest.ACat,
		Black: bidRequest.BCat,
	}

	var categories []string
	if site := bidRequest.Site; site != nil {
		if pubStr == PUBDefault && (site.Publisher != nil && site.Publisher.ID != "") {
			pubStr = site.Publisher.ID
		}
		if site.Domain != "" {
			siteStr = site.Domain
		} else if site.ID != "" {
			siteStr = site.ID
		}
		if site.Page != "" {
			slotStr = fmt.Sprintf("%x", md5.Sum([]byte(site.Page)))
		}
		if site.Cat != nil {
			categories = append(categories, site.Cat...)
		}
		if site.SectionCat != nil {
			categories = append(categories, site.SectionCat...)
		}
		if site.PageCat != nil {
			categories = append(categories, site.PageCat...)
		}
	} else if app := bidRequest.App; app != nil {
		if pubStr == PUBDefault && (app.Publisher != nil && app.Publisher.ID != "") {
			pubStr = app.Publisher.ID
		}
		if app.Bundle != "" {
			siteStr = app.Bundle
		} else if app.Domain != "" {
			siteStr = app.Domain
		} else if app.ID != "" {
			siteStr = app.ID
		}
		if app.Cat != nil {
			categories = append(categories, app.Cat...)
		}
		if app.SectionCat != nil {
			categories = append(categories, app.SectionCat...)
		}
		if app.PageCat != nil {
			categories = append(categories, app.PageCat...)
		}
	}
	if slotStr == SLOTDefault && bidRequest.Imp[0].TagID != "" {
		slotStr = bidRequest.Imp[0].TagID
	}

	a.PubStr = pubStr
	a.SiteStr = siteStr
	a.SlotStr = slotStr
	a.Categories = categories

	return a
}
