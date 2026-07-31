package dsp

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/internal/safehttp"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/maxmind"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"

	maxBidRequestBodyBytes = 1 << 20
)

type Controller struct {
	C                      *Config
	Ips                    *maxmind.IPSearch
	Redis                  radix.Client
	DB                     *sql.DB
	Nc                     *nats.Conn
	Logger                 *zap.Logger
	IsLocal                bool
	localMu                sync.Mutex
	local                  *localStaticCache
	auditMu                sync.Mutex
	audit                  *auditPublisher
	auditClosed            bool
	auditFactory           func(*nats.Conn, int) *auditPublisher
	client                 *http.Client
	middlemanStore         middlemanCallbackStore
	callbackURLGuard       func(context.Context, string) error
	middlemanRuntime       middlemanRuntime
	middlemanRoute         atomic.Pointer[middlemanRouteState]
	middlemanRouteMu       sync.Mutex
	middlemanRouteWait     chan struct{}
	middlemanRouteLoad     func(context.Context) (*match.MiddlemanRouteCache, error)
	middlemanRouteNow      func() time.Time
	middlemanRouteWaitHook func()
	trackingNotifyOnce     func(context.Context, Status, string, time.Duration) (bool, error)
	trackingEventOnce      func(context.Context, Status, url.Values, time.Duration) (bool, error)
	publishWinLossFunc     func([]byte) error
	ownRedis               bool
	ownDB                  bool
	ownNATS                bool
}

type controllerOptions struct {
	nats             bool
	maxmind          bool
	redis            radix.Client
	db               *sql.DB
	nc               *nats.Conn
	ips              *maxmind.IPSearch
	httpClient       *http.Client
	logger           *zap.Logger
	callbackURLGuard func(context.Context, string) error
	callbackStore    middlemanCallbackStore
}

type bidAudit struct {
	Attr    *match.Attribute
	One     match.RAdv
	Elapsed time.Duration
}

type auditSource struct {
	Source   string
	Contract string
}

type auditEnvelope struct {
	Source   string          `json:"source"`
	Contract string          `json:"contract"`
	Payload  json.RawMessage `json:"payload"`
}

var (
	auditSourceADX = auditSource{Source: "adx", Contract: "openrtb"}
	auditSourceSSP = auditSource{Source: "ssp", Contract: "pz-v1"}
)

