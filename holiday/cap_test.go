package holiday

import (
	"testing"
	"time"
//	"math"
)

func TestCap(t *testing.T) {
	cap := &Cap{CapNumber:4, CapPeriod:11, CapThrottle:0, ClickNumber:2, ClickPeriod:20}

	n := time.Now();
	t1 := n.Add(-10*time.Minute)
	f := CreateFcap(t1);
// served 1 time
	if cap.CanServeImp(t1, f) == false {
		t.Errorf("%d vs %d", cap.CapNumber, f.Total)
	}

	t1 = t1.Add(1*time.Minute)
	f.Refresh(t1);
// served 2 times
	if cap.CanServeImp(t1, f) == false {
		t.Errorf("%d vs %d", cap.CapNumber, f.Total)
	}

	t1 = t1.Add(1*time.Minute)
	f.Refresh(t1);
// served 3 times
	if cap.CanServeImp(t1, f) == false {
		t.Errorf("%d vs %d", cap.CapNumber, f.Total)
	}

	t1 = t1.Add(1*time.Minute)
	f.Refresh(t1);
// served 4 times
	if cap.CanServeImp(t1, f) {
		t.Errorf("equal not served, but %d vs %d", cap.CapNumber, f.Total)
	}

	t1 = t1.Add(1*time.Minute)
	f.Refresh(t1);
// served 5 times
	if cap.CanServeImp(t1, f) {
		t.Errorf("exceeding not served, but %d vs %d", cap.CapNumber, f.Total)
	}
	if cap.CanServeImp(n, f) {
		t.Errorf("exceeding not served, but %d vs %d", cap.CapNumber, f.Total)
	}

	cap.CapPeriod = 5
	if cap.CanServeImp(n, f) == false  {
		t.Errorf("%d should be less than %d", f.SinceStart(n), cap.CapPeriod)
	}

	cap.CapThrottle = 12 // since last has to be 12 mintues or longer
	if cap.CanServeImp(n, f) {
		t.Errorf("%d should be less than %d", f.SinceLast(n), cap.CapThrottle)
	}
	cap.CapThrottle = 7 // since last has to be 7 mintues or longer
	if cap.CanServeImp(n, f) {
		t.Errorf("%d should be less than %d", f.SinceLast(n), cap.CapThrottle)
	}
	cap.CapThrottle = 6 // since last has to be 6 mintues or longer
	if cap.CanServeImp(n, f)==false {
		t.Errorf("%d should be less than %d", f.SinceLast(n), cap.CapThrottle)
	}
}
