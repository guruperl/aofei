package dsp

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"expvar"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func TestTrafficGateRateLimitsConfiguredPartnersIndependently(t *testing.T) {
	config := &Config{
		TrafficDefault: TrafficPolicy{QPS: 1, Burst: 1, MaxConcurrency: 2, TimeoutMS: 500, MaxBodyBytes: 1024},
		TrafficPartners: map[string]TrafficPolicy{
			"adx:a.example": {},
			"adx:b.example": {},
		},
	}
	gate := NewTrafficGate(config)
	fixed := time.Now()
	gate.now = func() time.Time { return fixed }
	handler := gate.Handler("adx", func(r *http.Request) string {
		return "adx:" + r.Header.Get("X-Test-Partner")
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := func(partner string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/bid/"+partner, nil)
		req.Header.Set("X-Test-Partner", partner)
		response := httptest.NewRecorder()
		handler(response, req)
		return response
	}
	if got := request("a.example").Code; got != http.StatusNoContent {
		t.Fatalf("first partner A status = %d, want 204", got)
	}
	if got := request("a.example").Code; got != http.StatusTooManyRequests {
		t.Fatalf("second partner A status = %d, want 429", got)
	}
	if got := request("b.example").Code; got != http.StatusNoContent {
		t.Fatalf("partner B status = %d, want independent 204", got)
	}
	if strings.Contains(expvar.Get("aofei_traffic_requests_total").String(), "a.example") {
		t.Fatal("traffic metrics exposed a partner identifier")
	}
}

func TestTrafficGateRejectsConcurrentOverloadWithoutBlocking(t *testing.T) {
	config := &Config{TrafficDefault: TrafficPolicy{QPS: 1000, Burst: 10, MaxConcurrency: 1, TimeoutMS: 1000, MaxBodyBytes: 1024}}
	gate := NewTrafficGate(config)
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := gate.Handler("ssp", func(*http.Request) string { return "ssp" }, func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})

	var first *httptest.ResponseRecorder
	done := make(chan struct{})
	go func() {
		defer close(done)
		first = httptest.NewRecorder()
		handler(first, httptest.NewRequest(http.MethodPost, "/pz", nil))
	}()
	<-entered
	second := httptest.NewRecorder()
	handler(second, httptest.NewRequest(http.MethodPost, "/pz", nil))
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("overload status = %d, want 503", second.Code)
	}
	close(release)
	<-done
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want 204", first.Code)
	}
}

