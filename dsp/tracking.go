package dsp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"
)

const (
	trackingSignatureParam          = "sig"
	trackingSignatureTimestampParam = "sig_ts"
	defaultTrackingSignatureTTL     = 24 * time.Hour
	maxTrackingSignatureFutureSkew  = 5 * time.Minute
)

var winLossMacroKeys = map[string]struct{}{
	"auction_id":       {},
	"auction_bid_id":   {},
	"auction_imp_id":   {},
	"auction_price":    {},
	"auction_currency": {},
}

func signTrackingValues(secret, path string, args url.Values) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonicalTrackingPayload(path, args)))
	return hex.EncodeToString(mac.Sum(nil))
}

func addTrackingSignature(secret, path string, args url.Values) {
	if secret == "" {
		return
	}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues(secret, path, args))
}

func validateTrackingSignature(secret, path string, args url.Values, ttl ...time.Duration) error {
	if secret == "" {
		return fmt.Errorf("tracking signature secret is not configured")
	}
	got := args.Get(trackingSignatureParam)
	if got == "" {
		return fmt.Errorf("tracking signature missing")
	}
	if err := validateTrackingSignatureTimestamp(args, ttl...); err != nil {
		return err
	}
	want := signTrackingValues(secret, path, args)
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return fmt.Errorf("tracking signature malformed")
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return err
	}
	if !hmac.Equal(gotBytes, wantBytes) {
		return fmt.Errorf("tracking signature invalid")
	}
	return nil
}

func equalHexSignature(got, want string) bool {
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return false
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return false
	}
	return hmac.Equal(gotBytes, wantBytes)
}

func validateTrackingSignatureTimestamp(args url.Values, ttl ...time.Duration) error {
	raw := args.Get(trackingSignatureTimestampParam)
	if raw == "" {
		return fmt.Errorf("tracking signature timestamp missing")
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("tracking signature timestamp malformed")
	}
	signedAt := time.Unix(seconds, 0)
	limit := defaultTrackingSignatureTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		limit = ttl[0]
	}
	now := time.Now()
	if signedAt.After(now.Add(maxTrackingSignatureFutureSkew)) {
		return fmt.Errorf("tracking signature timestamp is in the future")
	}
	if now.Sub(signedAt) > limit {
		return fmt.Errorf("tracking signature expired")
	}
	return nil
}

func canonicalTrackingPayload(path string, args url.Values) string {
	canonical := make(url.Values, len(args))
	for key, values := range args {
		if key == trackingSignatureParam {
			continue
		}
		if path == "/win" || path == "/loss" {
			if _, ok := winLossMacroKeys[key]; ok {
				continue
			}
		}
		copied := append([]string(nil), values...)
		sort.Strings(copied)
		canonical[key] = copied
	}
	return path + "?" + canonical.Encode()
}
