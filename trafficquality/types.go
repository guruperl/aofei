// Package trafficquality implements explainable, versioned invalid-traffic
// rules and the review boundary around their decisions. It deliberately does
// not contain an automatic or learned scoring model.
package trafficquality

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

type Signal string
type Action string
type RuleMode string
type ScopeType string
type EvidenceState string
type ReviewStatus string
type BillingDisposition string

const (
	SignalReplay             Signal = "Replay"
	SignalImpossibleSequence Signal = "ImpossibleSequence"
	SignalInvalidOriginApp   Signal = "InvalidOriginApp"
	SignalMalformedIdentity  Signal = "MalformedIdentity"
	SignalAbnormalRate       Signal = "AbnormalRate"
	SignalAbnormalCTR        Signal = "AbnormalCTR"
	SignalAutomation         Signal = "Automation"
	SignalPartnerPolicy      Signal = "PartnerPolicy"

	ActionObserve    Action = "Observe"
	ActionFlag       Action = "Flag"
	ActionThrottle   Action = "Throttle"
	ActionReject     Action = "Reject"
	ActionQuarantine Action = "Quarantine"

	ModeDraft    RuleMode = "Draft"
	ModeObserve  RuleMode = "Observe"
	ModeCanary   RuleMode = "Canary"
	ModeActive   RuleMode = "Active"
	ModeDisabled RuleMode = "Disabled"

	ScopeGlobal     ScopeType = "Global"
	ScopeAdvertiser ScopeType = "Advertiser"
	ScopePublisher  ScopeType = "Publisher"
	ScopePartner    ScopeType = "Partner"

	EvidenceComplete EvidenceState = "Complete"
	EvidencePartial  EvidenceState = "Partial"
	EvidenceMissing  EvidenceState = "Missing"

	ReviewOpen           ReviewStatus = "Open"
	ReviewValidTraffic   ReviewStatus = "ValidTraffic"
	ReviewInvalidTraffic ReviewStatus = "InvalidTraffic"
	ReviewAppealed       ReviewStatus = "Appealed"
	ReviewAppealUpheld   ReviewStatus = "AppealUpheld"
	ReviewAppealDenied   ReviewStatus = "AppealDenied"

	BillingObserve BillingDisposition = "Observe"
	BillingExclude BillingDisposition = "Exclude"
	BillingHold    BillingDisposition = "Hold"
	BillingReverse BillingDisposition = "Reverse"
)

var (
	ErrConflict     = errors.New("traffic-quality state conflict")
	ErrForbidden    = errors.New("traffic-quality action forbidden")
	ErrNotFound     = errors.New("traffic-quality object not found")
	safeRuleKey     = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)
	safeReasonCode  = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)
	safeActorRole   = regexp.MustCompile(`^(admin|adv|pub|agent|analyst)$`)
	safeActorID     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@._:-]{0,127}$`)
	unixActorID     = regexp.MustCompile(`^unix-uid:[0-9]+$`)
	safeEventKey    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,255}$`)
	safePartnerKey  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	safeAuditReason = regexp.MustCompile(`^[^\x00-\x1f\x7f]{1,500}$`)
)

var allSignals = map[Signal]bool{
	SignalReplay: true, SignalImpossibleSequence: true, SignalInvalidOriginApp: true,
	SignalMalformedIdentity: true, SignalAbnormalRate: true, SignalAbnormalCTR: true,
	SignalAutomation: true, SignalPartnerPolicy: true,
}

var allActions = map[Action]bool{
	ActionObserve: true, ActionFlag: true, ActionThrottle: true,
	ActionReject: true, ActionQuarantine: true,
}

var allModes = map[RuleMode]bool{
	ModeDraft: true, ModeObserve: true, ModeCanary: true,
	ModeActive: true, ModeDisabled: true,
}

var allScopes = map[ScopeType]bool{
	ScopeGlobal: true, ScopeAdvertiser: true, ScopePublisher: true, ScopePartner: true,
}

