// go run summer.go --log_dir="../../logs/"
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/ssp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/nats-io/nats.go"

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
	flag.StringVar(&sConf, "s", os.Getenv("PZADX"), "Ssp Config")
	flag.Parse()
}

func main() {
	sc := getSsp(sConf)
	gc := getGenelet(gConf, sc)

	http.Handle(pzutil.SSPHandler, sc)
	http.Handle(pzutil.BIDHandler, sc)
	http.Handle(pzutil.CLK, sc)
	http.Handle(pzutil.WIN, sc)
	http.Handle("/", gc)

	log.Fatal(http.ListenAndServe(":"+gc.C.ServerPort, nil))
}

func getGenelet(fn string, sc *ssp.Controller) *genelet.Controller {
	c := genelet.NewConfig(fn)
	c.Db = sc.C.ConnectArray
	c.DocumentRoot = sc.C.DocumentRoot
	c.ServerURL = sc.C.ServerURL

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
		comp := genelet.NewComponent(c.ProjectRoot + "/summer/" + k + "/component.json")
		genelet.Invoke0(models[k], "Initialize", comp)
		genelet.Invoke0(storage[k], "Initialize", comp)
		genelet.Invoke0(filters[k], "Initialize", comp)
	}

	db := sc.Db
	storage["Redis"] = sc.Redis
	storage["Ssp"] = sc.C
	return &genelet.Controller{C: c, Db: db, Models: models, Filters: filters, Storage: storage}
}

func getSsp(fn string) *ssp.Controller {
	c := pzutil.NewConfig(fn)

	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		panic(err)
	}

	ips, err := ipsearch.LoadIPData(c.Ips)
	if err != nil {
		panic(err)
	}

	redis, err := pool.New(c.Redis.Network, c.Redis.Addr, c.Redis.Size)
	if err != nil {
		panic(err)
	}

	return &ssp.Controller{C: c, Ips: ips, Redis: redis, Db: db, Nc: nc}
}
