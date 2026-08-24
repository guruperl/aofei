package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	quality "github.com/guruperl/aofei/trafficquality"
)

type fakeQualityService struct {
	window quality.Window
	actor  quality.Actor
}

func (f *fakeQualityService) AssessWindow(_ context.Context, window quality.Window) ([]quality.Decision, error) {
	f.window = window
	return []quality.Decision{{
		ID: 1, RuleKey: "replay.window", RuleVersion: 2, Signal: quality.SignalReplay,
		AppliedAction: quality.ActionFlag, Scope: window.Scope,
		Evidence: window.Evidence, ReasonCode: "replay.threshold",
		BillingDisposition: quality.BillingObserve,
	}}, nil
}

func (f *fakeQualityService) PruneEvidence(_ context.Context, actor quality.Actor, _ int, _ string) (int64, error) {
	f.actor = actor
	return 3, nil
}

func (f *fakeQualityService) RuleHealth(_ context.Context, actor quality.Actor, _ time.Time) ([]quality.RuleHealth, error) {
	f.actor = actor
	return []quality.RuleHealth{{RuleID: 1, RuleKey: "replay.window"}}, nil
}

func TestAssessWindowUsesBoundedStrictJSONAndDoesNotPrintDigests(t *testing.T) {
	service := new(fakeQualityService)
	input := `{"Scope":{"Type":"Publisher","ID":7},"WindowKey":"window-1","StartedAt":"2026-08-01T00:00:00Z","EndedAt":"2026-08-01T00:01:00Z","Requests":2,"UniqueEvents":1,"Evidence":"Complete"}`
	var output bytes.Buffer
	if err := run(context.Background(), service, options{action: "assess-window"}, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if service.window.WindowKey != "window-1" || strings.Contains(output.String(), "Digest") || strings.Contains(output.String(), "window-1") {
		t.Fatalf("window/output=%#v %s", service.window, output.String())
	}
	if _, err := decodeWindow(strings.NewReader(strings.TrimSuffix(input, "}") + `,"unknown":1}`)); err == nil {
		t.Fatal("unknown aggregate field accepted")
	}
}

func TestMaintenanceActionsUseBoundedAdministratorAttribution(t *testing.T) {
	service := new(fakeQualityService)
	var output bytes.Buffer
	if err := run(context.Background(), service, options{action: "prune-evidence", limit: 10, reason: "scheduled retention"}, nil, &output); err != nil {
		t.Fatal(err)
	}
	if service.actor.Role != "admin" || service.actor.ID != fmt.Sprintf("unix-uid:%d", os.Geteuid()) || service.actor.RecentMFA || !service.actor.Can(quality.PermissionRetentionPrune) {
		t.Fatalf("maintenance actor=%#v", service.actor)
	}
	if output.String() != "pruned traffic_quality_evidence_rows=3\n" {
		t.Fatalf("output=%q", output.String())
	}
	if _, err := maintenanceActor(-1, quality.PermissionEvidenceRead); err == nil {
		t.Fatal("invalid effective Unix UID accepted")
	}
}
