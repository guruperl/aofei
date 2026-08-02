package dsp

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type SSPSiteToken string

func (self *SSPSiteToken) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*self = SSPSiteToken(text)
		return nil
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*self = SSPSiteToken(obj.ID)
	return nil
}

type SSPRequest struct {
	ID             string           `json:"id,omitempty"`
	Platform       string           `json:"platform,omitempty"`
	ResponseFormat string           `json:"responseFormat,omitempty"`
	Site           SSPSiteToken     `json:"site"`
	App            *openrtb2.App    `json:"app,omitempty"`
	Device         *openrtb2.Device `json:"device,omitempty"`
	User           *openrtb2.User   `json:"user,omitempty"`
	Regs           *openrtb2.Regs   `json:"regs,omitempty"`
	AdUnits        []SSPAdUnit      `json:"adUnits"`
}

type SSPAdUnit struct {
	Code       string        `json:"code"`
	Slot       string        `json:"slot,omitempty"`
	MediaTypes SSPMediaTypes `json:"mediaTypes,omitempty"`
	Floor      float64       `json:"floor,omitempty"`

	LegacyBanner *SSPBanner `json:"banner,omitempty"`
	LegacyVideo  *SSPVideo  `json:"video,omitempty"`
	LegacyNative *SSPNative `json:"native,omitempty"`
}

type SSPMediaTypes struct {
	Banner *SSPBanner `json:"banner,omitempty"`
	Video  *SSPVideo  `json:"video,omitempty"`
	Native *SSPNative `json:"native,omitempty"`

	unsupported []string
}

func (self *SSPMediaTypes) UnmarshalJSON(data []byte) error {
	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, value := range raw {
		switch key {
		case "banner":
			if err := json.Unmarshal(value, &self.Banner); err != nil {
				return err
			}
		case "video":
			if err := json.Unmarshal(value, &self.Video); err != nil {
				return err
			}
		case "native":
			if err := json.Unmarshal(value, &self.Native); err != nil {
				return err
			}
		default:
			self.unsupported = append(self.unsupported, key)
		}
	}
	sort.Strings(self.unsupported)
	return nil
}

type SSPBanner struct {
	Size []uint16 `json:"size,omitempty"`
}

type SSPVideo struct {
	Context    string   `json:"context,omitempty"`
	PlayerSize []uint16 `json:"playerSize,omitempty"`
	MIMEs      []string `json:"mimes,omitempty"`
}

type SSPNative struct {
	Image       []uint16 `json:"image,omitempty"`
	Icon        []uint16 `json:"icon,omitempty"`
	Title       bool     `json:"title,omitempty"`
	SponsoredBy bool     `json:"sponsoredBy,omitempty"`
	Body        bool     `json:"body,omitempty"`
}

type SSPValidatedUnit struct {
	Code            string
	Site            string
	Slot            string
	SiteStr         string
	SlotStr         string
	SiteType        acl.SiteType
	ConfiguredFloor float64
	RPub            match.RPub
}

func ParseSSPRequest(data []byte) (*SSPRequest, error) {
	req := &SSPRequest{}
	if err := json.Unmarshal(data, req); err != nil {
		return nil, err
	}
	if req.Site == "" {
		return nil, fmt.Errorf("ssp request missing site")
	}
	if len(req.AdUnits) == 0 {
		return nil, fmt.Errorf("ssp request has no adUnits")
	}
	if _, err := req.NormalizedPlatform(); err != nil {
		return nil, err
	}
	if _, err := req.NormalizedResponseFormat(); err != nil {
		return nil, err
	}
	for i := range req.AdUnits {
		if err := req.AdUnits[i].validateStatic(); err != nil {
			return nil, fmt.Errorf("adUnits[%d] %w", i, err)
		}
	}
	return req, nil
}

func (self *SSPRequest) NormalizedPlatform() (string, error) {
	if self == nil {
		return "", fmt.Errorf("ssp request is nil")
	}
	platform := strings.ToLower(strings.TrimSpace(self.Platform))
	if platform == "" {
		platform = "browser"
	}
	switch platform {
	case "browser", "sdk":
		return platform, nil
	default:
		return "", fmt.Errorf("unsupported platform %q", self.Platform)
	}
}

func (self *SSPRequest) NormalizedResponseFormat() (string, error) {
	if self == nil {
		return "", fmt.Errorf("ssp request is nil")
	}
	format := strings.ToLower(strings.TrimSpace(self.ResponseFormat))
	if format == "" {
		format = "html"
	}
	switch format {
	case "html", "json", "openrtb":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported responseFormat %q", self.ResponseFormat)
	}
}

func (self SSPAdUnit) legacySlotToken() string {
	if self.Slot != "" {
		return self.Slot
	}
	return self.Code
}

func (self SSPAdUnit) EffectiveMediaTypes() SSPMediaTypes {
	if self.MediaTypes.Banner != nil || self.MediaTypes.Video != nil || self.MediaTypes.Native != nil || len(self.MediaTypes.unsupported) != 0 {
		return self.MediaTypes
	}
	return SSPMediaTypes{
		Banner: self.LegacyBanner,
		Video:  self.LegacyVideo,
		Native: self.LegacyNative,
	}
}

func (self SSPMediaTypes) Validate() error {
	if len(self.unsupported) != 0 {
		return fmt.Errorf("unsupported mediaTypes: %v", self.unsupported)
	}
	count := 0
	if self.Banner != nil {
		count++
	}
	if self.Video != nil {
		count++
	}
	if self.Native != nil {
		count++
	}
	if count == 0 {
		return fmt.Errorf("missing supported mediaTypes")
	}
	if count != 1 {
		return fmt.Errorf("exactly one supported media type is required")
	}
	return nil
}

