package dmp

import (
	"net/url"
	"strconv"
	"testing"
)

func TestAudience(t *testing.T) {
	dmmp := GetDmpSample()
	aud := GetDmpAudienceSample()
	if aud.Sex != 1 || aud.Bplaces[0] != 9 || aud.Brands[0] != 11 || aud.Holds[0] != 1 ||
		aud.Games[0] != 1 || aud.Medias[0] != 8 || aud.Medias[1] != 9 ||
		aud.Healths[0] != 1 || aud.Healths[1] != 2 || aud.Learns[0] != 3 || aud.Learns[1] != 4 ||
		aud.Reports[0] != 4 || aud.Others[0] != 1 {
		t.Errorf("%v", aud)
	}

	bs, err := aud.Pack()
	if err != nil {
		t.Fatal(err)
	}
	//t.Errorf("%d", len(bs))
	unp, err := UnpackAudience(bs)
	if err != nil {
		t.Fatal(err)
	}
	if unp.Sex != 1 || unp.Bplaces[0] != 9 || unp.Brands[0] != 11 || unp.Holds[0] != 1 ||
		unp.Games[0] != 1 || unp.Medias[0] != 8 || unp.Medias[1] != 9 ||
		unp.Healths[0] != 1 || unp.Healths[1] != 2 || unp.Learns[0] != 3 || unp.Learns[1] != 4 ||
		unp.Reports[0] != 4 || unp.Others[0] != 1 {
		t.Errorf("%v\n%v", aud, unp)
	}

	if !aud.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", aud, dmmp)
	}

	unp.Others[0] = 5
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}
	unp.Others[0] = 1
	unp.Reports[0] = 5
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}
	unp.Reports[0] = 5
	unp.Learns[0] = 4
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}

	ARGS := url.Values{}
	campaign_id := uint32(25)
	ARGS.Set("campaign_id", strconv.FormatUint(uint64(campaign_id), 10))
	aud.ToArgs(ARGS)
	if ARGS.Get("Sex") != "1" || ARGS.Get("Bplace") != "9" ||
		ARGS.Get("Brand") != "11" || ARGS.Get("Hold") != "1" ||
		ARGS.Get("Game") != "1" || ARGS.Get("Media") != "8" ||
		ARGS.Get("Health") != "1" || ARGS.Get("Learn") != "3" ||
		ARGS.Get("Report") != "4" || ARGS.Get("Other") != "1" {
		t.Errorf("%v\n%v", aud, ARGS)
	}

	aud0 := DmpAudienceFromArgs(ARGS)
	if aud0.Sex != 1 || aud0.Bplaces[0] != 9 || aud0.Brands[0] != 11 || aud0.Holds[0] != 1 ||
		aud0.Games[0] != 1 || aud0.Medias[0] != 8 || aud0.Medias[1] != 9 ||
		aud0.Healths[0] != 1 || aud0.Healths[1] != 2 || aud0.Learns[0] != 3 || aud0.Learns[1] != 4 ||
		aud0.Reports[0] != 4 || aud0.Others[0] != 1 {
		t.Errorf("%v\n%v", ARGS, aud0)
	}
}
