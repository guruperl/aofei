package match

func GetSiteSample() *Site {
	return &Site{
		PubID:      uint32(1),
		Company:    "Pub company",
		SiteID:     uint32(2),
		SiteName:   "site name",
		SiteURL:    "http://www.site url",
		Referers:   []string{"aaa.com", "bb.com/cc"},
		ChannelIds: []uint16{33, 44, 55}}
}

func GetWeightSamples() []Weight {
	return []Weight{
		{WeightID: uint32(6), ItemID: uint32(1), Endx: uint32(0), CampaignID: uint32(2), CapNumber: uint8(3), CapPeriod: uint16(50),
			CapThrottle: uint16(30), ClickNumber: uint8(3), ClickPeriod: uint16(50), Weight: float32(0.8)},
		{WeightID: uint32(66), ItemID: uint32(11), Endx: uint32(0), CampaignID: uint32(22), CapNumber: uint8(3), CapPeriod: uint16(50),
			CapThrottle: uint16(30), ClickNumber: uint8(3), ClickPeriod: uint16(50), Weight: float32(0.88)},
	}
}

func getItemSample() *Item {
	creatives := []*Creative{{1, 0.9, "js1"}, {2, 0.1, "js1"}}
	return &Item{ItemID: 222, AdvID: 333, SizeID: 444, Creatives: creatives}
}
