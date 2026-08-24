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
	"math"
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
	"github.com/guruperl/aofei/publisherauth"
	"github.com/guruperl/aofei/trafficquality"
)

const (
	SUBJECTRequest   = "request"
	SUBJECTResponse  = "response"
	SUBJECTAttribute = "attribute"
	SUBJECTWinLoss   = "winloss"

	maxBidRequestBodyBytes = 1 << 20
)

type Controller struct {
	C                       *Config
	Ips                     *maxmind.IPSearch
	Redis                   radix.Client
	DB                      *sql.DB
	Nc                      *nats.Conn
	Logger                  *zap.Logger
	IsLocal                 bool
	localMu                 sync.Mutex
	local                   *localStaticCache
	localReloadMu           sync.Mutex
	localReloadCancel       context.CancelFunc
	localReloadDone         chan struct{}
	localReloadInterval     time.Duration
	auditMu                 sync.Mutex
	audit                   *auditPublisher
	auditClosed             bool
	auditFactory            func(*nats.Conn, int) *auditPublisher
	client                  *http.Client
	middlemanStore          middlemanCallbackStore
	callbackURLGuard        func(context.Context, string) error
	middlemanRuntime        middlemanRuntime
	middlemanRoute          atomic.Pointer[middlemanRouteState]
	middlemanRouteMu        sync.Mutex
	middlemanRouteWait      chan struct{}
	middlemanRouteLoad      func(context.Context) (*match.MiddlemanRouteCache, error)
	middlemanRouteNow       func() time.Time
	middlemanRouteWaitHook  func()
	trackingEventOnce       func(context.Context, Status, url.Values, time.Duration) (bool, error)
	publishWinLossFunc      func([]byte) error
	qualityService          *trafficquality.Service
	qualitySnapshot         atomic.Pointer[trafficquality.Snapshot]
	qualityReloadMu         sync.Mutex
	qualityReloadCancel     context.CancelFunc
	qualityReloadDone       chan struct{}
	directTokens            *acl.DirectTokenCodec
	publisherAuth           *publisherauth.Service
	publisherAuthReloadMu   sync.Mutex
	publisherAuthCancel     context.CancelFunc
	publisherAuthReloadDone chan struct{}
	ownRedis                bool
	ownDB                   bool
	ownNATS                 bool
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
	qualityService   *trafficquality.Service
}

type bidAudit struct {
	Attr          *match.Attribute
	One           match.RAdv
	Elapsed       time.Duration
	PrivacyMode   string
	PrivacyReason string
}

type auditSource struct {
	Source   string
	Contract string
}

