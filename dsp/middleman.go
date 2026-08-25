package dsp

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/internal/safehttp"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/trafficquality"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
	"golang.org/x/net/http/httpguts"
)

type middlemanFallbackImp struct {
	Index        int
	Attr         *match.Attribute
	TriggerModes []string
}

type middlemanCandidate struct {
	ImpIndex int
	Attr     *match.Attribute
	Entry    match.MiddlemanRouteEntry
}

type middlemanAssignment struct {
	Entry       match.MiddlemanRouteEntry
	EntriesByID map[string]match.MiddlemanRouteEntry
	AttrsByID   map[string]*match.Attribute
}

type middlemanDownstreamBid struct {
	ImpIndex           int
	Seat               string
	Bid                openrtb2.Bid
	Audit              bidAudit
	ResponseBidID      string
	Entry              match.MiddlemanRouteEntry
	DownstreamSeat     string
	DownstreamAdID     string
	DownstreamNURL     string
	DownstreamBURL     string
	DownstreamLURL     string
	DownstreamBidPrice float64
	UpstreamBidPrice   float64
	DownstreamBidCPM   accounting.CPM
	UpstreamBidCPM     accounting.CPM
	ClickRequestToken  string
}

type middlemanRuntime interface {
	Fallback(context.Context, *openrtb2.BidRequest, []byte, time.Time, []middlemanFallbackImp) ([]middlemanDownstreamBid, error)
}

type noopMiddlemanRuntime struct{}

func (noopMiddlemanRuntime) Fallback(context.Context, *openrtb2.BidRequest, []byte, time.Time, []middlemanFallbackImp) ([]middlemanDownstreamBid, error) {
	return nil, nil
}

type controllerMiddlemanRuntime struct {
	controller *Controller
}

type middlemanRouteState struct {
	cache     *match.MiddlemanRouteCache
	err       error
	expiresAt time.Time
}

func (r controllerMiddlemanRuntime) Fallback(ctx context.Context, bid *openrtb2.BidRequest, rawRequest []byte, started time.Time, fallbackImps []middlemanFallbackImp) ([]middlemanDownstreamBid, error) {
	return r.controller.middlemanFallback(ctx, bid, rawRequest, started, fallbackImps)
}

func (self *Controller) middleman() middlemanRuntime {
	if self == nil || self.C == nil || !self.C.MiddlemanEnabled {
		return noopMiddlemanRuntime{}
	}
	if self.middlemanRuntime != nil {
		return self.middlemanRuntime
	}
	return controllerMiddlemanRuntime{controller: self}
}

func (self *Controller) middlemanFallback(ctx context.Context, bid *openrtb2.BidRequest, rawRequest []byte, started time.Time, fallbackImps []middlemanFallbackImp) ([]middlemanDownstreamBid, error) {
	if self.C == nil || !self.C.MiddlemanEnabled || self.Redis == nil || len(fallbackImps) == 0 {
		return nil, nil
	}
	cache, err := self.middlemanRoutes(ctx)
	if err != nil {
		return nil, err
	}
	if len(cache.Entries) == 0 {
		return nil, nil
	}

	assignments := self.middlemanAssignments(bid, cache, fallbackImps)
	if len(assignments) == 0 {
		return nil, nil
	}

	client := self.client
	logger := self.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	results := make(chan []middlemanDownstreamBid, len(assignments))
	var wg sync.WaitGroup
	for _, assignment := range assignments {
		assignment := assignment
		wg.Add(1)
		go func() {
			defer wg.Done()
			bids, err := self.callMiddlemanBidder(ctx, client, bid, rawRequest, started, assignment)
			if err == nil && len(bids) > 0 {
				results <- bids
			} else if err != nil {
				logger.Warn("middleman bidder request failed",
					zap.String("request_id_hash", safeOpenRTBRequestIDHash(bid.ID)),
					zap.Uint32("bidder_id", assignment.Entry.BidderID),
					zap.String("reason", middlemanSafeFailureReason(err)),
				)
			}
		}()
	}
	wg.Wait()
	close(results)

	best := make(map[int]middlemanDownstreamBid)
	for bids := range results {
		for _, candidate := range bids {
			current, ok := best[candidate.ImpIndex]
			if !ok || candidate.Bid.Price > current.Bid.Price {
				best[candidate.ImpIndex] = candidate
			}
		}
	}

	winners := make([]middlemanDownstreamBid, 0, len(best))
	for _, imp := range fallbackImps {
		if selected, ok := best[imp.Index]; ok {
			winners = append(winners, selected)
		}
	}
	return winners, nil
}

