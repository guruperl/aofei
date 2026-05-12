package match

import (
	"context"
	"testing"
	"time"

	"github.com/genelet/winter/acl"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestNewAttributeForImpUsesPerImpressionFormatPriority(t *testing.T) {
	wBanner, hBanner := int64(300), int64(250)
	wVideo, hVideo := int64(640), int64(360)
	bid := &openrtb2.BidRequest{
		ID:     "req",
		Device: &openrtb2.Device{IP: "203.0.113.1", UA: "test"},
		Site:   &openrtb2.Site{Domain: "example.com"},
		Imp: []openrtb2.Imp{{
			ID:     "imp",
			TagID:  "slot",
			Banner: &openrtb2.Banner{W: &wBanner, H: &hBanner},
			Video:  &openrtb2.Video{W: &wVideo, H: &hVideo},
			Native: &openrtb2.Native{Request: `{"native":{"ver":"1.1","assets":[{"id":1,"img":{"wmin":64,"hmin":64}}]}}`},
		}},
	}
	pub := &acl.Pub{
		PubID:            1,
		DefaultWebSiteID: 2,
		DefaultWebSlotID: 3,
	}

	attr, err := NewAttributeForImp(context.Background(), nil, bid, 0, pub, time.Now(), "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if attr.NativeFormat == nil {
		t.Fatal("NativeFormat is nil, want native preferred over video/banner")
	}
	if attr.SizeID != SizeID2To1(64, 64) {
		t.Fatalf("SizeID = %d, want native image size", attr.SizeID)
	}
}

func TestNewAttributeForImpFallsBackWhenNativeHasNoRequest(t *testing.T) {
	wBanner, hBanner := int64(300), int64(250)
	bid := &openrtb2.BidRequest{
		ID:     "req",
		Device: &openrtb2.Device{IP: "203.0.113.1", UA: "test"},
		Site:   &openrtb2.Site{Domain: "example.com"},
		Imp: []openrtb2.Imp{{
			ID:     "imp",
			TagID:  "slot",
			Banner: &openrtb2.Banner{W: &wBanner, H: &hBanner},
			Native: &openrtb2.Native{},
		}},
	}
	pub := &acl.Pub{
		PubID:            1,
		DefaultWebSiteID: 2,
		DefaultWebSlotID: 3,
	}

	attr, err := NewAttributeForImp(context.Background(), nil, bid, 0, pub, time.Now(), "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if attr.NativeFormat != nil {
		t.Fatalf("NativeFormat = %+v, want nil fallback to banner", attr.NativeFormat)
	}
	if attr.SizeID != SizeID2To1(300, 250) {
		t.Fatalf("SizeID = %d, want banner size", attr.SizeID)
	}
}
