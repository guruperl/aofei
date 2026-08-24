package dsp

import (
	"math"
	"strings"
	"testing"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestValidateMiddlemanBannerContract(t *testing.T) {
	w, h := int64(300), int64(250)
	secure := int8(1)
	imp := &openrtb2.Imp{ID: "imp", Banner: &openrtb2.Banner{W: &w, H: &h}, Secure: &secure}
	attr := &match.Attribute{RPub: match.RPub{SizeID: match.SizeID2To1(300, 250)}}
	valid := openrtb2.Bid{
		ID: "bid", ImpID: "imp", Price: 2, W: 300, H: 250, MType: openrtb2.MarkupBanner,
		AdM:  `<script src="https://cdn.example/ad.js"></script><img src="https://cdn.example/pixel" srcset="https://cdn.example/pixel-1x 1x, https://cdn.example/pixel-2x 2x"><a href="https://advertiser.example" ping="https://tracker.example/ping">open</a>`,
		NURL: "https://bidder.example/win?price=${AUCTION_PRICE}",
	}
	if err := validateMiddlemanDownstreamBid(imp, attr, valid); err != nil {
		t.Fatalf("valid banner: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*openrtb2.Bid)
		want string
	}{
		{name: "non-finite price", edit: func(b *openrtb2.Bid) { b.Price = math.NaN() }, want: "finite positive"},
		{name: "wrong dimensions", edit: func(b *openrtb2.Bid) { b.W = 320 }, want: "do not match"},
		{name: "wrong markup type", edit: func(b *openrtb2.Bid) { b.MType = openrtb2.MarkupVideo }, want: "does not match"},
		{name: "unsafe callback", edit: func(b *openrtb2.Bid) { b.NURL = "javascript:alert(1)" }, want: "absolute HTTP(S)"},
		{name: "insecure secure-inventory callback", edit: func(b *openrtb2.Bid) { b.NURL = "http://bidder.example/win" }, want: "HTTPS"},
		{name: "container escape", edit: func(b *openrtb2.Bid) { b.AdM = `<script>top.location='https://evil.example'</script>` }, want: "container-escape"},
		{name: "bracket container escape", edit: func(b *openrtb2.Bid) { b.AdM = `<script>window [ "top" ].location='https://evil.example'</script>` }, want: "container-escape"},
		{name: "entity encoded unsafe URL", edit: func(b *openrtb2.Bid) { b.AdM = `<a href="java&#x73;cript:alert(1)">open</a>` }, want: "forbidden URL scheme"},
		{name: "entity encoded event escape", edit: func(b *openrtb2.Bid) {
			b.AdM = `<img src="https://cdn.example/ad.png" onerror="window&#x2e;top.location='https://evil.example'">`
		}, want: "container-escape"},
		{name: "nested srcdoc", edit: func(b *openrtb2.Bid) { b.AdM = `<iframe srcdoc="<p>nested</p>"></iframe>` }, want: "nested markup"},
		{name: "srcset unsafe URL", edit: func(b *openrtb2.Bid) {
			b.AdM = `<img src="https://cdn.example/ad.png" srcset="https://cdn.example/ad.png 1x, java&#x73;cript:alert(1) 2x">`
		}, want: "forbidden URL scheme"},
		{name: "ping unsafe URL", edit: func(b *openrtb2.Bid) {
			b.AdM = `<a href="https://advertiser.example" ping="https://tracker.example/ping java&#x73;cript:alert(1)">open</a>`
		}, want: "forbidden URL scheme"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			test.edit(&candidate)
			if err := validateMiddlemanDownstreamBid(imp, attr, candidate); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateMiddlemanNativeAndVideoContracts(t *testing.T) {
	format := &match.NativeFormat{Ver: "1.2", Assets: []*match.AssetFormat{{
		AssetFormat: adcom1.AssetFormat{ID: 7, Img: &adcom1.ImageAssetFormat{Type: adcom1.ImageAssetMain, MIME: []string{"image/png"}}},
		Required:    1,
	}}}
	attr := &match.Attribute{RPub: match.RPub{SizeID: match.SizeID2To1(300, 250)}, NativeFormat: format}
	native := &match.Native{
		Ver:    "1.2",
		Assets: []match.Asset{{ID: 7, Required: 1, Img: &adcom1.ImageAsset{URL: "https://cdn.example/main.png"}}},
		Link:   &match.LinkAsset{URL: "https://advertiser.example/landing"},
	}
	adm, err := native.AdM(native.Link.URL, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	nativeImp := &openrtb2.Imp{ID: "native", Native: &openrtb2.Native{}}
	if err := validateMiddlemanDownstreamBid(nativeImp, attr, openrtb2.Bid{
		ID: "bid", ImpID: "native", Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: adm,
	}); err != nil {
		t.Fatalf("valid native: %v", err)
	}
	native.Assets[0].Img.URL = "https://cdn.example/main.js"
	adm, _ = native.AdM(native.Link.URL, "", nil, nil)
	if err := validateMiddlemanDownstreamBid(nativeImp, attr, openrtb2.Bid{
		ID: "bid", ImpID: "native", Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: adm,
	}); err == nil || !strings.Contains(err.Error(), "MIME") {
		t.Fatalf("native MIME error = %v", err)
	}
	native.Assets[0].Img.URL = "https://cdn.example/main.png"
	native.Ver = "1.1"
	adm, _ = native.AdM(native.Link.URL, "", nil, nil)
	if err := validateMiddlemanDownstreamBid(nativeImp, attr, openrtb2.Bid{
		ID: "bid", ImpID: "native", Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: adm,
	}); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("native version error = %v", err)
	}
	native.Ver = "1.2"
	native.Assets[0] = match.Asset{ID: 7, Required: 1, Title: &adcom1.TitleAsset{Text: "wrong shape"}}
	adm, _ = native.AdM(native.Link.URL, "", nil, nil)
	if err := validateMiddlemanDownstreamBid(nativeImp, attr, openrtb2.Bid{
		ID: "bid", ImpID: "native", Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: adm,
	}); err == nil || !strings.Contains(err.Error(), "wrong image shape") {
		t.Fatalf("native asset shape error = %v", err)
	}
	unknown := strings.Replace(adm, `"link":`, `"jstracker":"alert(1)","link":`, 1)
	if err := validateMiddlemanDownstreamBid(nativeImp, attr, openrtb2.Bid{
		ID: "bid", ImpID: "native", Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: unknown,
	}); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unknown active native field error = %v", err)
	}

	vw, vh := int64(640), int64(360)
	videoSecure := int8(1)
	videoImp := &openrtb2.Imp{ID: "video", Video: &openrtb2.Video{W: &vw, H: &vh, Protocols: []adcom1.MediaCreativeSubtype{2}}, Secure: &videoSecure}
	videoAttr := &match.Attribute{RPub: match.RPub{SizeID: match.SizeID2To1(640, 360)}, IsVideo: true}
	videoBid := openrtb2.Bid{ID: "bid", ImpID: "video", Price: 1, W: 640, H: 360, MType: openrtb2.MarkupVideo, Protocol: 2, AdM: `<VAST version="3.0"></VAST>`}
	if err := validateMiddlemanDownstreamBid(videoImp, videoAttr, videoBid); err != nil {
		t.Fatalf("valid video: %v", err)
	}
	videoBid.Protocol = 3
	if err := validateMiddlemanDownstreamBid(videoImp, videoAttr, videoBid); err == nil || !strings.Contains(err.Error(), "protocol") {
		t.Fatalf("video protocol error = %v", err)
	}
	videoBid.Protocol = 2
	videoBid.AdM = `<VAST>`
	if err := validateMiddlemanDownstreamBid(videoImp, videoAttr, videoBid); err == nil || !strings.Contains(err.Error(), "not VAST") {
		t.Fatalf("malformed VAST error = %v", err)
	}
	videoBid.AdM = `<VAST version="3.0"><Ad><InLine><Creatives><Creative><Linear><MediaFiles><MediaFile>http://cdn.example/video.mp4</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	if err := validateMiddlemanDownstreamBid(videoImp, videoAttr, videoBid); err == nil || !strings.Contains(err.Error(), "not VAST") {
		t.Fatalf("insecure VAST resource error = %v", err)
	}
	videoBid.AdM = `<VAST version="3.0"><Ad><InLine><Creatives><Creative><Linear><MediaFiles><MediaFile>video.mp4</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	if err := validateMiddlemanDownstreamBid(videoImp, videoAttr, videoBid); err == nil || !strings.Contains(err.Error(), "not VAST") {
		t.Fatalf("relative VAST resource error = %v", err)
	}
}

func TestValidateMiddlemanNativeVideoVASTActiveContent(t *testing.T) {
	format := &match.NativeFormat{Ver: "1.2", Assets: []*match.AssetFormat{{
		AssetFormat: adcom1.AssetFormat{ID: 9, Video: &adcom1.VideoPlacement{}},
		Required:    1,
	}}}
	attr := &match.Attribute{RPub: match.RPub{SizeID: match.SizeID2To1(300, 250)}, NativeFormat: format}
	native := &match.Native{
		Ver: "1.2",
		Assets: []match.Asset{{ID: 9, Required: 1, Video: &adcom1.VideoAsset{
			AdM: `<VAST version="3.0"><Ad><InLine><Creatives><Creative><Linear><AdParameters><![CDATA[<script>top.location='https://evil.example'</script>]]></AdParameters></Linear></Creative></Creatives></InLine></Ad></VAST>`,
		}}},
		Link: &match.LinkAsset{URL: "https://advertiser.example/landing"},
	}
	adm, err := native.AdM(native.Link.URL, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	imp := &openrtb2.Imp{ID: "native-video", Native: &openrtb2.Native{}}
	err = validateMiddlemanDownstreamBid(imp, attr, openrtb2.Bid{
		ID: "bid", ImpID: imp.ID, Price: 1, W: 300, H: 250, MType: openrtb2.MarkupNative, AdM: adm,
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("native VAST active-content error = %v", err)
	}
}
