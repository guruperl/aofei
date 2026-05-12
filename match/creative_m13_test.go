package match

import (
	"strings"
	"testing"
)

func TestCreativeAdMAppBannerUsesIframeWithImpressionPixels(t *testing.T) {
	creative := &Creative{
		CreativeContent: "https://cdn.example/banner.html",
		SizeID:          SizeID2To1(300, 250),
		ImpTrackers:     []string{"https://tracker.example/imp2"},
		ClickTrackers:   []string{"https://tracker.example/click2"},
	}

	adm, err := creative.AdM(&Attribute{IsApp: true}, "https://tracker.example/imp1", "https://tracker.example/click1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(adm, "<iframe") {
		t.Fatalf("adm = %q, want iframe", adm)
	}
	if strings.Contains(adm, `"native"`) {
		t.Fatalf("adm = %q, did not expect native markup", adm)
	}
	if !strings.Contains(adm, "https://tracker.example/imp1") || !strings.Contains(adm, "https://tracker.example/imp2") {
		t.Fatalf("adm = %q, want impression trackers", adm)
	}
	if strings.Contains(adm, "https://tracker.example/click") {
		t.Fatalf("adm = %q, did not expect click tracker rewrite", adm)
	}
}
