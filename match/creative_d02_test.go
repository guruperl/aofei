package match

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestCreativeValidateForImpFormatsDimensionsMIMEAndSecureInventory(t *testing.T) {
	w, h := int64(300), int64(250)
	secure := int8(1)
	sizeID := SizeID2To1(uint16(w), uint16(h))
	banner := &Creative{
		CreativeName: "banner", CreativeContent: "https://cdn.example/banner.html",
		SizeID: sizeID, MediaType: CreativeMediaBanner, MIME: "text/html",
		Landing: "https://advertiser.example/landing",
	}
	imp := &openrtb2.Imp{Banner: &openrtb2.Banner{W: &w, H: &h, MIMEs: []string{"text/html"}}, Secure: &secure}
	attr := &Attribute{RPub: RPub{SizeID: sizeID}}
	if err := banner.ValidateForImp(imp, attr); err != nil {
		t.Fatalf("valid secure banner: %v", err)
	}

	bad := *banner
	bad.CreativeContent = "http://cdn.example/banner.html"
	if err := bad.ValidateForImp(imp, attr); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("insecure banner error = %v, want HTTPS rejection", err)
	}
	bad = *banner
	bad.CreativeContent = `<script>top.location='https://evil.example'</script>`
	if err := bad.ValidateForImp(imp, attr); err == nil || !strings.Contains(err.Error(), "URL") {
		t.Fatalf("hostile markup error = %v, want URL-only rejection", err)
	}
	bad = *banner
	bad.SizeID = SizeID2To1(320, 50)
	if err := bad.ValidateForImp(imp, attr); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("size mismatch error = %v", err)
	}
}

