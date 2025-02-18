// Package ssp provides functionalities for user management and matching items.
package ssp

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dmp"
	"github.com/genelet/winter/match"
	ipsearch "github.com/genelet/winter/maxmind"
	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/uadevice"
)

type User struct {
	Visitor
	uadevice.PzUa
	IsNew      bool
	IsDailyNew bool
	FullTime   time.Time
	IP32       uint32
	ICaps      map[uint32]match.Fcap
	CCaps      map[uint32]match.Fcap
}

func CreateUser(r *http.Request, c *pzutil.Config, ips *ipsearch.IPSearch, current time.Time) (*User, error) {
	xf := r.Header.Get("X-Forwarded-For")
	if xf == "" {
		xf = r.Header.Get("X-Real-IP")
	}
	var ipstr string
	if xf == "" {
		str, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return nil, err
		}
		ipstr = str
	} else {
		strs := strings.Split(xf, ",")
		ipstr = strs[0]
	}
	ip32 := pzutil.IP2Uint(net.ParseIP(ipstr))

	cookieVal := ""
	if ucookie, _ := r.Cookie(c.Ucookie); ucookie != nil {
		cookieVal = ucookie.Value
	}

	if r.UserAgent() == "" {
		return nil, errors.New("user agent not found")
	}
	pzua := uadevice.CreateTwoUa(r.UserAgent())
	visitor, needCookie, isNew, err := RetrieveVisitor(cookieVal, ips, pzua, current, ipstr, ip32)
	if err != nil {
		return nil, err
	}

	var icaps, ccaps map[uint32]match.Fcap
	if icookie, _ := r.Cookie(c.Icookie); icookie != nil {
		icaps, _ = match.UnpackFcaps(current, icookie.Value)
	}
	if ccookie, _ := r.Cookie(c.Ccookie); ccookie != nil {
		ccaps, _ = match.UnpackFcaps(current, ccookie.Value)
	}

	return &User{*visitor, *pzua, isNew, needCookie, current, ip32, icaps, ccaps}, nil
}

func (self *User) ToImp(status pzutil.Status, rpub match.RPub) match.Imp {
	return match.Imp{
		Status16: status.Pack(),
		Nano64:   self.FullTime.UnixNano(),
		IP32:     self.IP32,
		PzUa32:   self.PzUa.Pack(),
		Pid:      self.Pid,
		PubID:    rpub.PubID,
		SiteID:   rpub.SiteID,
	}
}

func (self *User) ToImpID() string {
	pid := match.Pid{
		StartNano: self.FullTime.UnixNano(),
		StartIP:   self.IP32,
		StartUa:   self.PzUa.Pack(),
	}
	return pid.PackHex()
}

func (self *User) ToCustomData(n1, n2 int) string {
	p, _ := self.Visitor.Pack()
	istr := ""
	cstr := ""
	if n1 > 0 {
		if str, err := match.PackFcaps(self.ICaps); err == nil {
			istr = str
		}
	}
	if n2 > 0 {
		if str, err := match.PackFcaps(self.CCaps); err == nil {
			cstr = str
		}
	}
	return p + "," + istr + "," + cstr
}

func (self *User) SetCookies(w http.ResponseWriter, c *pzutil.Config, n1, n2 int) {
	if self.IsNew || self.IsDailyNew {
		p, _ := self.Visitor.Pack()
		http.SetCookie(w, &http.Cookie{Name: c.Ucookie, Value: p, Path: "/", MaxAge: c.UcookieMaxAge})
	}
	if n1 > 0 {
		if str, err := match.PackFcaps(self.ICaps); err == nil {
			http.SetCookie(w, &http.Cookie{Name: c.Icookie, Value: str, Path: "/", MaxAge: c.IcookieMaxAge})
		}
	}
	if n2 > 0 {
		if str, err := match.PackFcaps(self.CCaps); err == nil {
			http.SetCookie(w, &http.Cookie{Name: c.Ccookie, Value: str, Path: "/", MaxAge: c.CcookieMaxAge})
		}
	}
}

