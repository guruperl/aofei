package match

import (
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

const (
	PUBDefault    = "default"
	SITEDefault   = "default"
	SLOTDefault   = "default"
	PUBIDDefault  = 1
	SITEIDDefault = 1
	SLOTIDDefault = 1
)

// PubMap is a map of publisher name to ID
var PubMap map[string]uint32 = map[string]uint32{
	PUBDefault: PUBIDDefault,
}

// SiteMap is a map of site name to ID
var SiteMap map[string]map[string]uint32 = map[string]map[string]uint32{
	PUBDefault: {
		SITEDefault: SITEIDDefault,
	},
}

// SlotMap is a map of slot name to ID
var SlotMap map[string]map[string]map[string]uint32 = map[string]map[string]map[string]uint32{
	PUBDefault: {
		SITEDefault: {
			SLOTDefault: SLOTIDDefault,
		},
	},
}

type RPub struct {
	PubID  uint32
	SiteID uint32
	SlotID uint32
	SizeID uint32
}

// getRPub returns the RPub object from the bid request.
func getRPub(bidRequest *openrtb2.BidRequest) RPub {
	pubStr := PUBDefault
	siteStr := SITEDefault
	slotStr := SLOTDefault
	if site := bidRequest.Site; site != nil {
		if site.Publisher != nil && site.Publisher.ID != "" {
			pubStr = site.Publisher.ID
		}
		if site.ID != "" {
			siteStr = site.ID
		}
		if site.Page != "" {
			slotStr = site.Page
		}
	} else if app := bidRequest.App; app != nil {
		if app.Publisher != nil && app.Publisher.ID != "" {
			pubStr = app.Publisher.ID
		}
		if app.ID != "" {
			siteStr = app.ID
		}
	}
	if len(bidRequest.Imp) > 0 && bidRequest.Imp[0].TagID != "" {
		slotStr = bidRequest.Imp[0].TagID
	}

	if _, ok := PubMap[pubStr]; !ok {
		pubStr = PUBDefault
	}
	if _, ok := SiteMap[pubStr][siteStr]; !ok {
		siteStr = SITEDefault
	}
	if _, ok := SlotMap[pubStr][siteStr][slotStr]; !ok {
		slotStr = SLOTDefault
	}

	var rpub RPub
	if id, ok := PubMap[pubStr]; ok {
		rpub.PubID = id
		if id, ok := SiteMap[pubStr][siteStr]; ok {
			rpub.SiteID = id
			if id, ok := SlotMap[pubStr][siteStr][slotStr]; ok {
				rpub.SlotID = id
			}
		}
	}

	return rpub
}

fun (self RPub) GetRAdvs() []RAdv {
	