func (self SSPMediaTypes) ValidateForSize(w, h uint16) error {
	if err := self.Validate(); err != nil {
		return err
	}
	switch {
	case self.Banner != nil:
		return validateSSPMediaDimensions("banner.size", self.Banner.Size, w, h, true)
	case self.Video != nil:
		return validateSSPMediaDimensions("video.playerSize", self.Video.PlayerSize, w, h, true)
	case self.Native != nil:
		if err := validateSSPMediaDimensions("native.image", self.Native.Image, w, h, true); err != nil {
			return err
		}
		return validateSSPMediaDimensions("native.icon", self.Native.Icon, 0, 0, false)
	default:
		return fmt.Errorf("missing supported mediaTypes")
	}
}

func validateSSPMediaDimensions(field string, dimensions []uint16, wantW, wantH uint16, authoritative bool) error {
	if len(dimensions) == 0 {
		return nil
	}
	if len(dimensions) != 2 || dimensions[0] == 0 || dimensions[1] == 0 {
		return fmt.Errorf("%s must contain two positive dimensions", field)
	}
	if authoritative && (dimensions[0] != wantW || dimensions[1] != wantH) {
		return fmt.Errorf("%s %dx%d does not match configured slot %dx%d", field, dimensions[0], dimensions[1], wantW, wantH)
	}
	return nil
}

func (self SSPAdUnit) validateStatic() error {
	if !validSSPCode(self.Code) {
		return fmt.Errorf("code must be 1-128 URL/DOM-safe characters")
	}
	if self.Floor < 0 || math.IsNaN(self.Floor) || math.IsInf(self.Floor, 0) {
		return fmt.Errorf("floor must be a finite non-negative USD CPM value")
	}
	return self.EffectiveMediaTypes().Validate()
}

func validSSPCode(code string) bool {
	if len(code) == 0 || len(code) > 128 {
		return false
	}
	for _, character := range code {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-' || character == '_' || character == '.' || character == ':':
		default:
			return false
		}
	}
	return true
}

func (self *SSPRequest) ValidateSupply(pub *acl.DirectPub) ([]SSPValidatedUnit, error) {
	if self == nil {
		return nil, fmt.Errorf("ssp request is nil")
	}
	if self.Site == "" {
		return nil, fmt.Errorf("ssp request missing site")
	}
	if len(self.AdUnits) == 0 {
		return nil, fmt.Errorf("ssp request has no adUnits")
	}
	platform, err := self.NormalizedPlatform()
	if err != nil {
		return nil, err
	}

	pubID, siteID, err := acl.UnpackDirectToken(string(self.Site))
	if err != nil {
		return nil, fmt.Errorf("invalid site token: %w", err)
	}
	if pub == nil || pub.Pub == nil {
		return nil, fmt.Errorf("publisher %d not found", pubID)
	}
	if pub.Pub.PubID != pubID {
		return nil, fmt.Errorf("site token pub_id %d does not match cached pub_id %d", pubID, pub.Pub.PubID)
	}
	seenCodes := make(map[string]int, len(self.AdUnits))

	units := make([]SSPValidatedUnit, 0, len(self.AdUnits))
	for i, adUnit := range self.AdUnits {
		if err := adUnit.validateStatic(); err != nil {
			return nil, fmt.Errorf("adUnits[%d] %w", i, err)
		}
		if first, ok := seenCodes[adUnit.Code]; ok {
			return nil, fmt.Errorf("adUnits[%d] duplicate code %q already used by adUnits[%d]", i, adUnit.Code, first)
		}
		seenCodes[adUnit.Code] = i
		slotToken := adUnit.Slot
		if slotToken == "" {
			return nil, fmt.Errorf("adUnits[%d] missing slot", i)
		}
		slotID, sizeID, err := acl.UnpackDirectToken(slotToken)
		if err != nil {
			return nil, fmt.Errorf("adUnits[%d] invalid slot token: %w", i, err)
		}
		siteStr, slotStr, siteType, configuredFloor, ok := pub.CommercialSlot(siteID, slotID, sizeID)
		if !ok {
			return nil, fmt.Errorf("adUnits[%d] inventory is inactive, unapproved, stale, or does not match site/slot/size: site_id=%d slot_id=%d size_id=%d", i, siteID, slotID, sizeID)
		}
		if (platform == "browser" && siteType != acl.SiteTypeWeb) || (platform == "sdk" && siteType != acl.SiteTypeAPP) {
			return nil, fmt.Errorf("adUnits[%d] platform %q does not match configured site type", i, platform)
		}
		w, h := match.SizeID1To2(sizeID)
		if w == 0 || h == 0 {
			return nil, fmt.Errorf("adUnits[%d] configured slot has empty size", i)
		}
		if err := adUnit.EffectiveMediaTypes().ValidateForSize(w, h); err != nil {
			return nil, fmt.Errorf("adUnits[%d] %w", i, err)
		}
		units = append(units, SSPValidatedUnit{
			Code:            adUnit.Code,
			Site:            string(self.Site),
			Slot:            slotToken,
			SiteStr:         siteStr,
			SlotStr:         slotStr,
			SiteType:        siteType,
			ConfiguredFloor: configuredFloor,
			RPub: match.RPub{
				PubID:  pubID,
				SiteID: siteID,
				SlotID: slotID,
				SizeID: sizeID,
			},
		})
	}
	return units, nil
}