type bidWinner struct {
	ImpIndex      int
	Seat          string
	Bid           openrtb2.Bid
	Audit         bidAudit
	ResponseBidID string
	ImpressionURL string
	ClickURL      string
	EffectiveCPM  float64
	Comparable    bool
	Local         bool
	Middleman     *middlemanDownstreamBid
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

func WithRedis(redis radix.Client) ControllerOption {
	return func(opts *controllerOptions) {
		opts.redis = redis
	}
}

func WithDB(db *sql.DB) ControllerOption {
	return func(opts *controllerOptions) {
		opts.db = db
	}
}

func WithNATS(nc *nats.Conn) ControllerOption {
	return func(opts *controllerOptions) {
		opts.nc = nc
		opts.nats = false
	}
}

func WithIPSearch(ips *maxmind.IPSearch) ControllerOption {
	return func(opts *controllerOptions) {
		opts.ips = ips
		opts.maxmind = false
	}
}

func WithHTTPClient(client *http.Client) ControllerOption {
	return func(opts *controllerOptions) {
		opts.httpClient = client
	}
}

func WithLogger(logger *zap.Logger) ControllerOption {
	return func(opts *controllerOptions) {
		opts.logger = logger
	}
}

func WithCallbackURLGuard(guard func(context.Context, string) error) ControllerOption {
	return func(opts *controllerOptions) {
		opts.callbackURLGuard = guard
	}
}

func withMiddlemanCallbackStore(store middlemanCallbackStore) ControllerOption {
	return func(opts *controllerOptions) {
		opts.callbackStore = store
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
	if err := c.Validate(ConfigModeBid); err != nil {
		return nil, err
	}
	options := applyControllerOptions(opts...)
	redis := options.redis
	db := options.db
	ownRedis := false
	ownDB := false
	if redis == nil || db == nil {
		openedRedis, openedDB, err := c.GetRedisDB(ctx)
		if err != nil {
			return nil, err
		}
		if redis == nil {
			redis = openedRedis
			ownRedis = true
		} else {
			openedRedis.Close()
		}
		if db == nil {
			db = openedDB
			ownDB = true
		} else {
			openedDB.Close()
		}
	}
	controller := &Controller{
		C:      c,
		Redis:  redis,
		DB:     db,
		local:  newLocalStaticCache(),
		client: safehttp.NewCallbackClient(nil),
		callbackURLGuard: func(ctx context.Context, raw string) error {
			return safehttp.ValidateCallbackURL(ctx, raw)
		},
		Logger:         options.logger,
		middlemanStore: options.callbackStore,
		ownRedis:       ownRedis,
		ownDB:          ownDB,
	}
	if options.httpClient != nil {
		controller.client = options.httpClient
	}
	if options.callbackURLGuard != nil {
		controller.callbackURLGuard = options.callbackURLGuard
	}
	if c.IsLocal {
		if err := controller.ReloadLocalStaticCache(); err != nil {
			return nil, err
		}
	}

	if options.nc != nil {
		controller.Nc = options.nc
		controller.audit = newAuditPublisher(options.nc, defaultAuditQueueSize)
	} else if options.nats {
		nc, err := nats.Connect(c.NatsURL, nats.ReconnectWait(10*time.Second))
		if err != nil {
			return nil, err
		}
		controller.Nc = nc
		controller.audit = newAuditPublisher(nc, defaultAuditQueueSize)
		controller.ownNATS = true
	}

	if options.ips != nil {
		controller.Ips = options.ips
	} else if options.maxmind {
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
	self.auditMu.Lock()
	self.auditClosed = true
	publisher := self.audit
	self.auditMu.Unlock()
	if publisher != nil {
		publisher.Close()
	}
	if self.Redis != nil && self.ownRedis {
		self.Redis.Close()
	}
	if self.DB != nil && self.ownDB {
		self.DB.Close()
	}
	if self.Nc != nil && self.ownNATS {
		self.Nc.Close()
	}
	if self.Logger != nil {
		self.Logger.Sync()
	}
}

func (self *Controller) ServeBid(w http.ResponseWriter, r *http.Request) {
	metricBidRequests.Add(1)
	logger := self.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	current := time.Now()
	ctx := r.Context()

	bidStr, err := io.ReadAll(io.LimitReader(r.Body, maxBidRequestBodyBytes+1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("bid request body read failed", zap.Error(err))
		return
	}
	r.Body.Close()
	if len(bidStr) > maxBidRequestBodyBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		logger.Debug("bid request body rejected",
			zap.Int("body_bytes", len(bidStr)),
			zap.Int("max_body_bytes", maxBidRequestBodyBytes),
		)
		return
	}

	bid := &openrtb2.BidRequest{}
	if err = json.Unmarshal(bidStr, bid); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Debug("malformed bid request rejected", zap.Error(err))
		return
	}
	if err = validateBidRequest(bid); err != nil {
		w.WriteHeader(http.StatusNoContent)
		logger.Debug("invalid bid request rejected",
			zap.String("request_id", bid.ID),
			zap.Error(err),
		)
		return
	}

	pubStr, pubObj, err := self.getPub(ctx, r, bid)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		logger.Debug("bid request publisher rejected",
			zap.String("request_id", bid.ID),
			zap.Error(err),
		)
		return
	}

	finalWinners, localWinners := self.auctionBidWinners(ctx, bid, pubObj, current, pubStr, bidStr, logger)
	seatOrder, seatBids, audits, responseBidID, _ := self.materializeBidWinners(ctx, bid, finalWinners, localWinners, logger)
	if len(audits) == 0 {
		w.WriteHeader(http.StatusNoContent)
		metricBidNoBids.Add(1)
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
		logger.Error("bid response marshal failed",
			zap.String("request_id", bid.ID),
			zap.Error(err),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(rspnStr); err != nil {
		logger.Warn("bid response write failed",
			zap.String("request_id", bid.ID),
			zap.Error(err),
		)
	}
	metricBidResponses.Add(1)

	elapsed := time.Since(current)
	for i := range audits {
		audits[i].Elapsed = elapsed
	}
	if self.Nc != nil {
		if err = self.publishBidAudits(bidStr, rspnStr, audits); err != nil {
			logger.Warn("bid audit publish failed",
				zap.String("request_id", bid.ID),
				zap.Error(err),
			)
		}
	}
}

func (self *Controller) middlemanAlwaysEnabled() bool {
	return self != nil && self.C != nil && self.C.MiddlemanEnabled && self.C.MiddlemanAlwaysEnabled
}

func (self *Controller) middlemanTriggerModesForNoBid() []string {
	return []string{"Fallback"}
}

func (self *Controller) auctionBidWinners(ctx context.Context, bid *openrtb2.BidRequest, pubObj *acl.Pub, current time.Time, pubStr string, rawMiddlemanRequest []byte, logger *zap.Logger) (map[int]bidWinner, map[int]bidWinner) {
	var fallbackImps []middlemanFallbackImp
	localWinners := make(map[int]bidWinner)
	finalWinners := make(map[int]bidWinner)
	for impIndex := range bid.Imp {
		dspBid, audit, err := self.bidForImp(ctx, bid, pubObj, current, pubStr, impIndex)
		if err != nil {
			if logger != nil && !isLocalNoBid(err) {
				logger.Warn("bid impression evaluation failed",
					zap.String("request_id", bid.ID),
					zap.Int("imp_index", impIndex),
					zap.Error(err),
				)
			}
			if audit.Attr != nil && isLocalNoBid(err) {
				fallbackImps = append(fallbackImps, middlemanFallbackImp{
					Index:        impIndex,
					Attr:         audit.Attr,
					TriggerModes: self.middlemanTriggerModesForNoBid(),
				})
			}
			continue
		}
		winloss := dspBid.WinLoss(StatusBid)
		rspnsBid, err := dspBid.NewBid(winloss)
		if err != nil {
			if logger != nil {
				logger.Warn("bid impression materialization failed",
					zap.String("request_id", bid.ID),
					zap.Int("imp_index", impIndex),
					zap.Error(err),
				)
			}
			continue
		}
		effectiveCPM, comparable := localBidEffectiveCPM(dspBid, audit, rspnsBid)
		winner := bidWinner{
			ImpIndex:      impIndex,
			Seat:          dspBid.seat(),
			Bid:           rspnsBid,
			Audit:         audit,
			ResponseBidID: dspBid.bidID(),
			ImpressionURL: winloss.ImpURL(),
			ClickURL:      clickURLForSSPBid(dspBid, winloss),
			EffectiveCPM:  effectiveCPM,
			Comparable:    comparable,
			Local:         true,
		}
		localWinners[impIndex] = winner
		finalWinners[impIndex] = winner
		if self.middlemanAlwaysEnabled() && comparable && audit.Attr != nil {
			fallbackImps = append(fallbackImps, middlemanFallbackImp{
				Index:        impIndex,
				Attr:         audit.Attr,
				TriggerModes: []string{"Always"},
			})
		}
	}
	if len(fallbackImps) != 0 {
		middlemanBids, err := self.middleman().Fallback(ctx, bid, rawMiddlemanRequest, current, fallbackImps)
		if err != nil {
			if logger != nil {
				logger.Warn("middleman fallback failed",
					zap.String("request_id", bid.ID),
					zap.Error(err),
				)
			}
		}
		for _, middlemanBid := range middlemanBids {
			current, ok := finalWinners[middlemanBid.ImpIndex]
			if winner, replace := chooseMiddlemanWinner(current, ok, middlemanBid); replace {
				finalWinners[middlemanBid.ImpIndex] = winner
			}
		}
	}
	return finalWinners, localWinners
}

func localBidEffectiveCPM(dspBid *DSP, audit bidAudit, bid openrtb2.Bid) (float64, bool) {
	if dspBid == nil {
		return 0, false
	}
	_, ok := audit.One.ECPM()
	if !ok {
		metricECPMErrors.Add(1)
		return 0, false
	}
	return bid.Price, true
}

func chooseMiddlemanWinner(current bidWinner, hasCurrent bool, middlemanBid middlemanDownstreamBid) (bidWinner, bool) {
	if hasCurrent && current.Local && (!current.Comparable || middlemanBid.Bid.Price <= current.EffectiveCPM) {
		return current, false
	}
	mb := middlemanBid
	return bidWinner{
		ImpIndex:      mb.ImpIndex,
		Seat:          mb.Seat,
		Bid:           mb.Bid,
		Audit:         mb.Audit,
		ResponseBidID: mb.ResponseBidID,
		EffectiveCPM:  mb.Bid.Price,
		Comparable:    true,
		Local:         false,
		Middleman:     &mb,
	}, true
}

func (self *Controller) materializeBidWinners(ctx context.Context, bid *openrtb2.BidRequest, winners map[int]bidWinner, localWinners map[int]bidWinner, logger *zap.Logger) ([]string, map[string][]openrtb2.Bid, []bidAudit, string, []bidWinner) {
	var seatOrder []string
	seatBids := make(map[string][]openrtb2.Bid)
	var audits []bidAudit
	var materialized []bidWinner
	responseBidID := ""
	for impIndex := range bid.Imp {
		winner, ok := winners[impIndex]
		if !ok {
			continue
		}
		if !winner.Local {
			if winner.Middleman == nil {
				continue
			}
			middlemanBid := *winner.Middleman
			if err := self.prepareMiddlemanCallback(ctx, bid, &middlemanBid); err != nil {
				metricMiddlemanCallbackSetupFailures.Add(1)
				if logger != nil {
					logger.Warn("middleman callback setup failed",
						zap.String("request_id", bid.ID),
						zap.Int("imp_index", impIndex),
						zap.Uint32("bidder_id", middlemanBid.Entry.BidderID),
						zap.Error(err),
					)
				}
				local, ok := localWinners[impIndex]
				if !ok {
					continue
				}
				winner = local
			} else {
				winner.Bid = middlemanBid.Bid
				winner.Audit = middlemanBid.Audit
				winner.ResponseBidID = middlemanBid.ResponseBidID
				winner.Seat = middlemanBid.Seat
			}
		}
		if _, ok := seatBids[winner.Seat]; !ok {
			seatOrder = append(seatOrder, winner.Seat)
		}
		if responseBidID == "" {
			responseBidID = winner.ResponseBidID
		}
		seatBids[winner.Seat] = append(seatBids[winner.Seat], winner.Bid)
		audits = append(audits, winner.Audit)
		materialized = append(materialized, winner)
	}
	return seatOrder, seatBids, audits, responseBidID, materialized
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
	return self.publishBidAuditsFor(auditSourceADX, bidStr, rspnStr, audits)
}

func (self *Controller) publishSSPBidAudits(bidStr, rspnStr []byte, audits []bidAudit) error {
	request, err := wrapAuditPayload(auditSourceSSP, bidStr)
	if err != nil {
		return err
	}
	response, err := wrapAuditPayload(auditSourceSSP, rspnStr)
	if err != nil {
		return err
	}
	return self.publishBidAuditsFor(auditSourceSSP, request, response, audits)
}

func wrapAuditPayload(source auditSource, payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		payload = []byte("null")
	}
	return json.Marshal(auditEnvelope{
		Source:   source.Source,
		Contract: source.Contract,
		Payload:  json.RawMessage(payload),
	})
}

func (self *Controller) publishBidAuditsFor(source auditSource, bidStr, rspnStr []byte, audits []bidAudit) error {
	publisher := self.bidAuditPublisher()
	if publisher == nil {
		return nil
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
			Elapsed:   audit.Elapsed.Milliseconds(),
			Source:    source.Source,
			Contract:  source.Contract,
		})
		if err != nil {
			return err
		}
		publisher.Enqueue(SUBJECTAttribute, bs)
	}
	return nil
}

