package dsp

import (
	"github.com/genelet/winter/match"

	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

// convertSite converts the site or app information in the bid request to the corresponding site information
// in the match package, where app is optional to mark whether the bid request is for an app.
func convertRPub(bidRequest *openrtb2.BidRequest) match.RPub {
	DefaultPubIDByName := map[string]uint32{
		"default": 1,
		"pub1":    2,
	}
	DefaultDSPSiteIDByName := map[string]map[string]uint32{
		"default": {
			"default": 1,
			"site1":   2,
		},
		"pub1": {
			"default": 3,
			"app1":    4,
		},
	}
	DefaultDSPSlotIDByName := map[string]map[string]map[string]uint32{
		"default": {
			"default": {
				"default": 1,
				"slot1":   2,
			},
			"site1": {
				"default":      3,
				"http://page1": 4,
			},
		},
		"pub1": {
			"default": {
				"default": 5,
				"slot2":   6,
			},
			"app1": {
				"default":        7,
				"http://siteurl": 8,
			},
		},
	}

	pub := "default"
	site := "default"
	slot := "default"

	if rtbSite := bidRequest.Site; rtbSite != nil {
		if rtbSite.Publisher != nil {
			if _, ok := DefaultPubIDByName[rtbSite.Publisher.ID]; ok {
				pub = rtbSite.Publisher.ID
			}
		}

		if rtbSite.ID != "" {
			if _, ok := DefaultDSPSiteIDByName[pub]; ok {
				if _, ok = DefaultDSPSiteIDByName[pub][rtbSite.ID]; ok {
					site = rtbSite.ID
				}
			}
		}

		if rtbSite.Page != "" {
			if _, ok := DefaultDSPSlotIDByName[pub]; ok {
				if _, ok = DefaultDSPSlotIDByName[pub][site]; ok {
					if _, ok = DefaultDSPSlotIDByName[pub][site][rtbSite.Page]; ok {
						slot = rtbSite.Page
					}
				}
			}
		}
	} else if rtbSite := bidRequest.App; rtbSite != nil {
		if rtbSite.Publisher != nil {
			if _, ok := DefaultPubIDByName[rtbSite.Publisher.ID]; ok {
				pub = rtbSite.Publisher.ID
			}
		}

		if rtbSite.ID != "" {
			if _, ok := DefaultDSPSiteIDByName[pub]; ok {
				if _, ok = DefaultDSPSiteIDByName[pub][rtbSite.ID]; ok {
					site = rtbSite.ID
				}
			}
		}

		if rtbSite.StoreURL != "" {
			if _, ok := DefaultDSPSlotIDByName[pub]; ok {
				if _, ok = DefaultDSPSlotIDByName[pub][site]; ok {
					if _, ok = DefaultDSPSlotIDByName[pub][site][rtbSite.StoreURL]; ok {
						slot = rtbSite.StoreURL
					}
				}
			}
		}
	}

	return match.RPub{
		PubID:  DefaultPubIDByName[pub],
		SiteID: DefaultDSPSiteIDByName[pub][site],
		SlotID: DefaultDSPSlotIDByName[pub][site][slot],
	}
}
