package dsp

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"mime"
	"net/url"
	"path"
	"strings"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
	xhtml "golang.org/x/net/html"
)

func validateMiddlemanDownstreamBid(imp *openrtb2.Imp, attr *match.Attribute, bid openrtb2.Bid) error {
	if imp == nil || attr == nil {
		return fmt.Errorf("middleman bid is missing impression metadata")
	}
	if strings.TrimSpace(bid.ID) == "" || strings.TrimSpace(bid.ImpID) == "" {
		return fmt.Errorf("middleman bid identifiers are required")
	}
	if bid.ImpID != imp.ID {
		return fmt.Errorf("middleman bid impression id %q does not match %q", bid.ImpID, imp.ID)
	}
	if bid.Price <= 0 || math.IsNaN(bid.Price) || math.IsInf(bid.Price, 0) {
		return fmt.Errorf("middleman bid price is not a finite positive USD CPM value")
	}
	if strings.TrimSpace(bid.AdM) == "" {
		return fmt.Errorf("middleman bid adm is required")
	}
	w, h := match.SizeID1To2(attr.SizeID)
	if w == 0 || h == 0 || bid.W != int64(w) || bid.H != int64(h) {
		return fmt.Errorf("middleman bid dimensions %dx%d do not match impression %dx%d", bid.W, bid.H, w, h)
	}

	secure := imp.Secure != nil && *imp.Secure == 1
	for _, target := range []struct {
		name string
		raw  string
	}{
		{name: "nurl", raw: bid.NURL},
		{name: "burl", raw: bid.BURL},
		{name: "lurl", raw: bid.LURL},
		{name: "iurl", raw: bid.IURL},
	} {
		if target.raw == "" {
			continue
		}
		if err := validateMiddlemanCreativeURL(target.name, target.raw, secure); err != nil {
			return err
		}
	}

	expected := openrtb2.MarkupBanner
	switch {
	case attr.NativeFormat != nil:
		if imp.Native == nil {
			return fmt.Errorf("middleman native impression metadata is missing")
		}
		expected = openrtb2.MarkupNative
	case attr.IsVideo:
		if imp.Video == nil {
			return fmt.Errorf("middleman video impression metadata is missing")
		}
		expected = openrtb2.MarkupVideo
	case imp.Banner == nil:
		return fmt.Errorf("middleman impression media type is unsupported")
	}
	if bid.MType != 0 && bid.MType != expected {
		return fmt.Errorf("middleman bid markup type %d does not match impression type %d", bid.MType, expected)
	}

	switch expected {
	case openrtb2.MarkupNative:
		return validateMiddlemanNativeAdM(attr.NativeFormat, bid.AdM, secure)
	case openrtb2.MarkupVideo:
		if err := validateVASTMarkup(bid.AdM, secure); err != nil {
			return fmt.Errorf("middleman video adm is not VAST markup: %w", err)
		}
		if bid.Protocol != 0 && len(imp.Video.Protocols) != 0 {
			allowed := false
			for _, protocol := range imp.Video.Protocols {
				if protocol == bid.Protocol {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("middleman video protocol %d is not accepted", bid.Protocol)
			}
		}
		return validateContainedAdMarkup(bid.AdM, secure)
	default:
		return validateContainedAdMarkup(bid.AdM, secure)
	}
}

func validateContainedAdMarkup(adm string, secure bool) error {
	lower := strings.ToLower(adm)
	for _, forbidden := range []string{
		"javascript:", "vbscript:", "data:text/html", "<base", "http-equiv=refresh",
		"http-equiv=\"refresh", "http-equiv='refresh", "target=\"_top", "target='_top",
		"target=\"_parent", "target='_parent", "window.top", "top.location",
		"parent.location", "window.parent", "parent.document", "top.document",
		"document.domain", "allow-top-navigation", "<!doctype", "<!entity",
	} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("middleman adm contains forbidden container-escape content")
		}
	}
	compact := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "").Replace(lower)
	for _, forbidden := range []string{
		"window[\"top\"]", "window['top']", "window[\"parent\"]", "window['parent']",
		"top[\"location\"]", "top['location']", "parent[\"location\"]", "parent['location']",
	} {
		if strings.Contains(compact, forbidden) {
			return fmt.Errorf("middleman adm contains forbidden container-escape content")
		}
	}
	if secure && strings.Contains(lower, "http://") {
		return fmt.Errorf("middleman adm contains an insecure URL for secure inventory")
	}
	tokenizer := xhtml.NewTokenizer(strings.NewReader(adm))
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case xhtml.ErrorToken:
			if err := tokenizer.Err(); err != nil && err != io.EOF {
				return fmt.Errorf("middleman adm is malformed: %w", err)
			}
			return nil
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if strings.EqualFold(token.Data, "base") {
				return fmt.Errorf("middleman adm contains forbidden container-escape content")
			}
			for _, attr := range token.Attr {
				name := strings.ToLower(attr.Key)
				value := strings.TrimSpace(attr.Val)
				switch name {
				case "srcdoc":
					return fmt.Errorf("middleman adm contains forbidden nested markup")
				case "target":
					if strings.EqualFold(value, "_top") || strings.EqualFold(value, "_parent") {
						return fmt.Errorf("middleman adm contains forbidden container-escape content")
					}
				case "sandbox":
					if strings.Contains(strings.ToLower(value), "allow-top-navigation") {
						return fmt.Errorf("middleman adm contains forbidden container-escape content")
					}
				case "http-equiv":
					if strings.EqualFold(value, "refresh") {
						return fmt.Errorf("middleman adm contains forbidden container-escape content")
					}
				case "href", "src", "action", "formaction", "poster", "data", "xlink:href":
					if err := validateContainedMarkupURL(value, secure); err != nil {
						return err
					}
				}
			}
		}
	}
}