var allEvidenceStates = map[EvidenceState]bool{
	EvidenceComplete: true, EvidencePartial: true, EvidenceMissing: true,
}

// Scope is the authorization and enforcement boundary for a rule or decision.
// Global scope always has ID zero. Other scopes require a positive numeric ID;
// a partner's opaque name is separately digested and never used as a metric
// label.
type Scope struct {
	Type ScopeType
	ID   uint64
}

func (s Scope) Validate() error {
	if !allScopes[s.Type] {
		return fmt.Errorf("invalid traffic-quality scope %q", s.Type)
	}
	if s.Type == ScopeGlobal && s.ID != 0 {
		return fmt.Errorf("global traffic-quality scope must use id zero")
	}
	if s.Type != ScopeGlobal && s.ID == 0 {
		return fmt.Errorf("scoped traffic-quality object requires a positive id")
	}
	return nil
}

// Rule is one immutable version of an explainable threshold rule. A new
// behavior is represented by a new version rather than rewriting history.
type Rule struct {
	ID                     uint64
	Key                    string
	Version                uint32
	Signal                 Signal
	Action                 Action
	Mode                   RuleMode
	Scope                  Scope
	Threshold              float64
	WindowSeconds          uint32
	CanaryBasisPoints      uint16
	ReasonCode             string
	EvidenceRetentionHrs   uint32
	AggregateRetentionDays uint32
	FalsePositiveLimitBPS  uint16
	CreatedBy              string
	CreatedAt              time.Time
	ActivatedAt            *time.Time
}

func (r Rule) Validate() error {
	if !safeRuleKey.MatchString(r.Key) {
		return fmt.Errorf("rule key must be a lowercase stable name of 3 through 64 characters")
	}
	if r.Version == 0 {
		return fmt.Errorf("rule version must be positive")
	}
	if !allSignals[r.Signal] {
		return fmt.Errorf("invalid traffic-quality signal %q", r.Signal)
	}
	if !allActions[r.Action] {
		return fmt.Errorf("invalid traffic-quality action %q", r.Action)
	}
	if !allModes[r.Mode] {
		return fmt.Errorf("invalid traffic-quality mode %q", r.Mode)
	}
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if math.IsNaN(r.Threshold) || math.IsInf(r.Threshold, 0) || r.Threshold < 0 || r.Threshold > 1_000_000_000 {
		return fmt.Errorf("rule threshold must be finite and between 0 and 1000000000")
	}
	if r.WindowSeconds == 0 || r.WindowSeconds > 30*24*60*60 {
		return fmt.Errorf("rule window must be between 1 second and 30 days")
	}
	if r.CanaryBasisPoints > 10_000 {
		return fmt.Errorf("rule canary basis points must not exceed 10000")
	}
	if r.Mode == ModeCanary && r.CanaryBasisPoints == 0 {
		return fmt.Errorf("canary rules require a non-zero sample")
	}
	if !safeReasonCode.MatchString(r.ReasonCode) {
		return fmt.Errorf("rule reason code must be a lowercase stable name of 3 through 64 characters")
	}
	if r.EvidenceRetentionHrs < 1 || r.EvidenceRetentionHrs > 30*24 {
		return fmt.Errorf("evidence retention must be between 1 and 720 hours")
	}
	if r.AggregateRetentionDays < 365 || r.AggregateRetentionDays > 2555 {
		return fmt.Errorf("aggregate retention must be between 365 and 2555 days")
	}
	if r.FalsePositiveLimitBPS > 10_000 {
		return fmt.Errorf("false-positive limit must not exceed 10000 basis points")
	}
	if !safeActorID.MatchString(r.CreatedBy) {
		return fmt.Errorf("rule creator is invalid")
	}
	return nil
}