type auditEnvelope struct {
	Source        string          `json:"source"`
	Contract      string          `json:"contract"`
	PrivacyMode   string          `json:"privacy_mode,omitempty"`
	PrivacyReason string          `json:"privacy_reason,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

var (
	auditSourceADX = auditSource{Source: "adx", Contract: "openrtb"}
	auditSourceSSP = auditSource{Source: "ssp", Contract: "pz-v1"}
)

type bidWinner struct {
	ImpIndex            int
	Seat                string
	Bid                 openrtb2.Bid
	Audit               bidAudit
	ResponseBidID       string
	ImpressionURL       string
	ClickURL            string
	EffectiveCPM        float64
	Comparable          bool
	Local               bool
	Middleman           *middlemanDownstreamBid
	DeliveryReservation string
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

// WithHTTPClient preserves caller timeouts and supported *http.Transport
// settings, then installs the mandatory safe bidder/callback transport.
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

// WithCallbackURLGuard adds an application/test guard after the mandatory
// safehttp URL policy; it cannot relax the network boundary.
func WithCallbackURLGuard(guard func(context.Context, string) error) ControllerOption {
	return func(opts *controllerOptions) {
		opts.callbackURLGuard = guard
	}
}

// WithTrafficQualityService supplies an already initialized service. It is
// primarily useful to tests and embeddings that own the service lifecycle.
func WithTrafficQualityService(service *trafficquality.Service) ControllerOption {
	return func(opts *controllerOptions) {
		opts.qualityService = service
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
	directTokenIssuer, err := NewDirectSSPTokenIssuer(c)
	if err != nil {
		return nil, err
	}
	directTokens := directTokenIssuer.codec
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
		C:              c,
		Redis:          redis,
		DB:             db,
		local:          newLocalStaticCache(),
		client:         safehttp.NewCallbackClient(options.httpClient),
		Logger:         options.logger,
		middlemanStore: options.callbackStore,
		directTokens:   directTokens,
		ownRedis:       ownRedis,
		ownDB:          ownDB,
	}
	publisherAuthService, err := publisherauth.NewService(c.DirectSSPAuth, db, redis)
	if err != nil {
		closeOwnedControllerStores(redis, db, ownRedis, ownDB)
		return nil, fmt.Errorf("initialize direct SSP authentication: %w", err)
	}
	controller.publisherAuth = publisherAuthService
	if publisherAuthService != nil {
		metricSSPPublisherAuthLoadedAt.Set(publisherAuthService.SnapshotGeneratedAt().Unix())
	}
	qualityService := options.qualityService
	if qualityService == nil {
		qualityService, err = trafficquality.NewService(c.TrafficQuality, db)
		if err != nil {
			closeOwnedControllerStores(redis, db, ownRedis, ownDB)
			return nil, fmt.Errorf("initialize traffic-quality service: %w", err)
		}
	}
	controller.qualityService = qualityService
	if qualityService != nil {
		qualityParent := ctx
		if qualityParent == nil {
			qualityParent = context.Background()
		}
		qualityCtx, qualityCancel := context.WithTimeout(context.WithoutCancel(qualityParent), 2*time.Second)
		snapshot, loadErr := qualityService.LoadEnforcementSnapshot(qualityCtx)
		qualityCancel()
		if loadErr != nil {
			closeOwnedControllerStores(redis, db, ownRedis, ownDB)
			return nil, fmt.Errorf("load traffic-quality enforcement: %w", loadErr)
		}
		controller.qualitySnapshot.Store(snapshot)
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
	controller.StartLocalStaticCacheReload()
	controller.startQualityEnforcementRefresh()
	controller.startPublisherAuthRefresh()

	return controller, nil
}

func closeOwnedControllerStores(redis radix.Client, db *sql.DB, ownRedis, ownDB bool) {
	if ownRedis && redis != nil {
		_ = redis.Close()
	}
	if ownDB && db != nil {
		_ = db.Close()
	}
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
	self.stopPublisherAuthRefresh()
	self.stopQualityEnforcementRefresh()
	self.stopLocalStaticCacheReload()
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
	w.Header().Set("x-openrtb-version", "2.5")
	if status, err := validateOpenRTB25HTTP(r); err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	logger := self.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	current := time.Now()
	ctx := r.Context()

	bidStr, err := io.ReadAll(io.LimitReader(r.Body, maxBidRequestBodyBytes+1))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
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
			zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
			zap.String("reason", "request_validation"),
		)
		return
	}
	privacy := self.privacyDecisionForBid(r, bid)
	recordPrivacyDecision(privacy)
	if err = self.applyPrivacyPolicy(bid, privacy); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Debug("bid request privacy sanitation failed",
			zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
			zap.String("reason", "privacy_policy"),
		)
		return
	}
	privacyMiddlemanRequest, err := json.Marshal(bid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pubStr, pubObj, err := self.getPub(ctx, r, bid)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		logger.Debug("bid request publisher rejected",
			zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
			zap.String("reason", "publisher_policy"),
		)
		return
	}

	finalWinners, localWinners := self.auctionBidWinners(ctx, bid, pubObj, current, pubStr, privacyMiddlemanRequest, logger, privacy)
	seatOrder, seatBids, audits, responseBidID, materialized := self.materializeBidWinners(ctx, bid, finalWinners, localWinners, logger)
	recordWinnerSourceLatency(current, materialized)
	if len(audits) == 0 {
		self.releaseMaterializedDeliveries(ctx, materialized)
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
		self.releaseMaterializedDeliveries(ctx, materialized)
		w.WriteHeader(http.StatusNoContent)
		logger.Error("bid response marshal failed",
			zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
			zap.String("reason", "response_marshal"),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if written, writeErr := w.Write(rspnStr); writeErr != nil || written != len(rspnStr) {
		self.releaseMaterializedDeliveries(ctx, materialized)
		logger.Warn("bid response write failed",
			zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
			zap.String("reason", "response_write"),
		)
		return
	}
	metricBidResponses.Add(1)

	elapsed := time.Since(current)
	for i := range audits {
		audits[i].Elapsed = elapsed
		audits[i].PrivacyMode = string(privacy.Mode)
		audits[i].PrivacyReason = privacy.Reason
	}
	if self.Nc != nil {
		if err = self.publishBidAudits(privacyMiddlemanRequest, rspnStr, audits); err != nil {
			logger.Warn("bid audit publish failed",
				zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
				zap.String("reason", "audit_dependency"),
			)
		}
	}
}

func (self *Controller) releaseMaterializedDeliveries(ctx context.Context, winners []bidWinner) {
	for _, winner := range winners {
		if winner.Local {
			_ = self.releaseDeliveryReservation(ctx, winner.DeliveryReservation)
		}
	}
}

func (self *Controller) middlemanAlwaysEnabled() bool {
	return self != nil && self.C != nil && self.C.MiddlemanEnabled && self.C.MiddlemanAlwaysEnabled
}

func (self *Controller) middlemanTriggerModesForNoBid() []string {
	return []string{"Fallback"}
}

func (self *Controller) auctionBidWinners(ctx context.Context, bid *openrtb2.BidRequest, pubObj *acl.Pub, current time.Time, pubStr string, rawMiddlemanRequest []byte, logger *zap.Logger, decisions ...privacyDecision) (map[int]bidWinner, map[int]bidWinner) {
	var privacy privacyDecision
	if len(decisions) != 0 {
		privacy = decisions[0].normalized()
	}
	var fallbackImps []middlemanFallbackImp
	localWinners := make(map[int]bidWinner)
	finalWinners := make(map[int]bidWinner)
	for impIndex := range bid.Imp {
		dspBid, audit, err := self.bidForImp(ctx, bid, pubObj, current, pubStr, impIndex, privacy)
		if err != nil {
			if logger != nil && !isLocalNoBid(err) {
				logger.Warn("bid impression evaluation failed",
					zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
					zap.Int("imp_index", impIndex),
					zap.String("reason", "evaluation_error"),
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
			_ = self.releaseDeliveryReservation(ctx, dspBid.deliveryReservation)
			if logger != nil {
				logger.Warn("bid impression materialization failed",
					zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
					zap.Int("imp_index", impIndex),
					zap.String("reason", "materialization_error"),
				)
			}
			continue
		}
		effectiveCPM, comparable := localBidEffectiveCPM(dspBid, audit, rspnsBid)
		winner := bidWinner{
			ImpIndex:            impIndex,
			Seat:                dspBid.seat(),
			Bid:                 rspnsBid,
			Audit:               audit,
			ResponseBidID:       dspBid.bidID(),
			ImpressionURL:       winloss.ImpURL(),
			ClickURL:            clickURLForSSPBid(dspBid, winloss),
			EffectiveCPM:        effectiveCPM,
			Comparable:          comparable,
			Local:               true,
			DeliveryReservation: dspBid.deliveryReservation,
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
	if len(fallbackImps) != 0 && (privacy.Mode == "" || privacy.AllowMiddleman) {
		middlemanBids, err := self.middleman().Fallback(ctx, bid, rawMiddlemanRequest, current, fallbackImps)
		if err != nil {
			if logger != nil {
				logger.Warn("middleman fallback failed",
					zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
					zap.String("reason", middlemanSafeFailureReason(err)),
				)
			}
		}
		for _, middlemanBid := range middlemanBids {
			current, ok := finalWinners[middlemanBid.ImpIndex]
			if winner, replace := chooseMiddlemanWinner(current, ok, middlemanBid); replace {
				finalWinners[middlemanBid.ImpIndex] = winner
			}
		}
	} else if len(fallbackImps) != 0 {
		metricPrivacyMiddlemanBlocked.Add(1)
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
						zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
						zap.Int("imp_index", impIndex),
						zap.Uint32("bidder_id", middlemanBid.Entry.BidderID),
						zap.String("reason", "callback_setup"),
					)
				}
				local, ok := localWinners[impIndex]
				if !ok {
					continue
				}
				winner = local
			} else {
				if local, ok := localWinners[impIndex]; ok {
					_ = self.releaseDeliveryReservation(ctx, local.DeliveryReservation)
				}
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

func (self *Controller) bidForImp(ctx context.Context, bid *openrtb2.BidRequest, pubObj *acl.Pub, current time.Time, pubStr string, impIndex int, decisions ...privacyDecision) (*DSP, bidAudit, error) {
	attr, err := match.NewAttributeForImp(ctx, self.Ips, bid, impIndex, pubObj, current, pubStr)
	if err != nil {
		return nil, bidAudit{}, err
	}
	if len(decisions) != 0 && decisions[0].Mode != "" {
		self.protectPrivacyAttributeForBid(attr, bid, decisions[0])
	}
	audit := bidAudit{Attr: attr}
	qualityEnabled := self.trafficQualityEnabled()
	qualityKey := ""
	if qualityEnabled {
		qualityKey = qualityEventKey(bid, impIndex)
		if action, enforcementID := self.qualityAction(trafficquality.Scope{Type: trafficquality.ScopePublisher, ID: uint64(attr.RPub.PubID)}, qualityKey, current); qualityBlocks(action) {
			return nil, audit, noBidErrorf("publisher traffic withheld by reviewed quality enforcement %d", enforcementID)
		}
	}
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
	if qualityEnabled {
		qualityByAdvertiser := make(map[uint32]bool)
		qualityChecked := make(map[uint32]bool)
		qualityCandidates := make(match.RAdvs, 0, len(monitors))
		for _, candidate := range monitors {
			blocked, checked := qualityByAdvertiser[candidate.AdvID], qualityChecked[candidate.AdvID]
			if !checked {
				action, _ := self.qualityAction(trafficquality.Scope{Type: trafficquality.ScopeAdvertiser, ID: uint64(candidate.AdvID)}, qualityKey, current)
				blocked = qualityBlocks(action)
				qualityByAdvertiser[candidate.AdvID] = blocked
				qualityChecked[candidate.AdvID] = true
			}
			if !blocked {
				qualityCandidates = append(qualityCandidates, candidate)
			}
		}
		monitors = qualityCandidates
		if len(monitors) == 0 {
			return nil, audit, noBidErrorf("no ad after reviewed traffic-quality enforcement")
		}
	}
	deliveryCandidates := make(match.RAdvs, 0, len(monitors))
	for _, candidate := range monitors {
		if eligible, reason := candidate.Delivery.EligibleAt(current, self.deliveryCacheMaxAge()); eligible {
			deliveryCandidates = append(deliveryCandidates, candidate)
		} else {
			recordDeliveryEligibilityRejection(reason)
		}
	}
	monitors = deliveryCandidates
	if len(monitors) == 0 {
		return nil, audit, noBidErrorf("no ad after delivery schedule and cached budget checks")
	}
	if self.Redis == nil && radvsNeedCaps(monitors) {
		return nil, audit, fmt.Errorf("redis mutable state unavailable for frequency caps")
	}

	capStarted := time.Now()
	observeCap := radvsNeedCaps(monitors)
	candidates, bothcaps, err := monitors.FilterByCaps(ctx, self.Redis, current, attr.UserID)
	if observeCap {
		recordBidPathLatency("cap", time.Since(capStarted))
	}
	if err != nil {
		return nil, audit, err
	}
	if len(candidates) == 0 {
		return nil, audit, noBidErrorf("no ad after frequency-cap evaluation")
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

	audienceStarted := time.Now()
	radvs, auds, err := self.priorityAudienceMatches(ctx, bid, candidates, audiences, attr)
	recordBidPathLatency("audience", time.Since(audienceStarted))
	if err != nil {
		return nil, audit, err
	}
	if len(radvs) == 0 {
		return nil, audit, noBidErrorf("no ad after matching audience")
	}

	imp := bid.Imp[impIndex]
	var one match.RAdv
	var selectedAudience *match.Audience
	var bidPrice float32
	var deliveryReservation string
	var creative *match.Creative
	for len(radvs) != 0 {
		index, selectedPrice := radvs.PickIndexPrice(imp.BidFloor, imp.BidFloorCur)
		if index < 0 {
			return nil, audit, noBidErrorf("no ad to match for bid floor %f %s", imp.BidFloor, imp.BidFloorCur)
		}
		candidate := radvs[index]
		if self.C.IsLocal {
			creative, err = self.localCreative(top, candidate.CreativeID)
		} else {
			creative, err = match.CreativeFromRedis(ctx, self.Redis, candidate.CreativeID)
		}
		if err != nil || creative.ValidateForImp(&imp, attr) != nil {
			metricCreativeRejections.Add(1)
			radvs, auds = removeCandidateAt(radvs, auds, index)
			continue
		}
		spend, exact := candidate.ImpressionSpendNano()
		if !exact {
			return nil, audit, noBidErrorf("candidate has no exact USD CPM")
		}
		reservation, reserveErr := self.reserveDelivery(ctx, candidate, current, spend)
		if reserveErr == nil {
			one = candidate
			bidPrice = selectedPrice
			deliveryReservation = reservation
			selectedAudience = auds[index]
			break
		}
		if !errors.Is(reserveErr, errDeliveryLimit) {
			return nil, audit, noBidErrorf("delivery reservation unavailable: %v", reserveErr)
		}
		radvs, auds = removeDemandUnit(radvs, auds, candidate)
	}
	if len(radvs) == 0 || one.CreativeID == 0 {
		return nil, audit, noBidErrorf("no ad after creative, live budget, and pacing checks")
	}
	var bothcap *match.BothCap
	if bothcaps != nil {
		if b, ok := bothcaps[one.ItemID]; ok {
			bothcap = &b
		}
	}
	dspBid := NewDSPForImp(bid, impIndex, attr, one, bothcap, creative, selectedAudience, bidPrice, self.C.ServerURL).
		WithTrackingSecret(self.C.TrackingSecret).
		withDeliveryReservation(deliveryReservation)
	_, _, _, actionTokenTTL, _ := self.actionPolicy()
	dspBid.withActionTokenTTL(actionTokenTTL)
	return dspBid, bidAudit{Attr: attr, One: one}, nil
}

func removeCandidateAt(radvs match.RAdvs, auds match.Audiences, index int) (match.RAdvs, match.Audiences) {
	radvs = append(radvs[:index], radvs[index+1:]...)
	auds = append(auds[:index], auds[index+1:]...)
	return radvs, auds
}

func removeDemandUnit(radvs match.RAdvs, auds match.Audiences, selected match.RAdv) (match.RAdvs, match.Audiences) {
	keptRAdvs := make(match.RAdvs, 0, len(radvs))
	keptAudiences := make(match.Audiences, 0, len(auds))
	for i, candidate := range radvs {
		if candidate.AdvID == selected.AdvID && candidate.CampaignID == selected.CampaignID && candidate.ItemID == selected.ItemID {
			continue
		}
		keptRAdvs = append(keptRAdvs, candidate)
		keptAudiences = append(keptAudiences, auds[i])
	}
	return keptRAdvs, keptAudiences
}

func recordWinnerSourceLatency(started time.Time, winners []bidWinner) {
	var local, middleman bool
	for _, winner := range winners {
		if winner.Local {
			local = true
		} else {
			middleman = true
		}
	}
	elapsed := time.Since(started)
	if local {
		recordBidPathLatency("local", elapsed)
	}
	if middleman {
		recordBidPathLatency("middleman", elapsed)
	}
}

func recordDeliveryEligibilityRejection(reason string) {
	metricDeliveryScheduleRejected.Add(1)
	switch reason {
	case "stale delivery cache", "delivery cache timestamp is in the future":
		metricDeliveryCacheStaleRejected.Add(1)
	case "campaign schedule", "item schedule":
		metricDeliveryWindowRejected.Add(1)
	case "cached budget exhausted":
		metricDeliveryCachedBudgetRejected.Add(1)
	default:
		metricDeliveryPolicyErrors.Add(1)
	}
}

func radvsNeedCaps(radvs match.RAdvs) bool {
	for _, radv := range radvs {
		if radv.CapNumber != 0 || radv.CapThrottle != 0 || radv.ClickNumber != 0 {
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
	if err := validateOpenRTBIdentifier("request", bid.ID); err != nil {
		return err
	}
	if len(bid.Imp) == 0 || len(bid.Imp) > 64 {
		return fmt.Errorf("bid request has no impressions")
	}
	if bid.Device == nil {
		return fmt.Errorf("bid request has no device")
	}
	if bid.AT < 0 || bid.AT > 2 {
		return fmt.Errorf("bid request auction type is unsupported")
	}
	if bid.TMax < 0 || bid.TMax > 60_000 {
		return fmt.Errorf("bid request tmax must be between 0 and 60000 milliseconds")
	}
	for _, currency := range bid.Cur {
		if !strings.EqualFold(strings.TrimSpace(currency), "USD") {
			return fmt.Errorf("bid request currency must be USD")
		}
	}
	seen := make(map[string]struct{}, len(bid.Imp))
	for i := range bid.Imp {
		imp := &bid.Imp[i]
		if err := validateOpenRTBIdentifier("impression", imp.ID); err != nil {
			return err
		}
		if _, duplicate := seen[imp.ID]; duplicate {
			return fmt.Errorf("bid request impression id is duplicated")
		}
		seen[imp.ID] = struct{}{}
		if imp.BidFloor < 0 || math.IsNaN(imp.BidFloor) || math.IsInf(imp.BidFloor, 0) {
			return fmt.Errorf("bid request floor must be finite and nonnegative")
		}
		media := 0
		if imp.Banner != nil {
			media++
		}
		if imp.Video != nil {
			media++
		}
		if imp.Native != nil {
			media++
		}
		if media == 0 {
			return fmt.Errorf("bid request impression must contain a supported media type")
		}
		if imp.Secure != nil && *imp.Secure != 0 && *imp.Secure != 1 {
			return fmt.Errorf("bid request secure flag must be 0 or 1")
		}
	}
	return nil
}

func validateOpenRTB25HTTP(r *http.Request) (int, error) {
	if r == nil {
		return http.StatusBadRequest, fmt.Errorf("request is nil")
	}
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, fmt.Errorf("OpenRTB requires POST")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if contentType != "" && contentType != "application/json" && contentType != "application/openrtb+json" {
		return http.StatusUnsupportedMediaType, fmt.Errorf("OpenRTB content type is unsupported")
	}
	if version := strings.TrimSpace(r.Header.Get("x-openrtb-version")); version != "" && version != "2.5" {
		return http.StatusBadRequest, fmt.Errorf("OpenRTB version must be 2.5")
	}
	return 0, nil
}

func validateOpenRTBIdentifier(name, value string) error {
	if value == "" || value != strings.TrimSpace(value) || len(value) > 256 || strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return fmt.Errorf("bid %s id must be nonempty and at most 256 bytes without control characters", name)
	}
	return nil
}

func (self *Controller) publishWinLoss(bs []byte) error {
	var wl WinLoss
	if err := json.Unmarshal(bs, &wl); err != nil {
		return err
	}
	if unpacked, err := UnpackBidID(wl.AuctionBidID); err == nil {
		unpacked.UserID = ""
		wl.AuctionBidID = unpacked.String()
	} else {
		wl.AuctionBidID = ""
	}
	privacySafe, err := json.Marshal(&wl)
	if err != nil {
		return err
	}
	if self.publishWinLossFunc != nil {
		return self.publishWinLossFunc(privacySafe)
	}
	if self.Nc == nil {
		return nil
	}
	return self.Nc.Publish(SUBJECTWinLoss, privacySafe)
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
	return self.publishSSPBidAuditsWithPrivacy(bidStr, rspnStr, audits, privacyDecisionFromAudits(audits))
}

func (self *Controller) publishSSPBidAuditsWithPrivacy(bidStr, rspnStr []byte, audits []bidAudit, privacy privacyDecision) error {
	request, err := wrapAuditPayloadWithPrivacy(auditSourceSSP, bidStr, privacy)
	if err != nil {
		return err
	}
	response, err := wrapAuditPayloadWithPrivacy(auditSourceSSP, rspnStr, privacy)
	if err != nil {
		return err
	}
	return self.publishBidAuditsFor(auditSourceSSP, request, response, audits)
}

func wrapAuditPayload(source auditSource, payload []byte) ([]byte, error) {
	return wrapAuditPayloadWithPrivacy(source, payload, privacyDecision{})
}

func wrapAuditPayloadWithPrivacy(source auditSource, payload []byte, privacy privacyDecision) ([]byte, error) {
	if len(payload) == 0 {
		payload = []byte("null")
	}
	sanitized, err := privacySanitizeJSON(payload, true)
	if err != nil {
		return nil, err
	}
	privacy = privacy.normalized()
	return json.Marshal(auditEnvelope{
		Source:        source.Source,
		Contract:      source.Contract,
		PrivacyMode:   string(privacy.Mode),
		PrivacyReason: privacy.Reason,
		Payload:       json.RawMessage(sanitized),
	})
}

func privacyDecisionFromAudits(audits []bidAudit) privacyDecision {
	for _, audit := range audits {
		if audit.PrivacyMode != "" {
			return privacyDecision{Mode: privacyMode(audit.PrivacyMode), Reason: audit.PrivacyReason}
		}
	}
	return privacyDecision{}
}

func (self *Controller) publishBidAuditsFor(source auditSource, bidStr, rspnStr []byte, audits []bidAudit) error {
	publisher := self.bidAuditPublisher()
	if publisher == nil {
		return nil
	}
	safeRequest, err := privacySanitizeAuditRequest(source, bidStr)
	if err != nil {
		return err
	}
	safeResponse, err := privacySanitizeAuditResponse(source, rspnStr)
	if err != nil {
		return err
	}
	publisher.Enqueue(SUBJECTRequest, safeRequest)
	publisher.Enqueue(SUBJECTResponse, safeResponse)
	for _, audit := range audits {
		if audit.Attr == nil {
			continue
		}
		safeAttr := privacySafeAttribute(audit.Attr)
		bs, err := json.Marshal(match.AttributePlus{
			Attribute:     *safeAttr,
			RAdv:          audit.One,
			Elapsed:       audit.Elapsed.Milliseconds(),
			Source:        source.Source,
			Contract:      source.Contract,
			PrivacyMode:   audit.PrivacyMode,
			PrivacyReason: audit.PrivacyReason,
		})
		if err != nil {
			return err
		}
		publisher.Enqueue(SUBJECTAttribute, bs)
	}
	return nil
}

func privacySanitizeAuditRequest(source auditSource, raw []byte) ([]byte, error) {
	if source.Source != auditSourceADX.Source || len(raw) == 0 {
		return privacySanitizeJSON(raw, true)
	}
	var bid openrtb2.BidRequest
	if err := json.Unmarshal(raw, &bid); err != nil {
		return nil, err
	}
	typed, err := json.Marshal(&bid)
	if err != nil {
		return nil, err
	}
	return privacySanitizeJSON(typed, true)
}

func privacySanitizeAuditResponse(source auditSource, raw []byte) ([]byte, error) {
	if source.Source != auditSourceADX.Source || len(raw) == 0 {
		return privacySanitizeJSON(raw, true)
	}
	var response openrtb2.BidResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	for i := range response.SeatBid {
		response.SeatBid[i].Ext = nil
		for j := range response.SeatBid[i].Bid {
			bid := &response.SeatBid[i].Bid[j]
			bid.AdM = ""
			bid.NURL = ""
			bid.BURL = ""
			bid.LURL = ""
			bid.Ext = nil
		}
	}
	typed, err := json.Marshal(&response)
	if err != nil {
		return nil, err
	}
	return privacySanitizeJSON(typed, true)
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
				var retryableErr *retryableCallbackError
				if !errors.As(err, &retryableErr) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if self.Logger != nil {
					self.Logger.Debug("click tracking dependency failed before redirect", zap.String("reason", "tracking_dependency"))
				}
			}
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
	}

	if err := self.serveStatus(ctx, status, current, r.URL.Query()); err != nil {
		responseStatus := http.StatusBadRequest
		var retryableErr *retryableCallbackError
		if errors.As(err, &retryableErr) {
			responseStatus = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), responseStatus)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type retryableCallbackError struct {
	err error
}

func (e *retryableCallbackError) Error() string { return e.err.Error() }
func (e *retryableCallbackError) Unwrap() error { return e.err }

func retryableCallback(err error) error {
	if err == nil {
		return nil
	}
	var retryableErr *retryableCallbackError
	if errors.As(err, &retryableErr) {
		return err
	}
	return &retryableCallbackError{err: err}
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
			metricTrackingClaimReleaseErrors.Add(1)
		} else {
			metricTrackingClaimReleases.Add(1)
		}
	}()
	wl := &WinLoss{
		Current:             current,
		Status:              status,
		AuctionID:           args.Get("auction_id"),
		AuctionBidID:        args.Get("auction_bid_id"),
		AuctionImpID:        args.Get("auction_imp_id"),
		DeliveryReservation: args.Get("delivery_reservation"),
	}
	if wl.Reporting, err = reportingDimensionsFromTracking(args); err != nil {
		return err
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
			wl.RAdv.CostType = match.CostTypeCPM
		}
	} else if status == StatusTrackClk || status == StatusTrackImp {
		return err
	}

	switch status {
	case StatusTrackClk, StatusTrackImp:
		u := args.Get("cap")
		var bid bidID
		if u != "" {
			if wl.RAdv.Cap, err = match.UnpackCapString(u); err != nil {
				return err
			}
			if bid, err = UnpackBidID(wl.AuctionBidID); err != nil {
				return err
			}
		}
		replayClaim = self.claimTrackingEvent(ctx, status, args, signatureValidUntil)
		if !replayClaim.records() {
			if replayClaim.completed() {
				if err := self.applyTrackingDeliverySideEffect(ctx, status, wl, signatureValidUntil); err != nil {
					return retryableCallback(err)
				}
			} else if wl.DeliveryReservation != "" {
				return retryableCallback(fmt.Errorf("tracking callback delivery is still processing"))
			}
			return nil
		}
		releaseReplayClaim = replayClaim.owned()
		if u == "" {
			break
		}
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
	default:
	}

	if status == StatusWin || status == StatusLoss {
		replayClaim = self.claimTrackingNotify(ctx, status, wl.AuctionBidID, signatureValidUntil)
		if !replayClaim.records() {
			if replayClaim.completed() {
				if err := self.applyTrackingDeliverySideEffect(ctx, status, wl, signatureValidUntil); err != nil {
					return retryableCallback(err)
				}
			} else if wl.DeliveryReservation != "" {
				return retryableCallback(fmt.Errorf("tracking callback delivery is still processing"))
			}
			return nil
		}
		releaseReplayClaim = replayClaim.owned()
	}

	bs, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	if err := self.publishWinLoss(bs); err != nil {
		if replayClaim.owned() {
			metricTrackingRetryablePublishErrors.Add(1)
		}
		return retryableCallback(err)
	}
	measurementPublished = true
	self.recordAttributionTouch(wl)
	var completionErr error
	if replayClaim.owned() {
		releaseReplayClaim = false
		if completionErr = self.completeTrackingEventClaim(replayClaim, signatureValidUntil); completionErr != nil {
			metricTrackingReplayRedisErrors.Add(1)
			metricTrackingReplayFailOpen.Add(1)
		}
	}
	deliveryErr := self.applyTrackingDeliverySideEffect(ctx, status, wl, signatureValidUntil)
	if completionErr != nil || deliveryErr != nil {
		return retryableCallback(errors.Join(completionErr, deliveryErr))
	}
	return nil
}

func (self *Controller) applyTrackingDeliverySideEffect(ctx context.Context, status Status, wl *WinLoss, validUntil time.Time) error {
	if wl == nil {
		return nil
	}
	switch status {
	case StatusTrackImp:
		return self.finalizeDeliveryReservation(ctx, wl.DeliveryReservation, validUntil)
	case StatusTrackClk:
		return self.recordDeliveryClick(ctx, wl.DeliveryReservation)
	case StatusLoss:
		return self.releaseDeliveryReservation(ctx, wl.DeliveryReservation)
	default:
		return nil
	}
}

func reportingDimensionsFromTracking(args url.Values) (*ReportingDimensions, error) {
	keys := []string{
		"report_country_id", "report_state_id", "report_device_os", "report_device_type",
		"report_environment", "report_integration", "report_media_intent", "report_placement",
		"report_render_context", "report_refresh_mode", "report_refresh_seconds", "report_ad_density", "report_traffic_quality",
		"report_source_quality", "report_management", "report_seller_type", "report_seller_id",
	}
	present := false
	for _, key := range keys {
		if args.Get(key) != "" {
			present = true
			break
		}
	}
	if !present {
		return nil, nil
	}
	parse := func(key string, bits int) (uint64, error) {
		raw := args.Get(key)
		if raw == "" {
			return 0, nil
		}
		value, err := strconv.ParseUint(raw, 10, bits)
		if err != nil {
			return 0, fmt.Errorf("invalid reporting dimension %s", key)
		}
		return value, nil
	}
	countryID, err := parse("report_country_id", 32)
	if err != nil {
		return nil, err
	}
	stateID, err := parse("report_state_id", 32)
	if err != nil {
		return nil, err
	}
	deviceOS, err := parse("report_device_os", 8)
	if err != nil {
		return nil, err
	}
	deviceType, err := parse("report_device_type", 8)
	if err != nil {
		return nil, err
	}
	controlled := func(key string, allowed []string) (string, error) {
		value := strings.TrimSpace(args.Get(key))
		if value == "" {
			value = "Unknown"
		}
		for _, candidate := range allowed {
			if value == candidate {
				return value, nil
			}
		}
		return "", fmt.Errorf("invalid reporting dimension %s", key)
	}
	environment, err := controlled("report_environment", acl.InventoryEnvironments)
	if err != nil {
		return nil, err
	}
	integration, err := controlled("report_integration", acl.IntegrationModes)
	if err != nil {
		return nil, err
	}
	mediaIntent, err := controlled("report_media_intent", acl.MediaIntents)
	if err != nil {
		return nil, err
	}
	placement, err := controlled("report_placement", acl.Placements)
	if err != nil {
		return nil, err
	}
	renderContext, err := controlled("report_render_context", acl.RenderContexts)
	if err != nil {
		return nil, err
	}
	refreshMode, err := controlled("report_refresh_mode", acl.RefreshModes)
	if err != nil {
		return nil, err
	}
	refreshSeconds, err := parse("report_refresh_seconds", 16)
	if err != nil {
		return nil, err
	}
	if (refreshMode == "Timed" && (refreshSeconds < 15 || refreshSeconds > 3600)) ||
		(refreshMode != "Timed" && refreshSeconds != 0) {
		return nil, fmt.Errorf("invalid reporting dimension report_refresh_seconds")
	}
	adDensity, err := controlled("report_ad_density", acl.AdDensities)
	if err != nil {
		return nil, err
	}
	trafficQuality, err := controlled("report_traffic_quality", acl.TrafficQualities)
	if err != nil {
		return nil, err
	}
	sourceQuality, err := controlled("report_source_quality", acl.SourceQualities)
	if err != nil {
		return nil, err
	}
	management, err := controlled("report_management", acl.ManagementControls)
	if err != nil {
		return nil, err
	}
	sellerType, err := controlled("report_seller_type", []string{"Unknown", "Publisher", "Intermediary"})
	if err != nil {
		return nil, err
	}
	sellerID := strings.TrimSpace(args.Get("report_seller_id"))
	if err := (acl.SellerMetadata{ID: sellerID, Type: "Publisher"}).Validate(); err != nil {
		return nil, fmt.Errorf("invalid reporting dimension report_seller_id")
	}
	if sellerType == "Unknown" && sellerID != "" {
		return nil, fmt.Errorf("report_seller_id requires an approved seller type")
	}
	return &ReportingDimensions{
		CountryID: uint32(countryID), StateID: uint32(stateID), DeviceOS: uint8(deviceOS), DeviceType: uint8(deviceType),
		Environment: environment, IntegrationMode: integration, MediaIntent: mediaIntent, Placement: placement,
		RenderContext: renderContext, RefreshMode: refreshMode, RefreshSeconds: uint16(refreshSeconds), AdDensity: adDensity, TrafficQuality: trafficQuality,
		SourceQuality: sourceQuality, ManagementControl: management, SellerType: sellerType, SellerID: sellerID,
	}, nil
}

func (self *Controller) claimTrackingNotify(ctx context.Context, status Status, auctionBidID string, validUntil time.Time) trackingEventClaim {
	if auctionBidID == "" {
		metricTrackingReplayUnkeyed.Add(1)
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{outcome: trackingClaimUnkeyed}
	}
	return self.claimTrackingKey(ctx, trackingNotifyKey(status, auctionBidID), validUntil)
}

func trackingNotifyKey(status Status, auctionBidID string) string {
	return "tracking:notify:" + status.path() + ":" + url.PathEscape(auctionBidID)
}

var (
	// "done" is a non-secret terminal-state label. Only the random owner token
	// returned by a successful claim may complete or release a processing claim;
	// no HTTP field or static marker value can establish ownership.
	claimTrackingEventScript = radix.NewEvalScript(`
local current = redis.call("GET", KEYS[1])
if current then
  if current == "done" then
    return 2
  end
  return 0
end
redis.call("SET", KEYS[1], ARGV[1], "PXAT", ARGV[2])
return 1`)
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
	return self.claimTrackingKey(ctx, key, validUntil)
}

func (self *Controller) claimTrackingKey(ctx context.Context, key string, validUntil time.Time) trackingEventClaim {
	remaining := time.Until(validUntil)
	if remaining <= 0 {
		metricTrackingReplayFailOpen.Add(1)
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
	var result int
	err = self.Redis.Do(redisCtx, claimTrackingEventScript.Cmd(&result, []string{key}, token, strconv.FormatInt(processingDeadline.UnixMilli(), 10)))
	if err != nil {
		metricTrackingReplayRedisErrors.Add(1)
		metricTrackingReplayFailOpen.Add(1)
		return trackingEventClaim{key: key, outcome: trackingClaimRedisFailOpen}
	}
	if result != 1 {
		metricTrackingReplaySuppressed.Add(1)
		if result == 2 {
			return trackingEventClaim{key: key, outcome: trackingClaimCompleted}
		}
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
	if err := self.Redis.Do(ctx, releaseTrackingEventClaimScript.Cmd(&released, []string{claim.key}, claim.token)); err != nil {
		return err
	}
	if released != 1 {
		return fmt.Errorf("tracking event claim ownership lost before release")
	}
	return nil
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
