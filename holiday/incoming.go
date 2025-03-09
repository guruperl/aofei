package holiday

import (
	//"log"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	uadevice "github.com/genelet/winter/advice"

	ipsearch "github.com/genelet/winter/maxmind"
	adcom1 "github.com/mxmCherry/openrtb/adcom1"
)

var Platform2Sources = map[string][]interface{}{
	"web":     {Browser, SSPServer},
	"mobile":  {MobileNative, SSPServer},
	"email":   {EmailSource, SSPServer},
	"video":   {VideoSource, SSPServer},
	"device":  {DeviceSource, SSPServer},
	"wechat":  {WechatSource, OpenID},
	"browser": {Browser, SSPServer},
}

type Incoming struct {
	ID      string         `json:"id,omitempty"`
	Adunits []*Adunit      `json:"adUnits"`
	CData   string         `json:"cdata,omitempty"`
	User    *User          `json:"user,omitempty"`
	Site    *adcom1.Site   `json:"site,omitempty"`
	App     *adcom1.App    `json:"app,omitempty"`
	Device  *adcom1.Device `json:"device,omitempty"`

	Platform string `json:"platform,omitempty"`
}

func NewIncoming(data []byte) (*Incoming, error) {
	incoming := &Incoming{}
	err := json.Unmarshal(data, incoming)
	if err != nil {
		return nil, err
	}

	pubsite := ""
	if incoming.Site != nil {
		pubsite = incoming.Site.ID
	} else if incoming.App != nil {
		pubsite = incoming.App.ID
	} else {
		return nil, errors.New("App or Site ID missing")
	}
	pub_id, site_id, err := UnpackTwo(pubsite)
	if err != nil {
		return nil, err
	}

	for i, adunit := range incoming.Adunits {
		slot_id, size_id, err := UnpackTwo(adunit.Code)
		if err != nil {
			return nil, err
		}
		incoming.Adunits[i].RPub = RPub{pub_id, site_id, slot_id}
		incoming.Adunits[i].SizeId = size_id
	}

	return incoming, nil
}

func (self *Incoming) UserGeoDeviceStatus(c *Config, ips *ipsearch.IpSearch, r *http.Request) (*Geo, *Device, Status, bool, error) {

	status := Status{RequestType: IMPR}
	switch r.URL.Path {
	case c.Handlers["ssp"]: // browser source and id
		status.Source = Browser
		status.IdSource = SSPServer
	case c.Handlers["api"]: // mobile api or other api for we have platform
		if self.Platform != "" {
			if platform, ok := Platform2Sources[self.Platform]; ok {
				status.Source = platform[0].(T_SOURCE)
				status.IdSource = platform[1].(ID_SOURCE)
			}
		}
	case c.Handlers["tencent"]: // if incoming has androidid or idfa
	case c.Handlers["statis"]:
		return nil, nil, status, true, nil
	default:
		return nil, nil, status, false, errors.New("Wrong request path")
	}

	id_str := ""
	if user := self.User; user != nil {
		status.IsUser = true
		id_str = user.ID
		if id_str == "" && user.BuyerUID != "" {
			id_str = user.BuyerUID
		}
	} else {
		self.User = new(User)
	}

	if id_str == "" && self.CData != "" {
		id_str = self.CData
	}
	if id_str == "" {
		if ucookie, _ := r.Cookie(c.Ucookie); ucookie != nil {
			id_str = ucookie.Value
		}
	}
	if id_str != "" {
		bts, err := hex.DecodeString(id_str)
		if err != nil {
			return nil, nil, status, false, err
		} else if len(bts) != 8 {
			return nil, nil, status, false, errors.New("Wrong uid hex")
		}
		self.User.UserId = Byte64Int(bts)
	}

	geo := &Geo{Ips: ips}
	ua := r.UserAgent()
	device := &Device{UA: ua, PzUA32: uadevice.CreateTwoUa(ua).Pack()}
	if self.Device != nil {
		status.IsDevice = true
		device.IncomingDevice = self.Device
		if self.Device.Geo != nil {
			geo.DeviceGeo = self.Device.Geo
		}
		if self.Device.IP != "" {
			status.IsIp = true
			geo.IPString = self.Device.IP
		}
	}
	if geo.IPString == "" {
		geo.IPString = get_ip(r)
	}

	return geo, device, status, false, nil
}

func get_ip(r *http.Request) string {
	xf := r.Header.Get("X-Forwarded-For")
	if xf == "" {
		xf = r.Header.Get("X-Real-IP")
	}
	if xf == "" {
		ip_str, _, _ := net.SplitHostPort(r.RemoteAddr)
		return ip_str
	}

	strs := strings.SplitN(xf, ",", -1)
	return strs[0]
}