func (self *Controller) middlemanRoutes(ctx context.Context) (*match.MiddlemanRouteCache, error) {
	countedMiss := false
	for {
		now := self.middlemanRouteTime()
		if state := self.middlemanRoute.Load(); state != nil && now.Before(state.expiresAt) {
			metricMiddlemanRouteCacheHits.Add(1)
			return state.cache, state.err
		}
		if !countedMiss {
			metricMiddlemanRouteCacheMisses.Add(1)
			countedMiss = true
		}

		self.middlemanRouteMu.Lock()
		now = self.middlemanRouteTime()
		if state := self.middlemanRoute.Load(); state != nil && now.Before(state.expiresAt) {
			self.middlemanRouteMu.Unlock()
			metricMiddlemanRouteCacheHits.Add(1)
			return state.cache, state.err
		}
		wait := self.middlemanRouteWait
		startedRefresh := false
		if wait == nil {
			wait = make(chan struct{})
			self.middlemanRouteWait = wait
			startedRefresh = true
		}
		self.middlemanRouteMu.Unlock()

		if startedRefresh {
			go self.refreshMiddlemanRoutes(ctx, wait)
		} else if self.middlemanRouteWaitHook != nil {
			self.middlemanRouteWaitHook()
		}
		select {
		case <-wait:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (self *Controller) refreshMiddlemanRoutes(parent context.Context, wait chan struct{}) {
	metricMiddlemanRouteCacheRefreshes.Add(1)
	loader := self.middlemanRouteLoad
	if loader == nil {
		loader = func(ctx context.Context) (*match.MiddlemanRouteCache, error) {
			return match.MiddlemanRouteCacheFromRedis(ctx, self.Redis)
		}
	}
	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), self.middlemanRouteLoadTimeout())
	cache, err := loader(loadCtx)
	cancel()
	if err != nil {
		metricMiddlemanRouteCacheErrors.Add(1)
	}
	self.middlemanRoute.Store(&middlemanRouteState{
		cache:     cache,
		err:       err,
		expiresAt: self.middlemanRouteTime().Add(self.middlemanRouteTTL()),
	})

	self.middlemanRouteMu.Lock()
	if self.middlemanRouteWait == wait {
		self.middlemanRouteWait = nil
		close(wait)
	}
	self.middlemanRouteMu.Unlock()
}

func (self *Controller) middlemanRouteTime() time.Time {
	if self != nil && self.middlemanRouteNow != nil {
		return self.middlemanRouteNow()
	}
	return time.Now()
}

func (self *Controller) middlemanRouteTTL() time.Duration {
	if self == nil || self.C == nil || self.C.MiddlemanRouteCacheTTLMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(self.C.MiddlemanRouteCacheTTLMS) * time.Millisecond
}

func (self *Controller) middlemanRouteLoadTimeout() time.Duration {
	if self == nil || self.C == nil || self.C.MiddlemanTimeoutMS <= 0 {
		return 100 * time.Millisecond
	}
	return time.Duration(self.C.MiddlemanTimeoutMS) * time.Millisecond
}

func (self *Controller) middlemanAssignments(bid *openrtb2.BidRequest, cache *match.MiddlemanRouteCache, fallbackImps []middlemanFallbackImp) []middlemanAssignment {
	assignments := make(map[uint32]*middlemanAssignment)
	for _, fallback := range fallbackImps {
		if fallback.Attr == nil {
			continue
		}
		candidates := middlemanCandidatesForImp(cache, fallback, self.C.MiddlemanMaxBiddersPerImp)
		for _, candidate := range candidates {
			if self.trafficQualityEnabled() {
				action, _ := self.qualityAction(
					trafficquality.Scope{Type: trafficquality.ScopePartner, ID: uint64(candidate.Entry.BidderID)},
					qualityPartnerEventKey(bid, fallback.Index, candidate.Entry.BidderID), fallback.Attr.When)
				if qualityBlocks(action) {
					continue
				}
			}
			assignment := assignments[candidate.Entry.BidderID]
			if assignment == nil {
				assignment = &middlemanAssignment{
					Entry:       candidate.Entry,
					EntriesByID: make(map[string]match.MiddlemanRouteEntry),
					AttrsByID:   make(map[string]*match.Attribute),
				}
				assignments[candidate.Entry.BidderID] = assignment
			}
			impID := candidateImpID(bid, candidate)
			if impID == "" {
				continue
			}
			if _, ok := assignment.EntriesByID[impID]; ok {
				continue
			}
			assignment.EntriesByID[impID] = candidate.Entry
			assignment.AttrsByID[impID] = candidate.Attr
		}
	}

	out := make([]middlemanAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		out = append(out, *assignment)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Entry.BidderID < out[j].Entry.BidderID
	})
	return out
}

