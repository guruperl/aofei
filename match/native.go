package match

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// AssetFormat is the struct for the asset format
// note that adcom1 has no required field
type AssetFormat struct {
	adcom1.AssetFormat
	Required int `json:"required,omitempty"`
}

// NativeFormat is the struct for the native format
// note that adcom1 has no required ver
type NativeFormat struct {
	Ver    string                 `json:"ver"`
	Assets []*AssetFormat         `json:"assets"`
	Ext    map[string]interface{} `json:"ext,omitempty"`
}

// GetSizes return the width and height of the native format
func (self *NativeFormat) GetSizes() (uint16, uint16) {
	w, h, _ := self.validatedSizes()
	return w, h
}

func (self *NativeFormat) validatedSizes() (uint16, uint16, error) {
	if self == nil {
		return 0, 0, nil
	}
	var selectedWidth, selectedHeight uint16
	for index, asset := range self.Assets {
		if asset == nil {
			return 0, 0, fmt.Errorf("native asset %d is nil", index)
		}
		if img := asset.Img; img != nil {
			width, height, err := nativeImageSize(img)
			if err != nil {
				return 0, 0, err
			}
			if selectedWidth == 0 && width != 0 {
				selectedWidth, selectedHeight = width, height
			}
		}
		if video := asset.Video; video != nil {
			// Native video dimensions are optional: 0x0 means omitted. A
			// partial or out-of-range explicit pair remains malformed.
			if video.W != 0 || video.H != 0 {
				width, height, err := validatedSizePair("native video", video.W, video.H)
				if err != nil {
					return 0, 0, err
				}
				if selectedWidth == 0 {
					selectedWidth, selectedHeight = width, height
				}
			}
		}
	}
	return selectedWidth, selectedHeight, nil
}

// nativeImageSize derives the representative native image size. A complete
// preferred pair is authoritative; an incomplete or absent preferred pair
// falls back to a complete minimum pair. Negative or oversized values and
// incomplete minimum pairs are rejected.
func nativeImageSize(img *adcom1.ImageAssetFormat) (uint16, uint16, error) {
	if img == nil {
		return 0, 0, nil
	}
	if img.W < 0 || img.H < 0 || img.W > maxSizeDimension || img.H > maxSizeDimension {
		return 0, 0, fmt.Errorf("native image dimensions %dx%d are outside the supported range 0..%d", img.W, img.H, maxSizeDimension)
	}
	if img.WMin < 0 || img.HMin < 0 || img.WMin > maxSizeDimension || img.HMin > maxSizeDimension {
		return 0, 0, fmt.Errorf("native image minimum dimensions %dx%d are outside the supported range 0..%d", img.WMin, img.HMin, maxSizeDimension)
	}
	if img.W != 0 && img.H != 0 {
		return uint16(img.W), uint16(img.H), nil
	}
	if img.WMin == 0 && img.HMin == 0 {
		return 0, 0, nil
	}
	if img.WMin == 0 || img.HMin == 0 {
		return 0, 0, fmt.Errorf("native image minimum dimensions %dx%d are incomplete", img.WMin, img.HMin)
	}
	return uint16(img.WMin), uint16(img.HMin), nil
}

func requestStringToNativeFormat(bs []byte) (*NativeFormat, error) {
	x := map[string]*NativeFormat{}
	if err := json.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x["native"], nil
}

// NewNativeFormat creates a new NativeFormat from penrtb2 native
func NewNativeFormat(native *openrtb2.Native) (*NativeFormat, error) {
	if native != nil && native.Request != "" {
		return requestStringToNativeFormat([]byte(native.Request))
	}
	return nil, nil
}

// Asset is the simplest struct for the asset
// note that adcom1 has no required Img nor Required
type Asset struct {
	ID       int64              `json:"id"`
	Required int8               `json:"required,omitempty"`
	Title    *adcom1.TitleAsset `json:"title,omitempty"`
	Img      *adcom1.ImageAsset `json:"img,omitempty"`
	Video    *adcom1.VideoAsset `json:"video,omitempty"`
	Data     *adcom1.DataAsset  `json:"data,omitempty"`
}

// LinkAsset is the struct for the link asset
// note that adcom1 Link is very different
type LinkAsset struct {
	URL           string   `json:"url,omitempty"`
	Clicktrackers []string `json:"clicktrackers,omitempty"`
	Fallback      string   `json:"fallback,omitempty"`
}

// Native is the struct for the native
// note that adcom1 has no ImpTrackers nor Ver, and Link is different
type Native struct {
	Ver         string                 `json:"ver"`
	Assets      []Asset                `json:"assets"`
	Link        *LinkAsset             `json:"link,omitempty"`
	ImpTrackers []string               `json:"imptrackers,omitempty"`
	Ext         map[string]interface{} `json:"ext,omitempty"`
}

// AdM returns the adm string
func (self *Native) AdM(landing, failback string, impTrackers, clickTrackers []string) (string, error) {
	if self.Link == nil {
		self.Link = &LinkAsset{}
	}
	self.Link.URL = landing
	self.Link.Fallback = failback
	self.ImpTrackers = impTrackers
	self.Link.Clicktrackers = clickTrackers
	x := map[string]*Native{"native": self}
	bs, err := json.Marshal(x)
	return string(bs), err
}

// UnpackAdm returns the native from the adm string
func UnpackAdm(bs []byte) (*Native, error) {
	x := map[string]*Native{}
	if err := json.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x["native"], nil
}

// DefaultImgNative creates a default image native
func DefaultImgNative(url string, title string, w, h uint16) *Native {
	return &Native{
		Ver: "1.1",
		Assets: []Asset{
			{
				Required: 1,
				ID:       1,
				Title: &adcom1.TitleAsset{
					Text: title,
				},
			}, {
				Required: 1,
				ID:       2,
				Img: &adcom1.ImageAsset{
					Type: 1,
					URL:  url,
					W:    int64(w),
					H:    int64(h),
				},
			},
		},
	}
}

