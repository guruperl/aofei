// Package acl provides functionality for handling acl and category operations.
package acl

type ACL struct {
	// publisher name, publish.id for web, bundle for app
	PubStr string `json:"bundle"`
	// site name, i.e. site.id for web and app.id for app
	SiteStr string `json:"app_id"`
	// slot name, request_domain or tagid. Not used in bw logic
	SlotStr string `json:"slot"`
	// black list advertisers by domain names
	BAdv []string
	// black list applications by store id, i.e. platform-specific application identifiers
	BApp []string
	// white list of content categories
	White []string
	// black list of content categories
	Black []string
	// my content categories
	Categories []string
}