func middlemanCandidatesForImp(cache *match.MiddlemanRouteCache, fallback middlemanFallbackImp, maxBidders int) []middlemanCandidate {
	if maxBidders <= 0 {
		maxBidders = 5
	}
	bestByBidder := make(map[uint32]middlemanCandidate)
	for _, entry := range cache.Entries {
		recordMiddlemanCandidate("considered", 1)
		if !entry.Eligible(fallback.Attr) {
			continue
		}
		if !middlemanTriggerAllowed(entry.TriggerMode, fallback.TriggerModes) {
			continue
		}
		recordMiddlemanCandidate("eligible", 1)
		candidate := middlemanCandidate{ImpIndex: fallback.Index, Attr: fallback.Attr, Entry: entry}
		current, ok := bestByBidder[entry.BidderID]
		if !ok || middlemanCandidateLess(candidate, current) {
			bestByBidder[entry.BidderID] = candidate
		}
	}
	candidates := make([]middlemanCandidate, 0, len(bestByBidder))
	for _, candidate := range bestByBidder {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return middlemanCandidateLess(candidates[i], candidates[j])
	})
	if len(candidates) > maxBidders {
		candidates = candidates[:maxBidders]
	}
	recordMiddlemanCandidate("assigned", len(candidates))
	return candidates
}

func middlemanTriggerAllowed(entryMode string, allowed []string) bool {
	if entryMode == "" {
		entryMode = "Fallback"
	}
	if len(allowed) == 0 {
		return entryMode == "Fallback"
	}
	for _, mode := range allowed {
		if mode == entryMode {
			return true
		}
	}
	return false
}

func middlemanCandidateLess(a, b middlemanCandidate) bool {
	if a.Entry.RouteBidderPriority != b.Entry.RouteBidderPriority {
		return a.Entry.RouteBidderPriority < b.Entry.RouteBidderPriority
	}
	if a.Entry.TargetPriority != b.Entry.TargetPriority {
		return a.Entry.TargetPriority < b.Entry.TargetPriority
	}
	if a.Entry.Specificity() != b.Entry.Specificity() {
		return a.Entry.Specificity() > b.Entry.Specificity()
	}
	if a.Entry.TargetID != b.Entry.TargetID {
		return a.Entry.TargetID < b.Entry.TargetID
	}
	if a.Entry.RouteBidderID != b.Entry.RouteBidderID {
		return a.Entry.RouteBidderID < b.Entry.RouteBidderID
	}
	return a.Entry.BidderID < b.Entry.BidderID
}