// Check status ----  0: no cap; -1: capped, not serve; 1: capped, ok
func (self *User) Check(weight match.Weight) (int, int) {
	when := self.FullTime
	cid := weight.CampaignID
	capped := 0
	clicked := 0
	if weight.Endx > 0 && weight.Endx < uint32(when.Unix()) {
		delete(self.ICaps, cid)
		delete(self.CCaps, cid)
		return -1, -1
	}

	if weight.CapNumber > 0 {
		capped = -1
		if fcap, ok := self.ICaps[cid]; ok {
			if fcap.SinceStart(when) > weight.CapPeriod {
				delete(self.ICaps, cid)
				capped = 1
			} else if fcap.Total < uint8(weight.CapNumber) {
				if weight.CapThrottle == 0 || (weight.CapThrottle > 0 && fcap.SinceLast(when) > weight.CapThrottle) {
					capped = 1
				}
			}
		} else {
			capped = 1
		}
	} else if _, ok := self.ICaps[cid]; ok { //campaign changed to remove cap
		delete(self.ICaps, cid)
		capped = 1
	}

	if weight.ClickNumber > 0 {
		clicked = -1
		if fcap, ok := self.CCaps[cid]; ok {
			if fcap.SinceStart(when) > weight.ClickPeriod {
				delete(self.CCaps, cid)
				clicked = 1
			} else if fcap.Total < uint8(weight.ClickNumber) {
				clicked = 1
			}
		} else {
			clicked = 1
		}
	} else if _, ok := self.CCaps[cid]; ok { //campaign changed to remove cap
		delete(self.CCaps, cid)
		clicked = 1
	}

	return capped, clicked
}

func (self *User) FilterCappedWeights(selects []match.Weight) (int, int, []match.Weight, []match.Weight) {
	var n1, n2 int
	weights := make([]match.Weight, 0)
	basics := make([]match.Weight, 0)

	for _, weight := range selects {
		if weight.Weight > 0 {
			capped, clicked := self.Check(weight)
			if capped == 1 {
				n1++
			}
			if clicked == 1 {
				n2++
			}
			if capped != -1 && clicked != -1 {
				weights = append(weights, weight)
			}
		} else {
			weight.Weight = float32(-1.0) * weight.Weight
			basics = append(basics, weight)
		}
	}
	return n1, n2, weights, basics
}

func basicItem(basics []match.Weight) (int, *match.Weight) {
	if len(basics) < 1 {
		return -2, nil
	}
	probs := make([]float32, 0)
	for _, b := range basics {
		probs = append(probs, b.Weight)
	}
	k := match.SelectOne(probs)
	return -1, &basics[k]
}

func (self *User) MatchItem(demo *demo.Demo, dmmp *dmp.Dmp, audiences []*match.Audience, weights []match.Weight, basics []match.Weight) (int, *match.Weight) {
	if audiences == nil || len(audiences) < 1 {
		return basicItem(basics)
	}
	probs := make([]float32, 0)
	index := 0
	markers := make(map[int]int)
	for k, aud := range audiences {
		matched := false
		if aud == nil {
			matched = true
		} else if aud.MatchDmp(dmmp) && aud.MatchGeo(self.Geo) && aud.MatchUa(self.PzUa) && aud.MatchWeekTime(self.FullTime) {
			matched = true
		}
		if matched {
			probs = append(probs, weights[k].Weight)
			markers[index] = k
			index++
		}
	}
	if index == 0 {
		return basicItem(basics)
	}
	marker := markers[match.SelectOne(probs)]
	weight := weights[marker]
	n := 0
	if weight.CapNumber > 0 {
		if self.ICaps == nil {
			self.ICaps = make(map[uint32]match.Fcap)
		}
		match.UpdateFcaps(&(self.ICaps), weight.CampaignID, self.FullTime)
		n = 1
	}
	return n, &weight
}

func (self *User) MatchItemRef(demo *demo.Demo, dmmp *dmp.Dmp, ref map[uint32]*match.Audience, weights []match.Weight, basics []match.Weight, refItems map[uint32]bool) (int, *match.Weight) {
	probs := make([]float32, 0)
	index := 0
	markers := make(map[int]int)
	for k, weight := range weights {
		if refItems[weight.ItemID] {
			continue
		} // item already selected
		matched := false
		aud := ref[weight.CampaignID]
		if aud == nil {
			matched = true
		} else if aud.MatchDmp(dmmp) && aud.MatchGeo(self.Geo) && aud.MatchUa(self.PzUa) && aud.MatchWeekTime(self.FullTime) {
			matched = true
		}
		if matched {
			probs = append(probs, weight.Weight)
			markers[index] = k
			index++
		}
	}
	if index == 0 {
		return basicItem(basics)
	}
	marker := markers[match.SelectOne(probs)]
	weight := weights[marker]
	n := 0
	if weight.CapNumber > 0 {
		if self.ICaps == nil {
			self.ICaps = make(map[uint32]match.Fcap)
		}
		match.UpdateFcaps(&(self.ICaps), weight.CampaignID, self.FullTime)
		n = 1
	}
	return n, &weight
}
