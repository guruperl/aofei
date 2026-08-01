package match

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRequestStringToNativeFromat parse the strings of using adm.json to native types
func TestRequestStringToNativeFromat(t *testing.T) {
	bs, err := os.ReadFile(filepath.Join("..", "etc", "samples", "sample_native.json"))
	if err != nil {
		t.Fatal(err)
	}
	req, err := requestStringToNativeFormat(bs)
	if err != nil {
		t.Fatal(err)
	}
	if req.Ver != "1.1" ||
		req.Assets[0].ID != 1 ||
		req.Assets[0].Required != 1 ||
		req.Assets[0].Title.Len != 100 {
		t.Errorf("%#v", req.Assets[0].Title)
	}
	if req.Assets[1].ID != 2 ||
		req.Assets[1].Required != 1 ||
		req.Assets[1].Img.WMin != 64 ||
		req.Assets[1].Img.HMin != 64 {
		t.Errorf("%#v", req.Assets[1])
	}

	bs, err = os.ReadFile(filepath.Join("..", "etc", "samples", "sample_adm.json"))
	if err != nil {
		t.Fatal(err)
	}
	nat, err := UnpackAdm(bs)
	if err != nil {
		t.Fatal(err)
	}
	if nat.Ver != "1.1" ||
		nat.Assets[0].ID != 1 ||
		nat.Assets[0].Required != 1 ||
		nat.Assets[0].Title.Text != "Example native advertisement" {
		t.Errorf("%#v", nat.Assets[0].Title)
	}
	if nat.Assets[1].ID != 2 ||
		nat.Assets[1].Required != 1 ||
		nat.Assets[1].Img.W != 64 ||
		nat.Assets[1].Img.H != 64 {
		t.Errorf("%#v", nat.Assets[1])
	}
	if nat.Link.URL != "https://advertiser.example.test/landing?click=example" {
		t.Errorf("%#v", nat.Link.URL)
	}
	if nat.Link.Clicktrackers[0] != "https://tracking.example.test/click/example-auction-001" {
		t.Errorf("%#v", nat.Link.URL)
	}
	if nat.Link.Fallback != "https://advertiser.example.test/fallback" {
		t.Errorf("%#v", nat.Link.Fallback)
	}
	if nat.ImpTrackers[0] != "https://tracking.example.test/impression/example-auction-001" {
		t.Errorf("%#v", nat.ImpTrackers[0])
	}

	impTrackers := []string{"https://tracking.example.test/impression/round-trip"}
	str, err := nat.AdM("landing url", "failback url", impTrackers, nil)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := UnpackAdm([]byte(str))
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Link.URL != "landing url" ||
		roundTrip.Link.Fallback != "failback url" ||
		roundTrip.ImpTrackers[0] != impTrackers[0] {
		t.Fatalf("round-trip adm mismatch: %#v", roundTrip)
	}
	if len(roundTrip.Link.Clicktrackers) != 0 {
		t.Fatalf("round-trip click trackers = %#v; want none", roundTrip.Link.Clicktrackers)
	}
}