func validateContainedMarkupURL(raw string, secure bool) error {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "#") {
		return nil
	}
	if strings.HasPrefix(value, "/") {
		return fmt.Errorf("middleman adm contains a relative or scheme-relative URL")
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("middleman adm contains an invalid URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("middleman adm contains a forbidden URL scheme")
	}
	if secure && scheme == "http" {
		return fmt.Errorf("middleman adm contains an insecure URL for secure inventory")
	}
	return nil
}

func validateVASTMarkup(adm string, secure bool) error {
	decoder := xml.NewDecoder(strings.NewReader(adm))
	var stack []string
	rootSeen := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(value.Name.Local)
			if !rootSeen {
				if name != "vast" {
					return fmt.Errorf("root element is %q", value.Name.Local)
				}
				rootSeen = true
			}
			stack = append(stack, name)
		case xml.EndElement:
			if len(stack) != 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			content := strings.TrimSpace(string(value))
			if content == "" {
				continue
			}
			name := stack[len(stack)-1]
			switch name {
			case "htmlresource":
				if err := validateContainedAdMarkup(content, secure); err != nil {
					return fmt.Errorf("VAST HTMLResource: %w", err)
				}
			case "impression", "error", "tracking", "mediafile", "clickthrough",
				"clicktracking", "customclick", "vastadtaguri", "iframeresource",
				"staticresource", "javascriptresource", "interactivecreativefile",
				"closedcaptionfile":
				if err := validateMiddlemanCreativeURL("VAST "+name, content, secure); err != nil {
					return err
				}
			}
		}
	}
	if !rootSeen {
		return fmt.Errorf("VAST root element is missing")
	}
	return nil
}

