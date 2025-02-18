package ssp

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/match"
	ipsearch "github.com/genelet/winter/maxmind"
	"github.com/genelet/winter/pzutil"
	"github.com/golang/glog"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/nats-io/nats.go"
)

type Controller struct {
	C     *pzutil.Config
	Ips   *ipsearch.IPSearch
	Redis *pool.Pool
	Db    *sql.DB
	Nc    *nats.Conn
}

func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Info("0: initial")
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Max-Age", "1728000")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	}
	if acrh := r.Header.Get("Access-Control-Request-Headers"); acrh != "" {
		w.Header().Set("Access-Control-Allow-Headers", acrh)
	}
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	c := self.C
	glog.Info("1: collect request info")
	status, _, adImps, clk, _, err := match.GetPathIds(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	current := time.Now()
	var user *User
	var dmo *demo.Demo

	switch status.Source {
	case pzutil.DSP:
		/*
		           ip_str = bid.Device.IP
		           ua_str = bid.Device.UA
		           if bid.User.Gender == "M" { dmo.Gender=1 }
		   		else if bid.User.Gender == "F" { dmo.Gender=2 }
		           if bid.User.Yob > 0 { dmo.Yob = uint32(bid.User.Yob) }
		*/
	case pzutil.BROWSER, pzutil.MOBILE, pzutil.SDK:
		glog.Info("2: create user")
		user, err = CreateUser(r, c, self.Ips, current)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if status.Source == pzutil.BROWSER && (user.PzUa.Device == 1 || user.PzUa.Device == 2) {
			status.Source = pzutil.MOBILE
		}
		dmo = demo.CreateArgsDemo(r.Form, "gender", "yob", "married", "income", "child", "household", "ethnicity", "education", "occupation")
	default:
	}

	switch status.Request {
	case pzutil.CLIC:
		glog.Infof("3: serve click %#v %#v", status, clk)
		self.serveClick(w, status, user, adImps[0], clk)
		return
	case pzutil.UnknownRequest:
		path := r.URL.Path
		if strings.Contains(path, `..`) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		} else {
			glog.Info("3: serve file")
			http.ServeFile(w, r, c.DocumentRoot+path)
		}
		return
	default:
	}

	glog.Info("4: check dmp/site")
	dmmp, site, err := self.RedisGetPublisher(adImps[0].SiteID, string(user.Pid.PackBytes()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	glog.Info("5: check domain")
	if !site.DomainMatch(r.URL) {
		http.Error(w, "bad domain", http.StatusBadRequest)
		return
	}

	glog.Info("6: get redis slots")
	nImps := len(adImps)
	slots, err := self.RedisGetSlots(adImps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var m1, m2, n1, n2, n int

	ok := make([]bool, nImps)
	sav_weights := make([][]match.Weight, nImps)
	sav_basics := make([][]match.Weight, nImps)
	camps := make(map[uint32]bool)
	glog.Info("7: get each slot")
	for i, slot := range slots {
		after := make([]match.Weight, 0)
		for _, w := range slot.Weights {
			if adImps[i].MatchMime(w.Mime8) {
				after = append(after, w)
			}
		}
		if len(after) < 1 {
			continue
		}
		ok[i] = true
		n1, n2, sav_weights[i], sav_basics[i] = user.FilterCappedWeights(after)
		m1 += n1
		m2 += n2
		for _, weight := range sav_weights[i] {
			camps[weight.CampaignID] = true
		}
	}
	glog.Info("8: get audiences")
	ref, err := self.getReference(camps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sav_final := make([]*match.Weight, nImps)
	ref_items := make(map[uint32]bool)
	glog.Info("9: get each item")
	for i := 0; i < nImps; i++ {
		if !ok[i] {
			continue
		}
		n, sav_final[i] = user.MatchItemRef(dmo, dmmp, ref, sav_weights[i], sav_basics[i], ref_items)
		if n > 0 {
			m1 += n // n=1, get normal item
			//} else if n == 0 { // normal item which isnt capped
			//} else if n == -1 { // get baseline item
		} else if n == -2 { // even baseline item not found
			ok[i] = false
			continue
		}
		ref_items[sav_final[i].ItemID] = true
	}
	glog.Info("10: get details of selecte items")
	ids := make([]uint32, 0)
	for id := range ref_items {
		ids = append(ids, id)
	}
	items, err := self.RedisGetItems(user, ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	htmls := make([]string, nImps)
	wins := make([]match.Win, nImps)
	id := user.ToImpID()
	glog.Info("11: build html")
	for i := 0; i < nImps; i++ {
		if ok[i] {
			item := items[i]
			creative := item.SelectCreative()
			htmls[i], wins[i] = self.getHtml(id, item, sav_final[i], creative, adImps[i])
		} else {
			status.IsPSA = true
			htmls[i], wins[i] = self.getPSA(adImps[i])
		}
	}

	glog.Info("11: set up cookie")
	user.SetCookies(w, c, m1, m2)

	w.Header().Set("Content-Type", status.ContentType())
	w.WriteHeader(200)
	glog.Infof("12: send page %v", status)
	if status.Mime == pzutil.JSON {
		glog.Infof("12: %s", htmls[0])
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.Encode(htmls)
		//b, _ := json.Marshal(htmls)
		//w.Write(b)
	} else {
		w.Write([]byte(htmls[0]))
	}

	record := match.Record{
		Imp:  user.ToImp(status, adImps[0].RPub),
		Wins: wins,
	}
	glog.Info("13: send to msg")
	if bs, err := record.Pack(); err == nil {
		self.Nc.Publish("user", bs)
		self.Nc.Flush()
	}
}

func (self *Controller) getReference(camps map[uint32]bool) (map[uint32]*match.Audience, error) {
	ids := make([]uint32, 0)
	for id := range camps {
		ids = append(ids, id)
	}

	audiences, err := self.RedisGetAudiences(ids)
	if err != nil {
		return nil, err
	}
	ref := make(map[uint32]*match.Audience)
	for i, audience := range audiences {
		ref[ids[i]] = audience
	}

	return ref, nil
}

func sizeString(size_id uint32) string {
	w, h := pzutil.GetSizes(size_id)
	return `width='` + strconv.Itoa(int(w)) + `' height='` + strconv.Itoa(int(h)) + `'`
}

func (self *Controller) getHtml(id string, item *match.Item, weight *match.Weight, creative *match.Creative, adImp *match.AdImp) (string, match.Win) {
	c := self.C
	radv := match.GetRAdv(item, weight, creative)
	rpub := adImp.RPub

	win := match.Win{
		SlotID: adImp.SlotID,
		RAdv:   radv,
	}

	escaped := url.PathEscape(url.QueryEscape(item.Click))
	//glog.Infof("escaped: %s", escaped)
	pp, _ := rpub.Pack()
	ap, _ := radv.Pack()
	click := c.ServerURL + c.Handle["click"] + "/" + id + "/" + pp + "/" + ap + "." + escaped
	content := strings.Replace(creative.Content, "LANDING", click, -1)
	if item.IsImage {
		return `<a href='` + click + `'><img src='` + c.ServerURL + content + `' ` + sizeString(rpub.SizeID) + `></a>`, win
	} else if item.IsHTML {
		return `<iframe frameborder=0 src="data:text/html; charset=UTF-8,` + content + `" ` + sizeString(rpub.SizeID) + `></iframe>`, win
	} else if item.IsJs {
		//return `<iframe frameborder=0 srcdoc="<script>`+content+`</script>" `+sizeString(rpub.SizeID)+`></iframe>`, win
		return `<iframe frameborder=0 src="data:text/html; charset=UTF-8,<script>` + content + `</script>" ` + sizeString(rpub.SizeID) + `></iframe>`, win
	} else if item.IsVideo {
		return `<video controls><source src="` + content + `"></video>`, win
	}
	return `<a href='` + click + `'><img src='` + c.ServerURL + content + `' ` + sizeString(rpub.SizeID) + `></a>`, win
}

func (self *Controller) getPSA(adImp *match.AdImp) (string, match.Win) {
	c := self.C
	size_id := adImp.GetSizeID()
	psa := (c.Sizes)[size_id]

	win := match.Win{
		SlotID: adImp.SlotID,
		RAdv: match.RAdv{
			AdvID:      0,
			CampaignID: 0,
			ItemID:     0,
			CreativeID: 0,
			Price:      0.0,
		},
	}

	if psa.Click == "" {
		return psa.Display, win
	}

	pp, _ := adImp.RPub.Pack()
	return `<a href='` + c.ServerURL + c.Handle["click"] + "/" + pp + "." + url.PathEscape(url.QueryEscape(psa.Click)) + `'><img src='` + psa.Display + `' ` + sizeString(size_id) + `></a>`, win
}
