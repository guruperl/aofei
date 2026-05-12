package dsp

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/genelet/winter/match"
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
