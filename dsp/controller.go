package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
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

	maxBidRequestBodyBytes = 1 << 20
)

type Controller struct {
	C                  *Config
	Ips                *maxmind.IPSearch
	Redis              radix.Client
	DB                 *sql.DB
	Nc                 *nats.Conn
	Logger             *zap.Logger
	IsLocal            bool
	localMu            sync.Mutex
	local              *localStaticCache
	audit              *auditPublisher
	client             *http.Client
	middlemanStore     middlemanCallbackStore
	publishWinLossFunc func([]byte) error
}

type controllerOptions struct {
	nats    bool
	maxmind bool
}

type bidAudit struct {
	Attr    *match.Attribute
	One     match.RAdv
	Elapsed time.Duration
}

type localNoBidError struct {
	err error
}

func (e localNoBidError) Error() string {
	return e.err.Error()
}

func (e localNoBidError) Unwrap() error {
	return e.err
}

func noBidErrorf(format string, args ...any) error {
	return localNoBidError{err: fmt.Errorf(format, args...)}
}

func isLocalNoBid(err error) bool {
	var noBid localNoBidError
	return errors.As(err, &noBid)
}

// ControllerOption configures optional external services for a Controller.
type ControllerOption func(*controllerOptions)

// WithoutNATS disables the Controller's NATS connection.
func WithoutNATS() ControllerOption {
	return func(opts *controllerOptions) {
		opts.nats = false
	}
}

// WithoutMaxMind disables loading the MaxMind IP search data.
func WithoutMaxMind() ControllerOption {
	return func(opts *controllerOptions) {
		opts.maxmind = false
	}
}

func NewController(ctx context.Context, filename string, offline ...string) (*Controller, error) {
	if len(offline) == 0 {
		return NewControllerWithOptions(ctx, filename)
	}
	switch offline[0] {
	case "nats":
		return NewControllerWithOptions(ctx, filename, WithoutMaxMind())
	case "maxmind":
		return NewControllerWithOptions(ctx, filename, WithoutNATS())
	default:
		return NewControllerWithOptions(ctx, filename, WithoutNATS(), WithoutMaxMind())
	}
}

func NewControllerWithOptions(ctx context.Context, filename string, opts ...ControllerOption) (*Controller, error) {
	c, err := NewConfig(filename)
	if err != nil {
		return nil, err
	}
	redis, db, err := c.GetRedisDB(ctx)
	if err != nil {
		return nil, err
	}
	controller := &Controller{
		C:      c,
		Redis:  redis,
		DB:     db,
		local:  newLocalStaticCache(),
		client: http.DefaultClient,
	}
	if c.IsLocal {
		if err := controller.ReloadLocalStaticCache(); err != nil {
			return nil, err
		}
	}

	options := applyControllerOptions(opts...)
	if options.nats {
		nc, err := nats.Connect(c.NatsURL, nats.ReconnectWait(10*time.Second))
		if err != nil {
			return nil, err
		}
		controller.Nc = nc
		controller.audit = newAuditPublisher(nc, defaultAuditQueueSize)
	}

	if options.maxmind {
		ips, err := maxmind.LoadIPData(c.Ips)
		if err != nil {
			return nil, err
		}
		controller.Ips = ips
	}

	return controller, nil
}

