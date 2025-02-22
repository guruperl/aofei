package summerbeacon

import (
	"database/sql"

	"github.com/genelet/winter/genelet"

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

func NewController(fn string) (*genelet.Controller, error) {
	c, err := genelet.NewConfig(fn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		return nil, err
	}

	models := map[string]interface{}{
		"agent": new(agent.Model), "manage": new(manage.Model), "payment": new(payment.Model), "alipay": new(alipay.Model), "wechat": new(wechat.Model), "cheque": new(cheque.Model), "cc": new(cc.Model), "ac": new(ac.Model), "address": new(address.Model), "adv": new(adv.Model), "attrname": new(attrname.Model), "campaign": new(campaign.Model), "chac": new(chac.Model), "channel": new(channel.Model), "balance": new(balance.Model), "ledger": new(ledger.Model), "creative": new(creative.Model), "item": new(item.Model), "pub": new(pub.Model), "site": new(site.Model), "slot": new(slot.Model), "targetname": new(targetname.Model), "weight": new(weight.Model),
	}

	storage := map[string]interface{}{
		"agent": new(agent.Model), "manage": new(manage.Model), "payment": new(payment.Model), "alipay": new(alipay.Model), "wechat": new(wechat.Model), "cheque": new(cheque.Model), "cc": new(cc.Model), "ac": new(ac.Model), "address": new(address.Model), "adv": new(adv.Model), "attrname": new(attrname.Model), "campaign": new(campaign.Model), "chac": new(chac.Model), "channel": new(channel.Model), "balance": new(balance.Model), "ledger": new(ledger.Model), "creative": new(creative.Model), "item": new(item.Model), "pub": new(pub.Model), "site": new(site.Model), "slot": new(slot.Model), "targetname": new(targetname.Model), "weight": new(weight.Model),
	}

	filters := map[string]interface{}{
		"agent": new(agent.Filter), "manage": new(manage.Filter), "payment": new(payment.Filter), "alipay": new(alipay.Filter), "wechat": new(wechat.Filter), "cheque": new(cheque.Filter), "cc": new(cc.Filter), "ac": new(ac.Filter), "address": new(address.Filter), "adv": new(adv.Filter), "attrname": new(attrname.Filter), "campaign": new(campaign.Filter), "chac": new(chac.Filter), "channel": new(channel.Filter), "balance": new(balance.Filter), "ledger": new(ledger.Filter), "creative": new(creative.Filter), "item": new(item.Filter), "pub": new(pub.Filter), "site": new(site.Filter), "slot": new(slot.Filter), "targetname": new(targetname.Filter), "weight": new(weight.Filter),
	}

	for k := range models {
		comp := genelet.NewComponent(c.ProjectRoot + "/src/github.com/genelet/winter/summer/" + k + "/component.json")
		genelet.Invoke0(models[k], "Initialize", comp)
		genelet.Invoke0(storage[k], "Initialize", comp)
		genelet.Invoke0(filters[k], "Initialize", comp)
	}

	return &genelet.Controller{C: c, Db: db, Models: models, Filters: filters, Storage: storage}, nil
}
