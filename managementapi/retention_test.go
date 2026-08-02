package managementapi

import (
	"context"
	"testing"
)

func TestPruneAuditRejectsUnsafeInputsBeforeDatabaseWork(t *testing.T) {
	tests := []struct {
		name      string
		actor     Actor
		retention int
		limit     int
		reason    string
	}{
		{"actor", Actor{Role: "adv", ID: 1}, 400, 1, "scheduled"},
		{"retention", Actor{Role: "admin", ID: 1}, 30, 1, "scheduled"},
		{"limit", Actor{Role: "admin", ID: 1}, 400, 0, "scheduled"},
		{"reason", Actor{Role: "admin", ID: 1}, 400, 1, "line one\nline two"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PruneAudit(context.Background(), nil, test.actor, test.retention, test.limit, test.reason); err == nil {
				t.Fatal("unsafe prune input was accepted")
			}
		})
	}
}
