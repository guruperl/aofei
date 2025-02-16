package match

import (
	"testing"
	"time"
//	"math"
)

func TestFcap(t *testing.T) {
	n := time.Now();
	loc := n.Location();
	f := NewFcap(n);
	str  := n.String();
	n0 := f.GetStart(loc);
	str0 := n0.String();
	if str[:16] != str0[:16] {
		t.Errorf("%s %s", str, str0);
	}

	num1 := f.Total
	m := n.AddDate(0, 0, 1);
	f = RefreshFcap(f, m);
	num2 := f.Total
	if f.Total != 2 {
		t.Errorf("2 is expected: %d --- %d, %d", f.Total, num1, num2);
	}

	hash := make(map[uint32]Fcap);
	hash[uint32(123)] = f;
	hash[uint32(456)] = f;
	b64, err := PackFcaps(hash);
	if err != nil { t.Errorf("%v", err); }

	out, err := UnpackFcaps(n, b64);
	if err != nil { t.Errorf("%v", err); }
	f1 := out[123];
	f2 := out[456];
	if f1.Total != f2.Total || f1.GetStart(loc) != f2.GetStart(loc) || f1.GetLast(loc) != f2.GetLast(loc) {
		t.Errorf("%v", out);
	}

	t1 := n.Add(-10*time.Minute)
	t2 := n.Add(-20*time.Minute)
	t3 := n.Add(-30*time.Minute)
	t4 := n.Add(-40*time.Minute)
	t5 := n.Add(-50*time.Minute)
	c1 := NewFcap(t1)
	c2 := NewFcap(t2)
	c3 := NewFcap(t3)
	c4 := NewFcap(t4)
	c5 := NewFcap(t5)
	c1.Total = 1
	c2.Total = 2
	c3.Total = 3
	c4.Total = 4
	c5.Total = 5
	hash = make(map[uint32]Fcap)
	hash[uint32(111)] = c1
	hash[uint32(222)] = c2
	hash[uint32(333)] = c3
	hash[uint32(444)] = c4
	hash[uint32(555)] = c5
	b64, err = PackFcaps(hash)
	if err != nil { t.Fatal(err) }
	out, err = UnpackFcaps(n, b64)
	if err != nil { t.Fatal(err) }
	if out[111].Total != uint8(1) ||
	out[222].Total != uint8(2) ||
	out[333].Total != uint8(3) ||
	out[444].Total != uint8(4) ||
	out[555].Total != uint8(5) {
		t.Errorf("%v %v %v %v %v\n%v %v %v %v %v", hash[111], hash[222], hash[333], hash[444], hash[555], out[111], out[222], out[333], out[444], out[555])
	}

	ups := make(map[uint32]Fcap)
	UpdateFcaps(&ups,1,t1)
	if ups[1].Total!=1 { t.Errorf("%v", ups) }
	UpdateFcaps(&ups,2,t2)
	if ups[2].Total!=1 { t.Errorf("%v", ups) }
	UpdateFcaps(&ups,3,t3)
	if ups[3].Total!=1 { t.Errorf("%v", ups) }
	UpdateFcaps(&ups,4,t4)
	if ups[4].Total!=1 { t.Errorf("%v", ups) }
	UpdateFcaps(&ups,5,t5)
	if ups[5].Total!=1 { t.Errorf("%v", ups) }
}
