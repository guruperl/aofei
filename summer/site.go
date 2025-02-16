package summer

import (
	"net/url"
	"strconv"
)

type Site struct {
	Internet uint32
	World    uint32
	Local    uint32
	Domain   uint32
	Age      uint32
	Visual   uint32
	Popup    uint32
	Crowd    uint32
	Traffic  uint32
	Source   uint32
	Control  uint32
}

func GetSiteAttrs() map[string]string {
	return map[string]string{
		"s_internet": "Internet", "s_world": "World", "s_local": "Local", "s_domain": "Domain", "s_age": "Age", "s_visual": "Visual", "s_popup": "Popup", "s_crowd": "Crowd", "s_traffic": "Traffic", "s_source": "Source", "s_control": "Control"}
}

func (self *Site) InHash() map[string]uint32 {
	return map[string]uint32{
		"s_internet": self.Internet, "s_world": self.World, "s_local": self.Local, "s_domain": self.Domain, "s_age": self.Age, "s_visual": self.Visual, "s_popup": self.Popup, "s_crowd": self.Crowd, "s_traffic": self.Traffic, "s_source": self.Source, "s_control": self.Control}
}

func GetSiteNames() map[string]map[uint32]string {
	return map[string]map[uint32]string{
		"s_internet": SiteBrandName, "s_world": SiteBrandName, "s_local": SiteBrandName, "s_domain": SiteDomainName, "s_age": SiteAgeName, "s_visual": SiteVisualName, "s_popup": SitePopupName, "s_crowd": SiteCrowdName, "s_traffic": SiteTrafficName, "s_source": SiteSourceName, "s_control": SiteControlName}
}

func GetSiteScoreArgs(ARGS url.Values) uint32 {
	v := Map([]string{"s_internet", "s_world", "s_local", "s_domain", "s_age", "s_visual", "s_popup", "s_crowd", "s_traffic", "s_source", "s_control"}, ARGS.Get)
	site := CreateSite(v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10])
	return site.Pack()
}

/*
func SetSiteScoreArgs(num uint32, ARGS url.Values) {
	S_attrs := GetSiteAttrs()
	site := UnpackSite(num)
	for i, name := range site.ToNames() {
		ARGS.Set(S_attrs[i], name)
	}
}

func SetSiteScoreItem(num uint32, item map[string]interface{}) {
	S_attrs := GetSiteAttrs()
	site := UnpackSite(num)
	for i, name := range site.ToNames() {
		item[S_attrs[i]] = name
	}
}
*/

type SiteBrand uint32

const (
	SiteBrandUnknown SiteBrand = iota
	SiteBrandNormal
	SiteBrandSometimes
	SiteBrandFamous
)

var SiteBrandName = map[uint32]string{
	1: "Normal",
	3: "Famous",
	2: "Sometimes",
	0: "Unknow/New",
}
var SiteBrandScore = map[uint32]float32{
	1: 0.0,
	3: 10.0,
	2: 2.0,
	0: -1.0,
}
var SiteBrandValue = map[string]uint32{
	"Normal":    1,
	"Famous":    3,
	"Sometimes": 2,
	"Unknown":   0,
}

type SiteDomain uint32

const (
	SiteDomainSubpoor SiteDomain = iota
	SiteDomainPoordomain
	SiteDomainNormal
	SiteDomainTopdomain
)

var SiteDomainName = map[uint32]string{
	2: "Normal Domain Name",
	3: "Top/Short Name",
	1: "Poor Name",
	0: "Sub of Poor Domain Name",
}
var SiteDomainScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -2.0,
	0: -3.0,
}
var SiteDomainValue = map[string]uint32{
	"Normal":     2,
	"Topdomain":  3,
	"Poordomain": 1,
	"Subpoor":    0,
}

type SiteAge uint32

const (
	SiteAge1Year SiteAge = iota
	SiteAgeNormal
	SiteAge10Years
	SiteAge20Years
)

var SiteAgeName = map[uint32]string{
	1: "Normal, 1-10 Years",
	2: "10-20 Years",
	3: "20 or more Years",
	0: "Less 1 Year",
}
var SiteAgeScore = map[uint32]float32{
	1: 0.0,
	2: 1.0,
	3: 2.0,
	0: -1.0,
}
var SiteAgeValue = map[string]uint32{
	"Normal":  1,
	"10Years": 2,
	"20Years": 3,
	"1Year":   0,
}

type SiteVisual uint32

