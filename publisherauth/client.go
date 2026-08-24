package publisherauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	credentialPrefix = "w8m_pz_v1_"
	requestDomain    = "w8m-pz-request-v1"
)

// NewRequestNonce returns a canonical 128-bit nonce for one request.
func NewRequestNonce() (string, error) {
	raw := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// SignRequest creates the four P03 headers from a one-time-issued private
// credential. Callers sign the exact decompressed JSON bytes sent to /pz.
func SignRequest(privateCredential string, timestamp time.Time, nonce string, body []byte) (http.Header, error) {
	publicID, seed, err := parsePrivateCredential(privateCredential)
	if err != nil {
		return nil, err
	}
	if err := validateNonce(nonce); err != nil {
		return nil, err
	}
	credential := credentialPrefix + publicID
	timestampText := strconv.FormatInt(timestamp.UTC().Unix(), 10)
	message := canonicalRequest(credential, timestampText, nonce, body)
	privateKey := ed25519.NewKeyFromSeed(seed)
	signature := ed25519.Sign(privateKey, message)
	headers := make(http.Header, 4)
	headers.Set(HeaderCredential, credential)
	headers.Set(HeaderTimestamp, timestampText)
	headers.Set(HeaderNonce, nonce)
	headers.Set(HeaderSignature, base64.RawURLEncoding.EncodeToString(signature))
	return headers, nil
}

func parsePrivateCredential(raw string) (string, []byte, error) {
	if !strings.HasPrefix(raw, credentialPrefix) {
		return "", nil, fmt.Errorf("private publisher credential is malformed")
	}
	remainder := strings.TrimPrefix(raw, credentialPrefix)
	parts := strings.SplitN(remainder, "_", 2)
	if len(parts) != 2 || !publicIDPattern.MatchString(parts[0]) {
		return "", nil, fmt.Errorf("private publisher credential is malformed")
	}
	seed, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(seed) != ed25519.SeedSize || base64.RawURLEncoding.EncodeToString(seed) != parts[1] {
		return "", nil, fmt.Errorf("private publisher credential is malformed")
	}
	return parts[0], seed, nil
}

func parseCredentialReference(raw string) (string, error) {
	if !strings.HasPrefix(raw, credentialPrefix) {
		return "", ErrInvalid
	}
	publicID := strings.TrimPrefix(raw, credentialPrefix)
	if !publicIDPattern.MatchString(publicID) {
		return "", ErrInvalid
	}
	return publicID, nil
}

func validateNonce(nonce string) error {
	raw, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil || len(raw) < 16 || len(raw) > 32 || base64.RawURLEncoding.EncodeToString(raw) != nonce {
		return fmt.Errorf("publisher request nonce must be canonical base64url for 16 through 32 bytes")
	}
	return nil
}

func canonicalRequest(credential, timestamp, nonce string, body []byte) []byte {
	digest := sha256.Sum256(body)
	return []byte(strings.Join([]string{
		requestDomain,
		http.MethodPost,
		"/pz",
		credential,
		timestamp,
		nonce,
		base64.RawURLEncoding.EncodeToString(digest[:]),
	}, "\n"))
}
