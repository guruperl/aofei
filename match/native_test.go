package match

import (
	"os"
	"testing"
)

// TestRequestStringToNativeType parse the strings of using adm.json to native types
func TestRequestStringToNativeType(t *testing.T) {
	bs, err := os.ReadFile("native_bid.json")
	if err != nil {
		t.Fatal(err)
	}
	req, err := requestStringToNativeType(bs)
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

	bs, err = os.ReadFile("adm.json")
	if err != nil {
		t.Fatal(err)
	}
	req, err = requestStringToNativeType(bs)
	if err != nil {
		t.Fatal(err)
	}
	if req.Ver != "1.1" ||
		req.Assets[0].ID != 1 ||
		req.Assets[0].Required != 1 ||
		req.Assets[0].Title.Text != "KION – фильмы, сериалы и тв" {
		t.Errorf("%#v", req.Assets[0].Title)
	}
	if req.Assets[1].ID != 2 ||
		req.Assets[1].Required != 1 ||
		req.Assets[1].Img.W != 64 ||
		req.Assets[1].Img.H != 64 {
		t.Errorf("%#v", req.Assets[1])
	}
	if req.Link.URL != "http://api-ad.example-ad.com/api/click?a=10068&o=14801&aff_click_id=400DD1F2BDFAA27563A2D51736911100148&sub_affid=89941&gaid=00000000-0000-0000-0000-000000000000&ip=31.43.223.170&ua=Mozilla%2F5.0+%28Linux%3B+Android+13%3B+RMX3834+Build%2FTP1A.220624.014%3B+wv%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Version%2F4.0+Chrome%2F131.0.6778.201+Mobile+Safari%2F537.36" {
		t.Errorf("%#v", req.Link.URL)
	}
	if req.Link.Clicktrackers[0] != "https://hiveads.example-dsp-bid.com/receiver/click/400DD1F2BDFAA27563A2D51736911100148?client=40&supply=221&campaign=4198&group=12610&ad=89941&creative=7615" {
		t.Errorf("%#v", req.Link.URL)
	}
	if req.Link.Fallback != "https://play.google.com/store/apps/details?id=ru.mts.mtstv" {
		t.Errorf("%#v", req.Link.Fallback)
	}
	if req.ImpTrackers[0] != "https://hiveads.example-dsp-bid.com/receiver/impression/400DD1F2BDFAA27563A2D51736911100148?win_price=0.5&client=40&supply=221&campaign=4198&group=12610&ad=89941&creative=7615" {
		t.Errorf("%#v", req.ImpTrackers[0])
	}
}