const (
	SiteVisualUgly SiteVisual = iota
	SiteVisualPoor
	SiteVisualNormal
	SiteVisualGood
)

var SiteVisualName = map[uint32]string{
	2: "Normal",
	3: "Good",
	1: "Poor/Negative etc.",
	0: "Ugly/Blank/Body etc.",
}
var SiteVisualScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -1.0,
	0: -2.0,
}
var SiteVisualValue = map[string]uint32{
	"Normal": 2,
	"Good":   3,
	"Poor":   1,
	"Ugly":   0,
}

type SitePopup uint32

const (
	SitePopup5Popups SitePopup = iota
	SitePopup2Popups
	SitePopup1Popups
	SitePopupNormal
)

var SitePopupName = map[uint32]string{
	3: "Normal",
	2: "1 Popups",
	1: "2-4 Popups",
	0: "5 or more popups",
}
var SitePopupScore = map[uint32]float32{
	3: 0.0,
	2: -1.0,
	1: -2.0,
	0: -5.0,
}
var SitePopupValue = map[string]uint32{
	"Normal":  3,
	"1Popups": 2,
	"2Popups": 1,
	"5Popups": 0,
}

type SiteCrowd uint32

const (
	SiteCrowd10Ads SiteCrowd = iota
	SiteCrowd5Ads
	SiteCrowdNormal
	SiteCrowdClean
)

var SiteCrowdName = map[uint32]string{
	2: "Normal 1-5 per page",
	3: "1 or no ad",
	1: "5-10 ads",
	0: "10 or more ads",
}
var SiteCrowdScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -2.0,
	0: -4.0,
}
var SiteCrowdValue = map[string]uint32{
	"Normal": 2,
	"Clean":  3,
	"5Ads":   1,
	"10Ads":  0,
}

type SiteTraffic uint32

const (
	SiteTrafficPoor SiteTraffic = iota
	SiteTrafficNormal
	SiteTrafficGood
	SiteTrafficExcellent
)

var SiteTrafficName = map[uint32]string{
	1: "Normal Traffic",
	3: "Excellent",
	2: "Good",
	0: "Poor",
}
var SiteTrafficScore = map[uint32]float32{
	1: 0.0,
	3: 2.0,
	2: 1.0,
	0: -2.0,
}
var SiteTrafficValue = map[string]uint32{
	"Normal":    1,
	"Excellent": 3,
	"Good":      2,
	"Poor":      0,
}

type SiteSource uint32

const (
	SiteSourceSpiderware SiteSource = iota
	SiteSourceProxy
	SiteSourceHijack
	SiteSourceNormal
)

var SiteSourceName = map[uint32]string{
	3: "Normal Site/App",
	2: "Hijack/Plugin",
	1: "Proxy",
	0: "Spiderware",
}
var SiteSourceScore = map[uint32]float32{
	3: 0.0,
	2: -1.0,
	1: -2.0,
	0: -10.0,
}
var SiteSourceValue = map[string]uint32{
	"Normal":     3,
	"Hijack":     2,
	"Proxy":      1,
	"Spiderware": 0,
}

type SiteControl uint32

const (
	SiteControlUser SiteControl = iota
	SiteControlCopied
	SiteControlNormal
)

var SiteControlName = map[uint32]string{
	2: "Normal Managed",
	1: "Copied Site",
	0: "No or User Uploaded",
}
var SiteControlScore = map[uint32]float32{
	2: 0.0,
	1: -1.0,
	0: -2.0,
}
var SiteControlValue = map[string]uint32{
	"Normal":   2,
	"Copied":   1,
	"Uploaded": 0,
}

