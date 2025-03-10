package holiday

import (
	"fmt"
	"time"

	adcom1 "github.com/mxmCherry/openrtb/adcom1"
)

type User struct {
	adcom1.User
	UserId int64
	Tags   map[string]*Tags
}

func (self *User) UpdateTags(device *Device, geo *Geo, current time.Time, refMap map[string]*TagMap) {
	userTagMap := make(map[string]*Tags)
	if self.User.Data != nil {
		for _, data := range self.User.Data {
			provider := data.ID
			if tagMap, ok := refMap[provider]; ok {
				if self.Tags == nil {
					self.Tags = make(map[string]*Tags)
				}
				userTagMap[provider] = tagMap.GetTagsFromCodes(data.Segment[0].Value)
			}
		}
	}
	userTagMap["device"] = device.GetTags()
	userTagMap["geo"] = geo.GetTags()
	dayhour := &Dayhour{current}
	userTagMap["dayhour"] = dayhour.GetTags()
	if self.YOB == 0 || self.Gender != "" {
		demo := new(Demo)
		if self.YOB != 0 {
			demo.YOB = self.YOB
		}
		if self.Gender != "" {
			demo.GENDER = self.Gender
		}
		userTagMap["demo"] = demo.GetTags()
	}
	self.Tags = userTagMap
	return
}

func (self *User) MergeTags() *Tags {
	hash := make(map[uint32][]uint32)
	for _, tags := range self.Tags {
		if tags != nil && tags.TagHashArray != nil {
			for attrID, value_ids := range tags.TagHashArray {
				if _, ok := hash[attrID]; !ok {
					hash[attrID] = make([]uint32, 0)
				}
				hash[attrID] = append(hash[attrID], value_ids...)
			}
		}
	}
	return &Tags{hash}
}

func (self *User) Top10Tags() map[string]interface{} {
	args := make(map[string]interface{})
	k := 0
	for provider, tags := range self.Tags {
		if IndexString([]string{"device", "geo", "dayhour"}, provider) >= 0 {
			continue
		}
		for attrname_id, values := range tags.TagHashArray {
			for _, value_id := range values {
				if k >= 10 {
					return args
				}
				// we shrink to use only uint16
				args[fmt.Sprintf("tag%d", k)] = getSizeID(uint16(attrname_id), uint16(value_id))
				k++
			}
		}
	}
	return args
}

/*
type Segment struct {
    //   ID of the data segment specific to the data provider.
    ID  string `json:"id,omitempty"`
    //   Displayable name of the data segment specific to the data provider.
    Name string `json:"name,omitempty"`
    //   String representation of the data segment value.
    Value string `json:"value,omitempty"`
    //   Optional vendor-specific extensions.
    Ext json.RawMessage `json:"ext,omitempty"`
}

type Data struct {
    //   Vendor-specific ID for the data provider.
    ID  string `json:"id,omitempty"`
    //   Vendor-specific displayable name for the data provider.
    Name string `json:"name,omitempty"`
    Segment []Segment `json:"segment,omitempty"`
    //   Optional vendor-specific extensions.
    Ext json.RawMessage `json:"ext,omitempty"`
}

type User struct {
    //   Vendor-specific ID for the user.
    ID  string `json:"id,omitempty"`
    //   Buyer-specific ID for the user as mapped by an exchange for the buyer.
    BuyerUID string `json:"buyeruid,omitempty"`
    //   Year of birth as a 4-digit integer.
    YOB int64 `json:"yob,omitempty"`
    //   Gender, where “M” = male, “F” = female, “O” = known to be other
    Gender string `json:"gender,omitempty"`
    //   Comma separated list of keywords, interests, or intent.
    Keywords string `json:"keywords,omitempty"`
    //   GDPR consent string if applicable
    Consent string `json:"consent,omitempty"`
    //   Location of the user's home base (i.e., not necessarily their current location).
    Geo *Geo `json:"geo,omitempty"`
    Data []Data `json:"data,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}
*/