func (self *Controller) bidAuditPublisher() *auditPublisher {
	self.auditMu.Lock()
	defer self.auditMu.Unlock()
	if self.auditClosed {
		return nil
	}
	if self.audit != nil {
		return self.audit
	}
	if self.Nc == nil {
		return nil
	}
	factory := self.auditFactory
	if factory == nil {
		factory = newAuditPublisher
	}
	self.audit = factory(self.Nc, defaultAuditQueueSize)
	return self.audit
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
					self.Logger.Debug("click tracking skipped before redirect", zap.Error(err))
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
	if _, err := validateTrackingSignature(self.trackingSecret(), "/clk", args, self.trackingSignatureTTL()); err != nil {
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
	var signatureValidUntil time.Time
	switch status {
	case StatusTrackClk, StatusTrackImp, StatusWin, StatusLoss:
		var err error
		signatureValidUntil, err = validateTrackingSignature(self.trackingSecret(), status.path(), args, self.trackingSignatureTTL())
		if err != nil {
			return err
		}
	}

	var err error
	var replayClaim trackingEventClaim
	releaseReplayClaim := false
	capUpdateFailOpen := false
	measurementPublished := false
	defer func() {
		if capUpdateFailOpen && measurementPublished {
			metricTrackingCapUpdateFailOpen.Add(1)
		}
		if !releaseReplayClaim {
			return
		}
		if err := self.releaseTrackingEventClaim(replayClaim); err != nil {
			metricTrackingReplayRedisErrors.Add(1)
		}
	}()
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
		replayClaim = self.claimTrackingEvent(ctx, status, args, signatureValidUntil)
		if !replayClaim.records() {
			return nil
		}
		releaseReplayClaim = replayClaim.owned()
		u := args.Get("cap")
		if u == "" {
			break
		}
		if wl.RAdv.Cap, err = match.UnpackCapString(u); err == nil {
			var bid bidID
			if bid, err = UnpackBidID(wl.AuctionBidID); err == nil {
				if bid.UserID != "" {
					if !replayClaim.keyed() {
						capUpdateFailOpen = true
					} else {
						redisCtx, cancel := trackingRedisOperationContext(ctx)
						_, capErr := match.MustRefreshBothCapOnceWithTTL(redisCtx, self.Redis, current, bid.UserID, wl.RAdv.ItemID, wl.RAdv.Cap, self.capStateTTL(), replayClaim.capMarkerKey(), time.Until(signatureValidUntil), status == StatusTrackImp, status == StatusTrackClk)
						cancel()
						if capErr != nil {
							capUpdateFailOpen = true
						}
					}
				}
			}
		}
		if err != nil {
			return err
		}
	default:
	}

	if status == StatusWin || status == StatusLoss {
		first, err := self.setTrackingNotifyOnce(ctx, status, wl.AuctionBidID, signatureValidUntil)
		if err != nil {
			return err
		}
		if !first {
			return nil
		}
	}

	bs, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	if err := self.publishWinLoss(bs); err != nil {
		return err
	}
	measurementPublished = true
	if replayClaim.owned() {
		releaseReplayClaim = false
		if err := self.completeTrackingEventClaim(replayClaim, signatureValidUntil); err != nil {
			metricTrackingReplayRedisErrors.Add(1)
			metricTrackingReplayFailOpen.Add(1)
		}
	}
	return nil
}