func TestTrafficGateTimeoutKeepsConcurrencyOwnedUntilHandlerStops(t *testing.T) {
	config := &Config{TrafficDefault: TrafficPolicy{QPS: 1000, Burst: 10, MaxConcurrency: 1, TimeoutMS: 20, MaxBodyBytes: 1024}}
	gate := NewTrafficGate(config)
	stopped := make(chan struct{})
	var once sync.Once
	handler := gate.Handler("adx", func(*http.Request) string { return "adx:fixture" }, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			<-r.Context().Done()
			once.Do(func() { close(stopped) })
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	slow := httptest.NewRecorder()
	handler(slow, httptest.NewRequest(http.MethodPost, "/slow", nil))
	if slow.Code != http.StatusServiceUnavailable {
		t.Fatalf("timeout status = %d, want 503", slow.Code)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("timed-out handler did not observe cancellation")
	}

	deadline := time.Now().Add(time.Second)
	for {
		fast := httptest.NewRecorder()
		handler(fast, httptest.NewRequest(http.MethodPost, "/fast", nil))
		if fast.Code == http.StatusNoContent {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("concurrency was not released after handler stopped; last status %d", fast.Code)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestTrafficGateEnforcesDeclaredAndStreamingBodyLimits(t *testing.T) {
	config := &Config{TrafficDefault: TrafficPolicy{QPS: 1000, Burst: 10, MaxConcurrency: 2, TimeoutMS: 500, MaxBodyBytes: 8}}
	gate := NewTrafficGate(config)
	handler := gate.Handler("ssp", func(*http.Request) string { return "ssp" }, func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	declaredReq := httptest.NewRequest(http.MethodPost, "/pz", strings.NewReader("012345678"))
	declared := httptest.NewRecorder()
	handler(declared, declaredReq)
	if declared.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("declared body status = %d, want 413", declared.Code)
	}

	streamReq := httptest.NewRequest(http.MethodPost, "/pz", strings.NewReader("012345678"))
	streamReq.ContentLength = -1
	streamed := httptest.NewRecorder()
	handler(streamed, streamReq)
	if streamed.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("streamed body status = %d, want 413", streamed.Code)
	}
}

func TestTrafficGateBoundsGzipCompressedAndDecompressedBodies(t *testing.T) {
	policy := TrafficPolicy{
		QPS: 1000, Burst: 10, MaxConcurrency: 2, TimeoutMS: 500,
		MaxBodyBytes: 128, MaxDecompressedBodyBytes: 8,
	}
	gate := NewTrafficGate(&Config{TrafficDefault: policy})
	handler := gate.Handler("adx", nil, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	request := func(content []byte, encoding string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(content))
		req.Header.Set("Content-Encoding", encoding)
		response := httptest.NewRecorder()
		handler(response, req)
		return response
	}

	if got := request(gzipFixture(t, []byte("12345678")), "gzip"); got.Code != http.StatusOK || got.Body.String() != "12345678" {
		t.Fatalf("bounded gzip response = (%d, %q), want (200, body)", got.Code, got.Body.String())
	}
	if got := request(gzipFixture(t, []byte("123456789")), "gzip"); got.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expanded gzip status = %d, want 413", got.Code)
	}
	if got := request([]byte("not-gzip"), "gzip"); got.Code != http.StatusBadRequest {
		t.Fatalf("malformed gzip status = %d, want 400", got.Code)
	}
	if got := request([]byte("{}"), "br"); got.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported encoding status = %d, want 415", got.Code)
	}
	if got := request([]byte("{}"), "gzip, br"); got.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("stacked encoding status = %d, want 415", got.Code)
	}
}