func (self *Controller) callMiddlemanBidder(ctx context.Context, client *http.Client, bid *openrtb2.BidRequest, rawRequest []byte, started time.Time, assignment middlemanAssignment) ([]middlemanDownstreamBid, error) {
	callStarted := time.Now()
	defer func() { metricMiddlemanBidderLatency.Observe(time.Since(callStarted)) }()
	outcome := "configuration_error"
	defer func() { recordMiddlemanBidderOutcome(outcome) }()
	acceptedCount := 0
	defer func() {
		requestID := ""
		if bid != nil {
			requestID = bid.ID
		}
		self.logOpenRTBDiagnostic(requestID, assignment.Entry.BidderID, outcome, len(assignment.EntriesByID), acceptedCount, time.Since(callStarted))
	}()
	if err := assignment.Entry.ValidatePartnerProfile(); err != nil {
		recordMiddlemanBidRejection("profile")
		return nil, fmt.Errorf("middleman partner profile: %w", err)
	}
	if err := self.validateMiddlemanBidderEndpoint(ctx, assignment.Entry.EndpointURL); err != nil {
		recordMiddlemanBidRejection("endpoint")
		return nil, err
	}
	headers, err := middlemanCredentialHeaders(assignment.Entry.CredentialRef)
	if err != nil {
		recordMiddlemanBidRejection("credential")
		return nil, err
	}
	clickRequestToken, err := newMiddlemanToken()
	if err != nil {
		recordMiddlemanBidRejection("callback")
		return nil, err
	}
	clickURLs, err := self.middlemanClickNotifyURLs(clickRequestToken, assignment.EntriesByID)
	if err != nil {
		recordMiddlemanBidRejection("callback")
		return nil, err
	}
	allowedImpIDs := make(map[string]struct{}, len(assignment.EntriesByID))
	for impID := range assignment.EntriesByID {
		allowedImpIDs[impID] = struct{}{}
	}
	body, err := middlemanRequestBodyForAssignmentImps(rawRequest, self.C.MiddlemanExchangeDomain, clickURLs, allowedImpIDs)
	if err != nil {
		recordMiddlemanBidRejection("envelope")
		return nil, err
	}
	body, err = prepareMiddlemanOutboundRequest(body, bid.ID, assignment.AttrsByID)
	if err != nil {
		recordMiddlemanBidRejection("envelope")
		return nil, err
	}

	timeout, ok := middlemanAssignmentTimeout(bid, assignment, self.C.MiddlemanTimeoutMS, time.Since(started))
	if !ok {
		recordMiddlemanBidRejection("timeout")
		return nil, fmt.Errorf("middleman tmax exhausted")
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, assignment.Entry.EndpointURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, values := range headers {
		for _, value := range values {
			req.Header.Add(k, value)
		}
	}
	req.Header.Set("Content-Type", "application/openrtb+json")
	req.Header.Set("Accept", "application/openrtb+json, application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if assignment.Entry.OpenRTBVersion != "" {
		req.Header.Set("x-openrtb-version", assignment.Entry.OpenRTBVersion)
	}

	resp, err := client.Do(req)
	if err != nil {
		outcome = "dependency_error"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			outcome = "timeout"
			recordMiddlemanBidRejection("timeout")
		}
		return nil, err
	}
	defer resp.Body.Close()
	if err := callCtx.Err(); err != nil {
		outcome = "timeout"
		recordMiddlemanBidRejection("late")
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		outcome = "no_bid"
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		recordMiddlemanBidRejection("status")
		outcome = "invalid_response"
		if resp.StatusCode == http.StatusTooManyRequests {
			outcome = "overload"
		} else if resp.StatusCode >= http.StatusInternalServerError {
			outcome = "dependency_error"
		}
		return nil, fmt.Errorf("downstream status %d", resp.StatusCode)
	}
	if err := validateMiddlemanResponseHeaders(resp.Header); err != nil {
		outcome = "invalid_response"
		recordMiddlemanBidRejection("content_type")
		return nil, err
	}
	respBody, err := readMiddlemanResponseBody(resp)
	if err != nil {
		outcome = "dependency_error"
		var protocolError middlemanResponseProtocolError
		if errors.As(err, &protocolError) {
			outcome = "invalid_response"
		}
		recordMiddlemanBidRejection("body")
		return nil, err
	}
	if err := callCtx.Err(); err != nil {
		outcome = "timeout"
		recordMiddlemanBidRejection("late")
		return nil, err
	}
	if len(respBody) > maxBidRequestBodyBytes {
		outcome = "invalid_response"
		recordMiddlemanBidRejection("body")
		return nil, fmt.Errorf("middleman response exceeds %d bytes", maxBidRequestBodyBytes)
	}

	var downstream openrtb2.BidResponse
	if err := json.Unmarshal(respBody, &downstream); err != nil {
		outcome = "invalid_response"
		recordMiddlemanBidRejection("envelope")
		return nil, err
	}
	if strings.TrimSpace(downstream.ID) == "" || downstream.ID != bid.ID {
		outcome = "invalid_response"
		recordMiddlemanBidRejection("request_id")
		return nil, nil
	}
	if downstream.Cur != "" && !strings.EqualFold(downstream.Cur, "USD") {
		outcome = "invalid_response"
		recordMiddlemanBidRejection("currency")
		return nil, nil
	}

	var out []middlemanDownstreamBid
	sawInvalidBid := false
	seenBidIDs := make(map[string]struct{})
	for _, seatBid := range downstream.SeatBid {
		if seatBid.Group != 0 || (assignment.Entry.Seat != "" && seatBid.Seat != assignment.Entry.Seat) {
			recordMiddlemanBidRejection("seat")
			sawInvalidBid = true
			continue
		}
		for _, rspBid := range seatBid.Bid {
			recordMiddlemanCandidate("returned", 1)
			if strings.TrimSpace(rspBid.ID) == "" {
				recordMiddlemanBidRejection("identity")
				sawInvalidBid = true
				continue
			}
			if _, duplicate := seenBidIDs[rspBid.ID]; duplicate {
				recordMiddlemanBidRejection("identity")
				sawInvalidBid = true
				continue
			}
			seenBidIDs[rspBid.ID] = struct{}{}
			entry, ok := assignment.EntriesByID[rspBid.ImpID]
			if !ok {
				recordMiddlemanBidRejection("impression")
				sawInvalidBid = true
				continue
			}
			impIndex := middlemanImpIndexByID(bid, rspBid.ImpID)
			if impIndex < 0 || !supportedMiddlemanBidFloorCurrency(bid.Imp[impIndex].BidFloorCur) {
				recordMiddlemanBidRejection("currency")
				sawInvalidBid = true
				continue
			}
			auditAttr := assignment.AttrsByID[rspBid.ImpID]
			if err := validateMiddlemanDownstreamBid(&bid.Imp[impIndex], auditAttr, rspBid); err != nil {
				recordMiddlemanBidRejection(classifyMiddlemanBidValidation(err))
				sawInvalidBid = true
				continue
			}
			downstreamCPM, err := protocolCPM(rspBid.Price)
			if err != nil {
				recordMiddlemanBidRejection("price")
				sawInvalidBid = true
				continue
			}
			floorCPM, err := protocolCPM(bid.Imp[impIndex].BidFloor)
			if err != nil {
				recordMiddlemanBidRejection("floor")
				sawInvalidBid = true
				continue
			}
			if downstreamCPM < floorCPM {
				recordMiddlemanBidRejection("floor")
				continue
			}
			markedCPM, err := middlemanMarkedCPM(downstreamCPM, entry)
			if err != nil || markedCPM <= 0 {
				recordMiddlemanBidRejection("price")
				sawInvalidBid = true
				continue
			}
			markedPrice := markedCPM.Float64()
			downstreamPrice := rspBid.Price
			downstreamNURL := rspBid.NURL
			downstreamBURL := rspBid.BURL
			downstreamLURL := rspBid.LURL
			downstreamAdID := rspBid.AdID
			rspBid.Price = markedPrice
			rspBid.CID = strconv.FormatUint(uint64(entry.SyntheticCampaignID), 10)
			rspBid.CrID = strconv.FormatUint(uint64(entry.SyntheticCreativeID), 10)
			rspBid.AdID = strconv.FormatUint(uint64(entry.SyntheticCreativeID), 10)
			responseBidID := downstream.BidID
			if responseBidID == "" {
				responseBidID = rspBid.ID
			}
			out = append(out, middlemanDownstreamBid{
				ImpIndex: impIndex,
				Seat:     strconv.FormatUint(uint64(entry.SyntheticCampaignID), 10),
				Bid:      rspBid,
				Audit: bidAudit{
					Attr: auditAttr,
					One: match.RAdv{
						Demand: match.Demand{
							AdvID:      entry.AdvID,
							CampaignID: entry.SyntheticCampaignID,
							ItemID:     entry.SyntheticItemID,
							CreativeID: entry.SyntheticCreativeID,
						},
						CostType: match.CostTypeCPM,
						Cost:     float32(markedPrice),
						CostCPM:  markedCPM,
					},
				},
				ResponseBidID:      responseBidID,
				Entry:              entry,
				DownstreamSeat:     seatBid.Seat,
				DownstreamAdID:     downstreamAdID,
				DownstreamNURL:     downstreamNURL,
				DownstreamBURL:     downstreamBURL,
				DownstreamLURL:     downstreamLURL,
				DownstreamBidPrice: downstreamPrice,
				UpstreamBidPrice:   markedPrice,
				DownstreamBidCPM:   downstreamCPM,
				UpstreamBidCPM:     markedCPM,
				ClickRequestToken:  clickRequestToken,
			})
			acceptedCount++
			recordMiddlemanCandidate("accepted", 1)
		}
	}
	if err := callCtx.Err(); err != nil {
		outcome = "timeout"
		recordMiddlemanBidRejection("late")
		return nil, err
	}
	outcome = "no_bid"
	if len(out) != 0 {
		outcome = "fill"
	} else if sawInvalidBid {
		outcome = "invalid_response"
	}
	return out, nil
}

