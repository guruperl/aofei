package match

import (
	"testing"
	"time"
)

// TestFcap tests the Fcap struct.
func TestFcap(t *testing.T) {
	now := time.Now()
	fcap := NewFcap(now)
	if fcap.Total != 0 {
		t.Errorf("Total should be 0, but got %d", fcap.Total)
	}
	if fcap.StartYM != 0 {
		t.Errorf("StartYM should be 0, but got %d", fcap.StartYM)
	}
	if fcap.StartDHM != 0 {
		t.Errorf("StartDHM should be 0, but got %d", fcap.StartDHM)
	}
	if fcap.Last != 0 {
		t.Errorf("Last should be 0, but got %d", fcap.Last)
	}
}
