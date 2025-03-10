package holiday

const DefaultItemPrice float32 = 1.0
const (
	UnknownCost = iota
	CPMCost
	CPCCost
	CPACost
	CPDCost
)

type Item struct {
	RAdv
	Cap Cap
}

func itemFromTao(hash map[string]interface{}) *Item {
	item := &Item{RAdv: radvFromTao(hash), Cap: capFromTao(hash)}
	return item
}

func (self *Item) Eprice() float32 {
	if self.Price <= 0.0 {
		return DefaultItemPrice
	}
	switch int(self.CostType) {
	case CPMCost:
		return self.Price
	case CPCCost:
		return 10.0 * self.Price
	case CPACost:
		return 100.0 * self.Price
	case CPDCost:
		return 1000.0 * self.Price
	default:
	}
	return self.Price
}

// Weigh assigns a weight according to eprice, or whatever logic is
func (self *Item) Weight() float32 {
	ep := self.Eprice()
	return ep * ep
}

// PickItem generates a random item according to their weights
func PickItem(items []*Item) *Item {
	n := len(items)
	weights := make([]float32, n)
	for i, item := range items {
		weights[i] = item.Weight()
	}
	index := SelectOne(weights)
	return items[index]
}
