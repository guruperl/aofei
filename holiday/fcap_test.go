package holiday

import (
	"testing"
	"time"
	// "math"
)

// Make a hash map for DB
func (self *Fcap) hasInt() map[string]interface{} {
	return map[string]interface{}{"total": int8(self.Total), "ym": int8(self.StartYM), "dhm": int32(self.StartDHM), "ls": int32(self.Last)}
}

func TestFcap(t *testing.T) {
	n := time.Now()
	f := CreateFcap(n)
	str := n.String()
	n0 := f.GetStart()
	str0 := n0.String()
	if str[:16] != str0[:16] {
		t.Errorf("%s %s", str, str0)
	}

	num1 := f.Total
	m := n.AddDate(0, 0, 1)
	f.Refresh(m)
	num2 := f.Total
	if f.Total != 2 {
		t.Errorf("2 is expected: %d --- %d, %d", f.Total, num1, num2)
	}

	t1 := n.Add(-10 * time.Minute)
	cli := CreateFcap(t1)
	if cli.SinceLast(time.Now()) != 10 {
		t.Errorf("%v", f.GetStart().String())
		t.Errorf("%v", cli.GetStart().String())
	}
}

func TestSFcap(t *testing.T) {
	caps := map[uint32]*Cap{
		uint32(111): &Cap{CapNumber: 5, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(222): &Cap{CapNumber: 15, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(444): &Cap{CapNumber: 25, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20}}

	n := time.Now()
	t1 := n.Add(-10 * time.Minute)
	t2 := n.Add(-20 * time.Minute)
	t3 := n.Add(-30 * time.Minute)
	fcap1 := CreateFcap(t1)
	fcap1.Total = 10
	fcap2 := CreateFcap(t2)
	fcap2.Total = 20
	fcap3 := CreateFcap(t3)
	fcap3.Total = 30
	h1 := fcap1.hasInt()
	h1["campaign_id"] = 111
	h1["act"] = int8(1)
	h2 := fcap2.hasInt()
	h2["campaign_id"] = 222
	h2["act"] = int8(1)
	h3 := fcap3.hasInt()
	h3["campaign_id"] = 333
	h3["act"] = int8(1)
	lists := []map[string]interface{}{h3, h2, h1}

	sfcaps, _ := NewSFcaps(n, caps, lists)
	if len(caps) != 1 {
		// only 444 left
		t.Errorf("%d", len(caps))
		for k, item := range caps {
			t.Errorf("%d, %#v", k, item)
		}
	}
	// sfcaps is empty
	if len(sfcaps) != 0 {
		t.Errorf("%d", len(sfcaps))
	}
}

func TestSFcapMore(t *testing.T) {
	caps := map[uint32]*Cap{
		uint32(111): &Cap{CapNumber: 5, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(222): &Cap{CapNumber: 15, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(444): &Cap{CapNumber: 25, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20}}

	n := time.Now()
	t1 := n.Add(-10 * time.Minute)
	t2 := n.Add(-20 * time.Minute)
	t3 := n.Add(-30 * time.Minute)
	fcap1 := CreateFcap(t1)
	fcap1.Total = 10
	fcap2 := CreateFcap(t2)
	fcap2.Total = 10
	fcap3 := CreateFcap(t3)
	fcap3.Total = 30
	h1 := fcap1.hasInt()
	h1["campaign_id"] = 111
	h1["act"] = int8(1)
	h2 := fcap2.hasInt()
	h2["campaign_id"] = 222
	h2["act"] = int8(1)
	h3 := fcap3.hasInt()
	h3["campaign_id"] = 333
	h3["act"] = int8(1)
	lists := []map[string]interface{}{h3, h2, h1}

	sfcaps, denies := NewSFcaps(n, caps, lists)

	if len(caps) != 2 {
		// only 444 and 222
		t.Errorf("%d", len(caps))
		for k, item := range caps {
			t.Errorf("%d, %#v", k, item)
		}
	}
	// sfcaps has 222
	if len(sfcaps) != 1 {
		t.Errorf("%d", len(sfcaps))
		for k, item := range sfcaps {
			t.Errorf("%d, %#v", k, item)
		}
	}

	if _, ok := denies[111]; !ok {
		t.Errorf("%v", sfcaps)
		t.Errorf("%v", denies)
	}
}

func TestSFcapMore2(t *testing.T) {
	caps := map[uint32]*Cap{
		uint32(111): &Cap{CapNumber: 5, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(222): &Cap{CapNumber: 15, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20},
		uint32(444): &Cap{CapNumber: 25, CapPeriod: 40, CapThrottle: 0, ClickNumber: 2, ClickPeriod: 20}}

	n := time.Now()
	t1 := n.Add(-10 * time.Minute)
	t2 := n.Add(-20 * time.Minute)
	t3 := n.Add(-30 * time.Minute)
	fcap1 := CreateFcap(t1)
	fcap1.Total = 10
	fcap2 := CreateFcap(t2)
	fcap2.Total = 10
	fcap3 := CreateFcap(t3)
	fcap3.Total = 30
	h1 := fcap1.hasInt()
	h1["campaign_id"] = 111
	h1["act"] = int8(1)
	h2 := fcap2.hasInt()
	h2["campaign_id"] = 222
	h2["act"] = int8(1)
	h3 := fcap3.hasInt()
	h3["campaign_id"] = 333
	h3["act"] = int8(1)
	lists := []map[string]interface{}{h3, h2, h1}

	new_sfcaps := SFcapsFromTao(lists)
	new_denies := GetCapped(n, new_sfcaps, caps)
	if len(new_sfcaps) != 3 {
		t.Errorf("%v", new_sfcaps)
	}
	if _, ok := new_denies[111]; !ok {
		t.Errorf("%v", new_denies)
	}
}
