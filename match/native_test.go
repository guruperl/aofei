package match

import (
	"os"
	"testing"
)

// TestRequestStringToNativeFromat parse the strings of using adm.json to native types
func TestRequestStringToNativeFromat(t *testing.T) {
	bs, err := os.ReadFile("sample_native.json")
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

	bs, err = os.ReadFile("sample_adm.json")
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
		nat.Assets[0].Title.Text != "KION – фильмы, сериалы и тв" {
		t.Errorf("%#v", nat.Assets[0].Title)
	}
	if nat.Assets[1].ID != 2 ||
		nat.Assets[1].Required != 1 ||
		nat.Assets[1].Img.W != 64 ||
		nat.Assets[1].Img.H != 64 {
		t.Errorf("%#v", nat.Assets[1])
	}
	if nat.Link.URL != "http://api-ad.example-ad.com/api/click?a=10068&o=14801&aff_click_id=400DD1F2BDFAA27563A2D51736911100148&sub_affid=89941&gaid=00000000-0000-0000-0000-000000000000&ip=31.43.223.170&ua=Mozilla%2F5.0+%28Linux%3B+Android+13%3B+RMX3834+Build%2FTP1A.220624.014%3B+wv%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Version%2F4.0+Chrome%2F131.0.6778.201+Mobile+Safari%2F537.36" {
		t.Errorf("%#v", nat.Link.URL)
	}
	if nat.Link.Clicktrackers[0] != "https://hiveads.example-dsp-bid.com/receiver/click/400DD1F2BDFAA27563A2D51736911100148?client=40&supply=221&campaign=4198&group=12610&ad=89941&creative=7615" {
		t.Errorf("%#v", nat.Link.URL)
	}
	if nat.Link.Fallback != "https://play.google.com/store/apps/details?id=ru.mts.mtstv" {
		t.Errorf("%#v", nat.Link.Fallback)
	}
	if nat.ImpTrackers[0] != "https://hiveads.example-dsp-bid.com/receiver/impression/400DD1F2BDFAA27563A2D51736911100148?win_price=0.5&client=40&supply=221&campaign=4198&group=12610&ad=89941&creative=7615" {
		t.Errorf("%#v", nat.ImpTrackers[0])
	}

	str, err := nat.AdM("landing url", "failback url", []string{"https://hiveads.example-dsp-bid.com/receiver/impression/400DD1F2BDFAA27563A2D51736911100148?win_price=0.5&client=40&supply=221&campaign=4198&group=12610&ad=89941&creative=7615"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Errorf("%s", nat.Link.URL)
	t.Errorf("%s", str)
}
