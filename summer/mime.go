package summer

type MIMEType int

const (
	MimeUnknown MIMEType = iota
	XHTMLText
	XHTMLBanner
	JSMime
	Iframe
)

var MimeTypes = map[MIMEType]string{
	MimeUnknown: "Unknown",
	XHTMLText:   "XHTML Text Ad (usually mobile)",
	XHTMLBanner: "XHTML Banner Ad. (usually mobile)",
	JSMime:      "JavaScript Ad; must be valid XHTML (i.e., Script Tags Included)",
	Iframe:      "iframe ",
}