func (self *Controller) logOpenRTBDiagnostic(requestID string, bidderID uint32, outcome string, requestedImps, acceptedBids int, elapsed time.Duration) {
	if self == nil || self.C == nil || !self.C.OpenRTBDebugEnabled || self.Logger == nil {
		return
	}
	rate := self.C.OpenRTBDebugSampleRate
	if rate <= 0 || rate > 1 {
		return
	}
	sum := sha256.Sum256([]byte(requestID + "|" + strconv.FormatUint(uint64(bidderID), 10)))
	if rate < 1 {
		sample := binary.BigEndian.Uint64(sum[:8])
		if float64(sample)/float64(^uint64(0)) >= rate {
			return
		}
	}
	self.Logger.Debug("sampled middleman OpenRTB diagnostic",
		zap.String("request_id_hash", hex.EncodeToString(sum[:8])),
		zap.Uint32("bidder_id", bidderID),
		zap.String("outcome", outcome),
		zap.Int("requested_impressions", requestedImps),
		zap.Int("accepted_bids", acceptedBids),
		zap.Int64("elapsed_ms", elapsed.Milliseconds()),
	)
}

func safeOpenRTBRequestIDHash(requestID string) string {
	sum := sha256.Sum256([]byte(requestID))
	return hex.EncodeToString(sum[:8])
}

