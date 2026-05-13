package dsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

type middlemanFallbackImp struct {
	Index int
	Attr  *match.Attribute
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
	ClickRequestToken  string
}

func (self *Controller) middlemanFallback(ctx context.Context, bid *openrtb2.BidRequest, rawRequest []byte, started time.Time, fallbackImps []middlemanFallbackImp) ([]middlemanDownstreamBid, error) {
	if self.C == nil || !self.C.MiddlemanEnabled || self.Redis == nil || len(fallbackImps) == 0 {
		return nil, nil
	}
	cache, err := match.MiddlemanRouteCacheFromRedis(ctx, self.Redis)
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
	if client == nil {
		client = http.DefaultClient
	}
	logger := self.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	glog := logger.Sugar()
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
				glog.Infof("middleman bidder %d failed: %v", assignment.Entry.BidderID, err)
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
			if err := self.prepareMiddlemanCallback(ctx, bid, &selected); err != nil {
				glog.Infof("middleman bidder %d callback setup failed: %v", selected.Entry.BidderID, err)
				continue
			}
			winners = append(winners, selected)
		}
	}
	return winners, nil
}

func (self *Controller) middlemanAssignments(bid *openrtb2.BidRequest, cache *match.MiddlemanRouteCache, fallbackImps []middlemanFallbackImp) []middlemanAssignment {
	assignments := make(map[uint32]*middlemanAssignment)
	for _, fallback := range fallbackImps {
		if fallback.Attr == nil {
			continue
		}
		candidates := middlemanCandidatesForImp(cache, fallback, self.C.MiddlemanMaxBiddersPerImp)
		for _, candidate := range candidates {
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
		if !entry.Eligible(fallback.Attr) {
			continue
		}
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
	return candidates
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
	headers, err := middlemanCredentialHeaders(assignment.Entry.CredentialRef)
	if err != nil {
		return nil, err
	}
	clickRequestToken, err := newMiddlemanToken()
	if err != nil {
		return nil, err
	}
	clickURLs, err := self.middlemanClickNotifyURLs(clickRequestToken, assignment.EntriesByID)
	if err != nil {
		return nil, err
	}
	body, err := middlemanRequestBodyForAssignment(rawRequest, self.C.MiddlemanExchangeDomain, clickURLs)
	if err != nil {
		return nil, err
	}

	timeout, ok := middlemanAssignmentTimeout(bid, assignment, self.C.MiddlemanTimeoutMS, time.Since(started))
	if !ok {
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if assignment.Entry.OpenRTBVersion != "" {
		req.Header.Set("x-openrtb-version", assignment.Entry.OpenRTBVersion)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downstream status %d", resp.StatusCode)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBidRequestBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(respBody) > maxBidRequestBodyBytes {
		return nil, fmt.Errorf("middleman response exceeds %d bytes", maxBidRequestBodyBytes)
	}

	var downstream openrtb2.BidResponse
	if err := json.Unmarshal(respBody, &downstream); err != nil {
		return nil, err
	}
	if downstream.Cur != "" && !strings.EqualFold(downstream.Cur, "USD") {
		return nil, nil
	}

	var out []middlemanDownstreamBid
	for _, seatBid := range downstream.SeatBid {
		for _, rspBid := range seatBid.Bid {
			entry, ok := assignment.EntriesByID[rspBid.ImpID]
			if !ok || rspBid.Price <= 0 {
				continue
			}
			impIndex := middlemanImpIndexByID(bid, rspBid.ImpID)
			if impIndex < 0 || !supportedMiddlemanBidFloorCurrency(bid.Imp[impIndex].BidFloorCur) {
				continue
			}
			markedPrice := middlemanMarkedPrice(rspBid.Price, entry)
			if markedPrice < bid.Imp[impIndex].BidFloor {
				continue
			}
			downstreamPrice := rspBid.Price
			downstreamNURL := rspBid.NURL
			downstreamBURL := rspBid.BURL
			downstreamLURL := rspBid.LURL
			downstreamAdID := rspBid.AdID
			rspBid.Price = markedPrice
			rspBid.CID = strconv.FormatUint(uint64(entry.SyntheticCampaignID), 10)
			rspBid.CrID = strconv.FormatUint(uint64(entry.SyntheticCreativeID), 10)
			rspBid.AdID = strconv.FormatUint(uint64(entry.SyntheticCreativeID), 10)
			auditAttr := assignment.AttrsByID[rspBid.ImpID]
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
						CostType: 2,
						Cost:     float32(markedPrice),
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
				ClickRequestToken:  clickRequestToken,
			})
		}
	}
	return out, nil
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
	if exchangeDomain == "" {
		if len(clickNotifyURLs) == 0 || len(clickNotifyURLs[0]) == 0 {
			return append([]byte(nil), rawRequest...), nil
		}
	}
	root := make(map[string]json.RawMessage)
	if err := json.Unmarshal(rawRequest, &root); err != nil {
		return nil, err
	}
	ext := make(map[string]json.RawMessage)
	if rawExt := root["ext"]; len(rawExt) != 0 {
		if err := json.Unmarshal(rawExt, &ext); err != nil {
			ext = make(map[string]json.RawMessage)
		}
	}
	data, err := json.Marshal(exchangeDomain)
	if err != nil {
		return nil, err
	}
	if exchangeDomain != "" {
		ext["request_domain"] = data
	}
	if len(clickNotifyURLs) > 0 && len(clickNotifyURLs[0]) > 0 {
		payload := map[string]any{"click_notify_urls": clickNotifyURLs[0]}
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		ext["aofei_middleman"] = data
	}
	root["ext"], err = json.Marshal(ext)
	if err != nil {
		return nil, err
	}
	return json.Marshal(root)
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
		if middlemanBlockedHeader(k) {
			return nil, fmt.Errorf("middleman credential header %q is not allowed", k)
		}
		header.Set(k, v)
	}
	return header, nil
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

func middlemanMarkedPrice(price float64, entry match.MiddlemanRouteEntry) float64 {
	margin := price * entry.EffectiveMarginPct()
	if min := entry.EffectiveMinMarginCPM(); margin < min {
		margin = min
	}
	return price + margin
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