// DefaultVideoNative creates a default video native from cohntent only
func DefaultVideoNative(adm string) *Native {
	return &Native{
		Ver: "1.1",
		Assets: []Asset{
			{
				Required: 1,
				ID:       1,
				Video: &adcom1.VideoAsset{
					AdM: adm,
				},
			},
		},
	}
}

// NativeCreativeV1 is the source-only authoring contract stored in
// adv_creative.content for Native creatives. It contains data and URLs, never
// executable markup.
type NativeCreativeV1 struct {
	Version      string `json:"version"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	CTA          string `json:"cta"`
	IconURL      string `json:"icon_url,omitempty"`
	MainImageURL string `json:"main_image_url"`
}

func (n NativeCreativeV1) Validate() error {
	if n.Version != "1" {
		return fmt.Errorf("native creative version %q is unsupported", n.Version)
	}
	if strings.TrimSpace(n.Title) == "" {
		return fmt.Errorf("native creative title is required")
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "title", value: n.Title, max: 50},
		{name: "description", value: n.Description, max: 255},
		{name: "cta", value: n.CTA, max: 50},
	} {
		if utf8.RuneCountInString(field.value) > field.max {
			return fmt.Errorf("native creative %s exceeds %d characters", field.name, field.max)
		}
	}
	if strings.TrimSpace(n.MainImageURL) == "" {
		return fmt.Errorf("native creative main_image_url is required")
	}
	return nil
}

func ParseNativeCreativeV1(content string) (*NativeCreativeV1, error) {
	var creative NativeCreativeV1
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&creative); err != nil {
		return nil, fmt.Errorf("decode native creative: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode native creative: trailing data")
		}
		return nil, fmt.Errorf("decode native creative trailing data: %w", err)
	}
	if err := creative.Validate(); err != nil {
		return nil, err
	}
	return &creative, nil
}

func MarshalNativeCreativeV1(creative NativeCreativeV1) (string, error) {
	if err := creative.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(creative)
	return string(data), err
}

// NativeFromCreativeV1 materializes only assets requested by the publisher.
// Required unsupported assets fail the bid; optional unsupported assets are
// omitted.
func NativeFromCreativeV1(format *NativeFormat, creative *NativeCreativeV1, creativeName string, w, h uint16) (*Native, error) {
	if format == nil || creative == nil {
		return nil, fmt.Errorf("native format and creative are required")
	}
	if len(format.Assets) == 0 {
		return nil, fmt.Errorf("native request has no assets")
	}
	response := &Native{Ver: format.Ver}
	if response.Ver == "" {
		response.Ver = "1.1"
	}
	seenIDs := make(map[int64]struct{}, len(format.Assets))
	for _, requested := range format.Assets {
		if requested == nil {
			return nil, fmt.Errorf("native request contains a nil asset")
		}
		if requested.ID <= 0 {
			return nil, fmt.Errorf("native asset id %d is invalid", requested.ID)
		}
		if _, exists := seenIDs[requested.ID]; exists {
			return nil, fmt.Errorf("native asset id %d is duplicated", requested.ID)
		}
		seenIDs[requested.ID] = struct{}{}
		required := requested.Required != 0 || requested.Req != 0
		asset := Asset{ID: requested.ID}
		switch {
		case requested.Title != nil:
			asset.Title = &adcom1.TitleAsset{Text: truncateRunes(creative.Title, requested.Title.Len)}
		case requested.Img != nil:
			imageURL := creative.MainImageURL
			if requested.Img.Type == adcom1.ImageAssetIcon {
				imageURL = creative.IconURL
			}
			if imageURL != "" && len(requested.Img.MIME) != 0 && !mimeAllowed(inferCreativeMIME(imageURL), requested.Img.MIME) {
				if required {
					return nil, fmt.Errorf("required native image asset %d MIME is not accepted", requested.ID)
				}
				imageURL = ""
			}
			if imageURL != "" {
				width, height := requested.Img.W, requested.Img.H
				if width == 0 {
					width = requested.Img.WMin
				}
				if height == 0 {
					height = requested.Img.HMin
				}
				if width == 0 {
					width = int64(w)
				}
				if height == 0 {
					height = int64(h)
				}
				asset.Img = &adcom1.ImageAsset{URL: imageURL, W: width, H: height, Type: requested.Img.Type}
			}
		case requested.Data != nil:
			value := ""
			switch requested.Data.Type {
			case adcom1.DataAssetSponsored:
				value = creativeName
				if value == "" {
					value = creative.Title
				}
			case adcom1.DataAssetDesc, adcom1.DataAssetDesc2:
				value = creative.Description
			case adcom1.DataAssetCTAText:
				value = creative.CTA
			}
			if value != "" {
				value = truncateRunes(value, requested.Data.Len)
				asset.Data = &adcom1.DataAsset{Value: value, Len: int64(utf8.RuneCountInString(value)), Type: requested.Data.Type}
			}
		}
		if asset.Title == nil && asset.Img == nil && asset.Video == nil && asset.Data == nil {
			if required {
				return nil, fmt.Errorf("required native asset %d is unsupported or unavailable", requested.ID)
			}
			continue
		}
		if required {
			asset.Required = 1
		}
		response.Assets = append(response.Assets, asset)
	}
	if len(response.Assets) == 0 {
		return nil, fmt.Errorf("native creative produced no requested assets")
	}
	return response, nil
}

func truncateRunes(value string, max int64) string {
	if max <= 0 || int64(utf8.RuneCountInString(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}
