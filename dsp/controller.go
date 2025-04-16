package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/maxmind"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"
)

type Controller struct {
	C       *Config
	Ips     *maxmind.IPSearch
	Redis   radix.Client
	DB      *sql.DB
	Nc      *nats.Conn
	Logger  *zap.Logger
	IsLocal bool
}

func NewController(ctx context.Context, filename string, offline ...string) (*Controller, error) {
	c, err := NewConfig(filename)
	if err != nil {
		return nil, err
	}
	redis, db, err := c.GetRedisDB(ctx)
	if err != nil {
		return nil, err
	}
	controller := &Controller{
		C:     c,
		Redis: redis,
		DB:    db,
	}

	if len(offline) == 0 || offline[0] == "nats" {
		nc, err := nats.Connect(c.NatsURL, nats.ReconnectWait(10*time.Second))
		if err != nil {
			return nil, err
		}
		controller.Nc = nc
	}

	if len(offline) == 0 || offline[0] == "maxmind" {
		ips, err := maxmind.LoadIPData(c.Ips)
		if err != nil {
			return nil, err
		}
		controller.Ips = ips
	}

	return controller, nil
}

// Close closes the Controller.
func (self *Controller) Close() {
	if self.Redis != nil {
		self.Redis.Close()
	}
	if self.DB != nil {
		self.DB.Close()
	}
	if self.Nc != nil {
		self.Nc.Close()
	}
	if self.Logger != nil {
		self.Logger.Sync()
	}
}

func (self *Controller) ServeBid(w http.ResponseWriter, r *http.Request) {
	glog := self.Logger.Sugar()
	glog.Info("0: initial")

	current := time.Now()
	ctx := r.Context()
	c := self.C

	bidStr, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Infof("%v", err)
		return
	}
	r.Body.Close()

	bid := &openrtb2.BidRequest{}
	if err = json.Unmarshal(bidStr, bid); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		glog.Infof("%v", err)
		return
	}

	pubStr, pubObj, err := self.getPub(ctx, r, bid)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}

	glog.Infof("1: bid domain: %s => pub id: %d", pubStr, pubObj.PubID)
	attr, err := match.NewAttribute(ctx, self.Ips, bid, pubObj, current, pubStr)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}
	width, height := match.SizeID1To2(attr.SizeID)

	glog.Infof("2: size %d (%dx%d), siteID: %d, slotID: %d", attr.SizeID, width, height, attr.RPub.SiteID, attr.SlotID)
	var monitors match.RAdvs
	top := c.Spread
	if c.IsLocal {
		monitors, err = match.RAdvsFromIOBySizeIDSlotID(top, attr.SizeID, attr.RPub.SlotID)
	} else {
		monitors, err = match.RAdvsFromRedisBySizeIDSlotID(ctx, self.Redis, attr.SizeID, attr.RPub.SlotID)
	}
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}
	if len(monitors) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("no ad for slot %d and size %d", attr.RPub.SlotID, attr.SizeID)
		return
	}

	userID := attr.UserID
	glog.Infof("4: total # of candidates: %d", len(monitors))
	candidates, bothcaps, err := monitors.FilterByCaps(ctx, self.Redis, current, userID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}
	if len(candidates) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("no ad after fcap for user %s", userID)
		return
	}

	glog.Infof("5: total # after fcap: %d", len(candidates))
	var audiences match.Audiences
	if c.IsLocal {
		audiences, err = candidates.AudiencesFromIO(top)
	} else {
		audiences, err = candidates.AudiencesFromRedis(ctx, self.Redis)
	}
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}
	radvs, audiences, err := candidates.FilterByAudiences(audiences, attr)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}
	if len(radvs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("no ad after matching audience")
		return
	}

	glog.Infof("6: total # after audience: %d and audieces # %d", len(radvs), len(audiences))
	index := radvs.PickIndex(bid.Imp[0].BidFloor, bid.Imp[0].BidFloorCur)
	if index < 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("no ad to match for bid floor %f %s", bid.Imp[0].BidFloor, bid.Imp[0].BidFloorCur)
		return
	}

	one := radvs[index]
	glog.Infof("7: index %d and campaign %d item %d creative %d", index, one.CampaignID, one.ItemID, one.CreativeID)
	var bothcap *match.BothCap
	if bothcaps != nil {
		if b, ok := bothcaps[one.ItemID]; ok {
			bothcap = &b
		}
	}
	var creative *match.Creative
	if c.IsLocal {
		creative, err = match.CreativeFromIO(top, one.CreativeID)
	} else {
		creative, err = match.CreativeFromRedis(ctx, self.Redis, one.CreativeID)
	}
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}

	dsp := NewDSP(bid, attr, one, bothcap, creative, audiences[index], self.C.ServerURL)
	winloss := dsp.WinLoss(StatusBid)

	glog.Info("8: BidSeat Bid")
	rspnsBid, err := dsp.NewBid(winloss)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}

	glog.Info("9: rspnsBid")
	response := &openrtb2.BidResponse{
		ID:    bid.ID,
		BidID: dsp.bidID(),
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat:  dsp.seat(),
			Group: 0,
			Bid:   []openrtb2.Bid{rspnsBid},
		}},
	}

	rspnStr, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%v", err)
		return
	}

	glog.Info("10: response")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rspnStr)

	glog.Info("11: publish to nats")
	elapsed := time.Since(current)
	if err = self.Nc.Publish(SUBJECTRequest, bidStr); err == nil {
		if err = self.Nc.Publish(SUBJECTResponse, rspnStr); err == nil {
			if bidStr, err = json.Marshal(match.AttributePlus{
				Attribute: *attr,
				RAdv:      one,
				Elapsed:   time.Duration(elapsed.Milliseconds()),
			}); err == nil {
				err = self.Nc.Publish(SUBJECTAttribute, bidStr)
				self.Nc.Flush()
			}
		}
	}
	if err != nil {
		glog.Infof("nats error: %v", err)
	}
}

