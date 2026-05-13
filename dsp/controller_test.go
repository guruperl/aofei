package dsp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/guruperl/aofei/match"
	"go.uber.org/zap"
)

func TestControllerOptionsCanDisableNATSAndMaxMindIndependently(t *testing.T) {
	defaults := applyControllerOptions()
	if !defaults.nats || !defaults.maxmind {
		t.Fatalf("defaults = %+v, want both optional services enabled", defaults)
	}

	withoutNATS := applyControllerOptions(WithoutNATS())
	if withoutNATS.nats || !withoutNATS.maxmind {
		t.Fatalf("WithoutNATS = %+v, want nats disabled and maxmind enabled", withoutNATS)
	}

	withoutMaxMind := applyControllerOptions(WithoutMaxMind())
	if !withoutMaxMind.nats || withoutMaxMind.maxmind {
		t.Fatalf("WithoutMaxMind = %+v, want nats enabled and maxmind disabled", withoutMaxMind)
	}

	withoutBoth := applyControllerOptions(WithoutNATS(), WithoutMaxMind())
	if withoutBoth.nats || withoutBoth.maxmind {
		t.Fatalf("without both = %+v, want both disabled", withoutBoth)
	}

	guard := func(context.Context, string) error { return fmt.Errorf("guard") }
	client := &http.Client{}
	logger := zap.NewNop()
	withDeps := applyControllerOptions(WithHTTPClient(client), WithLogger(logger), WithCallbackURLGuard(guard), withMiddlemanCallbackStore(newMemoryMiddlemanCallbackStore()))
	if withDeps.httpClient != client || withDeps.logger != logger || withDeps.callbackURLGuard == nil || withDeps.callbackStore == nil {
		t.Fatalf("injected deps not retained: %+v", withDeps)
	}
}

func TestServeStatusRejectsInvalidTrackingAuctionPrice(t *testing.T) {
	err := (&Controller{}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), url.Values{
		"auction_price": []string{"bad-price"},
	})
	if err == nil {
		t.Fatal("expected invalid auction_price to return an error")
	}
}

func TestPublishBidAuditNoNATSIsNoop(t *testing.T) {
	err := (&Controller{}).publishBidAudit(nil, nil, nil, zeroRAdv(), 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServeStatusRequiresSignatureForCapMutation(t *testing.T) {
	capValue, err := (match.Cap{CapNumber: 1, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	err = (&Controller{C: &Config{TrackingSecret: "test-secret"}}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
		"cap":              []string{capValue},
	})
	if err == nil {
		t.Fatal("expected unsigned cap mutation to fail")
	}
}

func TestServeWinLossRequiresSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/win?auction_bid_id=bid", nil)
	rr := httptest.NewRecorder()
	(&Controller{C: &Config{TrackingSecret: "test-secret"}}).ServeWinLoss(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request", rr.Code)
	}
}

func TestServeWinLossPublishesWinOnce(t *testing.T) {
	winloss := NewWinLoss(
		StatusBid,
		time.Now(),
		match.RPub{PubID: 1, SiteID: 2, SlotID: 3},
		match.RAdv{Demand: match.Demand{AdvID: 4, CampaignID: 5, ItemID: 6, CreativeID: 7}, Cost: 1.25, CostType: 2},
		nil,
		"5",
		"auction",
		"bid",
		"imp",
		"7",
		"https://dsp.example",
	).WithTrackingSecret("test-secret")
	u, err := url.Parse(winloss.NURL())
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("auction_id", "auction")
	q.Set("auction_bid_id", "bid")
	q.Set("auction_imp_id", "imp")
	q.Set("auction_price", "1.25")
	q.Set("auction_currency", "USD")
	u.RawQuery = q.Encode()

	seen := false
	published := 0
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret"},
		trackingNotifyOnce: func(_ context.Context, _ Status, _ string, _ time.Duration) (bool, error) {
			if seen {
				return false, nil
			}
			seen = true
			return true, nil
		},
		publishWinLossFunc: func(_ []byte) error {
			published++
			return nil
		},
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
		rr := httptest.NewRecorder()
		controller.ServeWinLoss(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
		}
	}
	if published != 1 {
		t.Fatalf("published = %d, want one", published)
	}
}

func TestAuditPublisherDropsWhenQueueFull(t *testing.T) {
	publisher := newAuditPublisher(nil, 1)
	defer publisher.Close()

	publisher.Enqueue(SUBJECTRequest, []byte("one"))
	publisher.Enqueue(SUBJECTResponse, []byte("two"))
	if got := publisher.Dropped(); got != 1 {
		t.Fatalf("Dropped = %d, want 1", got)
	}
}

func zeroRAdv() match.RAdv {
	return match.RAdv{}
}
