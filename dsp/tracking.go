package dsp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
)

const trackingSignatureParam = "sig"

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
	args.Set(trackingSignatureParam, signTrackingValues(secret, path, args))
}

func validateTrackingSignature(secret, path string, args url.Values) error {
	if secret == "" {
		return fmt.Errorf("tracking signature secret is not configured")
	}
	got := args.Get(trackingSignatureParam)
	if got == "" {
		return fmt.Errorf("tracking signature missing")
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
