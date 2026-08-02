package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	payment "github.com/guruperl/aofei/hostedpayment"
)

type fakeService struct{ actor payment.Actor }

func (f *fakeService) OperationalHealth(context.Context) (payment.OperationalHealth, error) {
	return payment.OperationalHealth{UnresolvedExceptions: 2, StuckSubmitting: 1}, nil
}
func (f *fakeService) PruneEvents(_ context.Context, actor payment.Actor, _ int, _ string) (int64, error) {
	f.actor = actor
	return 7, nil
}

func TestRunHealthReturnsBoundedAggregateOnly(t *testing.T) {
	var output bytes.Buffer
	if err := run(context.Background(), new(fakeService), options{action: "health"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"unresolved_exceptions":2`) || strings.Contains(output.String(), "provider_token") {
		t.Fatalf("health output=%s", output.String())
	}
}

func TestRunPruneUsesExplicitAuditActorAndCannotMoveMoney(t *testing.T) {
	service := new(fakeService)
	var output bytes.Buffer
	if err := run(context.Background(), service, options{action: "prune-events", actorID: "9", reason: "retention schedule", limit: 100}, &output); err != nil {
		t.Fatal(err)
	}
	if service.actor.ID != "9" || !service.actor.RecentMFA || !strings.Contains(output.String(), "=7") {
		t.Fatalf("actor=%#v output=%q", service.actor, output.String())
	}
}