func CreateSite(internet, world, local, domain, age, visual, popup, crowd, traffic, source, control string) *Site {
	site := &Site{1, 1, 1, 2, 1, 2, 3, 2, 1, 3, 2}
	if internet != "" {
		if v, err := strconv.Atoi(internet); err == nil {
			site.Internet = uint32(v)
		}
	}
	if world != "" {
		if v, err := strconv.Atoi(world); err == nil {
			site.World = uint32(v)
		}
	}
	if local != "" {
		if v, err := strconv.Atoi(local); err == nil {
			site.Local = uint32(v)
		}
	}
	if domain != "" {
		if v, err := strconv.Atoi(domain); err == nil {
			site.Domain = uint32(v)
		}
	}
	if age != "" {
		if v, err := strconv.Atoi(age); err == nil {
			site.Age = uint32(v)
		}
	}
	if visual != "" {
		if v, err := strconv.Atoi(visual); err == nil {
			site.Visual = uint32(v)
		}
	}
	if popup != "" {
		if v, err := strconv.Atoi(popup); err == nil {
			site.Popup = uint32(v)
		}
	}
	if crowd != "" {
		if v, err := strconv.Atoi(crowd); err == nil {
			site.Crowd = uint32(v)
		}
	}
	if traffic != "" {
		if v, err := strconv.Atoi(traffic); err == nil {
			site.Traffic = uint32(v)
		}
	}
	if source != "" {
		if v, err := strconv.Atoi(source); err == nil {
			site.Source = uint32(v)
		}
	}
	if control != "" {
		if v, err := strconv.Atoi(control); err == nil {
			site.Control = uint32(v)
		}
	}

	return site
}

func (self *Site) Pack() uint32 {
	if self.Internet >= 4 {
		self.Internet = 1
	}
	if self.World >= 4 {
		self.World = 1
	}
	if self.Local >= 4 {
		self.Local = 1
	}
	if self.Domain >= 4 {
		self.Domain = 2
	}
	if self.Age >= 4 {
		self.Age = 1
	}
	if self.Visual >= 4 {
		self.Visual = 2
	}
	if self.Popup >= 4 {
		self.Popup = 3
	}
	if self.Crowd >= 4 {
		self.Crowd = 2
	}
	if self.Traffic >= 4 {
		self.Traffic = 1
	}
	if self.Source >= 4 {
		self.Source = 3
	}
	if self.Control >= 4 {
		self.Control = 2
	}

	return ((self.Internet & 3) << 0) +
		((self.World & 3) << 2) +
		((self.Local & 3) << 4) +
		((self.Domain & 3) << 6) +
		((self.Age & 3) << 8) +
		((self.Visual & 3) << 10) +
		((self.Popup & 3) << 12) +
		((self.Crowd & 3) << 14) +
		((self.Traffic & 3) << 16) +
		((self.Source & 3) << 18) +
		((self.Control & 3) << 20)
}

func UnpackSite(site uint32) *Site {
	a := site & 3
	b := (site >> 2) & 3
	c := (site >> 4) & 3
	d := (site >> 6) & 3
	e := (site >> 8) & 3
	f := (site >> 10) & 3
	g := (site >> 12) & 3
	h := (site >> 14) & 3
	i := (site >> 16) & 3
	j := (site >> 18) & 3
	k := (site >> 20) & 3
	return &Site{a, b, c, d, e, f, g, h, i, j, k}
}

func (self *Site) ToNames() []string {
	return []string{SiteBrandName[self.Internet], SiteBrandName[self.World], SiteBrandName[self.Local], SiteDomainName[self.Domain], SiteAgeName[self.Age], SiteVisualName[self.Visual], SitePopupName[self.Popup], SiteCrowdName[self.Crowd], SiteTrafficName[self.Traffic], SiteSourceName[self.Source], SiteControlName[self.Control]}
}

func (self *Site) TotalScore() float32 {
	return self.InternetScore() +
		self.WorldScore() +
		self.LocalScore() +
		self.DomainScore() +
		self.AgeScore() +
		self.VisualScore() +
		self.PopupScore() +
		self.CrowdScore() +
		self.TrafficScore() +
		self.SourceScore() +
		self.ControlScore()
}

func (self *Site) InternetScore() float32 {
	return SiteBrandScore[self.Internet]
}

func (self *Site) WorldScore() float32 {
	return SiteBrandScore[self.World]
}

func (self *Site) LocalScore() float32 {
	return SiteBrandScore[self.Local]
}

func (self *Site) DomainScore() float32 {
	return SiteDomainScore[self.Domain]
}

func (self *Site) AgeScore() float32 {
	return SiteAgeScore[self.Age]
}

func (self *Site) VisualScore() float32 {
	return SiteVisualScore[self.Visual]
}

func (self *Site) PopupScore() float32 {
	return SitePopupScore[self.Popup]
}

func (self *Site) CrowdScore() float32 {
	return SiteCrowdScore[self.Crowd]
}

func (self *Site) TrafficScore() float32 {
	return SiteTrafficScore[self.Traffic]
}

func (self *Site) SourceScore() float32 {
	return SiteSourceScore[self.Source]
}

func (self *Site) ControlScore() float32 {
	return SiteControlScore[self.Control]
}