// Observation is a single bounded signal. EventKey and PartnerKey exist only
// long enough to derive keyed digests; callers must not log them.
type Observation struct {
	Signal        Signal
	Scope         Scope
	EventKey      string
	PartnerKey    string
	ObservedValue float64
	Evidence      EvidenceState
	ObservedAt    time.Time
	SafeSummary   string
}

func (o Observation) Validate() error {
	if !allSignals[o.Signal] {
		return fmt.Errorf("invalid traffic-quality signal %q", o.Signal)
	}
	if err := o.Scope.Validate(); err != nil {
		return err
	}
	if !safeEventKey.MatchString(o.EventKey) {
		return fmt.Errorf("event key is invalid")
	}
	if o.PartnerKey != "" && !safePartnerKey.MatchString(o.PartnerKey) {
		return fmt.Errorf("partner key is invalid")
	}
	if math.IsNaN(o.ObservedValue) || math.IsInf(o.ObservedValue, 0) || o.ObservedValue < 0 || o.ObservedValue > 1_000_000_000 {
		return fmt.Errorf("observed value must be finite and between 0 and 1000000000")
	}
	if !allEvidenceStates[o.Evidence] {
		return fmt.Errorf("invalid evidence state %q", o.Evidence)
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("observation time is required")
	}
	if o.SafeSummary != strings.TrimSpace(o.SafeSummary) || len(o.SafeSummary) > 255 || strings.ContainsAny(o.SafeSummary, "\r\n") {
		return fmt.Errorf("safe summary must be a single bounded line")
	}
	return nil
}

// Decision is an immutable rule outcome. It contains only keyed digests and
// bounded classifications, never a raw cookie, device id, IP, auction id, or
// bearer token.
type Decision struct {
	ID                 uint64
	RuleID             uint64
	RuleKey            string
	RuleVersion        uint32
	Signal             Signal
	ConfiguredAction   Action
	AppliedAction      Action
	Mode               RuleMode
	Scope              Scope
	EventDigest        [32]byte
	PartnerDigest      [32]byte
	ObservedValue      float64
	Threshold          float64
	Matched            bool
	CanarySelected     bool
	Evidence           EvidenceState
	ReasonCode         string
	BillingDisposition BillingDisposition
	ObservedAt         time.Time
	EvidenceExpiresAt  time.Time
}

// Actor is supplied only from an authenticated S02 session. Permissions and
// scope are copied from verified server-side identity state, not request data.
type Actor struct {
	Role        string
	ID          string
	Scope       Scope
	Permissions map[string]bool
	RecentMFA   bool
}

func (a Actor) Name() string { return a.Role + ":" + a.ID }

func (a Actor) Validate() error {
	if !safeActorRole.MatchString(a.Role) || !safeActorID.MatchString(a.ID) {
		return fmt.Errorf("traffic-quality actor is invalid")
	}
	if err := a.Scope.Validate(); err != nil {
		return err
	}
	return nil
}

func (a Actor) Can(permission string) bool {
	return a.Permissions["*"] || a.Permissions[permission]
}

func (a Actor) isUnixMaintenance(permission string) bool {
	return a.Role == "admin" && unixActorID.MatchString(a.ID) &&
		a.Scope == (Scope{Type: ScopeGlobal}) && !a.RecentMFA &&
		len(a.Permissions) == 1 && a.Permissions[permission]
}

func (a Actor) CanRead(scope Scope) bool {
	if a.Role == "admin" && a.Can("quality.evidence.read") {
		return true
	}
	return a.Can("quality.evidence.read") && a.Scope == scope && scope.Type != ScopeGlobal
}

func validAuditReason(reason string) bool {
	return reason == strings.TrimSpace(reason) && safeAuditReason.MatchString(reason)
}

func blockingAction(action Action) bool {
	return action == ActionThrottle || action == ActionReject || action == ActionQuarantine
}

func billingDisposition(action Action) BillingDisposition {
	switch action {
	case ActionReject:
		return BillingExclude
	case ActionQuarantine:
		return BillingHold
	default:
		return BillingObserve
	}
}
