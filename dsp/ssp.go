package dsp

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
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
	ID       string       `json:"id,omitempty"`
	Platform string       `json:"platform,omitempty"`
	Site     SSPSiteToken `json:"site"`
	AdUnits  []SSPAdUnit  `json:"adUnits"`
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
	Code    string
	Site    string
	Slot    string
	SiteStr string
	SlotStr string
	RPub    match.RPub
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
	return req, nil
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
	if self.Banner == nil && self.Video == nil && self.Native == nil {
		return fmt.Errorf("missing supported mediaTypes")
	}
	return nil
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

	units := make([]SSPValidatedUnit, 0, len(self.AdUnits))
	for i, adUnit := range self.AdUnits {
		slotToken := adUnit.Slot
		if slotToken == "" {
			return nil, fmt.Errorf("adUnits[%d] missing slot", i)
		}
		slotID, sizeID, err := acl.UnpackDirectToken(slotToken)
		if err != nil {
			return nil, fmt.Errorf("adUnits[%d] invalid slot token: %w", i, err)
		}
		siteStr, slotStr, ok := pub.Validate(siteID, slotID)
		if !ok {
			return nil, fmt.Errorf("adUnits[%d] invalid site/slot pair: site_id=%d slot_id=%d", i, siteID, slotID)
		}
		units = append(units, SSPValidatedUnit{
			Code:    adUnit.Code,
			Site:    string(self.Site),
			Slot:    slotToken,
			SiteStr: siteStr,
			SlotStr: slotStr,
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
