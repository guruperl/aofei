package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"

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
	C      *Config
	Ips    *maxmind.IPSearch
	Redis  radix.Client
	DB     *sql.DB
	Nc     *nats.Conn
	PubMap match.PubMap
	Logger *zap.Logger
}

func NewController(ctx context.Context, filename string, offline ...string) (*Controller, error) {
	c, err := NewConfig(filename)
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

	controller := &Controller{
		C:     c,
		Redis: redis,
		DB:    db,
	}

	if len(offline) == 0 || offline[0] == "nats" {
		nc, err := nats.Connect(c.NatsURL)
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

	if len(offline) == 0 || offline[0] == "pubmap" {
		pubMap := make(match.PubMap)
		bs, err := os.ReadFile(c.RPubMap)
		if err != nil {
			return nil, err
		}
		if bs != nil {
			err = json.Unmarshal(bs, &pubMap)
			if err != nil {
				return nil, err
			}
		}
		controller.PubMap = pubMap
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (self *Controller) ServeBid(w http.ResponseWriter, r *http.Request) {
	glog := self.Logger.Sugar()
	glog.Info("0: initial")

	current := time.Now()
	ctx := r.Context()

	bidStr, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	bid := &openrtb2.BidRequest{}
	if err = json.Unmarshal(bidStr, bid); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		glog.Infof("%s: %d", err.Error(), http.StatusBadRequest)
		return
	}

	pubStr := r.PathValue("domain")
	if pubStr == "" {
		pubStr = match.PUBDefault
	}
	pubObj, err := self.getPubObj(ctx, pubStr)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}

	glog.Info("1: bid")
	attr, err := match.NewAttribute(ctx, self.Ips, bid, pubObj, current, pubStr)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	width, height := match.SizeID1To2(attr.SizeID)

	glog.Infof("2: width %d, height %d", width, height)
	glog.Infof("3: rpub %#v", attr.RPub)

	monitors, err := match.RAdvsFromRedis(ctx, self.Redis, attr.RPub.SlotID, attr.SizeID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	if len(monitors) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", "no ad", http.StatusNoContent)
		return
	}

	userID := attr.UserID
	if userID == "" {
		userID = attr.IFA
	}
	glog.Infof("4: userID %s => total # %d => %#v", userID, len(monitors), monitors[0])
	candidates, bothcaps, err := monitors.FilterByCaps(ctx, self.Redis, current, userID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	if len(candidates) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", "no ad", http.StatusNoContent)
		return
	}

	glog.Infof("5: total # after cap %d", len(candidates))
	radvs, audiences, err := candidates.FilterByAudiences(ctx, self.Redis, attr)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	if len(radvs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", "no ad", http.StatusNoContent)
		return
	}

	glog.Infof("6: radvs # %d, audieces # %d", len(radvs), len(audiences))
	index := radvs.PickIndex(bid.Imp[0].BidFloor, bid.Imp[0].BidFloorCur)
	if index < 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", "no ad", http.StatusNoContent)
		return
	}

	glog.Infof("7: index %d", index)
	one := radvs[index]
	var bothcap *match.BothCap
	if bothcaps != nil {
		if b, ok := bothcaps[one.ItemID]; ok {
			bothcap = &b
		}
	}

	bidID := match.NewBid(current, userID).BidID()
	winloss := NewWinLoss(current, StatusBid, attr.RPub, one, bothcap, bid.ID, bidID, bid.Imp[0].ID)

	glog.Info("8: winloss")
	rspnsBid, err := self.bidSeatBid(ctx, bid, one, audiences[index], winloss, attr, width, height)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}

	glog.Info("9: rspnsBid")
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
	w.WriteHeader(http.StatusOK)
	rspnStr, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(rspnStr)
	elapsed := time.Since(current)

	glog.Info("10: response")
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
		glog.Infof("%s: %d", err.Error(), http.StatusInternalServerError)
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
		NURL:    self.C.ServerURL + "/win?" + winloss.PackURLString(),
		LURL:    self.C.ServerURL + "/loss?" + winloss.PackURLString(),
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

	str := winloss.PackURLString(true)
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

// PubRedisName return the redis name for the PubMap.
func PubRedisName(fullname string) string {
	arr := strings.SplitN(filepath.Base(fullname), ".", -1)
	return arr[0]
}

// getPubObj returns the Pub object from the PubMap.
func (self *Controller) getPubObj(ctx context.Context, pubStr string) (*match.Pub, error) {
	var pubObj *match.Pub
	// note that we embed pubmap in the controller after restart, but also dynamically update it for each request.
	var bs []byte
	name := PubRedisName(self.C.RPubMap)
	err := self.Redis.Do(ctx, radix.Cmd(&bs, "MGET", name, pubStr))
	glog := self.Logger.Sugar()
	glog.Infof("getPubObj: %s, %s error %v bs %s", name, pubStr, err, bs)
	if err == nil && len(bs) > 2 {
		glog.Infof("getPubObj: bs %d", len(bs))
		pubObj, err = match.UnpackPub(bs)
	}
	if err != nil {
		return nil, err
	}
	if pubObj == nil {
		pubObj = self.PubMap[pubStr]
		if pubObj == nil {
			pubObj = self.PubMap[match.PUBDefault]
		}
	}
	glog.Infof("PubMap %#v: pubObj %#v", self.PubMap, pubObj)
	return pubObj, nil
}
