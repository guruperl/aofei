package summer

import (
	"net/url"
	"strconv"
)

type Campaign struct {
	Content   uint32
	Visual    uint32
	Act       uint32
	Download  uint32
	Speed     uint32
	Postclick uint32
}

func GetCampaignAttrs() map[string]string {
	return map[string]string{"c_content": "Content", "c_visual": "Visual", "c_act": "Act", "c_download": "Download", "c_speed": "Speed", "c_postclick": "Postclick"}
}

func (self *Campaign) InHash() map[string]uint32 {
	return map[string]uint32{
		"c_content": self.Content, "c_visual": self.Visual, "c_act": self.Act, "c_download": self.Download, "c_speed": self.Speed, "c_postclick": self.Postclick}
}

func GetCampaignNames() map[string]map[uint32]string {
	return map[string]map[uint32]string{
		"c_content": CampaignContentName, "c_visual": CampaignVisualName, "c_act": CampaignActName, "c_download": CampaignDownloadName, "c_speed": CampaignSpeedName, "c_postclick": CampaignPostclickName}
}

func GetCampaignScoreArgs(ARGS url.Values) uint32 {
	v := Map([]string{"c_content", "c_visual", "c_act", "c_download", "c_speed", "c_postclick"}, ARGS.Get)
	camp := CreateCampaign(v[0], v[1], v[2], v[3], v[4], v[5])
	return camp.Pack()
}

/*
func SetCampaignScoreArgs(num uint32, ARGS url.Values) {
    C_attrs := GetCampaignAttrs()
    camp := UnpackCampaign(num)
	for i, name := range camp.ToNames() {
		ARGS.Set(C_attrs[i], name)
    }
}

func SetCampaignScoreItem(num uint32, item map[string]interface{}) {
    C_attrs := GetCampaignAttrs()
    camp := UnpackCampaign(num)
    for i, name := range camp.ToNames() {
        item[C_attrs[i]] = name
    }
}
*/

type CampaignVisual uint32

const (
	CampaignVisualUgly CampaignVisual = 2 + iota
	CampaignVisualPoor
	CampaignVisualNormal
	CampaignVisualGood
	CampaignVisualExcellent
)

var CampaignVisualName = map[uint32]string{
	4: "Normal",
	6: "Excellent",
	5: "Good",
	3: "Poor/Negative etc.",
	2: "Ugly/Blank/Body etc.",
}
var CampaignVisualScore = map[uint32]float32{
	4: 0.0,
	6: 2.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
}
var CampaignVisualValue = map[string]uint32{
	"Normal":    4,
	"Excellent": 6,
	"Good":      5,
	"Poor":      3,
	"Ugly":      2,
}

type CampaignSpeed uint32

const (
	CampaignSpeedSlow CampaignSpeed = 3 + iota
	CampaignSpeedNormal
)

var CampaignSpeedName = map[uint32]string{
	4: "Normal",
	3: "Slow",
}
var CampaignSpeedScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
}
var CampaignSpeedValue = map[string]uint32{
	"Normal": 4,
	"Slow":   3,
}

type CampaignAct uint32

const (
	CampaignActAudio CampaignAct = 2 + iota
	CampaignActExpand
	CampaignActNormal
)

var CampaignActName = map[uint32]string{
	4: "Normal, No act",
	3: "Expand/Popup etc.",
	2: "Audio/Download etc.",
}
var CampaignActScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
	2: -2.0,
}
var CampaignActValue = map[string]uint32{
	"Normal": 4,
	"Expand": 3,
	"Audio":  2,
}

type CampaignDownload uint32

const (
	CampaignDownloadExecutable CampaignDownload = 1 + iota
	CampaignDownloadSoftware
	CampaignDownloadDocument
	CampaignDownloadNormal
)

var CampaignDownloadName = map[uint32]string{
	4: "Normal, No download",
	3: "Paper/Document etc.",
	2: "Wallpaper/Software etc.",
	1: "Executable",
}
var CampaignDownloadScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
	2: -5.0,
	1: -10.0,
}
var CampaignDownloadValue = map[string]uint32{
	"Normal":     4,
	"Document":   3,
	"Sofware":    2,
	"Executable": 1,
}

type CampaignContent uint32

const (
	CampaignContentDeceptive CampaignContent = 1 + iota
	CampaignContentProvocative
	CampaignContentDating
	CampaignContentNormal
	CampaignContentBrand
	CampaignContentTopBrand
)

