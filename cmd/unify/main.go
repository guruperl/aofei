// go run summer.go --log_dir="../../logs/"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/ssp"
	_ "github.com/go-sql-driver/mysql"

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

func usage() {
	fmt.Fprintf(os.Stderr, "usage: summer --g=web_config --s=ssp_config -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var gConf, sConf string

func init() {
	flag.Usage = usage
	flag.StringVar(&gConf, "g", os.Getenv("SUMMER"), "Genelet Config")
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "Ssp Config")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	sc, err := getSsp(ctx, sConf)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle(pzutil.SSPHandler, sc)
	http.Handle(pzutil.BIDHandler, sc)
	http.Handle(pzutil.CLK, sc)
	http.Handle(pzutil.WIN, sc)

	gc, err := getGenelet(gConf)
	if err != nil {
		panic(err)
	}
	gc.DB = sc.DB
	gc.Storage["Redis"] = sc.Redis
	gc.Storage["Ssp"] = sc.C
	if gc.C.ServerPort == "" {
		gc.C.ServerPort = sc.C.ServerPort
	}
	if gc.C.ServerURL == "" {
		gc.C.ServerURL = sc.C.ServerURL
	}
	if gc.C.DocumentRoot == "" {
		gc.C.DocumentRoot = sc.C.DocumentRoot
	}
	if gc.C.ConnectArray == nil {
		gc.C.ConnectArray = sc.C.ConnectArray
	}
	http.Handle("/", gc)

	log.Fatal(http.ListenAndServe(":"+gc.C.ServerPort, nil))
}

func getGenelet(fn string) (*genelet.Controller, error) {
	models := map[string]interface{}{
		"agent": new(agent.Model), "manage": new(manage.Model), "payment": new(payment.Model), "alipay": new(alipay.Model), "wechat": new(wechat.Model), "cheque": new(cheque.Model), "cc": new(cc.Model), "ac": new(ac.Model), "address": new(address.Model), "adv": new(adv.Model), "attrname": new(attrname.Model), "campaign": new(campaign.Model), "chac": new(chac.Model), "channel": new(channel.Model), "balance": new(balance.Model), "ledger": new(ledger.Model), "creative": new(creative.Model), "item": new(item.Model), "pub": new(pub.Model), "site": new(site.Model), "slot": new(slot.Model), "targetname": new(targetname.Model), "weight": new(weight.Model),
	}

	storage := map[string]interface{}{
		"agent": new(agent.Model), "manage": new(manage.Model), "payment": new(payment.Model), "alipay": new(alipay.Model), "wechat": new(wechat.Model), "cheque": new(cheque.Model), "cc": new(cc.Model), "ac": new(ac.Model), "address": new(address.Model), "adv": new(adv.Model), "attrname": new(attrname.Model), "campaign": new(campaign.Model), "chac": new(chac.Model), "channel": new(channel.Model), "balance": new(balance.Model), "ledger": new(ledger.Model), "creative": new(creative.Model), "item": new(item.Model), "pub": new(pub.Model), "site": new(site.Model), "slot": new(slot.Model), "targetname": new(targetname.Model), "weight": new(weight.Model),
	}

	filters := map[string]interface{}{
		"agent": new(agent.Filter), "manage": new(manage.Filter), "payment": new(payment.Filter), "alipay": new(alipay.Filter), "wechat": new(wechat.Filter), "cheque": new(cheque.Filter), "cc": new(cc.Filter), "ac": new(ac.Filter), "address": new(address.Filter), "adv": new(adv.Filter), "attrname": new(attrname.Filter), "campaign": new(campaign.Filter), "chac": new(chac.Filter), "channel": new(channel.Filter), "balance": new(balance.Filter), "ledger": new(ledger.Filter), "creative": new(creative.Filter), "item": new(item.Filter), "pub": new(pub.Filter), "site": new(site.Filter), "slot": new(slot.Filter), "targetname": new(targetname.Filter), "weight": new(weight.Filter),
	}

	c, err := genelet.NewConfig(fn)
	if err != nil {
		return nil, err
	}
	for k := range models {
		comp := genelet.NewComponent(c.ProjectRoot + "/summer/" + k + "/component.json")
		genelet.Invoke0(models[k], "Initialize", comp)
		genelet.Invoke0(storage[k], "Initialize", comp)
		genelet.Invoke0(filters[k], "Initialize", comp)
	}

	return &genelet.Controller{C: c, Models: models, Filters: filters, Storage: storage}, nil
}

func getSsp(ctx context.Context, fn string) (*ssp.Controller, error) {
	c, err := pzutil.NewConfig(fn)
	if err != nil {
		return nil, err
	}
	return ssp.NewController(ctx, c)
}
