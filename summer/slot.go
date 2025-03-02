package summer

import (
	"net/url"
	"strconv"
)

type Slot struct {
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

func GetSlotAttrs() map[string]string {
	return map[string]string{
		"s_internet": "Internet", "s_world": "World", "s_local": "Local", "s_domain": "Domain", "s_age": "Age", "s_visual": "Visual", "s_popup": "Popup", "s_crowd": "Crowd", "s_traffic": "Traffic", "s_source": "Source", "s_control": "Control"}
}

func (self *Slot) InHash() map[string]uint32 {
	return map[string]uint32{
		"s_internet": self.Internet, "s_world": self.World, "s_local": self.Local, "s_domain": self.Domain, "s_age": self.Age, "s_visual": self.Visual, "s_popup": self.Popup, "s_crowd": self.Crowd, "s_traffic": self.Traffic, "s_source": self.Source, "s_control": self.Control}
}

func GetSlotNames() map[string]map[uint32]string {
	return map[string]map[uint32]string{
		"s_internet": SlotBrandName, "s_world": SlotBrandName, "s_local": SlotBrandName, "s_domain": SlotDomainName, "s_age": SlotAgeName, "s_visual": SlotVisualName, "s_popup": SlotPopupName, "s_crowd": SlotCrowdName, "s_traffic": SlotTrafficName, "s_source": SlotSourceName, "s_control": SlotControlName}
}

func GetSlotScoreArgs(ARGS url.Values) uint32 {
	v := Map([]string{"s_internet", "s_world", "s_local", "s_domain", "s_age", "s_visual", "s_popup", "s_crowd", "s_traffic", "s_source", "s_control"}, ARGS.Get)
	site := CreateSlot(v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10])
	return site.Pack()
}

/*
func SetSlotScoreArgs(num uint32, ARGS url.Values) {
	S_attrs := GetSlotAttrs()
	site := UnpackSlot(num)
	for i, name := range site.ToNames() {
		ARGS.Set(S_attrs[i], name)
	}
}

func SetSlotScoreItem(num uint32, item map[string]interface{}) {
	S_attrs := GetSlotAttrs()
	site := UnpackSlot(num)
	for i, name := range site.ToNames() {
		item[S_attrs[i]] = name
	}
}
*/

type SlotBrand uint32

const (
	SlotBrandUnknown SlotBrand = iota
	SlotBrandNormal
	SlotBrandSometimes
	SlotBrandFamous
)

var SlotBrandName = map[uint32]string{
	1: "Normal",
	3: "Famous",
	2: "Sometimes",
	0: "Unknow/New",
}
var SlotBrandScore = map[uint32]float32{
	1: 0.0,
	3: 10.0,
	2: 2.0,
	0: -1.0,
}
var SlotBrandValue = map[string]uint32{
	"Normal":    1,
	"Famous":    3,
	"Sometimes": 2,
	"Unknown":   0,
}

type SlotDomain uint32

const (
	SlotDomainSubpoor SlotDomain = iota
	SlotDomainPoordomain
	SlotDomainNormal
	SlotDomainTopdomain
)

var SlotDomainName = map[uint32]string{
	2: "Normal Domain Name",
	3: "Top/Short Name",
	1: "Poor Name",
	0: "Sub of Poor Domain Name",
}
var SlotDomainScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -2.0,
	0: -3.0,
}
var SlotDomainValue = map[string]uint32{
	"Normal":     2,
	"Topdomain":  3,
	"Poordomain": 1,
	"Subpoor":    0,
}

type SlotAge uint32

const (
	SlotAge1Year SlotAge = iota
	SlotAgeNormal
	SlotAge10Years
	SlotAge20Years
)

var SlotAgeName = map[uint32]string{
	1: "Normal, 1-10 Years",
	2: "10-20 Years",
	3: "20 or more Years",
	0: "Less 1 Year",
}
var SlotAgeScore = map[uint32]float32{
	1: 0.0,
	2: 1.0,
	3: 2.0,
	0: -1.0,
}
var SlotAgeValue = map[string]uint32{
	"Normal":  1,
	"10Years": 2,
	"20Years": 3,
	"1Year":   0,
}

type SlotVisual uint32

const (
	SlotVisualUgly SlotVisual = iota
	SlotVisualPoor
	SlotVisualNormal
	SlotVisualGood
)

var SlotVisualName = map[uint32]string{
	2: "Normal",
	3: "Good",
	1: "Poor/Negative etc.",
	0: "Ugly/Blank/Body etc.",
}
var SlotVisualScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -1.0,
	0: -2.0,
}
var SlotVisualValue = map[string]uint32{
	"Normal": 2,
	"Good":   3,
	"Poor":   1,
	"Ugly":   0,
}

type SlotPopup uint32

const (
	SlotPopup5Popups SlotPopup = iota
	SlotPopup2Popups
	SlotPopup1Popups
	SlotPopupNormal
)

