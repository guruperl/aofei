package trafficquality

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"time"
)

// Snapshot is a bounded serving-time copy of reviewed enforcement rows. It is
// safe for concurrent readers and contains no raw traffic identity.
type Snapshot struct {
	LoadedAt time.Time
	entries  map[Scope][]Enforcement
}

// NewEnforcementSnapshot validates and copies reviewed enforcement rows. It is
// useful to cache loaders and deterministic tests; callers cannot mutate the
// returned snapshot through the input slice.
func NewEnforcementSnapshot(loadedAt time.Time, entries []Enforcement) (*Snapshot, error) {
	if loadedAt.IsZero() {
		return nil, fmt.Errorf("traffic-quality snapshot load time is required")
	}
	snapshot := &Snapshot{LoadedAt: loadedAt.UTC(), entries: make(map[Scope][]Enforcement)}
	for _, item := range entries {
		validRollout := item.State == "Canary" && item.CanaryBasisPoints > 0 && item.CanaryBasisPoints < 10_000 ||
			item.State == "Active" && (item.CanaryBasisPoints == 0 || item.CanaryBasisPoints == 10_000)
		if err := item.Scope.Validate(); err != nil || item.Scope.Type == ScopeGlobal || !blockingAction(item.Action) || !validRollout || item.ExpiresAt.IsZero() {
			return nil, fmt.Errorf("invalid traffic-quality enforcement row %d", item.ID)
		}
		snapshot.entries[item.Scope] = append(snapshot.entries[item.Scope], item)
	}
	return snapshot, nil
}

func (s *Service) LoadEnforcementSnapshot(ctx context.Context) (*Snapshot, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("traffic-quality database is nil")
	}
	now := s.currentTime()
	rows, err := s.DB.QueryContext(ctx, `
SELECT enforcement_id, rule_id, decision_id, scope_type, scope_id, enforcement_action,
 state, canary_basis_points, created_by, created_at, expires_at
FROM quality_enforcement
WHERE state IN ('Canary','Active') AND expires_at>?
ORDER BY enforcement_id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []Enforcement
	for rows.Next() {
		var item Enforcement
		if err := rows.Scan(&item.ID, &item.RuleID, &item.DecisionID,
			&item.Scope.Type, &item.Scope.ID, &item.Action, &item.State,
			&item.CanaryBasisPoints, &item.CreatedBy, &item.CreatedAt,
			&item.ExpiresAt); err != nil {
			return nil, err
		}
		if err := item.Scope.Validate(); err != nil || item.Scope.Type == ScopeGlobal || !blockingAction(item.Action) {
			return nil, fmt.Errorf("invalid traffic-quality enforcement row %d", item.ID)
		}
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return NewEnforcementSnapshot(now, entries)
}

// EnforcementAction returns the strongest reviewed action selected for this
// event. An expired snapshot or row never enforces. Callers should retain the
// prior unexpired snapshot when a refresh dependency fails.
func (s *Service) EnforcementAction(snapshot *Snapshot, scope Scope, eventKey string, now time.Time, maxAge time.Duration) (Action, uint64, error) {
	if s == nil || s.engine == nil || snapshot == nil {
		return ActionObserve, 0, nil
	}
	if err := scope.Validate(); err != nil || scope.Type == ScopeGlobal || !safeEventKey.MatchString(eventKey) || maxAge <= 0 {
		return ActionObserve, 0, fmt.Errorf("invalid traffic-quality enforcement input")
	}
	now = now.UTC()
	if snapshot.LoadedAt.IsZero() || now.Sub(snapshot.LoadedAt) > maxAge || snapshot.LoadedAt.After(now.Add(time.Minute)) {
		return ActionObserve, 0, nil
	}
	digest := s.engine.digest("event", eventKey)
	type candidate struct {
		action Action
		id     uint64
	}
	var selected []candidate
	for _, enforcement := range snapshot.entries[scope] {
		if !now.Before(enforcement.ExpiresAt) {
			continue
		}
		if enforcement.State == "Canary" && !enforcementCanarySelected(enforcement, digest) {
			continue
		}
		selected = append(selected, candidate{action: enforcement.Action, id: enforcement.ID})
	}
	if len(selected) == 0 {
		return ActionObserve, 0, nil
	}
	sort.Slice(selected, func(i, j int) bool {
		left, right := enforcementPriority(selected[i].action), enforcementPriority(selected[j].action)
		if left != right {
			return left > right
		}
		return selected[i].id < selected[j].id
	})
	return selected[0].action, selected[0].id, nil
}

func enforcementCanarySelected(enforcement Enforcement, digest [32]byte) bool {
	if enforcement.CanaryBasisPoints == 0 {
		return false
	}
	if enforcement.CanaryBasisPoints >= 10_000 {
		return true
	}
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], enforcement.ID)
	h := sha256.New()
	_, _ = h.Write([]byte("w8m-quality-enforcement-v1\n"))
	_, _ = h.Write(id[:])
	_, _ = h.Write(digest[:])
	bucket := binary.BigEndian.Uint16(h.Sum(nil)[:2]) % 10_000
	return bucket < enforcement.CanaryBasisPoints
}

func enforcementPriority(action Action) int {
	switch action {
	case ActionQuarantine:
		return 3
	case ActionReject:
		return 2
	case ActionThrottle:
		return 1
	default:
		return 0
	}
}
