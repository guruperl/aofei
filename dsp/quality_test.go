package dsp

import (
	"bytes"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/guruperl/aofei/trafficquality"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestControllerQualityEnforcementUsesReviewedFreshSnapshot(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service, err := trafficquality.NewServiceWithKey(db, bytes.Repeat([]byte{0x61}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	scope := trafficquality.Scope{Type: trafficquality.ScopePublisher, ID: 9}
	snapshot, err := trafficquality.NewEnforcementSnapshot(now, []trafficquality.Enforcement{{
		ID: 4, Scope: scope, Action: trafficquality.ActionQuarantine, State: "Active",
		CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}})
	if err != nil {
		t.Fatal(err)
	}
	controller := &Controller{
		C: &Config{TrafficQuality: trafficquality.Config{
			Enabled: true, EnforcementRefreshSeconds: 30, EnforcementMaxAgeSeconds: 120,
		}},
		qualityService: service,
	}
	controller.qualitySnapshot.Store(snapshot)
	action, id := controller.qualityAction(scope, "request:abcdef", now)
	if action != trafficquality.ActionQuarantine || id != 4 {
		t.Fatalf("action=%s id=%d", action, id)
	}
	action, id = controller.qualityAction(scope, "request:abcdef", now.Add(3*time.Minute))
	if action != trafficquality.ActionObserve || id != 0 {
		t.Fatalf("stale action=%s id=%d", action, id)
	}
}

func TestQualityEventKeyNeverContainsRawRequestIdentity(t *testing.T) {
	request := &openrtb2.BidRequest{ID: "request with unsafe/raw identity", Imp: []openrtb2.Imp{{ID: "imp/identity"}}}
	key := qualityEventKey(request, 0)
	if key == "" || key == request.ID || bytes.Contains([]byte(key), []byte("unsafe")) || bytes.Contains([]byte(key), []byte("imp/identity")) {
		t.Fatalf("quality event key leaks request identity: %q", key)
	}
}