func TestTrafficGateNegotiatesJSONResponseGzip(t *testing.T) {
	gate := NewTrafficGate(&Config{TrafficDefault: TrafficPolicy{
		QPS: 1000, Burst: 10, MaxConcurrency: 2, TimeoutMS: 500,
		MaxBodyBytes: 1024, MaxDecompressedBodyBytes: 1024,
	}})
	handler := gate.Handler("adx", nil, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/no-bid" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/openrtb+json; charset=utf-8")
		_, _ = w.Write([]byte(`{"id":"fixture"}`))
	})

	request := func(path, accept string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Accept-Encoding", accept)
		response := httptest.NewRecorder()
		handler(response, req)
		return response
	}

	compressed := request("/bid", "br, gzip; q=0.8")
	if compressed.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("content encoding = %q, want gzip", compressed.Header().Get("Content-Encoding"))
	}
	reader, err := gzip.NewReader(compressed.Body)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"id":"fixture"}` {
		t.Fatalf("decompressed response = %q", raw)
	}
	if !strings.Contains(compressed.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("Vary = %q, want Accept-Encoding", compressed.Header().Get("Vary"))
	}

	for _, accept := range []string{"", "gzip;q=0", "gzip;q=bogus"} {
		plain := request("/bid", accept)
		if got := plain.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("Accept-Encoding %q produced %q", accept, got)
		}
	}
	noBid := request("/no-bid", "gzip")
	if noBid.Code != http.StatusNoContent || noBid.Header().Get("Content-Encoding") != "" {
		t.Fatalf("no-bid response = (%d, %q), want uncompressed 204", noBid.Code, noBid.Header().Get("Content-Encoding"))
	}
}

func gzipFixture(t testing.TB, raw []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestAuctionHandlersMapTrafficBodyLimitErrorsTo413(t *testing.T) {
	for name, serve := range map[string]http.HandlerFunc{
		"adx": (&Controller{}).ServeBid,
		"ssp": (&Controller{}).ServeSSP,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("012345678"))
			request.ContentLength = -1
			response := httptest.NewRecorder()
			request.Body = http.MaxBytesReader(response, request.Body, 8)
			serve(response, request)
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413", response.Code)
			}
		})
	}
}

func TestProtectedMetricsHandlerUsesDirectPeerAllowlist(t *testing.T) {
	handler := ProtectedMetricsHandler(&Config{MetricsAllowedCIDRs: []string{"10.0.0.0/8"}}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	allowedReq := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
	allowedReq.RemoteAddr = "10.2.3.4:1234"
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, allowedReq)
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("allowed status = %d, want 204", allowed.Code)
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
	blockedReq.RemoteAddr = "203.0.113.8:1234"
	blockedReq.Header.Set("X-Forwarded-For", "10.2.3.4")
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, blockedReq)
	if blocked.Code != http.StatusNotFound {
		t.Fatalf("blocked status = %d, want 404", blocked.Code)
	}
}

func TestControllerMetricsHandlerProbesDependenciesOnlyForAuthorizedScrapes(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectPing()
	controller := &Controller{C: &Config{}, DB: db}
	handler := controller.MetricsHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	blockedReq := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
	blockedReq.RemoteAddr = "203.0.113.8:1234"
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, blockedReq)
	if blocked.Code != http.StatusNotFound {
		t.Fatalf("blocked status = %d, want 404", blocked.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
	allowedReq.RemoteAddr = "127.0.0.1:1234"
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, allowedReq)
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("allowed status = %d, want 204", allowed.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyMetricsProbeRedis(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	controller := &Controller{Redis: client}
	controller.refreshDependencyMetrics(context.Background())
	if got := dependencyUpRedis.Value(); got != 1 {
		t.Fatalf("redis dependency up = %d, want 1", got)
	}
	server.Close()
	controller.refreshDependencyMetrics(context.Background())
	if got := dependencyUpRedis.Value(); got != 0 {
		t.Fatalf("closed redis dependency up = %d, want 0", got)
	}
}

func TestDependencyMetricsRecordMySQLFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectPing().WillReturnError(errors.New("database unavailable"))
	controller := &Controller{DB: db}
	controller.refreshDependencyMetrics(context.Background())
	if got := dependencyUpMySQL.Value(); got != 0 {
		t.Fatalf("mysql dependency up = %d, want 0", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLatencyHistogramPublishesBoundedPercentiles(t *testing.T) {
	histogram := newLatencyHistogram()
	for _, duration := range []time.Duration{time.Millisecond, 2 * time.Millisecond, 20 * time.Millisecond, 200 * time.Millisecond} {
		histogram.Observe(duration)
	}
	raw := histogram.String()
	for _, want := range []string{`"count":4`, `"p50_ms":2`, `"p95_ms":200`, `"p99_ms":200`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("histogram %s does not contain %s", raw, want)
		}
	}
}

func TestO01MetricContractUsesFixedShapes(t *testing.T) {
	for _, name := range []string{
		"aofei_traffic_requests_total", "aofei_traffic_responses_total",
		"aofei_traffic_rejections_total", "aofei_traffic_in_flight",
		"aofei_bid_path_latency_ms", "aofei_dependency_up",
		"aofei_dependency_check_last_ms", "aofei_dependency_check_errors_total",
		"aofei_db_pool", "aofei_middleman_bidder_outcomes_total",
		"aofei_middleman_bid_rejections_total", "aofei_middleman_candidates_total",
		"aofei_middleman_bidder_latency_ms",
	} {
		if expvar.Get(name) == nil {
			t.Fatalf("required metric %s is not published", name)
		}
	}
	wantShapes := map[string]bool{
		"adx": true, "ssp": true, "local": true, "middleman": true,
		"cap": true, "audience": true, "compressed": true, "fill": true,
		"no_fill": true, "rejection": true, "overload": true,
	}
	seen := map[string]bool{}
	metricBidPathLatency.Do(func(keyValue expvar.KeyValue) { seen[keyValue.Key] = true })
	if len(seen) != len(wantShapes) {
		t.Fatalf("latency shapes = %v, want %v", seen, wantShapes)
	}
	for shape := range seen {
		if !wantShapes[shape] {
			t.Fatalf("unexpected latency shape %q", shape)
		}
	}
}

func TestI01MiddlemanMetricsRejectDynamicLabels(t *testing.T) {
	before := expvarMapInt64(metricMiddlemanBidRejections, "other")
	recordMiddlemanBidRejection("partner-controlled-value")
	if got := expvarMapInt64(metricMiddlemanBidRejections, "other") - before; got != 1 {
		t.Fatalf("other rejection delta = %d, want 1", got)
	}
	if metricMiddlemanBidRejections.Get("partner-controlled-value") != nil {
		t.Fatal("middleman rejection metric accepted a dynamic label")
	}
	beforeCandidate := expvarMapInt64(metricMiddlemanCandidates, "eligible")
	recordMiddlemanCandidate("partner-controlled-stage", 3)
	if got := expvarMapInt64(metricMiddlemanCandidates, "eligible") - beforeCandidate; got != 0 {
		t.Fatalf("eligible candidates changed by dynamic stage: %d", got)
	}
	if metricMiddlemanCandidates.Get("partner-controlled-stage") != nil {
		t.Fatal("middleman candidate metric accepted a dynamic stage")
	}
}

func TestMiddlemanBidderOutcomeMetricRejectsDynamicLabels(t *testing.T) {
	before := expvarMapInt64(metricMiddlemanBidderOutcomes, "configuration_error")
	recordMiddlemanBidderOutcome("partner-controlled-value")
	if got := expvarMapInt64(metricMiddlemanBidderOutcomes, "configuration_error") - before; got != 1 {
		t.Fatalf("configuration-error outcome delta = %d, want 1", got)
	}
	if metricMiddlemanBidderOutcomes.Get("partner-controlled-value") != nil {
		t.Fatal("middleman outcome metric accepted a dynamic label")
	}
}

func BenchmarkTrafficGateAccepted(b *testing.B) {
	gate := NewTrafficGate(&Config{TrafficDefault: TrafficPolicy{
		QPS: 1_000_000, Burst: 1_000_000, MaxConcurrency: 100_000,
		TimeoutMS: 1000, MaxBodyBytes: 1024,
	}})
	handler := gate.Handler("adx", func(*http.Request) string { return "adx:fixture" }, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	var badStatus atomic.Int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest(http.MethodPost, "/bid/fixture", nil))
			if response.Code != http.StatusNoContent {
				badStatus.CompareAndSwap(0, int64(response.Code))
			}
		}
	})
	if got := badStatus.Load(); got != 0 {
		b.Fatalf("status = %d, want %d", got, http.StatusNoContent)
	}
}

func BenchmarkTrafficGateGzipRequest(b *testing.B) {
	gate := NewTrafficGate(&Config{TrafficDefault: TrafficPolicy{
		QPS: 1_000_000, Burst: 1_000_000, MaxConcurrency: 100_000,
		TimeoutMS: 1000, MaxBodyBytes: 1024, MaxDecompressedBodyBytes: 4096,
	}})
	body := gzipFixture(b, bytes.Repeat([]byte(`{"imp":[{"id":"1"}]}`), 100))
	handler := gate.Handler("adx", nil, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusNoContent)
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := httptest.NewRequest(http.MethodPost, "/bid/fixture", bytes.NewReader(body))
		request.Header.Set("Content-Encoding", "gzip")
		handler(httptest.NewRecorder(), request)
	}
}
