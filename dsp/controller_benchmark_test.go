package dsp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/guruperl/aofei/match"
)

func BenchmarkServeBidLocalTwoImpressions(b *testing.B) {
	controller := newLocalBidPathController(b)
	body := marshalBidRequest(b, localBidRequest("USD", "USD"))
	var badStatus atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/bid/pub.example", bytes.NewReader(body))
			req.SetPathValue("domain", "pub.example")
			rsp := httptest.NewRecorder()
			controller.ServeBid(rsp, req)
			if rsp.Code != http.StatusOK {
				badStatus.CompareAndSwap(0, int64(rsp.Code))
			}
		}
	})
	if got := badStatus.Load(); got != 0 {
		b.Fatalf("ServeBid status = %d, want %d", got, http.StatusOK)
	}
}

func BenchmarkServeSSPLocalTwoAdUnits(b *testing.B) {
	controller := newLocalBidPathController(b)
	body := sspRequestBody(b, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
		{Code: "unit-two", SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 1},
	})
	var badStatus atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
			req.Header.Set("Origin", "https://example.com")
			rsp := httptest.NewRecorder()
			controller.ServeSSP(rsp, req)
			if rsp.Code != http.StatusOK {
				badStatus.CompareAndSwap(0, int64(rsp.Code))
			}
		}
	})
	if got := badStatus.Load(); got != 0 {
		b.Fatalf("ServeSSP status = %d, want %d", got, http.StatusOK)
	}
}