/*

type Publisher struct {
    ID  string `json:"id,omitempty"`
    Name string `json:"name,omitempty"`
    Domain string `json:"domain,omitempty"`
    Cat []string `json:"cat,omitempty"`
    CatTax CategoryTaxonomy `json:"cattax,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}

type ContentContext int8
const (
    ContentVideo   ContentContext = 1 // 1 Video (i.e., video file or stream such as Internet TV broadcasts)
    ContentGame    ContentContext = 2 // 2 Game (i.e., an interactive software game)
    ContentMusic   ContentContext = 3 // 3 Music (i.e., audio file or stream such as Internet radio broadcasts)
    ContentApp     ContentContext = 4 // 4 Application (i.e., an interactive software application)
    ContentText    ContentContext = 5 // 5 Text (i.e., primarily textual document such as a web page, eBook, or news article)
    ContentOther   ContentContext = 6 // 6 Other (i.e., none of the other categories applies)
    ContentUnknown ContentContext = 7 // 7 Unknown
)

type MediaRating int8
const (
    MediaRatingAll    MediaRating = 1 // All Audiences
    MediaRatingOver12 MediaRating = 2 // Everyone Over Age 12
    MediaRatingMature MediaRating = 3 // Mature Audiences
)

type Content struct {
    ID  string `json:"id,omitempty"`
    Episode int64 `json:"episode,omitempty"`
    Title string `json:"title,omitempty"`
    Series string `json:"series,omitempty"`
    Season string `json:"season,omitempty"`
    Artist string `json:"artist,omitempty"`
    Genre string `json:"genre,omitempty"`
    Album string `json:"album,omitempty"`
    ISRC string `json:"isrc,omitempty"`
    URL string `json:"url,omitempty"`
    Cat []string `json:"cat,omitempty"`
    CatTax int8 `json:"cattax,omitempty"`
    ProdQ int8 `json:"prodq,omitempty"`
    Context ContentContext `json:"context,omitempty"`
    Rating string `json:"rating,omitempty"`
    URating string `json:"urating,omitempty"`
    MRating MediaRating `json:"mrating,omitempty"`
    Keywords string `json:"keywords,omitempty"`
    Live int8 `json:"live,omitempty"`
    SrcRel int8 `json:"srcrel,omitempty"`
    Len int64 `json:"len,omitempty"`
    Lang string `json:"lang,omitempty"`
    Embed int8 `json:"embed,omitempty"`
    Producer string `json:"producer,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}

type DistributionChannel struct {
	ID  string `json:"id,omitempty"`
    Name string `json:"name,omitempty"`
    Pub *Publisher `json:"pub,omitempty"`
    Content *Content `json:"content,omitempty"`
}

type CategoryTaxonomy int
const (
    CatTaxIABContent0 CategoryTaxonomy = 0 // Other, by this system
    CatTaxIABContent1 CategoryTaxonomy = 1 // 1	IAB Content Category Taxonomy 1.0.
    CatTaxIABContent2 CategoryTaxonomy = 2 // 2	IAB Content Category Taxonomy 2.0: www.iab.com/guidelines/taxonomy
    CatTaxIABProduct1 CategoryTaxonomy = 3 // 3	IAB Ad Product Taxonomy 1.0.
)

type App struct {
	DistributionChannel
	Domain string `json:"domain,omitempty"`
	//   Array of content categories describing the app
    Cat []string `json:"cat,omitempty"`
	//   Array of content categories describing the current section of the app
    SectCat []string `json:"sectcat,omitempty"`
	//   Array of content categories describing the current page/view of the app
    PageCat []string `json:"pagecat,omitempty"`
    CatTax CategoryTaxonomy `json:"cattax,omitempty"`
	//   Indicates if the app has a privacy policy, where 0 = no, 1 = yes
    PrivPolicy int8 `json:"privpolicy,omitempty"`
	//   Comma separated list of keywords about the app.
    Keywords string `json:"keywords,omitempty"`
		//   Bundle or package name of the app (e.g., “com.foo.mygame”) and should NOT be app store ID s (e.g., not iTunes store IDs).
		Bundle string `json:"bundle,omitempty"`
		//   The ID of the app in an app store (e.g. Apple iTunes, Google Play).
		StoreID string `json:"storeid,omitempty"`
		//   App store URL for an installed app; for IQG 2.1 compliance.
		StoreURL string `json:"storeurl,omitempty"`
		//   Application version.
		Ver string `json:"ver,omitempty"`
		//   Indicator of whether or not this is a paid app, 0 = free, 1 = paid.
		Paid int8 `json:"paid,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}

type Site struct {
	DistributionChannel
	Domain string `json:"domain,omitempty"`
	//   Array of content categories describing the app
    Cat []string `json:"cat,omitempty"`
	//   Array of content categories describing the current section of the app
    SectCat []string `json:"sectcat,omitempty"`
	//   Array of content categories describing the current page/view of the app
    PageCat []string `json:"pagecat,omitempty"`
    CatTax CategoryTaxonomy `json:"cattax,omitempty"`
	//   Indicates if the app has a privacy policy, where 0 = no, 1 = yes
    PrivPolicy int8 `json:"privpolicy,omitempty"`
	//   Comma separated list of keywords about the app.
    Keywords string `json:"keywords,omitempty"`
		//   URL of the page within the site.
		Page string `json:"page,omitempty"`
		//   Referrer URL that caused navigation to the current page.
		Ref string `json:"ref,omitempty"`
		//   Search string that caused navigation to the current page.
		Search string `json:"search,omitempty"`
		//   Indicates if the site has been programmed to optimize layout when viewed on mobile devices, where 0 = no, 1 = yes.
		Mobile int8 `json:"mobile,omitempty"`
		//   Indicates if the page is AMP HTML, where 0 = no, 1 = yes.
		AMP int8 `json:"amp,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}
*/