func middlemanSafeFailureReason(err error) string {
	if err == nil {
		return "other"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "profile"), strings.Contains(message, "credential"):
		return "configuration"
	case strings.Contains(message, "unsafe"), strings.Contains(message, "endpoint"):
		return "endpoint"
	case strings.Contains(message, "response"), strings.Contains(message, "downstream"), strings.Contains(message, "json"):
		return "invalid_response"
	default:
		return "dependency"
	}
}

func validateMiddlemanResponseHeaders(header http.Header) error {
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(header.Get("Content-Type"), ";")[0]))
	if contentType != "application/json" && contentType != "application/openrtb+json" {
		return fmt.Errorf("middleman response content type is not OpenRTB JSON")
	}
	if version := strings.TrimSpace(header.Get("x-openrtb-version")); version != "" && version != "2.5" {
		return fmt.Errorf("middleman response OpenRTB version is not 2.5")
	}
	return nil
}

func readMiddlemanResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, middlemanResponseProtocolError{fmt.Errorf("middleman response body is missing")}
	}
	if resp.ContentLength > maxBidRequestBodyBytes {
		return nil, middlemanResponseProtocolError{fmt.Errorf("middleman compressed response exceeds %d bytes", maxBidRequestBodyBytes)}
	}
	raw := io.LimitReader(resp.Body, maxBidRequestBodyBytes+1)
	var reader io.Reader = raw
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	switch encoding {
	case "", "identity":
	case "gzip":
		compressed, err := gzip.NewReader(raw)
		if err != nil {
			return nil, middlemanResponseProtocolError{fmt.Errorf("middleman gzip response is invalid: %w", err)}
		}
		defer compressed.Close()
		reader = compressed
	default:
		return nil, middlemanResponseProtocolError{fmt.Errorf("middleman response content encoding is unsupported")}
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxBidRequestBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBidRequestBodyBytes {
		return nil, middlemanResponseProtocolError{fmt.Errorf("middleman decompressed response exceeds %d bytes", maxBidRequestBodyBytes)}
	}
	return body, nil
}

type middlemanResponseProtocolError struct {
	err error
}

func (e middlemanResponseProtocolError) Error() string { return e.err.Error() }
func (e middlemanResponseProtocolError) Unwrap() error { return e.err }

func classifyMiddlemanBidValidation(err error) string {
	if err == nil {
		return "other"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "dimension"):
		return "size"
	case strings.Contains(message, "price"):
		return "price"
	case strings.Contains(message, "identifier"):
		return "identity"
	case strings.Contains(message, "impression"):
		return "impression"
	case strings.Contains(message, "nurl"), strings.Contains(message, "burl"), strings.Contains(message, "lurl"), strings.Contains(message, "callback"):
		return "callback"
	case strings.Contains(message, "markup type"), strings.Contains(message, "media type"):
		return "media"
	case strings.Contains(message, "adm"), strings.Contains(message, "vast"), strings.Contains(message, "native"), strings.Contains(message, "container"), strings.Contains(message, "url"):
		return "markup"
	default:
		return "other"
	}
}

func (self *Controller) validateMiddlemanBidderEndpoint(ctx context.Context, raw string) error {
	if err := safehttp.ValidateCallbackURL(ctx, raw); err != nil {
		return err
	}
	if self != nil && self.callbackURLGuard != nil {
		return self.callbackURLGuard(ctx, raw)
	}
	return nil
}

func (self *Controller) middlemanClickNotifyURLs(requestToken string, entries map[string]match.MiddlemanRouteEntry) (map[string]string, error) {
	if requestToken == "" || len(entries) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(entries))
	for impID := range entries {
		u, err := self.middlemanClickProxyURL(requestToken, impID)
		if err != nil {
			return nil, err
		}
		out[impID] = u
	}
	return out, nil
}

func middlemanRequestBodyForAssignment(rawRequest []byte, exchangeDomain string, clickNotifyURLs ...map[string]string) ([]byte, error) {
	var clickURLs map[string]string
	if len(clickNotifyURLs) != 0 {
		clickURLs = clickNotifyURLs[0]
	}
	return middlemanRequestBodyForAssignmentImps(rawRequest, exchangeDomain, clickURLs, nil)
}

