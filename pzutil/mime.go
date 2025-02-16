package pzutil

import "strings"

// TMime 3 bits, 8 values
type TMime int

const (
	UnknownMime TMime = iota
	JS
	JSON
	HTML
	GIF
	PNG
)

func StringToMime(s string) TMime {
	switch strings.ToLower(s) {
	case "json":
		return JSON
	case "html":
		return HTML
	case "js":
		return JS
	case "gif":
		return GIF
	case "png":
		return PNG
	default:
		return UnknownMime
	}
}

func (self *Status) MimeString() string {
	switch self.Mime {
	case GIF:
		return "gif"
	case PNG:
		return "png"
	case JS:
		return "js"
	case JSON:
		return "json"
	case HTML:
		return "html"
	default:
		return ""
	}
}

// ContentType generate correct content type according to extension
func (self *Status) ContentType() string {
	switch self.Mime {
	case GIF:
		return "image/gif"
	case PNG:
		return "image/png"
	case JS:
		return "application/javascript"
	case JSON:
		return "application/json"
	case HTML:
		return "text/html"
	default:
		return ""
	}
}
