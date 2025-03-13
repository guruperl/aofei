package match

type AdUnit struct {
	Slot       string                 `json:"slot"`
	Code       string                 `json:"code"`
	MediaTypes map[string]interface{} `json:"mediaTypes"`
}

type Incoming struct {
	Platform string    `json:"platform"`
	Site     string    `json:"site"`
	AdUnits  []*AdUnit `json:"adUnits"`
}

/*
func (incoming *Incoming) Unpack() ([]*AdImp, error) {
	pubID, siteID, err := pzutil.UnpackTwo(incoming.Site)
	if err != nil {
		return nil, err
	}

	i2sizes := func(value interface{}) []uint16 {
		size := make([]uint16, 0)
		for _, val := range value.([]interface{}) {
			size = append(size, uint16(val.(float64)))
		}
		return size
	}

	adImps := make([]*AdImp, len(incoming.AdUnits))

	for i, adunit := range incoming.AdUnits {
		slotID, sizeID, err := pzutil.UnpackTwo(adunit.Slot)
		if err != nil {
			return nil, err
		}

		var nt *NativeType
		var bt *BannerType
		var vt *VideoType

		for k, v := range adunit.MediaTypes {
			switch k {
			case "banner":
				bt = &BannerType{}
				for key, value := range v.(map[string]interface{}) {
					if key == "size" {
						bt.Size = i2sizes(value)
					}
				}
			case "native":
				nt = &NativeType{}
				for key, value := range v.(map[string]interface{}) {
					if key == "image" {
						nt.Image = i2sizes(value)
					}
					if key == "icon" {
						nt.Image = i2sizes(value)
					}
					if key == "sponsoredBy" {
						nt.SponsoredBy = value.(bool)
					}
					if key == "title" {
						nt.Title = value.(bool)
					}
					if key == "body" {
						nt.Body = value.(bool)
					}
				}
			case "video":
				vt = &VideoType{}
				for key, value := range v.(map[string]interface{}) {
					if key == "playerSize" {
						vt.PlayerSize = i2sizes(value)
					}
					if key == "context" {
						vt.Context = value.(string)
					}
				}
			default:
			}
		}
		if nt == nil && bt == nil && vt == nil {
			nt = &NativeType{}
			bt = &BannerType{}
			vt = &VideoType{}
		}
		adImps[i] = &AdImp{RPub{pubID, siteID, slotID, sizeID}, nt, bt, vt}
	}

	return adImps, nil
}
*/
