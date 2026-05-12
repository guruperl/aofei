package dsp

import (
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

func TestServeStatusRejectsInvalidAuctionPrice(t *testing.T) {
	err := (&Controller{}).serveStatus(nil, StatusWin, time.Now(), url.Values{
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

func zeroRAdv() match.RAdv {
	return match.RAdv{}
}
