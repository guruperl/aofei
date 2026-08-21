package dsp

import (
	"crypto/hmac"
	"crypto/rand"
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
	defaultTrackingProcessingTTL    = 30 * time.Second
	trackingClaimOperationTimeout   = 2 * time.Second
	defaultCapStateTTL              = 90 * 24 * time.Hour
	maxTrackingSignatureFutureSkew  = 5 * time.Minute
)

type trackingClaimOutcome uint8

const (
	trackingClaimOwner trackingClaimOutcome = iota
	trackingClaimDuplicate
	trackingClaimCompleted
	trackingClaimUnkeyed
	trackingClaimRedisFailOpen
)

type trackingEventClaim struct {
	key     string
	token   string
	outcome trackingClaimOutcome
}

func (c trackingEventClaim) owned() bool {
	return c.outcome == trackingClaimOwner && c.key != "" && c.token != ""
}

func (c trackingEventClaim) records() bool {
	return c.outcome != trackingClaimDuplicate && c.outcome != trackingClaimCompleted
}

func (c trackingEventClaim) completed() bool {
	return c.outcome == trackingClaimCompleted
}

func (c trackingEventClaim) keyed() bool {
	return c.key != ""
}

func (c trackingEventClaim) capMarkerKey() string {
	return c.key + ":cap"
}

func newTrackingEventClaimToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

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

// validateTrackingSignature validates args and returns the exact instant at
// which the signed request stops being valid. A timestamp accepted within the
// future-skew window therefore remains valid for the full configured TTL from
// that signed timestamp.
func validateTrackingSignature(secret, path string, args url.Values, ttl time.Duration) (time.Time, error) {
	if secret == "" {
		return time.Time{}, fmt.Errorf("tracking signature secret is not configured")
	}
	got := args.Get(trackingSignatureParam)
	if got == "" {
		return time.Time{}, fmt.Errorf("tracking signature missing")
	}
	validUntil, err := validateTrackingSignatureTimestamp(args, ttl)
	if err != nil {
		return time.Time{}, err
	}
	want := signTrackingValues(secret, path, args)
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return time.Time{}, fmt.Errorf("tracking signature malformed")
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return time.Time{}, err
	}
	if !hmac.Equal(gotBytes, wantBytes) {
		return time.Time{}, fmt.Errorf("tracking signature invalid")
	}
	return validUntil, nil
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

func validateTrackingSignatureTimestamp(args url.Values, ttl time.Duration) (time.Time, error) {
	raw := args.Get(trackingSignatureTimestampParam)
	if raw == "" {
		return time.Time{}, fmt.Errorf("tracking signature timestamp missing")
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("tracking signature timestamp malformed")
	}
	signedAt := time.Unix(seconds, 0)
	if ttl <= 0 {
		return time.Time{}, fmt.Errorf("tracking signature TTL must be positive")
	}
	now := time.Now()
	if signedAt.After(now.Add(maxTrackingSignatureFutureSkew)) {
		return time.Time{}, fmt.Errorf("tracking signature timestamp is in the future")
	}
	validUntil := signedAt.Add(ttl)
	if now.After(validUntil) {
		return time.Time{}, fmt.Errorf("tracking signature expired")
	}
	return validUntil, nil
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
