package dsp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
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