func (self *Controller) ServeWinLoss(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	current := time.Now()
	status := StatusBid
	switch r.URL.Path {
	case "/win":
		status = StatusWin
	case "/loss":
		status = StatusLoss
	case "/imp":
		status = StatusTrackImp
	case "/clk":
		status = StatusTrackClk
	default:
		http.Error(w, "Invalid path", http.StatusNotFound)
		return
	}

	if err := self.serveStatus(ctx, status, current, r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// serverStatus sends the win, loss, impression and click trackers, refresh cap, and notify the NATS server.
func (self *Controller) serveStatus(ctx context.Context, status Status, current time.Time, args url.Values) error {
	var err error
	wl := &WinLoss{
		Current:      current,
		Status:       status,
		AuctionID:    args.Get("auction_id"),
		AuctionBidID: args.Get("auction_bid_id"),
		AuctionImpID: args.Get("auction_imp_id"),
	}

	demand := args.Get("demand")
	supply := args.Get("supply")
	if demand != "" {
		wl.RAdv.Demand, err = match.UnpackDemandString(demand)
		if err != nil {
			return err
		}
	}
	if supply != "" {
		wl.RPub, err = match.UnpackRPubString(supply)
		if err != nil {
			return err
		}
	}

	price, err := strconv.ParseFloat(args.Get("auction_price"), 64)
	if err != nil {
		return err
	}
	wl.RAdv.Cost = float32(price)
	if v := args.Get("auction_currency"); v == "USD" {
		wl.RAdv.CostType = 1
	}

	switch status {
	case StatusTrackClk, StatusTrackImp:
		u := args.Get("cap")
		if u == "" {
			break
		}
		if wl.RAdv.Cap, err = match.UnpackCapString(u); err == nil {
			var bid bidID
			if bid, err = UnpackBidID(wl.AuctionBidID); err == nil {
				err = match.MustRefreshBothCap(ctx, self.Redis, current, bid.UserID, wl.RAdv.ItemID, status == StatusTrackImp, status == StatusTrackClk)
			}
		}
		if err != nil {
			return err
		}
	default:
	}

	bs, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	return self.Nc.Publish(SUBJECTWinLoss, bs)
}

// getPub returns the publisher string and object from the bid request
func (self *Controller) getPub(ctx context.Context, r *http.Request, bid *openrtb2.BidRequest) (string, *acl.Pub, error) {
	pubStr := r.PathValue("domain")
	if pubStr == "" {
		pubStr = acl.PUBDefault
	}
	var pubObj *acl.Pub
	var err error
	top := self.C.Spread
	if self.C.IsLocal {
		if pubObj, err = acl.PubFromIO(top, pubStr); err == nil && pubObj == nil {
			pubObj, err = acl.PubFromIO(top, acl.PUBDefault)
		}
	} else {
		if pubObj, err = acl.PubFromRedis(ctx, self.Redis, pubStr); err == nil && pubObj == nil {
			pubObj, err = acl.PubFromRedis(ctx, self.Redis, acl.PUBDefault)
		}
	}
	return pubStr, pubObj, err
}
