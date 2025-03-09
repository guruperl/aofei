package holiday

type RAdv struct {
	AdvId		uint32
	CampaignId	uint32
	ItemId		uint32
	CreativeId	uint32
	CostType	uint8
	Price		float32
}

func radvFromTao(hash map[string]interface{}) RAdv {
	radv := RAdv{
		AdvId     : uint32(hash["adv_id"].(int32)),
		CampaignId: uint32(hash["campaign_id"].(int32)),
		ItemId    : uint32(hash["item_id"].(int32)),
	}
	if hash["creative_id"] != nil {
		radv.CreativeId = uint32(hash["creative_id"].(int32))
	}
    if hash["price"] != nil {
        radv.Price = hash["price"].(float32)
    }
	if hash["cost_type"] != nil {
		radv.CostType = uint8(hash["cost_type"].(int8))
	}
	return radv
}

func (self RAdv) ToArgs() map[string]interface{} {
	return map[string]interface{}{
		"creative_id": self.CreativeId,
		"price"      : self.Price,
		"cost_type"  : self.CostType,
		"item_id"    : self.ItemId,
		"campaign_id": self.CampaignId,
		"adv_id"     : self.AdvId,
	}
}
