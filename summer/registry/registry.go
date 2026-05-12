package registry

import (
	"github.com/genelet/winter/summer/ac"
	"github.com/genelet/winter/summer/address"
	"github.com/genelet/winter/summer/adv"
	"github.com/genelet/winter/summer/agent"
	"github.com/genelet/winter/summer/alipay"
	"github.com/genelet/winter/summer/attrname"
	"github.com/genelet/winter/summer/balance"
	"github.com/genelet/winter/summer/campaign"
	"github.com/genelet/winter/summer/cc"
	"github.com/genelet/winter/summer/chac"
	"github.com/genelet/winter/summer/channel"
	"github.com/genelet/winter/summer/cheque"
	"github.com/genelet/winter/summer/creative"
	"github.com/genelet/winter/summer/item"
	"github.com/genelet/winter/summer/ledger"
	"github.com/genelet/winter/summer/manage"
	"github.com/genelet/winter/summer/payment"
	"github.com/genelet/winter/summer/pub"
	"github.com/genelet/winter/summer/site"
	"github.com/genelet/winter/summer/slot"
	"github.com/genelet/winter/summer/targetname"
	"github.com/genelet/winter/summer/wechat"
	"github.com/genelet/winter/summer/weight"
)

type Entry struct {
	Name       string
	NewModel   func() interface{}
	NewStorage func() interface{}
	NewFilter  func() interface{}
}

var Entries = []Entry{
	{"ac", func() interface{} { return new(ac.Model) }, func() interface{} { return new(ac.Model) }, func() interface{} { return new(ac.Filter) }},
	{"address", func() interface{} { return new(address.Model) }, func() interface{} { return new(address.Model) }, func() interface{} { return new(address.Filter) }},
	{"adv", func() interface{} { return new(adv.Model) }, func() interface{} { return new(adv.Model) }, func() interface{} { return new(adv.Filter) }},
	{"agent", func() interface{} { return new(agent.Model) }, func() interface{} { return new(agent.Model) }, func() interface{} { return new(agent.Filter) }},
	{"alipay", func() interface{} { return new(alipay.Model) }, func() interface{} { return new(alipay.Model) }, func() interface{} { return new(alipay.Filter) }},
	{"attrname", func() interface{} { return new(attrname.Model) }, func() interface{} { return new(attrname.Model) }, func() interface{} { return new(attrname.Filter) }},
	{"balance", func() interface{} { return new(balance.Model) }, func() interface{} { return new(balance.Model) }, func() interface{} { return new(balance.Filter) }},
	{"campaign", func() interface{} { return new(campaign.Model) }, func() interface{} { return new(campaign.Model) }, func() interface{} { return new(campaign.Filter) }},
	{"cc", func() interface{} { return new(cc.Model) }, func() interface{} { return new(cc.Model) }, func() interface{} { return new(cc.Filter) }},
	{"chac", func() interface{} { return new(chac.Model) }, func() interface{} { return new(chac.Model) }, func() interface{} { return new(chac.Filter) }},
	{"channel", func() interface{} { return new(channel.Model) }, func() interface{} { return new(channel.Model) }, func() interface{} { return new(channel.Filter) }},
	{"cheque", func() interface{} { return new(cheque.Model) }, func() interface{} { return new(cheque.Model) }, func() interface{} { return new(cheque.Filter) }},
	{"creative", func() interface{} { return new(creative.Model) }, func() interface{} { return new(creative.Model) }, func() interface{} { return new(creative.Filter) }},
	{"item", func() interface{} { return new(item.Model) }, func() interface{} { return new(item.Model) }, func() interface{} { return new(item.Filter) }},
	{"ledger", func() interface{} { return new(ledger.Model) }, func() interface{} { return new(ledger.Model) }, func() interface{} { return new(ledger.Filter) }},
	{"manage", func() interface{} { return new(manage.Model) }, func() interface{} { return new(manage.Model) }, func() interface{} { return new(manage.Filter) }},
	{"payment", func() interface{} { return new(payment.Model) }, func() interface{} { return new(payment.Model) }, func() interface{} { return new(payment.Filter) }},
	{"pub", func() interface{} { return new(pub.Model) }, func() interface{} { return new(pub.Model) }, func() interface{} { return new(pub.Filter) }},
	{"site", func() interface{} { return new(site.Model) }, func() interface{} { return new(site.Model) }, func() interface{} { return new(site.Filter) }},
	{"slot", func() interface{} { return new(slot.Model) }, func() interface{} { return new(slot.Model) }, func() interface{} { return new(slot.Filter) }},
	{"targetname", func() interface{} { return new(targetname.Model) }, func() interface{} { return new(targetname.Model) }, func() interface{} { return new(targetname.Filter) }},
	{"wechat", func() interface{} { return new(wechat.Model) }, func() interface{} { return new(wechat.Model) }, func() interface{} { return new(wechat.Filter) }},
	{"weight", func() interface{} { return new(weight.Model) }, func() interface{} { return new(weight.Model) }, func() interface{} { return new(weight.Filter) }},
}

func Build() (map[string]interface{}, map[string]interface{}, map[string]interface{}) {
	models := make(map[string]interface{}, len(Entries))
	storage := make(map[string]interface{}, len(Entries))
	filters := make(map[string]interface{}, len(Entries))
	for _, entry := range Entries {
		models[entry.Name] = entry.NewModel()
		storage[entry.Name] = entry.NewStorage()
		filters[entry.Name] = entry.NewFilter()
	}
	return models, storage, filters
}
