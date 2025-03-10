package holiday

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	adcom1 "github.com/prebid/openrtb/v20/adcom1"
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

type SeatBase struct {
	user    *User
	c       *Config
	current time.Time
	adunit  *Adunit
}

type Seat struct {
	SeatBase
	imp_id   int64
	raw_id   int64
	item     *Item
	creative *Creative
}

func (self *Seat) MakeClickBare() string {
	rawimp := &rawImp{self.imp_id, self.raw_id, self.user.UserId, self.item.RAdv, self.adunit.RPub}
	packed, _ := PackFixedURL(rawimp)
	return self.c.ServerURL + self.c.Handlers["click"] + "/" + packed
}

func (self *Seat) MakeClick() string {
	escaped := url.PathEscape(url.QueryEscape(self.creative.Click))
	return self.MakeClickBare() + "." + escaped
}

func (self *Seat) MakeLinkAsset() *adcom1.LinkAsset {
	click_url := self.MakeClick()
	return &adcom1.LinkAsset{
		URL:   click_url,
		URLFB: click_url,
		Trkr:  []string{self.MakeClickBare()}}
}

func (self *Seat) MakeBanner() *adcom1.Banner {
	//clickurl := self.MakeClick()
	return &adcom1.Banner{
		Img:  self.creative.Content,
		Link: self.MakeLinkAsset()}
}

func (self *Seat) MakeBannerAdM() string {
	clickurl := self.MakeClick()
	return `<a href='` + clickurl + `'><img src='` + self.c.ServerURL + self.creative.Content + `></a>`
}

func (self *Seat) SeatBid(status Status) openrtb2.SeatBid {
	click := self.MakeClick()
	content := strings.Replace(self.creative.Content, "LANDING", click, -1)
	w, h := GetSizes(self.adunit.SizeId)

	ad_id := fmt.Sprintf("ITEM%d", self.item.ItemId)
	bidmedia := adcom1.BidMedia{
		Ad: &adcom1.Ad{
			ID:      ad_id,
			ADomain: []string{"pzcom.com"},
			IURL:    click,
			Init:    self.current.UnixNano(),
			Display: &adcom1.Display{
				MIME:   "image/png",
				CType:  adcom1.CreativeImage,
				W:      int64(w),
				H:      int64(h),
				AdM:    content,
				Priv:   "http://www.pzcom.com/psa.html",
				Banner: self.MakeBanner(),
			},
		},
	}
	json_bts, _ := json.Marshal(bidmedia)
	return openrtb2.SeatBid{
		Seat:    fmt.Sprintf("ADV%d", self.item.AdvId),
		Package: int8(0),
		Bid: []openrtb2.Bid{
			{
				ID:    fmt.Sprintf("CAMPAIGN%d", self.item.CampaignId),
				Price: float64(self.item.Price),
				PURL:  self.creative.Click,
				BURL:  self.creative.Click,
				LURL:  self.creative.Click,
				Exp:   10,
				MID:   ad_id,
				Media: json_bts,
			},
		},
	}

}

func (self *SeatBase) PSABid() openrtb2.SeatBid {
	size_id := self.adunit.SizeID()
	psa, ok := self.c.PSAs[size_id]
	if !ok {
		psa = self.c.PSAs[0]
	}
	ad_id := fmt.Sprintf("PSA_AD%d", size_id)
	bidmedia := adcom1.BidMedia{
		Ad: &adcom1.Ad{
			ID:      ad_id,
			ADomain: []string{"pzcom.com"},
			IURL:    psa.Display,
			Init:    self.current.UnixNano(),
			Display: &adcom1.Display{
				MIME:  "image/png",
				CType: adcom1.CreativeImage,
				W:     int64(psa.W),
				H:     int64(psa.H),
				AdM:   `<a href='` + psa.Click + `'><img src='` + psa.Display + `></a>`,
				Priv:  "http://www.pzcom.com/psa.html",
				Banner: &adcom1.Banner{
					Img: psa.Display,
					Link: &adcom1.LinkAsset{
						URL:   psa.Click,
						URLFB: psa.Click,
						Trkr:  []string{psa.Click},
					},
				},
			},
		},
	}
	json_bts, _ := json.Marshal(bidmedia)
	//if err != nil { panic(err) }
	return openrtb2.SeatBid{
		Seat:    fmt.Sprintf("PSA%d", size_id),
		Package: int8(0),
		Bid: []openrtb2.Bid{
			{
				ID:    fmt.Sprintf("PSA_BID%d", size_id),
				Price: float64(psa.Price),
				PURL:  psa.Click,
				BURL:  psa.Click,
				LURL:  psa.Click,
				Exp:   10,
				MID:   ad_id,
				Media: json_bts,
			},
		},
	}
}
