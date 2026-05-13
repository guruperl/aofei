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
	if err := validateTrackingSignature("secret", "/imp", args); err == nil {
		t.Fatal("expected missing signature timestamp to fail")
	}
}

func TestTrackingSignatureRejectsExpiredTimestamp(t *testing.T) {
	args := url.Values{"auction_id": []string{"auction"}}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("secret", "/imp", args))
	if err := validateTrackingSignature("secret", "/imp", args, time.Hour); err == nil {
		t.Fatal("expected expired signature timestamp to fail")
	}
}

func TestAddTrackingSignatureAddsTimestamp(t *testing.T) {
	args := url.Values{"auction_id": []string{"auction"}}
	addTrackingSignature("secret", "/imp", args)
	if args.Get(trackingSignatureTimestampParam) == "" {
		t.Fatal("sig_ts was not added")
	}
	if err := validateTrackingSignature("secret", "/imp", args); err != nil {
		t.Fatal(err)
	}
}