var SlotPopupName = map[uint32]string{
	3: "Normal",
	2: "1 Popups",
	1: "2-4 Popups",
	0: "5 or more popups",
}
var SlotPopupScore = map[uint32]float32{
	3: 0.0,
	2: -1.0,
	1: -2.0,
	0: -5.0,
}
var SlotPopupValue = map[string]uint32{
	"Normal":  3,
	"1Popups": 2,
	"2Popups": 1,
	"5Popups": 0,
}

type SlotCrowd uint32

const (
	SlotCrowd10Ads SlotCrowd = iota
	SlotCrowd5Ads
	SlotCrowdNormal
	SlotCrowdClean
)

var SlotCrowdName = map[uint32]string{
	2: "Normal 1-5 per page",
	3: "1 or no ad",
	1: "5-10 ads",
	0: "10 or more ads",
}
var SlotCrowdScore = map[uint32]float32{
	2: 0.0,
	3: 1.0,
	1: -2.0,
	0: -4.0,
}
var SlotCrowdValue = map[string]uint32{
	"Normal": 2,
	"Clean":  3,
	"5Ads":   1,
	"10Ads":  0,
}

type SlotTraffic uint32

const (
	SlotTrafficPoor SlotTraffic = iota
	SlotTrafficNormal
	SlotTrafficGood
	SlotTrafficExcellent
)

var SlotTrafficName = map[uint32]string{
	1: "Normal Traffic",
	3: "Excellent",
	2: "Good",
	0: "Poor",
}
var SlotTrafficScore = map[uint32]float32{
	1: 0.0,
	3: 2.0,
	2: 1.0,
	0: -2.0,
}
var SlotTrafficValue = map[string]uint32{
	"Normal":    1,
	"Excellent": 3,
	"Good":      2,
	"Poor":      0,
}

type SlotSource uint32

const (
	SlotSourceSpiderware SlotSource = iota
	SlotSourceProxy
	SlotSourceHijack
	SlotSourceNormal
)

var SlotSourceName = map[uint32]string{
	3: "Normal Slot/App",
	2: "Hijack/Plugin",
	1: "Proxy",
	0: "Spiderware",
}
var SlotSourceScore = map[uint32]float32{
	3: 0.0,
	2: -1.0,
	1: -2.0,
	0: -10.0,
}
var SlotSourceValue = map[string]uint32{
	"Normal":     3,
	"Hijack":     2,
	"Proxy":      1,
	"Spiderware": 0,
}

type SlotControl uint32

const (
	SlotControlUser SlotControl = iota
	SlotControlCopied
	SlotControlNormal
)

var SlotControlName = map[uint32]string{
	2: "Normal Managed",
	1: "Copied Slot",
	0: "No or User Uploaded",
}
var SlotControlScore = map[uint32]float32{
	2: 0.0,
	1: -1.0,
	0: -2.0,
}
var SlotControlValue = map[string]uint32{
	"Normal":   2,
	"Copied":   1,
	"Uploaded": 0,
}

func CreateSlot(internet, world, local, domain, age, visual, popup, crowd, traffic, source, control string) *Slot {
	site := &Slot{1, 1, 1, 2, 1, 2, 3, 2, 1, 3, 2}
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

func (self *Slot) Pack() uint32 {
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

func UnpackSlot(site uint32) *Slot {
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
	return &Slot{a, b, c, d, e, f, g, h, i, j, k}
}

func (self *Slot) ToNames() []string {
	return []string{SlotBrandName[self.Internet], SlotBrandName[self.World], SlotBrandName[self.Local], SlotDomainName[self.Domain], SlotAgeName[self.Age], SlotVisualName[self.Visual], SlotPopupName[self.Popup], SlotCrowdName[self.Crowd], SlotTrafficName[self.Traffic], SlotSourceName[self.Source], SlotControlName[self.Control]}
}

func (self *Slot) TotalScore() float32 {
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

func (self *Slot) InternetScore() float32 {
	return SlotBrandScore[self.Internet]
}

func (self *Slot) WorldScore() float32 {
	return SlotBrandScore[self.World]
}

func (self *Slot) LocalScore() float32 {
	return SlotBrandScore[self.Local]
}

func (self *Slot) DomainScore() float32 {
	return SlotDomainScore[self.Domain]
}

func (self *Slot) AgeScore() float32 {
	return SlotAgeScore[self.Age]
}

func (self *Slot) VisualScore() float32 {
	return SlotVisualScore[self.Visual]
}

func (self *Slot) PopupScore() float32 {
	return SlotPopupScore[self.Popup]
}

func (self *Slot) CrowdScore() float32 {
	return SlotCrowdScore[self.Crowd]
}

func (self *Slot) TrafficScore() float32 {
	return SlotTrafficScore[self.Traffic]
}

func (self *Slot) SourceScore() float32 {
	return SlotSourceScore[self.Source]
}

func (self *Slot) ControlScore() float32 {
	return SlotControlScore[self.Control]
}
