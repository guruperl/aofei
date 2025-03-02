package summer

import (
	"net/url"
	"strconv"
)

type Item struct {
	Content   uint32
	Visual    uint32
	Act       uint32
	Download  uint32
	Speed     uint32
	Postclick uint32
}

func GetItemAttrs() map[string]string {
	return map[string]string{"c_content": "Content", "c_visual": "Visual", "c_act": "Act", "c_download": "Download", "c_speed": "Speed", "c_postclick": "Postclick"}
}

func (self *Item) InHash() map[string]uint32 {
	return map[string]uint32{
		"c_content": self.Content, "c_visual": self.Visual, "c_act": self.Act, "c_download": self.Download, "c_speed": self.Speed, "c_postclick": self.Postclick}
}

func GetItemNames() map[string]map[uint32]string {
	return map[string]map[uint32]string{
		"c_content": ItemContentName, "c_visual": ItemVisualName, "c_act": ItemActName, "c_download": ItemDownloadName, "c_speed": ItemSpeedName, "c_postclick": ItemPostclickName}
}

func GetItemScoreArgs(ARGS url.Values) uint32 {
	v := Map([]string{"c_content", "c_visual", "c_act", "c_download", "c_speed", "c_postclick"}, ARGS.Get)
	camp := CreateItem(v[0], v[1], v[2], v[3], v[4], v[5])
	return camp.Pack()
}

/*
func SetItemScoreArgs(num uint32, ARGS url.Values) {
    C_attrs := GetItemAttrs()
    camp := UnpackItem(num)
	for i, name := range camp.ToNames() {
		ARGS.Set(C_attrs[i], name)
    }
}

func SetCampaignScoreItem(num uint32, item map[string]interface{}) {
    C_attrs := GetItemAttrs()
    camp := UnpackItem(num)
    for i, name := range camp.ToNames() {
        item[C_attrs[i]] = name
    }
}
*/

type ItemVisual uint32

const (
	ItemVisualUgly ItemVisual = 2 + iota
	ItemVisualPoor
	ItemVisualNormal
	ItemVisualGood
	ItemVisualExcellent
)

var ItemVisualName = map[uint32]string{
	4: "Normal",
	6: "Excellent",
	5: "Good",
	3: "Poor/Negative etc.",
	2: "Ugly/Blank/Body etc.",
}
var ItemVisualScore = map[uint32]float32{
	4: 0.0,
	6: 2.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
}
var ItemVisualValue = map[string]uint32{
	"Normal":    4,
	"Excellent": 6,
	"Good":      5,
	"Poor":      3,
	"Ugly":      2,
}

type ItemSpeed uint32

const (
	ItemSpeedSlow ItemSpeed = 3 + iota
	ItemSpeedNormal
)

var ItemSpeedName = map[uint32]string{
	4: "Normal",
	3: "Slow",
}
var ItemSpeedScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
}
var ItemSpeedValue = map[string]uint32{
	"Normal": 4,
	"Slow":   3,
}

type ItemAct uint32

const (
	ItemActAudio ItemAct = 2 + iota
	ItemActExpand
	ItemActNormal
)

var ItemActName = map[uint32]string{
	4: "Normal, No act",
	3: "Expand/Popup etc.",
	2: "Audio/Download etc.",
}
var ItemActScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
	2: -2.0,
}
var ItemActValue = map[string]uint32{
	"Normal": 4,
	"Expand": 3,
	"Audio":  2,
}

type ItemDownload uint32

const (
	ItemDownloadExecutable ItemDownload = 1 + iota
	ItemDownloadSoftware
	ItemDownloadDocument
	ItemDownloadNormal
)

var ItemDownloadName = map[uint32]string{
	4: "Normal, No download",
	3: "Paper/Document etc.",
	2: "Wallpaper/Software etc.",
	1: "Executable",
}
var ItemDownloadScore = map[uint32]float32{
	4: 0.0,
	3: -1.0,
	2: -5.0,
	1: -10.0,
}
var ItemDownloadValue = map[string]uint32{
	"Normal":     4,
	"Document":   3,
	"Sofware":    2,
	"Executable": 1,
}

type ItemContent uint32

const (
	ItemContentDeceptive ItemContent = 1 + iota
	ItemContentProvocative
	ItemContentDating
	ItemContentNormal
	ItemContentBrand
	ItemContentTopBrand
)

var ItemContentName = map[uint32]string{
	4: "Normal",
	6: "Top Brand",
	5: "Brand",
	3: "Dating etc.",
	2: "Provocative/Puzzle/Casino etc.",
	1: "Deceptive",
}
var ItemContentScore = map[uint32]float32{
	4: 0.0,
	6: 2.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
	1: -5.0,
}
var ItemContentValue = map[string]uint32{
	"Normal":      4,
	"Top":         6,
	"Brand":       5,
	"Dating":      3,
	"Provocative": 2,
	"Deceptive":   1,
}

type ItemPostclick uint32

const (
	ItemPostclickWrong ItemPostclick = 2 + iota
	ItemPostclickPoor
	ItemPostclickNormal
	ItemPostclickGood
)

var ItemPostclickName = map[uint32]string{
	4: "Normal",
	5: "Good Looking Site",
	3: "Poor/Wrong Site",
	2: "Broken/Hangup",
}
var ItemPostclickScore = map[uint32]float32{
	4: 0.0,
	5: 1.0,
	3: -1.0,
	2: -2.0,
}
var ItemPostclickValue = map[string]uint32{
	"Normal": 4,
	"Good":   5,
	"Poor":   3,
	"Broken": 2,
}

func CreateItem(content, visual, act, download, speed, postclick string) *Item {
	campaign := &Item{4, 4, 4, 4, 4, 4}
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

func (self *Item) Pack() uint32 {
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

func UnpackItem(campaign uint32) *Item {
	a := campaign & 7
	b := (campaign >> 3) & 7
	c := (campaign >> 6) & 7
	d := (campaign >> 9) & 7
	e := (campaign >> 12) & 7
	f := (campaign >> 15) & 7
	return &Item{a, b, c, d, e, f}
}

func (self *Item) ToNames() []string {
	return []string{ItemContentName[self.Content], ItemVisualName[self.Visual], ItemActName[self.Act], ItemDownloadName[self.Download], ItemSpeedName[self.Speed], ItemPostclickName[self.Postclick]}
}

func (self *Item) TotalScore() float32 {
	return self.ContentScore() +
		self.VisualScore() +
		self.ActScore() +
		self.DownloadScore() +
		self.SpeedScore() +
		self.PostclickScore()
}

func (self *Item) ContentScore() float32 {
	return ItemContentScore[self.Content]
}

func (self *Item) VisualScore() float32 {
	return ItemVisualScore[self.Visual]
}

func (self *Item) ActScore() float32 {
	return ItemActScore[self.Act]
}

func (self *Item) DownloadScore() float32 {
	return ItemDownloadScore[self.Download]
}

func (self *Item) SpeedScore() float32 {
	return ItemSpeedScore[self.Speed]
}

func (self *Item) PostclickScore() float32 {
	return ItemPostclickScore[self.Postclick]
}
