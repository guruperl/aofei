package holiday

type Creative struct {
	CreativeId uint32
	Weight     float32
	Click      string
	Content    string
	Cap *Cap
}

func creativeFromTao(hash map[string]interface{}) *Creative {
	return &Creative{
		uint32(hash["creative_id"].(int32)),
		hash["weight"].(float32),
		hash["click"].(string),
		hash["content"].(string), capFromTao(hash)}
}

func PickCreativeFromTao(lists []map[string]interface{}) *Creative {
	creatives := make([]*Creative, 0)
	weights := make([]float32, 0)
	for _, hash := range lists {
		creative := creativeFromTao(hash)
		creatives = append(creatives, creative)
		weights = append(weights, creative.Weight)
	}
	index := SelectOne(weights)
	return creatives[index]
}
