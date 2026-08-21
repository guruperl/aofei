package dsp

import (
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestTrackingSignatureRequiresTimestamp(t *testing.T) {
	args := url.Values{"auction_id": []string{"auction"}}
	args.Set(trackingSignatureParam, signTrackingValues("secret", "/imp", args))
	if _, err := validateTrackingSignature("secret", "/imp", args, time.Hour); err == nil {
		t.Fatal("expected missing signature timestamp to fail")
	}
}

func TestTrackingSignatureRejectsExpiredTimestamp(t *testing.T) {
	args := url.Values{"auction_id": []string{"auction"}}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("secret", "/imp", args))
	if _, err := validateTrackingSignature("secret", "/imp", args, time.Hour); err == nil {
		t.Fatal("expected expired signature timestamp to fail")
	}
}

func TestAddTrackingSignatureAddsTimestamp(t *testing.T) {
	args := url.Values{"auction_id": []string{"auction"}}
	addTrackingSignature("secret", "/imp", args)
	if args.Get(trackingSignatureTimestampParam) == "" {
		t.Fatal("sig_ts was not added")
	}
	if _, err := validateTrackingSignature("secret", "/imp", args, time.Hour); err != nil {
		t.Fatal(err)
	}
}

func TestTrackingSignatureReturnsSignedTimestampDeadline(t *testing.T) {
	signedAt := time.Now().Add(4 * time.Minute).Truncate(time.Second)
	args := url.Values{"auction_id": []string{"auction"}}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(signedAt.Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("secret", "/imp", args))
	validUntil, err := validateTrackingSignature("secret", "/imp", args, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	want := signedAt.Add(time.Hour)
	if !validUntil.Equal(want) {
		t.Fatalf("valid until = %s, want %s", validUntil, want)
	}
}

func TestWinLossSignatureAllowsOnlyExchangeResolvedMacroChanges(t *testing.T) {
	args := url.Values{
		"auction_id":       []string{"${AUCTION_ID}"},
		"auction_bid_id":   []string{"${AUCTION_BID_ID}"},
		"auction_imp_id":   []string{"${AUCTION_IMP_ID}"},
		"auction_price":    []string{"${AUCTION_PRICE}"},
		"auction_currency": []string{"${AUCTION_CURRENCY}"},
		"demand":           []string{"signed-demand"},
	}
	addTrackingSignature("secret", "/win", args)
	args.Set("auction_id", "auction")
	args.Set("auction_bid_id", "bid")
	args.Set("auction_imp_id", "imp")
	args.Set("auction_price", "1.250")
	args.Set("auction_currency", "USD")
	if _, err := validateTrackingSignature("secret", "/win", args, time.Hour); err != nil {
		t.Fatalf("exchange macro resolution invalidated signature: %v", err)
	}
	args.Set("demand", "tampered-demand")
	if _, err := validateTrackingSignature("secret", "/win", args, time.Hour); err == nil {
		t.Fatal("signed immutable demand accepted after tampering")
	}
}