func middlemanRequestBodyForAssignmentImps(rawRequest []byte, exchangeDomain string, clickNotifyURLs map[string]string, allowedImpIDs map[string]struct{}) ([]byte, error) {
	var bid openrtb2.BidRequest
	if err := json.Unmarshal(rawRequest, &bid); err != nil {
		return nil, err
	}
	seenImpIDs := make(map[string]struct{}, len(bid.Imp))
	for _, imp := range bid.Imp {
		if imp.ID == "" {
			return nil, fmt.Errorf("middleman request contains an empty impression id")
		}
		if _, duplicate := seenImpIDs[imp.ID]; duplicate {
			return nil, fmt.Errorf("middleman request contains duplicate impression id %q", imp.ID)
		}
		seenImpIDs[imp.ID] = struct{}{}
	}
	if allowedImpIDs != nil {
		filtered := make([]openrtb2.Imp, 0, len(allowedImpIDs))
		for _, imp := range bid.Imp {
			if _, allowed := allowedImpIDs[imp.ID]; allowed {
				filtered = append(filtered, imp)
			}
		}
		bid.Imp = filtered
		if len(bid.Imp) != len(allowedImpIDs) {
			return nil, fmt.Errorf("middleman assignment contains %d requested ids but %d matching impressions", len(allowedImpIDs), len(bid.Imp))
		}
	}
	raw, err := json.Marshal(&bid)
	if err != nil {
		return nil, err
	}
	raw, err = privacySanitizeJSON(raw, false)
	if err != nil {
		return nil, err
	}
	var sanitized openrtb2.BidRequest
	if err := json.Unmarshal(raw, &sanitized); err != nil {
		return nil, err
	}
	bid = sanitized
	if err := validatePartnerSource(bid.Source); err != nil {
		return nil, err
	}
	if bid.Source != nil {
		// Payment-chain strings are not part of P02's public disclosure
		// allowlist. A validated schain remains available to the partner.
		bid.Source.PChain = ""
	}
	bid.Cur = []string{"USD"}
	for i := range bid.Imp {
		if !supportedMiddlemanBidFloorCurrency(bid.Imp[i].BidFloorCur) {
			return nil, fmt.Errorf("middleman request impression %q has unsupported floor currency", bid.Imp[i].ID)
		}
		bid.Imp[i].BidFloorCur = "USD"
	}
	ext := make(map[string]any)
	if exchangeDomain != "" {
		ext["request_domain"] = exchangeDomain
	}
	if len(clickNotifyURLs) > 0 {
		ext["aofei_middleman"] = map[string]any{"click_notify_urls": clickNotifyURLs}
	}
	if len(ext) != 0 {
		bid.Ext, err = json.Marshal(ext)
		if err != nil {
			return nil, err
		}
	} else {
		bid.Ext = nil
	}
	return json.Marshal(&bid)
}

func prepareMiddlemanOutboundRequest(raw []byte, requestID string, attrs map[string]*match.Attribute) ([]byte, error) {
	var bid openrtb2.BidRequest
	if err := json.Unmarshal(raw, &bid); err != nil {
		return nil, fmt.Errorf("middleman outbound request is malformed: %w", err)
	}
	if bid.ID == "" || bid.ID != requestID || len(bid.Imp) != len(attrs) || len(bid.Imp) == 0 {
		return nil, fmt.Errorf("middleman outbound request envelope does not match its assignment")
	}
	if len(bid.Cur) != 1 || bid.Cur[0] != "USD" {
		return nil, fmt.Errorf("middleman outbound request must use USD")
	}
	seen := make(map[string]struct{}, len(bid.Imp))
	for i := range bid.Imp {
		imp := &bid.Imp[i]
		if imp.ID == "" || imp.BidFloorCur != "USD" {
			return nil, fmt.Errorf("middleman outbound impression identity or currency is invalid")
		}
		if _, duplicate := seen[imp.ID]; duplicate {
			return nil, fmt.Errorf("middleman outbound impression id is duplicated")
		}
		seen[imp.ID] = struct{}{}
		attr := attrs[imp.ID]
		if attr == nil {
			return nil, fmt.Errorf("middleman outbound impression is missing selected media metadata")
		}
		switch {
		case attr.NativeFormat != nil:
			if imp.Native == nil {
				return nil, fmt.Errorf("middleman outbound native impression is missing native media")
			}
			imp.Banner, imp.Video, imp.Audio = nil, nil, nil
		case attr.IsVideo:
			if imp.Video == nil {
				return nil, fmt.Errorf("middleman outbound video impression is missing video media")
			}
			imp.Banner, imp.Native, imp.Audio = nil, nil, nil
		default:
			if imp.Banner == nil {
				return nil, fmt.Errorf("middleman outbound banner impression is missing banner media")
			}
			imp.Video, imp.Native, imp.Audio = nil, nil, nil
		}
	}
	return json.Marshal(&bid)
}

