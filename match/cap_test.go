package match

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCapRejectsNumberWithoutPeriod(t *testing.T) {
	for _, cap := range []Cap{
		{CapNumber: 1},
		{ClickNumber: 1},
	} {
		if err := cap.Validate(); err == nil || !strings.Contains(err.Error(), "requires a positive period") {
			t.Fatalf("Validate(%+v) = %v", cap, err)
		}
		if _, err := cap.Pack(); err == nil {
			t.Fatalf("Pack(%+v) succeeded", cap)
		}
		if cap.CanServe(time.Now(), BothCap{}) {
			t.Fatalf("invalid cap %+v served", cap)
		}
		if _, _, err := (RAdvs{{Demand: Demand{ItemID: 9}, Cap: cap}}).FilterByCaps(context.Background(), nil, time.Now(), "user"); err == nil {
			t.Fatalf("invalid cached cap %+v passed runtime filtering", cap)
		}
	}
}

func TestCapAllowsStandaloneThrottleAndCompleteNumberedWindows(t *testing.T) {
	for _, cap := range []Cap{
		{CapThrottle: 10},
		{CapNumber: 2, CapPeriod: 60},
		{ClickNumber: 2, ClickPeriod: 60},
	} {
		if err := cap.Validate(); err != nil {
			t.Fatalf("Validate(%+v) = %v", cap, err)
		}
	}
}