func TestCreativeValidateForImpVideoMIME(t *testing.T) {
	w, h := int64(640), int64(360)
	sizeID := SizeID2To1(uint16(w), uint16(h))
	creative := &Creative{
		CreativeName: "video", CreativeContent: "https://cdn.example/video.mp4",
		SizeID: sizeID, MediaType: CreativeMediaVideo, MIME: "video/mp4",
		Landing: "https://advertiser.example/landing",
	}
	imp := &openrtb2.Imp{Video: &openrtb2.Video{W: &w, H: &h, MIMEs: []string{"video/webm"}}}
	attr := &Attribute{RPub: RPub{SizeID: sizeID}, IsVideo: true}
	if err := creative.ValidateForImp(imp, attr); err == nil || !strings.Contains(err.Error(), "not accepted") {
		t.Fatalf("video MIME error = %v", err)
	}
	imp.Video.MIMEs = []string{"video/mp4"}
	if err := creative.ValidateForImp(imp, attr); err != nil {
		t.Fatalf("valid video: %v", err)
	}
	adm, err := creative.AdM(attr, "https://dsp.example/imp", "https://dsp.example/click", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(adm, `<VAST version="3.0">`) || !strings.Contains(adm, `type="video/mp4"`) {
		t.Fatalf("video adm = %q", adm)
	}
}

func TestNativeCreativeV1MaterializesRequestedAssets(t *testing.T) {
	content, err := MarshalNativeCreativeV1(NativeCreativeV1{
		Version: "1", Title: "Native title", Description: "Native description", CTA: "Learn more",
		IconURL: "https://cdn.example/icon.png", MainImageURL: "https://cdn.example/main.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	creative := &Creative{
		CreativeName: "Sponsor", CreativeContent: content, SizeID: SizeID2To1(1200, 627),
		MediaType: CreativeMediaNative, Landing: "https://advertiser.example/landing",
	}
	format := &NativeFormat{Ver: "1.2", Assets: []*AssetFormat{
		{AssetFormat: adcom1.AssetFormat{ID: 10, Title: &adcom1.TitleAssetFormat{Len: 50}}, Required: 1},
		{AssetFormat: adcom1.AssetFormat{ID: 20, Img: &adcom1.ImageAssetFormat{Type: adcom1.ImageAssetMain, W: 1200, H: 627}}, Required: 1},
		{AssetFormat: adcom1.AssetFormat{ID: 30, Data: &adcom1.DataAssetFormat{Type: adcom1.DataAssetDesc, Len: 255}}, Required: 1},
		{AssetFormat: adcom1.AssetFormat{ID: 40, Data: &adcom1.DataAssetFormat{Type: adcom1.DataAssetCTAText, Len: 50}}, Required: 1},
	}}
	attr := &Attribute{RPub: RPub{SizeID: creative.SizeID}, NativeFormat: format}
	imp := &openrtb2.Imp{Native: &openrtb2.Native{}}
	if err := creative.ValidateForImp(imp, attr); err != nil {
		t.Fatal(err)
	}
	adm, err := creative.AdM(attr, "https://dsp.example/imp", "https://dsp.example/click", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	native, err := UnpackAdm([]byte(adm))
	if err != nil {
		t.Fatal(err)
	}
	if len(native.Assets) != 4 || native.Assets[0].ID != 10 || native.Assets[1].Img.URL != "https://cdn.example/main.jpg" || native.Assets[2].Data.Value != "Native description" {
		t.Fatalf("native assets = %#v", native.Assets)
	}
}

func TestNativeCreativeOptionalCopyIsAllowedUntilRequested(t *testing.T) {
	content, err := MarshalNativeCreativeV1(NativeCreativeV1{
		Version: "1", Title: "Title", MainImageURL: "https://cdn.example/main.png",
	})
	if err != nil {
		t.Fatalf("optional native copy: %v", err)
	}
	creative, err := ParseNativeCreativeV1(content)
	if err != nil {
		t.Fatal(err)
	}
	format := &NativeFormat{Ver: "1.2", Assets: []*AssetFormat{{
		AssetFormat: adcom1.AssetFormat{ID: 1, Data: &adcom1.DataAssetFormat{Type: adcom1.DataAssetCTAText}},
		Required:    1,
	}}}
	if _, err := NativeFromCreativeV1(format, creative, "creative", 300, 250); err == nil || !strings.Contains(err.Error(), "required native asset") {
		t.Fatalf("missing requested CTA error = %v", err)
	}
}

func TestNativeCreativeRejectsDuplicateRequestAssetIDs(t *testing.T) {
	creative := &NativeCreativeV1{
		Version: "1", Title: "Title", MainImageURL: "https://cdn.example/main.png",
	}
	format := &NativeFormat{Ver: "1.2", Assets: []*AssetFormat{
		{AssetFormat: adcom1.AssetFormat{ID: 1, Title: &adcom1.TitleAssetFormat{Len: 50}}},
		{AssetFormat: adcom1.AssetFormat{ID: 1, Img: &adcom1.ImageAssetFormat{Type: adcom1.ImageAssetMain}}},
	}}
	if _, err := NativeFromCreativeV1(format, creative, "creative", 300, 250); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate native asset error = %v", err)
	}
}

func TestNativeCreativeDoesNotInventUnrequestedAssets(t *testing.T) {
	creative := &NativeCreativeV1{
		Version: "1", Title: "Title", MainImageURL: "https://cdn.example/main.png",
	}
	if _, err := NativeFromCreativeV1(&NativeFormat{Ver: "1.2"}, creative, "creative", 300, 250); err == nil || !strings.Contains(err.Error(), "no assets") {
		t.Fatalf("empty native request error = %v", err)
	}
}

func TestNativeCreativeHonorsRequestedImageMIME(t *testing.T) {
	creative := &NativeCreativeV1{
		Version: "1", Title: "Title", MainImageURL: "https://cdn.example/main.jpg",
	}
	format := &NativeFormat{Ver: "1.2", Assets: []*AssetFormat{{
		AssetFormat: adcom1.AssetFormat{ID: 1, Img: &adcom1.ImageAssetFormat{Type: adcom1.ImageAssetMain, MIME: []string{"image/png"}}},
		Required:    1,
	}}}
	if _, err := NativeFromCreativeV1(format, creative, "creative", 300, 250); err == nil || !strings.Contains(err.Error(), "MIME") {
		t.Fatalf("native image MIME error = %v", err)
	}
}

func TestNativeCreativeWithoutExactImageExtensionFailsClosed(t *testing.T) {
	creative := &NativeCreativeV1{
		Version: "1", Title: "Title", MainImageURL: "https://cdn.example/image?id=main",
	}
	format := &NativeFormat{Ver: "1.2", Assets: []*AssetFormat{{
		AssetFormat: adcom1.AssetFormat{ID: 1, Img: &adcom1.ImageAssetFormat{Type: adcom1.ImageAssetMain, MIME: []string{"image/jpeg"}}},
		Required:    1,
	}}}
	if _, err := NativeFromCreativeV1(format, creative, "creative", 300, 250); err == nil || !strings.Contains(err.Error(), "MIME") {
		t.Fatalf("extensionless required native image error = %v", err)
	}
}

func TestCreativeLegacyCacheFieldsFailClosed(t *testing.T) {
	creative := &Creative{CreativeName: "legacy", CreativeContent: "https://cdn.example/banner.html", SizeID: SizeID2To1(300, 250), Landing: "https://advertiser.example"}
	if err := creative.ValidateConfiguration(false); err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("legacy creative error = %v", err)
	}
}
