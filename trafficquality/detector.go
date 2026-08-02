package trafficquality

import (
	"context"
	"fmt"
	"time"
)

// Window is a bounded aggregate supplied by a trusted measurement worker. It
// contains counts and an opaque internal key, never a raw IP, cookie, device
// id, user agent, consent string, or auction payload.
type Window struct {
	Scope                   Scope
	WindowKey               string
	PartnerKey              string
	StartedAt               time.Time
	EndedAt                 time.Time
	Requests                uint64
	UniqueEvents            uint64
	Impressions             uint64
	Clicks                  uint64
	Actions                 uint64
	InvalidOriginApp        uint64
	MalformedIdentity       uint64
	AutomationMatches       uint64
	PartnerPolicyViolations uint64
	Evidence                EvidenceState
}

func (w Window) Validate() error {
	if err := w.Scope.Validate(); err != nil || w.Scope.Type == ScopeGlobal {
		return fmt.Errorf("traffic-quality window requires a concrete scope")
	}
	if !safeEventKey.MatchString(w.WindowKey) {
		return fmt.Errorf("traffic-quality window key is invalid")
	}
	if !safeEventKey.MatchString(w.WindowKey + ":" + string(SignalImpossibleSequence)) {
		return fmt.Errorf("traffic-quality window key is too long for derived signal identities")
	}
	if w.PartnerKey != "" && !safePartnerKey.MatchString(w.PartnerKey) {
		return fmt.Errorf("traffic-quality window partner key is invalid")
	}
	if w.StartedAt.IsZero() || !w.EndedAt.After(w.StartedAt) || w.EndedAt.Sub(w.StartedAt) > 30*24*time.Hour {
		return fmt.Errorf("traffic-quality window must be positive and no longer than 30 days")
	}
	if !allEvidenceStates[w.Evidence] {
		return fmt.Errorf("traffic-quality window evidence state is invalid")
	}
	if w.UniqueEvents > w.Requests {
		return fmt.Errorf("unique event count cannot exceed request count")
	}
	for name, value := range map[string]uint64{
		"requests": w.Requests, "unique_events": w.UniqueEvents,
		"impressions": w.Impressions, "clicks": w.Clicks, "actions": w.Actions,
		"invalid_origin_app": w.InvalidOriginApp, "malformed_identity": w.MalformedIdentity,
		"automation_matches":        w.AutomationMatches,
		"partner_policy_violations": w.PartnerPolicyViolations,
	} {
		if value > 1_000_000_000 {
			return fmt.Errorf("traffic-quality window %s exceeds the bounded counter maximum", name)
		}
	}
	return nil
}

// Observations derives the closed S03 signal taxonomy. Infrastructure errors,
// timeouts, malformed partner responses, and management API outcomes are not
// fields and therefore cannot be silently converted into IVT signals.
func (w Window) Observations() ([]Observation, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	duration := w.EndedAt.Sub(w.StartedAt).Seconds()
	replay := w.Requests - w.UniqueEvents
	impossible := uint64(0)
	if w.Clicks > w.Impressions {
		impossible += w.Clicks - w.Impressions
	}
	if w.Actions > w.Clicks {
		impossible += w.Actions - w.Clicks
	}
	ctrBPS := float64(0)
	if w.Impressions != 0 {
		ctrBPS = float64(w.Clicks) * 10_000 / float64(w.Impressions)
	} else if w.Clicks != 0 {
		ctrBPS = 1_000_000_000
	}
	values := []struct {
		signal  Signal
		value   float64
		summary string
	}{
		{SignalReplay, float64(replay), "replayed requests in the bounded measurement window"},
		{SignalImpossibleSequence, float64(impossible), "click or action sequence exceeded its prerequisite count"},
		{SignalInvalidOriginApp, float64(w.InvalidOriginApp), "origin or application identity failed the reviewed policy"},
		{SignalMalformedIdentity, float64(w.MalformedIdentity), "required traffic identity was malformed"},
		{SignalAbnormalRate, float64(w.Requests) / duration, "requests per second in the bounded measurement window"},
		{SignalAbnormalCTR, ctrBPS, "click-through rate in basis points for the bounded measurement window"},
		{SignalAutomation, float64(w.AutomationMatches), "reviewed automation-rule matches in the bounded measurement window"},
		{SignalPartnerPolicy, float64(w.PartnerPolicyViolations), "named partner-policy violations in the bounded measurement window"},
	}
	out := make([]Observation, 0, len(values))
	for _, value := range values {
		out = append(out, Observation{
			Signal: value.signal, Scope: w.Scope,
			EventKey: w.WindowKey + ":" + string(value.signal), PartnerKey: w.PartnerKey,
			ObservedValue: value.value, Evidence: w.Evidence,
			ObservedAt: w.EndedAt.UTC(), SafeSummary: value.summary,
		})
	}
	return out, nil
}

func (s *Service) AssessWindow(ctx context.Context, window Window) ([]Decision, error) {
	observations, err := window.Observations()
	if err != nil {
		return nil, err
	}
	var decisions []Decision
	for _, observation := range observations {
		current, err := s.Assess(ctx, observation)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, current...)
	}
	return decisions, nil
}
