package dmp

import (
	"testing"
)

func TestDmp(t *testing.T) {
	dmp := GetDmpSample()
	if dmp.Sex != 1 || dmp.Bplace != 9 || dmp.Brand != 11 || dmp.Hold != 1 ||
		dmp.Games[0] != 1 || dmp.Medias[0] != 8 || dmp.Medias[1] != 9 ||
		dmp.Healths[0] != 1 || dmp.Healths[1] != 2 || dmp.Learns[0] != 3 ||
		dmp.Learns[1] != 4 || dmp.Reports[0] != 4 || dmp.Others[0] != 1 {
		t.Errorf("%v", dmp)
	}

	bs, err := dmp.Pack()
	if err != nil {
		t.Fatal(err)
	}
	//t.Errorf("%d", len(bs))
	unp, err := UnpackDmp(bs)
	if err != nil {
		t.Fatal(err)
	}
	if unp.Sex != 1 || unp.Bplace != 9 || unp.Brand != 11 || unp.Hold != 1 ||
		unp.Games[0] != 1 || unp.Medias[0] != 8 || unp.Medias[1] != 9 ||
		unp.Healths[0] != 1 || unp.Healths[1] != 2 || unp.Learns[0] != 3 ||
		unp.Learns[1] != 4 || unp.Reports[0] != 4 || unp.Others[0] != 1 {
		t.Errorf("%v\n%v", dmp, unp)
	}

	data, err := dmp.PackSimple()
	if err != nil {
		t.Fatal(err)
	}
	//t.Errorf("%d", len(data))
	unp, err = UnpackDmpSimple(data)
	if err != nil {
		t.Fatal(err)
	}
	if unp.Sex != 1 || unp.Bplace != 9 || unp.Brand != 11 || unp.Hold != 1 ||
		unp.Games[0] != 1 || unp.Medias[0] != 8 || unp.Medias[1] != 9 ||
		unp.Healths[0] != 1 || unp.Healths[1] != 2 || unp.Learns[0] != 3 ||
		unp.Learns[1] != 4 || unp.Reports[0] != 4 || unp.Others[0] != 1 {
		t.Errorf("%v\n%v", dmp, unp)
	}

}
