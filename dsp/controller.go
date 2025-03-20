package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/genelet/winter/match"
	"github.com/genelet/winter/maxmind"
)

type Controller struct {
	C       *Config
	Ips     *maxmind.IPSearch
	Redis   radix.Client
	DB      *sql.DB
	Nc      *nats.Conn
	RPubMap *match.RPubMap
}

func NewController(ctx context.Context, filename string) (*Controller, error) {
	c, err := NewConfig(filename)
	if err != nil {
		return nil, err
	}

	ips, err := maxmind.LoadIPData(c.Ips)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		return nil, err
	}
	red := c.Redis
	cfg := radix.PoolConfig{
		Dialer: radix.Dialer{
			AuthUser: red.User,
			AuthPass: red.Pass,
		},
	}
	if red.Size != 0 {
		cfg.Size = red.Size
	}
	redis, err := cfg.New(ctx, red.Network, red.Addr)
	if err != nil {
		return nil, err
	}
	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		return nil, err
	}

	rpubMap := match.DefaultRPubMap()
	if c.RPubMap == "" {
		bs, err := os.ReadFile(c.RPubMap)
		if err != nil {
			return nil, err
		}
		if bs != nil {
			err = json.Unmarshal(bs, &rpubMap)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Controller{C: c, Ips: ips, Redis: redis, DB: db, Nc: nc, RPubMap: rpubMap}, err
}

// Close closes the Controller.
func (self *Controller) Close() {
	self.Redis.Close()
	self.DB.Close()
	self.Nc.Close()
}

func (self *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Info("0: initial")

	var bidStr []byte
	var bid *openrtb2.BidRequest
	var pubStr string
	var ok bool
	var err error

	current := time.Now()
	ctx := r.Context()

	switch r.Method {
	case "GET":
		switch r.URL.Path {
		case "/clk":
			err = self.serveStatus(ctx, StatusTrackClk, current, r.URL.Query())
		case "/imp":
			err = self.serveStatus(ctx, StatusTrackImp, current, r.URL.Query())
		case "/win":
			ok = true
			err = self.serveStatus(ctx, StatusWin, current, r.URL.Query())
		case "/loss":
			ok = true
			err = self.serveStatus(ctx, StatusLoss, current, r.URL.Query())
		default:
			glog.Errorf("%s: %d", "Not found", http.StatusNotFound)
			return
		}
	case "POST":
		pubStr, ok = strings.CutPrefix(r.URL.Path, "/bid")
		if !ok || (len(pubStr) >= 1 && pubStr[0:1] != "/") {
			glog.Errorf("%s: %d", "Not found", http.StatusNotFound)
			return
		}
		if len(pubStr) >= 1 {
			pubStr = pubStr[1:]
		}
		bidStr, err = io.ReadAll(r.Body)
		if err == nil {
			bid = &openrtb2.BidRequest{}
			err = json.Unmarshal(bidStr, bid)
		}
	case "OPTIONS":
		w.WriteHeader(200)
		return
	default:
		glog.Errorf("%s: %d", "Method not supported", http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusBadRequest)
		return
	}
	if ok {
		w.WriteHeader(200)
		return
	}

	attr, err := match.NewAttribute(ctx, self.Ips, bid, self.RPubMap, current, pubStr)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	width, height := match.SizeID1To2(attr.SizeID)

	what := "Web"
	if attr.IsApp {
		what = "App"
	}
	monitors, err := match.RAdvsFromRedis(ctx, self.Redis, what, attr.RPub.SlotID)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}

	userID := attr.UserID
	if userID == "" {
		userID = attr.IFA
	}
	candidates, bothcaps, err := monitors.FilterByCaps(ctx, self.Redis, current, userID)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}

	radvs, audiences, err := candidates.FilterByAudiences(ctx, self.Redis, attr)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	if len(radvs) == 0 {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", "No ad to show", http.StatusNoContent)
		return
	}

	index := radvs.PickIndex(bid.Imp[0].BidFloor, bid.Imp[0].BidFloorCur)
	if index < 0 {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", "No ad to show", http.StatusNoContent)
		return
	}

	one := radvs[index]
	bothcap := bothcaps[one.ItemID]
	bidID := match.NewBid(current, userID).BidID()
	winloss := NewWinLoss(current, StatusBid, attr.RPub, one, bothcap, bid.ID, bidID, bid.Imp[0].ID)

	rspnsBid, err := self.bidSeatBid(ctx, bid, one, audiences[one.ItemID], winloss, attr, width, height)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}

	response := &openrtb2.BidResponse{
		ID:    bid.ID,
		BidID: bidID,
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat:  fmt.Sprintf("%d", one.CampaignID),
			Group: 0,
			Bid:   []openrtb2.Bid{rspnsBid},
		}},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	rspnStr, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(204)
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(rspnStr)

	if err = self.Nc.Publish(SUBJECTRequest, bidStr); err == nil {
		if err = self.Nc.Publish(SUBJECTResponse, rspnStr); err == nil {
			if bidStr, err = json.Marshal(AttributePlus{
				Attribute: *attr,
				Adv:       one,
			}); err == nil {
				err = self.Nc.Publish(SUBJECTAttribute, bidStr)
				self.Nc.Flush()
			}
		}
	}
	if err != nil {
		glog.Errorf("%s: %d", err.Error(), http.StatusInternalServerError)
	}
}

// bidSeatBid returns the SeatBid for the bid response.
func (self *Controller) bidSeatBid(ctx context.Context, bidRequest *openrtb2.BidRequest, one match.RAdv, audience *match.Audience, winloss *WinLoss, attr *match.Attribute, width, height uint16) (openrtb2.Bid, error) {
	adm, err := self.admFromCreative(ctx, attr, winloss, one.CreativeID, width, height)
	if err != nil {
		return openrtb2.Bid{}, err
	}
	rspnsBid := openrtb2.Bid{
		ID:      NewSeatBidBid(attr.When, one.CreativeID).Pack(),
		ImpID:   bidRequest.Imp[0].ID,
		Price:   float64(one.Cost),
		NURL:    self.C.ServerURL + "/win?" + winloss.GetURLString(),
		LURL:    self.C.ServerURL + "/loss?" + winloss.GetURLString(),
		AdM:     adm,
		AdID:    fmt.Sprintf("%d", one.CreativeID),
		ADomain: []string{audience.AdvStr},
		Bundle:  audience.AppStr,
		CID:     fmt.Sprintf("%d", one.CampaignID),
		CrID:    fmt.Sprintf("%d", one.CreativeID),
		Cat:     audience.Categories,
		W:       int64(width),
		H:       int64(height),
	}

	return rspnsBid, nil
}

func (self *Controller) admFromCreative(ctx context.Context, attr *match.Attribute, winloss *WinLoss, creativeID uint32, width, height uint16) (string, error) {
	creative, err := match.CreativeFromRedis(ctx, self.Redis, creativeID)
	if err != nil {
		return "", err
	}

	str := winloss.GetURLString(true)
	trackers := []string{
		self.C.ServerURL + "/imp?" + str,
		self.C.ServerURL + "/clk?" + str,
	}

	if attr.NativeFormat != nil || attr.IsApp {
		return match.DefaultImgNative(creative.CreativeContent, creative.CreativeName, width, height).AdM(trackers...)
	} else if attr.IsVideo {
		return match.DefaultVideoNative(creative.CreativeContent).AdM(trackers...)
	}

	return fmt.Sprintf(`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" marginheight="0" marginwidth="0" topmargin="0" leftmargin="0"></iframe>`, creative.CreativeContent, width, height), nil
}