func applyControllerOptions(opts ...ControllerOption) controllerOptions {
	options := controllerOptions{nats: true, maxmind: true}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// Close closes the Controller.
func (self *Controller) Close() {
	if self.audit != nil {
		self.audit.Close()
	}
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
	logger := self.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	glog := logger.Sugar()
	glog.Info("0: initial")

	current := time.Now()
	ctx := r.Context()

	bidStr, err := io.ReadAll(io.LimitReader(r.Body, maxBidRequestBodyBytes+1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Infof("%v", err)
		return
	}
	r.Body.Close()
	if len(bidStr) > maxBidRequestBodyBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		glog.Infof("bid request body exceeds %d bytes", maxBidRequestBodyBytes)
		return
	}

	bid := &openrtb2.BidRequest{}
	if err = json.Unmarshal(bidStr, bid); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		glog.Infof("%v", err)
		return
	}
	if err = validateBidRequest(bid); err != nil {
		w.WriteHeader(http.StatusNoContent)
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
	var audits []bidAudit
	var fallbackImps []middlemanFallbackImp
	var seatOrder []string
	seatBids := make(map[string][]openrtb2.Bid)
	responseBidID := ""
	for impIndex := range bid.Imp {
		dspBid, audit, err := self.bidForImp(ctx, bid, pubObj, current, pubStr, impIndex)
		if err != nil {
			glog.Infof("imp %d: %v", impIndex, err)
			if audit.Attr != nil && isLocalNoBid(err) {
				fallbackImps = append(fallbackImps, middlemanFallbackImp{Index: impIndex, Attr: audit.Attr})
			}
			continue
		}
		seat := dspBid.seat()
		if _, ok := seatBids[seat]; !ok {
			seatOrder = append(seatOrder, seat)
		}
		winloss := dspBid.WinLoss(StatusBid)
		rspnsBid, err := dspBid.NewBid(winloss)
		if err != nil {
			glog.Infof("imp %d: %v", impIndex, err)
			continue
		}
		if responseBidID == "" {
			responseBidID = dspBid.bidID()
		}
		seatBids[seat] = append(seatBids[seat], rspnsBid)
		audits = append(audits, audit)
	}
	if len(fallbackImps) != 0 {
		middlemanBids, err := self.middlemanFallback(ctx, bid, bidStr, current, fallbackImps)
		if err != nil {
			glog.Infof("middleman fallback: %v", err)
		}
		for _, middlemanBid := range middlemanBids {
			seat := middlemanBid.Seat
			if _, ok := seatBids[seat]; !ok {
				seatOrder = append(seatOrder, seat)
			}
			if responseBidID == "" {
				responseBidID = middlemanBid.ResponseBidID
			}
			seatBids[seat] = append(seatBids[seat], middlemanBid.Bid)
			audits = append(audits, middlemanBid.Audit)
		}
	}
	if len(audits) == 0 {
		w.WriteHeader(http.StatusNoContent)
		glog.Infof("no impression produced a bid")
		return
	}

	var responseSeatBids []openrtb2.SeatBid
	for _, seat := range seatOrder {
		responseSeatBids = append(responseSeatBids, openrtb2.SeatBid{
			Seat:  seat,
			Group: 0,
			Bid:   seatBids[seat],
		})
	}
	response := &openrtb2.BidResponse{
		ID:      bid.ID,
		BidID:   responseBidID,
		Cur:     "USD",
		SeatBid: responseSeatBids,
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
	for i := range audits {
		audits[i].Elapsed = elapsed
	}
	if self.Nc == nil {
		glog.Info("nats unavailable: skipped bid audit publish")
	} else if err = self.publishBidAudits(bidStr, rspnStr, audits); err != nil {
		glog.Infof("nats error: %v", err)
	}
}

func (self *Controller) bidForImp(ctx context.Context, bid *openrtb2.BidRequest, pubObj *acl.Pub, current time.Time, pubStr string, impIndex int) (*DSP, bidAudit, error) {
	attr, err := match.NewAttributeForImp(ctx, self.Ips, bid, impIndex, pubObj, current, pubStr)
	if err != nil {
		return nil, bidAudit{}, err
	}
	audit := bidAudit{Attr: attr}
	var monitors match.RAdvs
	top := self.C.Spread
	if self.C.IsLocal {
		monitors, err = self.localRAdvs(top, attr.SizeID, attr.RPub.SlotID)
	} else {
		monitors, err = match.RAdvsFromRedisBySizeIDSlotID(ctx, self.Redis, attr.SizeID, attr.RPub.SlotID)
	}
	if err != nil {
		return nil, audit, err
	}
	if len(monitors) == 0 {
		return nil, audit, noBidErrorf("no ad for slot %d and size %d", attr.RPub.SlotID, attr.SizeID)
	}
	if self.Redis == nil && radvsNeedCaps(monitors) {
		return nil, audit, fmt.Errorf("redis mutable state unavailable for frequency caps")
	}

	candidates, bothcaps, err := monitors.FilterByCaps(ctx, self.Redis, current, attr.UserID)
	if err != nil {
		return nil, audit, err
	}
	if len(candidates) == 0 {
		return nil, audit, noBidErrorf("no ad after fcap for user %s", attr.UserID)
	}

	var audiences match.Audiences
	if self.C.IsLocal {
		audiences, err = self.localAudiences(top, candidates)
	} else {
		audiences, err = candidates.AudiencesFromRedis(ctx, self.Redis)
	}
	if err != nil {
		return nil, audit, err
	}
	if self.Redis == nil && audiencesNeedUploads(audiences) {
		return nil, audit, fmt.Errorf("redis mutable state unavailable for uploaded audiences")
	}

	radvs, auds, err := self.priorityAudienceMatches(ctx, bid, candidates, audiences, attr)
	if err != nil {
		return nil, audit, err
	}
	if len(radvs) == 0 {
		return nil, audit, noBidErrorf("no ad after matching audience")
	}

	imp := bid.Imp[impIndex]
	index, bidPrice := radvs.PickIndexPrice(imp.BidFloor, imp.BidFloorCur)
	if index < 0 {
		return nil, audit, noBidErrorf("no ad to match for bid floor %f %s", imp.BidFloor, imp.BidFloorCur)
	}

	one := radvs[index]
	var bothcap *match.BothCap
	if bothcaps != nil {
		if b, ok := bothcaps[one.ItemID]; ok {
			bothcap = &b
		}
	}
	var creative *match.Creative
	if self.C.IsLocal {
		creative, err = self.localCreative(top, one.CreativeID)
	} else {
		creative, err = match.CreativeFromRedis(ctx, self.Redis, one.CreativeID)
	}
	if err != nil {
		return nil, audit, err
	}

	dspBid := NewDSPForImp(bid, impIndex, attr, one, bothcap, creative, auds[index], bidPrice, self.C.ServerURL).
		WithTrackingSecret(self.C.TrackingSecret)
	return dspBid, bidAudit{Attr: attr, One: one}, nil
}

func radvsNeedCaps(radvs match.RAdvs) bool {
	for _, radv := range radvs {
		if radv.CapNumber != 0 || radv.ClickNumber != 0 {
			return true
		}
	}
	return false
}

func audiencesNeedUploads(audiences match.Audiences) bool {
	for _, aud := range audiences {
		if aud != nil && aud.UploadAudience != nil && aud.UploadAudience.Uploads != 0 {
			return true
		}
	}
	return false
}

func (self *Controller) priorityAudienceMatches(ctx context.Context, bid *openrtb2.BidRequest, candidates match.RAdvs, audiences match.Audiences, attr *match.Attribute) (match.RAdvs, match.Audiences, error) {
	var radvs match.RAdvs
	var auds match.Audiences
	for i, candidate := range candidates {
		aud := audiences[i]
		if aud == nil || aud.UploadAudience == nil || aud.UploadAudience.Uploads == 0 {
			continue
		}
		ok, err := aud.UploadAudience.Has(ctx, self.Redis, bid, candidate.AdvID)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			radvs = append(radvs, candidate)
			auds = append(auds, aud)
		}
	}
	if len(radvs) != 0 {
		return radvs, auds, nil
	}
	return candidates.FilterByAudiences(ctx, self.Redis, bid, audiences, attr)
}

func validateBidRequest(bid *openrtb2.BidRequest) error {
	if bid == nil {
		return fmt.Errorf("bid request is nil")
	}
	if len(bid.Imp) == 0 {
		return fmt.Errorf("bid request has no impressions")
	}
	if bid.Device == nil {
		return fmt.Errorf("bid request has no device")
	}
	return nil
}

func (self *Controller) publishWinLoss(bs []byte) error {
	if self.publishWinLossFunc != nil {
		return self.publishWinLossFunc(bs)
	}
	if self.Nc == nil {
		return nil
	}
	return self.Nc.Publish(SUBJECTWinLoss, bs)
}

func (self *Controller) publishBidAudit(bidStr, rspnStr []byte, attr *match.Attribute, one match.RAdv, elapsed time.Duration) error {
	return self.publishBidAudits(bidStr, rspnStr, []bidAudit{{
		Attr:    attr,
		One:     one,
		Elapsed: elapsed,
	}})
}

func (self *Controller) publishBidAudits(bidStr, rspnStr []byte, audits []bidAudit) error {
	if self.Nc == nil && self.audit == nil {
		return nil
	}
	publisher := self.audit
	if publisher == nil {
		publisher = newAuditPublisher(self.Nc, defaultAuditQueueSize)
		self.audit = publisher
	}
	publisher.Enqueue(SUBJECTRequest, bidStr)
	publisher.Enqueue(SUBJECTResponse, rspnStr)
	for _, audit := range audits {
		if audit.Attr == nil {
			continue
		}
		bs, err := json.Marshal(match.AttributePlus{
			Attribute: *audit.Attr,
			RAdv:      audit.One,
			Elapsed:   time.Duration(audit.Elapsed.Milliseconds()),
		})
		if err != nil {
			return err
		}
		publisher.Enqueue(SUBJECTAttribute, bs)
	}
	return nil
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

	if status == StatusTrackClk {
		if target, ok, err := self.clickRedirectTarget(r.URL.Query()); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if ok {
			if err := self.serveStatus(ctx, status, current, r.URL.Query()); err != nil {
				if self.Logger != nil {
					self.Logger.Sugar().Infof("click tracking skipped before redirect: %v", err)
				}
			}
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
	}

	if err := self.serveStatus(ctx, status, current, r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (self *Controller) clickRedirectTarget(args url.Values) (string, bool, error) {
	target := args.Get("redirect")
	if target == "" {
		return "", false, nil
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", true, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", true, fmt.Errorf("invalid click redirect scheme")
	}
	if u.Host == "" {
		return "", true, fmt.Errorf("invalid click redirect host")
	}
	for _, key := range []string{"auction_id", "auction_bid_id", "auction_imp_id", "auction_price", "demand", "supply"} {
		if args.Get(key) == "" {
			return "", true, fmt.Errorf("click redirect missing %s", key)
		}
	}
	if err := validateTrackingSignature(self.trackingSecret(), "/clk", args); err != nil {
		return "", true, err
	}
	return u.String(), true, nil
}

func (self *Controller) trackingSecret() string {
	if self == nil || self.C == nil {
		return ""
	}
	return self.C.TrackingSecret
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
	if err == nil {
		wl.RAdv.Cost = float32(price)
		if v := args.Get("auction_currency"); v == "USD" {
			wl.RAdv.CostType = 1
		}
	} else if status == StatusTrackClk || status == StatusTrackImp {
		return err
	}

	switch status {
	case StatusTrackClk, StatusTrackImp:
		u := args.Get("cap")
		if u == "" {
			break
		}
		if err := validateTrackingSignature(self.trackingSecret(), status.path(), args); err != nil {
			return err
		}
		if wl.RAdv.Cap, err = match.UnpackCapString(u); err == nil {
			var bid bidID
			if bid, err = UnpackBidID(wl.AuctionBidID); err == nil {
				err = match.MustRefreshBothCap(ctx, self.Redis, current, bid.UserID, wl.RAdv.ItemID, wl.RAdv.Cap, status == StatusTrackImp, status == StatusTrackClk)
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
	return self.publishWinLoss(bs)
}

func (self Status) path() string {
	switch self {
	case StatusWin:
		return "/win"
	case StatusLoss:
		return "/loss"
	case StatusTrackImp:
		return "/imp"
	case StatusTrackClk:
		return "/clk"
	default:
		return ""
	}
}

// getPub returns the publisher string and object from the bid request
func (self *Controller) getPub(ctx context.Context, r *http.Request, bid *openrtb2.BidRequest) (string, *acl.Pub, error) {
	pubStr := r.PathValue("domain")
	if pubStr == "" {
		return "", nil, fmt.Errorf("adx not found")
		//pubStr = acl.PUBDefault
	}
	var pubObj *acl.Pub
	var err error
	top := self.C.Spread
	if self.C.IsLocal {
		if pubObj, err = self.localPub(top, pubStr); err == nil && pubObj == nil {
			return "", nil, fmt.Errorf("%s not found", pubStr)
			//pubObj, err = acl.PubFromIO(top, acl.PUBDefault)
		}
	} else {
		if pubObj, err = acl.PubFromRedis(ctx, self.Redis, pubStr); err == nil && pubObj == nil {
			return "", nil, fmt.Errorf("%s not found", pubStr)
			//pubObj, err = acl.PubFromRedis(ctx, self.Redis, acl.PUBDefault)
		}
	}
	if err != nil {
		return "", nil, err
	}

	return pubStr, pubObj, err
}