func middlemanCredentialHeaders(ref string) (http.Header, error) {
	if ref == "" {
		return nil, fmt.Errorf("middleman credential ref is empty")
	}
	raw := os.Getenv(ref)
	if raw == "" {
		return nil, fmt.Errorf("middleman credential env %q is empty", ref)
	}
	values := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	header := make(http.Header)
	for k, v := range values {
		if !httpguts.ValidHeaderFieldName(k) {
			return nil, fmt.Errorf("middleman credential header name %q is invalid", k)
		}
		if !httpguts.ValidHeaderFieldValue(v) {
			return nil, fmt.Errorf("middleman credential header %q has an invalid value", k)
		}
		if middlemanBlockedHeader(k) {
			return nil, fmt.Errorf("middleman credential header %q is not allowed", k)
		}
		header.Set(k, v)
	}
	return header, nil
}

// ValidateMiddlemanCredentialRef confirms that an environment-backed bidder
// credential exists and contains only allowed outbound headers. It deliberately
// returns no header values so readiness tooling cannot expose credentials.
func ValidateMiddlemanCredentialRef(ref string) error {
	_, err := middlemanCredentialHeaders(ref)
	return err
}

func middlemanBlockedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "host", "connection", "content-length", "transfer-encoding", "upgrade", "trailer", "te", "proxy-authorization", "proxy-authenticate", "keep-alive":
		return true
	default:
		return false
	}
}

func middlemanAssignmentTimeout(bid *openrtb2.BidRequest, assignment middlemanAssignment, configTimeoutMS int, elapsed time.Duration) (time.Duration, bool) {
	var values []int64
	if bid != nil && bid.TMax > 0 {
		remaining := bid.TMax - durationCeilMillis(elapsed)
		if remaining <= 0 {
			return 0, false
		}
		values = append(values, remaining)
	}
	if configTimeoutMS > 0 {
		values = append(values, int64(configTimeoutMS))
	}
	for _, entry := range assignment.EntriesByID {
		if entry.GroupTimeoutMS > 0 {
			values = append(values, int64(entry.GroupTimeoutMS))
		}
		if timeout := entry.EffectiveTimeoutMS(); timeout > 0 {
			values = append(values, int64(timeout))
		}
	}
	if len(values) == 0 {
		return 100 * time.Millisecond, true
	}
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return time.Duration(minimum) * time.Millisecond, true
}

func durationCeilMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64((d + time.Millisecond - 1) / time.Millisecond)
}

func middlemanMarkedCPM(price accounting.CPM, entry match.MiddlemanRouteEntry) (accounting.CPM, error) {
	if price <= 0 || price > accounting.MaxCPM {
		return 0, fmt.Errorf("downstream CPM is outside the supported range")
	}
	units, minimum, err := entry.ExactMarginTerms()
	if err != nil {
		return 0, err
	}
	// The four-place percentage product rounds half away from zero once at the
	// six-place CPM boundary. Inputs are nonnegative, so adding half the divisor
	// implements the published rule without binary floating point.
	product := int64(price) * int64(units)
	margin := accounting.CPM((product + 5_000) / 10_000)
	if margin < minimum {
		margin = minimum
	}
	if margin > accounting.MaxCPM-price {
		return 0, fmt.Errorf("marked-up CPM exceeds %s", accounting.MaxCPM)
	}
	return price + margin, nil
}

func middlemanImpIndexByID(bid *openrtb2.BidRequest, impID string) int {
	if bid == nil {
		return -1
	}
	for i := range bid.Imp {
		if bid.Imp[i].ID == impID {
			return i
		}
	}
	return -1
}

func candidateImpID(bid *openrtb2.BidRequest, candidate middlemanCandidate) string {
	if bid == nil || candidate.ImpIndex < 0 || candidate.ImpIndex >= len(bid.Imp) {
		return ""
	}
	return bid.Imp[candidate.ImpIndex].ID
}

func supportedMiddlemanBidFloorCurrency(cur string) bool {
	return cur == "" || strings.EqualFold(cur, "USD")
}
