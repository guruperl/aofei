// Package acl provides functionality for handling acl and category operations.
package acl

const (
	PUBDefault     = "default"
	SITEDefaultApp = "defaultApp"
	SITEDefaultWeb = "defaultWeb"
	SLOTDefault    = "defaultSlot"

	defaultPubSlotSizeID = 4194368
)

func ParseSiteType(value string) SiteType {
	switch value {
	case "Web":
		return SiteTypeWeb
	case "App":
		return SiteTypeAPP
	default:
		return SiteTypeUnknown
	}
}

type SiteType uint8

const (
	SiteTypeUnknown SiteType = iota
	SiteTypeWeb
	SiteTypeAPP
)

type ACL struct {
	// publisher name, publish.id for web, bundle for app
	PubStr   string `json:"bundle"`
	SiteType `json:"site_type"`
	// site foreign id, i.e. site domain .id for web and app foreign .id for app
	SiteStr string `json:"app_id"`
	// slot name, request_domain or tagid. Not used in bw logic
	SlotStr string `json:"slot"`
	// black list advertisers by domain names
	BAdv []string `json:"badv,omitempty"`
	// black list applications by store id, i.e. platform-specific application identifiers
	BApp []string `json:"bapp,omitempty"`
	// white list of content categories
	White []string `json:"acat,omitempty"`
	// black list of content categories
	Black []string `json:"bcat,omitempty"`
	// my content categories
	Categories []string `json:"categories,omitempty"`
}