func (self *Controller) setTrackingNotifyOnce(ctx context.Context, status Status, auctionBidID string, validUntil time.Time) (bool, error) {
	if auctionBidID == "" {
		return true, nil
	}
	remaining := time.Until(validUntil)
	if remaining <= 0 {
		return true, nil
	}
	if self != nil && self.trackingNotifyOnce != nil {
		return self.trackingNotifyOnce(ctx, status, auctionBidID, remaining)
	}
	if self == nil || self.Redis == nil {
		return true, nil
	}
	redisCtx, cancel := trackingRedisOperationContext(ctx)
	defer cancel()
	var result string
	err := self.Redis.Do(redisCtx, radix.Cmd(&result, "SET", trackingNotifyKey(status, auctionBidID), "1", "EX", strconv.Itoa(ttlSeconds(remaining)), "NX"))
	return result == "OK", err
}

func trackingNotifyKey(status Status, auctionBidID string) string {
	return "tracking:notify:" + status.path() + ":" + url.PathEscape(auctionBidID)
}

var (
	completeTrackingEventClaimScript = radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  redis.call("SET", KEYS[1], "done", "EXAT", ARGV[2])
  return 1
end
return 0`)
	releaseTrackingEventClaimScript = radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
)

func (self *Controller) claimTrackingEvent(ctx context.Context, status Status, args url.Values, validUntil time.Time) trackingEventClaim {
	key, ok := trackingEventKey(status, args)
	if !ok {
		metricTrackingReplayUnkeyed.Add(1)
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{outcome: trackingClaimUnkeyed}
	}
	remaining := time.Until(validUntil)
	if remaining <= 0 {
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	if self != nil && self.trackingEventOnce != nil {
		first, err := self.trackingEventOnce(ctx, status, args, remaining)
		if err != nil {
			metricTrackingReplayRedisErrors.Add(1)
			metricTrackingReplayFailOpen.Add(1)
			return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
		}
		if !first {
			metricTrackingReplaySuppressed.Add(1)
			return trackingEventClaim{key: key, outcome: trackingClaimDuplicate}
		}
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	if self == nil || self.Redis == nil {
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	token, err := newTrackingEventClaimToken()
	if err != nil {
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	processingDeadline := time.Now().Add(defaultTrackingProcessingTTL)
	if validUntil.Before(processingDeadline) {
		processingDeadline = validUntil
	}
	redisCtx, cancel := trackingRedisOperationContext(ctx)
	defer cancel()
	var result string
	err = self.Redis.Do(redisCtx, radix.Cmd(&result, "SET", key, token, "PXAT", strconv.FormatInt(processingDeadline.UnixMilli(), 10), "NX"))
	if err != nil {
		metricTrackingReplayRedisErrors.Add(1)
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	if result != "OK" {
		metricTrackingReplaySuppressed.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimDuplicate}
	}
	return trackingEventClaim{key: key, token: token, outcome: trackingClaimOwner}
}

func (self *Controller) completeTrackingEventClaim(claim trackingEventClaim, validUntil time.Time) error {
	if !claim.owned() || self == nil || self.Redis == nil {
		return nil
	}
	if !validUntil.After(time.Now()) {
		return nil
	}
	ctx, cancel := trackingRedisOperationContext(context.Background())
	defer cancel()
	var completed int
	if err := self.Redis.Do(ctx, completeTrackingEventClaimScript.Cmd(&completed, []string{claim.key}, claim.token, strconv.FormatInt(validUntil.Unix(), 10))); err != nil {
		return err
	}
	if completed != 1 {
		return fmt.Errorf("tracking event claim ownership lost before completion")
	}
	return nil
}

func (self *Controller) releaseTrackingEventClaim(claim trackingEventClaim) error {
	if !claim.owned() || self == nil || self.Redis == nil {
		return nil
	}
	ctx, cancel := trackingRedisOperationContext(context.Background())
	defer cancel()
	var released int
	return self.Redis.Do(ctx, releaseTrackingEventClaimScript.Cmd(&released, []string{claim.key}, claim.token))
}

func trackingRedisOperationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), trackingClaimOperationTimeout)
}

func trackingEventKey(status Status, args url.Values) (string, bool) {
	if status != StatusTrackImp && status != StatusTrackClk {
		return "", false
	}
	parts := []string{
		status.path(),
		args.Get("auction_id"),
		args.Get("auction_bid_id"),
		args.Get("auction_imp_id"),
	}
	for _, part := range parts[1:] {
		if part == "" {
			return "", false
		}
	}
	payload := strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "tracking:notify:" + status.path() + ":" + hex.EncodeToString(sum[:]), true
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
