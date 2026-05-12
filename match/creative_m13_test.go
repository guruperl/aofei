package match

import (
	"net/url"
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

func TestCreativeAdMBannerReplacesClickMacros(t *testing.T) {
	creative := &Creative{
		CreativeContent: "https://cdn.example/banner.html?click={CLICK_URL}&landing={LANDING_URL}",
		Landing:         "https://advertiser.example/landing?campaign=1",
		SizeID:          SizeID2To1(300, 250),
	}

	adm, err := creative.AdM(&Attribute{}, "https://dsp.example/imp", "https://dsp.example/clk?redirect=https%3A%2F%2Fadvertiser.example%2Flanding%3Fcampaign%3D1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(adm, "{CLICK_URL}") || strings.Contains(adm, "{LANDING_URL}") {
		t.Fatalf("adm = %q, expected banner macros to be replaced", adm)
	}
	if !strings.Contains(adm, "https://dsp.example/clk?redirect=https%3A%2F%2Fadvertiser.example%2Flanding%3Fcampaign%3D1") {
		t.Fatalf("adm = %q, want click redirect URL", adm)
	}
	if !strings.Contains(adm, "https://advertiser.example/landing?campaign=1") {
		t.Fatalf("adm = %q, want direct landing URL macro", adm)
	}
}

func TestCreativeAdMNativeUsesClickRedirectURLAndConfiguredClickTrackers(t *testing.T) {
	creative := &Creative{
		CreativeContent: "https://cdn.example/native.png",
		CreativeName:    "native",
		SizeID:          SizeID2To1(300, 250),
		Landing:         "https://advertiser.example/landing",
		Failback:        "https://advertiser.example/fallback",
		ClickTrackers:   []string{"https://tracker.example/click"},
	}

	adm, err := creative.AdM(&Attribute{NativeFormat: &NativeFormat{}}, "https://dsp.example/imp", "https://dsp.example/clk?redirect=https%3A%2F%2Fadvertiser.example%2Flanding", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	native, err := UnpackAdm([]byte(adm))
	if err != nil {
		t.Fatal(err)
	}
	if native.Link.URL != "https://dsp.example/clk?redirect=https%3A%2F%2Fadvertiser.example%2Flanding" {
		t.Fatalf("native click URL = %q, want DSP redirect", native.Link.URL)
	}
	if native.Link.Fallback != "https://advertiser.example/fallback" {
		t.Fatalf("native fallback = %q, want direct fallback", native.Link.Fallback)
	}
	if len(native.Link.Clicktrackers) != 1 || native.Link.Clicktrackers[0] != "https://tracker.example/click" {
		t.Fatalf("native click trackers = %#v, want configured trackers only", native.Link.Clicktrackers)
	}
}

func TestApplyMacroPreservesRepeatedQueryValues(t *testing.T) {
	got, err := applyMacro(
		"https://tracker.example/pixel?keep=1&event=${AUCTION_ID}&event=static&name=pre-{DSP_CREATIVE_ID}&empty=",
		map[string]string{`${AUCTION_ID}`: "auction-1"},
		map[string]string{`{DSP_CREATIVE_ID}`: "77"},
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if got := query["event"]; len(got) != 2 || got[0] != "auction-1" || got[1] != "static" {
		t.Fatalf("event values = %#v, want replaced macro and preserved static value", got)
	}
	if got := query.Get("keep"); got != "1" {
		t.Fatalf("keep = %q, want preserved non-macro value", got)
	}
	if got := query.Get("name"); got != "pre-77" {
		t.Fatalf("name = %q, want embedded custom macro replaced", got)
	}
	if _, ok := query["empty"]; !ok {
		t.Fatal("empty query value was not preserved")
	}
}
