package dsp

import (
	"errors"
	"expvar"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/uploaded"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestLocalStaticCacheMissingEntries(t *testing.T) {
	top := t.TempDir()
	controller := &Controller{C: &Config{Spread: top, IsLocal: true}}

	pub, err := controller.localPub(top, "missing.example")
	if err != nil {
		t.Fatal(err)
	}
	if pub != nil {
		t.Fatalf("missing pub = %#v, want nil", pub)
	}

	radvs, err := controller.localRAdvs(top, 300250, 99)
	if err != nil {
		t.Fatal(err)
	}
	if radvs != nil {
		t.Fatalf("missing radvs = %#v, want nil", radvs)
	}

	audiences, err := controller.localAudiences(top, match.RAdvs{{Demand: match.Demand{ItemID: 77}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(audiences) != 1 || audiences[0] != nil {
		t.Fatalf("missing audiences = %#v, want one nil wildcard", audiences)
	}

	creative, err := controller.localCreative(top, 7)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing creative err = %v, want os.ErrNotExist", err)
	}
	if creative != nil {
		t.Fatalf("missing creative = %#v, want nil", creative)
	}
}

func TestLocalStaticCacheReloadsGeneration(t *testing.T) {
	top := t.TempDir()
	controller := &Controller{C: &Config{Spread: top, IsLocal: true}}

	writeCreativeSnapshot(t, top, 7, &match.Creative{CreativeName: "old", SizeID: 300250})
	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}
	creative, err := controller.localCreative(top, 7)
	if err != nil {
		t.Fatal(err)
	}
	if creative.CreativeName != "old" {
		t.Fatalf("creative name = %q, want old", creative.CreativeName)
	}

	writeCreativeSnapshot(t, top, 7, &match.Creative{CreativeName: "new", SizeID: 300250})
	creative, err = controller.localCreative(top, 7)
	if err != nil {
		t.Fatal(err)
	}
	if creative.CreativeName != "old" {
		t.Fatalf("creative name = %q, want old until explicit reload", creative.CreativeName)
	}

	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}
	creative, err = controller.localCreative(top, 7)
	if err != nil {
		t.Fatal(err)
	}
	if creative.CreativeName != "new" {
		t.Fatalf("creative name = %q, want new", creative.CreativeName)
	}
}

func TestLocalStaticCacheFreshnessMetricsAreAlertOnly(t *testing.T) {
	controller := &Controller{
		C:     &Config{IsLocal: true, LocalCacheMaxAgeSeconds: 60},
		local: newLocalStaticCache(),
	}
	loadedAt := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	controller.local.mu.Lock()
	controller.local.loadedAt = loadedAt
	controller.local.mu.Unlock()
	controller.publishLocalCacheFreshnessState()

	if got := expvar.Get("aofei_local_cache_loaded_at_unix").String(); got != strconv.FormatInt(loadedAt.Unix(), 10) {
		t.Fatalf("loaded_at metric = %s, want %d", got, loadedAt.Unix())
	}
	if got := localCacheAgeSecondsAt(loadedAt.Add(30 * time.Second)); got != 30 {
		t.Fatalf("fresh age metric = %d, want 30", got)
	}
	if got := localCacheStaleAt(loadedAt.Add(30 * time.Second)); got != 0 {
		t.Fatalf("fresh stale metric = %d, want 0", got)
	}

	if got := localCacheAgeSecondsAt(loadedAt.Add(90 * time.Second)); got != 90 {
		t.Fatalf("stale age metric = %d, want 90", got)
	}
	if got := localCacheStaleAt(loadedAt.Add(90 * time.Second)); got != 1 {
		t.Fatalf("stale metric = %d, want 1", got)
	}
}

func TestLocalStaticCacheFreshnessMetricsAdvanceWithoutCacheRead(t *testing.T) {
	loadedAt := time.Now().Add(-2 * time.Second)
	setLocalCacheFreshnessMetrics(loadedAt, 0)

	first, err := strconv.ParseInt(expvar.Get("aofei_local_cache_age_seconds").String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	second, err := strconv.ParseInt(expvar.Get("aofei_local_cache_age_seconds").String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if second <= first {
		t.Fatalf("age metric did not advance without cache read: first=%d second=%d", first, second)
	}
}

func TestServeBidLocalStaticCacheDoesNotRequireRedisWithoutMutableFeatures(t *testing.T) {
	controller := newLocalBidPathController(t)
	if controller.Redis != nil {
		t.Fatalf("test controller Redis = %#v, want nil", controller.Redis)
	}

	body := marshalBidRequest(t, localBidRequest("USD", "USD"))
	rr := serveSmokeBid(t, controller, "pub.example", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeBid status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestServeBidLocalStaticCacheRequiresRedisForCaps(t *testing.T) {
	controller := newLocalBidPathController(t)
	sizeID := match.SizeID2To1(300, 250)
	writeRAdvsSnapshot(t, controller.C.Spread, sizeID, 100, match.RAdvs{{
		Demand:   match.Demand{AdvID: 1, CampaignID: 10, ItemID: 1000, CreativeID: 10000},
		Weight:   1,
		CostType: 2,
		Cost:     2,
		Cap:      match.Cap{CapNumber: 1, CapPeriod: 60},
	}})
	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}

	bid := localBidRequest("USD", "USD")
	bid.Imp = bid.Imp[:1]
	body := marshalBidRequest(t, bid)
	rr := serveSmokeBid(t, controller, "pub.example", body)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestServeBidLocalStaticCacheRequiresRedisForUploads(t *testing.T) {
	controller := newLocalBidPathController(t)
	writeAudienceSnapshot(t, controller.C.Spread, 1000, &match.Audience{
		UploadAudience: &uploaded.UploadAudience{Uploads: uint32(1 << uploaded.UploadUserID)},
	})
	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}

	bid := localBidRequest("USD", "USD")
	bid.Imp = bid.Imp[:1]
	bid.User = &openrtb2.User{ID: "user-1"}
	body := marshalBidRequest(t, bid)
	rr := serveSmokeBid(t, controller, "pub.example", body)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("ServeBid status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func writeAudienceSnapshot(t *testing.T, top string, itemID uint32, audience *match.Audience) {
	t.Helper()
	path := filepath.Join(top, match.HashNameAudience, strconvUint(itemID))
	mkdirParent(t, path)
	data, err := audience.Pack()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