func validateMiddlemanNativeAdM(format *match.NativeFormat, adm string, secure bool) error {
	if format == nil {
		return fmt.Errorf("middleman native request format is missing")
	}
	native, err := decodeStrictMiddlemanNative(adm)
	if err != nil || native == nil {
		return fmt.Errorf("middleman native adm is invalid")
	}
	if native.Link == nil || native.Link.URL == "" {
		return fmt.Errorf("middleman native click URL is required")
	}
	if strings.TrimSpace(native.Ver) == "" || (strings.TrimSpace(format.Ver) != "" && native.Ver != format.Ver) {
		return fmt.Errorf("middleman native version %q does not match request version %q", native.Ver, format.Ver)
	}
	for _, target := range append([]string{native.Link.URL, native.Link.Fallback}, append(native.ImpTrackers, native.Link.Clicktrackers...)...) {
		if target == "" {
			continue
		}
		if err := validateMiddlemanCreativeURL("native asset", target, secure); err != nil {
			return err
		}
	}
	requested := make(map[int64]*match.AssetFormat, len(format.Assets))
	for _, asset := range format.Assets {
		if asset == nil {
			return fmt.Errorf("middleman native request contains a nil asset")
		}
		if asset.ID <= 0 {
			return fmt.Errorf("middleman native request asset id %d is invalid", asset.ID)
		}
		if _, exists := requested[asset.ID]; exists {
			return fmt.Errorf("middleman native request asset id %d is duplicated", asset.ID)
		}
		requested[asset.ID] = asset
	}
	if len(requested) == 0 {
		return fmt.Errorf("middleman native request has no assets")
	}
	provided := make(map[int64]bool, len(native.Assets))
	for _, asset := range native.Assets {
		request, ok := requested[asset.ID]
		if !ok {
			return fmt.Errorf("middleman native asset %d was not requested", asset.ID)
		}
		if provided[asset.ID] {
			return fmt.Errorf("middleman native asset %d is duplicated", asset.ID)
		}
		provided[asset.ID] = true
		switch {
		case request.Title != nil:
			if asset.Title == nil || strings.TrimSpace(asset.Title.Text) == "" || asset.Img != nil || asset.Video != nil || asset.Data != nil {
				return fmt.Errorf("middleman native asset %d has the wrong title shape", asset.ID)
			}
		case request.Img != nil:
			if asset.Img == nil || asset.Title != nil || asset.Video != nil || asset.Data != nil {
				return fmt.Errorf("middleman native asset %d has the wrong image shape", asset.ID)
			}
			if err := validateMiddlemanCreativeURL("native image", asset.Img.URL, secure); err != nil {
				return err
			}
			if len(request.Img.MIME) != 0 {
				got := strings.ToLower(strings.Split(mime.TypeByExtension(strings.ToLower(path.Ext(mustURLPath(asset.Img.URL)))), ";")[0])
				allowed := false
				for _, candidate := range request.Img.MIME {
					if strings.EqualFold(strings.TrimSpace(candidate), got) {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("middleman native image MIME %q is not accepted", got)
				}
			}
		case request.Video != nil:
			if asset.Video == nil || asset.Title != nil || asset.Img != nil || asset.Data != nil {
				return fmt.Errorf("middleman native asset %d has the wrong video shape", asset.ID)
			}
			if (asset.Video.AdM == "") == (asset.Video.CURL == "") {
				return fmt.Errorf("middleman native video asset %d must contain exactly one of adm or curl", asset.ID)
			}
			if asset.Video.CURL != "" {
				if err := validateMiddlemanCreativeURL("native video", asset.Video.CURL, secure); err != nil {
					return err
				}
			} else {
				if err := validateVASTMarkup(asset.Video.AdM, secure); err != nil {
					return fmt.Errorf("middleman native video asset %d is not VAST: %w", asset.ID, err)
				}
				if err := validateContainedAdMarkup(asset.Video.AdM, secure); err != nil {
					return fmt.Errorf("middleman native video asset %d is unsafe: %w", asset.ID, err)
				}
			}
		case request.Data != nil:
			if asset.Data == nil || strings.TrimSpace(asset.Data.Value) == "" || asset.Title != nil || asset.Img != nil || asset.Video != nil {
				return fmt.Errorf("middleman native asset %d has the wrong data shape", asset.ID)
			}
			if request.Data.Type != 0 && asset.Data.Type != 0 && request.Data.Type != asset.Data.Type {
				return fmt.Errorf("middleman native asset %d has data type %d, want %d", asset.ID, asset.Data.Type, request.Data.Type)
			}
		default:
			return fmt.Errorf("middleman native asset %d has an unsupported request shape", asset.ID)
		}
	}
	for _, request := range format.Assets {
		if request != nil && (request.Required != 0 || request.Req != 0) && !provided[request.ID] {
			return fmt.Errorf("middleman native response omitted required asset %d", request.ID)
		}
	}
	return nil
}

func decodeStrictMiddlemanNative(adm string) (*match.Native, error) {
	var envelope struct {
		Native *match.Native `json:"native"`
	}
	decoder := json.NewDecoder(strings.NewReader(adm))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing native response data")
		}
		return nil, err
	}
	if envelope.Native == nil {
		return nil, fmt.Errorf("native response is missing")
	}
	return envelope.Native, nil
}

func mustURLPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Path
}

func validateMiddlemanCreativeURL(name, raw string, secure bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("middleman %s URL: %w", name, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("middleman %s URL must be an absolute HTTP(S) URL without credentials", name)
	}
	if secure && scheme != "https" {
		return fmt.Errorf("middleman %s URL must use HTTPS for secure inventory", name)
	}
	return nil
}