var CampaignContentName = map[uint32]string{
	4: "Normal",
	6: "Top Brand",
	5: "Brand",
	3: "Dating etc.",
	2: "Provocative/Puzzle/Casino etc.",
	1: "Deceptive",
}
var CampaignContentScore = map[uint32]float32{
	4: 0.0,
	6: 2.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
	1: -5.0,
}
var CampaignContentValue = map[string]uint32{
	"Normal":      4,
	"Top":         6,
	"Brand":       5,
	"Dating":      3,
	"Provocative": 2,
	"Deceptive":   1,
}

type CampaignPostclick uint32

const (
	CampaignPostclickWrong CampaignPostclick = 2 + iota
	CampaignPostclickPoor
	CampaignPostclickNormal
	CampaignPostclickGood
)

var CampaignPostclickName = map[uint32]string{
	4: "Normal",
	5: "Good Looking Site",
	3: "Poor/Wrong Site",
	2: "Broken/Hangup",
}
var CampaignPostclickScore = map[uint32]float32{
	4: 0.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
}
var CampaignPostclickValue = map[string]uint32{
	"Normal": 4,
	"Good":   5,
	"Poor":   3,
	"Broken": 2,
}

func CreateCampaign(content, visual, act, download, speed, postclick string) *Campaign {
	campaign := &Campaign{4, 4, 4, 4, 4, 4}
	if content != "" {
		if v, err := strconv.Atoi(content); err == nil {
			campaign.Content = uint32(v)
		}
	}
	if visual != "" {
		if v, err := strconv.Atoi(visual); err == nil {
			campaign.Visual = uint32(v)
		}
	}
	if act != "" {
		if v, err := strconv.Atoi(act); err == nil {
			campaign.Act = uint32(v)
		}
	}
	if download != "" {
		if v, err := strconv.Atoi(download); err == nil {
			campaign.Download = uint32(v)
		}
	}
	if speed != "" {
		if v, err := strconv.Atoi(speed); err == nil {
			campaign.Speed = uint32(v)
		}
	}
	if postclick != "" {
		if v, err := strconv.Atoi(postclick); err == nil {
			campaign.Postclick = uint32(v)
		}
	}
	return campaign
}

func (self *Campaign) Pack() uint32 {
	if self.Content >= 8 {
		self.Content = 4
	}
	if self.Visual >= 8 {
		self.Visual = 4
	}
	if self.Act >= 8 {
		self.Act = 4
	}
	if self.Download >= 8 {
		self.Download = 4
	}
	if self.Speed >= 8 {
		self.Speed = 4
	}
	if self.Postclick >= 8 {
		self.Postclick = 4
	}

	return ((self.Content & 7) << 0) +
		((self.Visual & 7) << 3) +
		((self.Act & 7) << 6) +
		((self.Download & 7) << 9) +
		((self.Speed & 7) << 12) +
		((self.Postclick & 7) << 15)
}

func UnpackCampaign(campaign uint32) *Campaign {
	a := campaign & 7
	b := (campaign >> 3) & 7
	c := (campaign >> 6) & 7
	d := (campaign >> 9) & 7
	e := (campaign >> 12) & 7
	f := (campaign >> 15) & 7
	return &Campaign{a, b, c, d, e, f}
}

func (self *Campaign) ToNames() []string {
	return []string{CampaignContentName[self.Content], CampaignVisualName[self.Visual], CampaignActName[self.Act], CampaignDownloadName[self.Download], CampaignSpeedName[self.Speed], CampaignPostclickName[self.Postclick]}
}

func (self *Campaign) TotalScore() float32 {
	return self.ContentScore() +
		self.VisualScore() +
		self.ActScore() +
		self.DownloadScore() +
		self.SpeedScore() +
		self.PostclickScore()
}

func (self *Campaign) ContentScore() float32 {
	return CampaignContentScore[self.Content]
}

func (self *Campaign) VisualScore() float32 {
	return CampaignVisualScore[self.Visual]
}

func (self *Campaign) ActScore() float32 {
	return CampaignActScore[self.Act]
}

func (self *Campaign) DownloadScore() float32 {
	return CampaignDownloadScore[self.Download]
}

func (self *Campaign) SpeedScore() float32 {
	return CampaignSpeedScore[self.Speed]
}

func (self *Campaign) PostclickScore() float32 {
	return CampaignPostclickScore[self.Postclick]
}
