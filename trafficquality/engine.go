package trafficquality

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// Engine evaluates a closed rule set. Key must be a deployment secret and is
// used only to create non-reversible event and partner digests.
type Engine struct {
	key []byte
}

func NewEngine(key []byte) (*Engine, error) {
	if len(key) < 32 {
		return nil, fmt.Errorf("traffic-quality digest key must contain at least 32 bytes")
	}
	copyKey := append([]byte(nil), key...)
	return &Engine{key: copyKey}, nil
}

// Evaluate returns one decision for a compatible non-draft rule. A partial or
// missing evidence state can be observed but can never throttle, reject, or
// quarantine traffic.
func (e *Engine) Evaluate(rule Rule, observation Observation) (Decision, bool, error) {
	if e == nil || len(e.key) < 32 {
		return Decision{}, false, fmt.Errorf("traffic-quality engine is unavailable")
	}
	if err := rule.Validate(); err != nil {
		return Decision{}, false, err
	}
	if err := observation.Validate(); err != nil {
		return Decision{}, false, err
	}
	if rule.Mode == ModeDraft || rule.Mode == ModeDisabled || rule.Signal != observation.Signal || !scopeMatches(rule.Scope, observation.Scope) {
		return Decision{}, false, nil
	}
	eventDigest := e.digest("event", observation.EventKey)
	partnerDigest := [32]byte{}
	if observation.PartnerKey != "" {
		partnerDigest = e.digest("partner", observation.PartnerKey)
	}
	decision := Decision{
		RuleID: rule.ID, RuleKey: rule.Key, RuleVersion: rule.Version,
		Signal: rule.Signal, ConfiguredAction: rule.Action, AppliedAction: ActionObserve,
		Mode: rule.Mode, Scope: observation.Scope, EventDigest: eventDigest,
		PartnerDigest: partnerDigest, ObservedValue: observation.ObservedValue,
		Threshold: rule.Threshold, Evidence: observation.Evidence,
		ReasonCode: rule.ReasonCode, ObservedAt: observation.ObservedAt.UTC(),
		EvidenceExpiresAt: observation.ObservedAt.UTC().Add(
			timeHours(rule.EvidenceRetentionHrs)),
	}
	decision.Matched = observation.ObservedValue >= rule.Threshold
	if !decision.Matched || observation.Evidence != EvidenceComplete || rule.Mode == ModeObserve {
		decision.BillingDisposition = BillingObserve
		return decision, true, nil
	}
	if rule.Mode == ModeCanary {
		decision.CanarySelected = canarySelected(rule, eventDigest)
		if !decision.CanarySelected {
			decision.BillingDisposition = BillingObserve
			return decision, true, nil
		}
	}
	decision.AppliedAction = rule.Action
	decision.BillingDisposition = billingDisposition(rule.Action)
	return decision, true, nil
}

func (e *Engine) digest(domain, value string) [32]byte {
	h := hmac.New(sha256.New, e.key)
	_, _ = h.Write([]byte("w8m-quality-v1\n" + domain + "\n" + value))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func scopeMatches(rule, observed Scope) bool {
	return rule.Type == ScopeGlobal || rule == observed
}

func canarySelected(rule Rule, digest [32]byte) bool {
	if rule.CanaryBasisPoints >= 10_000 {
		return true
	}
	h := sha256.New()
	_, _ = h.Write([]byte(rule.Key))
	var version [4]byte
	binary.BigEndian.PutUint32(version[:], rule.Version)
	_, _ = h.Write(version[:])
	_, _ = h.Write(digest[:])
	sum := h.Sum(nil)
	bucket := binary.BigEndian.Uint16(sum[:2]) % 10_000
	return bucket < rule.CanaryBasisPoints
}

func timeHours(hours uint32) time.Duration {
	return time.Duration(hours) * time.Hour
}
