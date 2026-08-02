package dsp

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxConfiguredTrafficPartners = 256

var defaultMetricsAllowedCIDRs = []string{"127.0.0.1/32", "::1/128"}

// TrafficPolicy is one bounded admission policy. Partner overrides inherit
// zero-valued fields from traffic_default.
type TrafficPolicy struct {
	QPS                      float64 `json:"qps,omitempty"`
	Burst                    int     `json:"burst,omitempty"`
	MaxConcurrency           int     `json:"max_concurrency,omitempty"`
	TimeoutMS                int     `json:"timeout_ms,omitempty"`
	MaxBodyBytes             int64   `json:"max_body_bytes,omitempty"`
	MaxDecompressedBodyBytes int64   `json:"max_decompressed_body_bytes,omitempty"`
}

func defaultTrafficPolicy() TrafficPolicy {
	return TrafficPolicy{
		QPS:                      2000,
		Burst:                    500,
		MaxConcurrency:           256,
		TimeoutMS:                1000,
		MaxBodyBytes:             maxBidRequestBodyBytes,
		MaxDecompressedBodyBytes: maxBidRequestBodyBytes,
	}
}

func (p TrafficPolicy) withDefaults(fallback TrafficPolicy) TrafficPolicy {
	if p.QPS == 0 {
		p.QPS = fallback.QPS
	}
	if p.Burst == 0 {
		p.Burst = fallback.Burst
	}
	if p.MaxConcurrency == 0 {
		p.MaxConcurrency = fallback.MaxConcurrency
	}
	if p.TimeoutMS == 0 {
		p.TimeoutMS = fallback.TimeoutMS
	}
	if p.MaxBodyBytes == 0 {
		p.MaxBodyBytes = fallback.MaxBodyBytes
	}
	if p.MaxDecompressedBodyBytes == 0 {
		p.MaxDecompressedBodyBytes = fallback.MaxDecompressedBodyBytes
	}
	return p
}

func (p TrafficPolicy) validate(label string) error {
	if math.IsNaN(p.QPS) || math.IsInf(p.QPS, 0) || p.QPS <= 0 || p.QPS > 1_000_000 {
		return fmt.Errorf("%s qps must be greater than 0 and at most 1000000", label)
	}
	if p.Burst <= 0 || p.Burst > 1_000_000 {
		return fmt.Errorf("%s burst must be between 1 and 1000000", label)
	}
	if p.MaxConcurrency <= 0 || p.MaxConcurrency > 100_000 {
		return fmt.Errorf("%s max_concurrency must be between 1 and 100000", label)
	}
	if p.TimeoutMS <= 0 || p.TimeoutMS > 60_000 {
		return fmt.Errorf("%s timeout_ms must be between 1 and 60000", label)
	}
	if p.MaxBodyBytes <= 0 || p.MaxBodyBytes > maxBidRequestBodyBytes {
		return fmt.Errorf("%s max_body_bytes must be between 1 and %d", label, maxBidRequestBodyBytes)
	}
	if p.MaxDecompressedBodyBytes <= 0 || p.MaxDecompressedBodyBytes > maxBidRequestBodyBytes {
		return fmt.Errorf("%s max_decompressed_body_bytes must be between 1 and %d", label, maxBidRequestBodyBytes)
	}
	return nil
}

func (c *Config) validateTrafficPolicies() error {
	defaults := defaultTrafficPolicy()
	if c != nil {
		defaults = c.TrafficDefault.withDefaults(defaults)
	}
	if err := defaults.validate("traffic_default"); err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	if len(c.TrafficPartners) > maxConfiguredTrafficPartners {
		return fmt.Errorf("traffic_partners may contain at most %d entries", maxConfiguredTrafficPartners)
	}
	for key, policy := range c.TrafficPartners {
		if key != strings.TrimSpace(key) || len(key) > 128 || (key != "ssp" && (!strings.HasPrefix(key, "adx:") || len(key) == len("adx:"))) {
			return fmt.Errorf("traffic_partners key %q must be ssp or adx:<partner> and at most 128 bytes", key)
		}
		if err := policy.withDefaults(defaults).validate("traffic_partners[" + key + "]"); err != nil {
			return err
		}
	}
	return nil
}

type trafficLimiter struct {
	policy TrafficPolicy
	mu     sync.Mutex
	tokens float64
	last   time.Time
	active chan struct{}
}

func newTrafficLimiter(policy TrafficPolicy, now time.Time) *trafficLimiter {
	return &trafficLimiter{
		policy: policy,
		tokens: float64(policy.Burst),
		last:   now,
		active: make(chan struct{}, policy.MaxConcurrency),
	}
}

func (l *trafficLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Before(l.last) {
		l.last = now
	}
	elapsed := now.Sub(l.last).Seconds()
	l.tokens = math.Min(float64(l.policy.Burst), l.tokens+elapsed*l.policy.QPS)
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func (l *trafficLimiter) acquire() bool {
	select {
	case l.active <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *trafficLimiter) release() {
	<-l.active
}

// TrafficGate bounds only public auction traffic. Internal, account, tracker,
// and metrics routes are intentionally outside this gate.
type TrafficGate struct {
	defaultLimiter *trafficLimiter
	partners       map[string]*trafficLimiter
	now            func() time.Time
}

func NewTrafficGate(c *Config) *TrafficGate {
	now := time.Now()
	defaults := defaultTrafficPolicy()
	if c != nil {
		defaults = c.TrafficDefault.withDefaults(defaults)
	}
	gate := &TrafficGate{
		defaultLimiter: newTrafficLimiter(defaults, now),
		partners:       make(map[string]*trafficLimiter),
		now:            time.Now,
	}
	if c != nil {
		for key, policy := range c.TrafficPartners {
			gate.partners[key] = newTrafficLimiter(policy.withDefaults(defaults), now)
		}
	}
	return gate
}

func (g *TrafficGate) limiter(partner string) *trafficLimiter {
	if g != nil {
		if limiter := g.partners[partner]; limiter != nil {
			return limiter
		}
		if g.defaultLimiter != nil {
			return g.defaultLimiter
		}
	}
	return newTrafficLimiter(defaultTrafficPolicy(), time.Now())
}

// Handler applies body, QPS, concurrency, and request-time limits. surface is
// a fixed low-cardinality label (adx or ssp); partner keys are never exported.
func (g *TrafficGate) Handler(surface string, partner func(*http.Request) string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := g.now()
		recordTrafficRequest(surface)
		key := ""
		if partner != nil {
			key = partner(r)
		}
		limiter := g.limiter(key)
		if r.ContentLength > limiter.policy.MaxBodyBytes {
			recordTrafficRejection(surface, "body")
			recordBidPathLatency("rejection", time.Since(started))
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if !limiter.allow(started) {
			recordTrafficRejection(surface, "qps")
			recordBidPathLatency("overload", time.Since(started))
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if !limiter.acquire() {
			recordTrafficRejection(surface, "concurrency")
			recordBidPathLatency("overload", time.Since(started))
			w.Header().Set("Retry-After", "1")
			http.Error(w, "server overloaded", http.StatusServiceUnavailable)
			return
		}
		recordTrafficInFlight(surface, 1)

		released := false
		release := func() {
			if released {
				return
			}
			released = true
			limiter.release()
			recordTrafficInFlight(surface, -1)
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := prepareTrafficRequestBody(w, r, limiter.policy); err != nil {
			release()
			reason := "encoding"
			status := http.StatusBadRequest
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				reason = "body"
				status = http.StatusRequestEntityTooLarge
			} else if errors.Is(err, errUnsupportedContentEncoding) {
				status = http.StatusUnsupportedMediaType
			}
			recordTrafficRejection(surface, reason)
			recordBidPathLatency("rejection", time.Since(started))
			http.Error(w, http.StatusText(status), status)
			return
		}
		timeout := time.Duration(limiter.policy.TimeoutMS) * time.Millisecond
		done := make(chan struct{})
		wrapped := http.HandlerFunc(func(innerW http.ResponseWriter, innerR *http.Request) {
			defer close(done)
			next(innerW, innerR)
		})
		compressed := newNegotiatedResponseWriter(w, r.Header.Get("Accept-Encoding"))
		defer compressed.Close()
		status := &trafficStatusWriter{ResponseWriter: compressed}
		http.TimeoutHandler(wrapped, timeout, "request timeout\n").ServeHTTP(status, r)
		select {
		case <-done:
			release()
			recordTrafficResponse(surface, status.statusCode(), time.Since(started))
			return
		default:
			recordTrafficRejection(surface, "timeout")
			recordBidPathLatency("rejection", time.Since(started))
			go func() {
				<-done
				release()
			}()
		}
	}
}

var errUnsupportedContentEncoding = fmt.Errorf("unsupported content encoding")

// prepareTrafficRequestBody keeps the compressed admission bound independent
// from the JSON parser's decompressed bound. Only a single gzip coding is
// accepted; stacked or unknown codings are rejected before auction work.
func prepareTrafficRequestBody(w http.ResponseWriter, r *http.Request, policy TrafficPolicy) error {
	raw := http.MaxBytesReader(w, r.Body, policy.MaxBodyBytes)
	encoding := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
	switch encoding {
	case "", "identity":
		r.Body = http.MaxBytesReader(w, raw, policy.MaxDecompressedBodyBytes)
		return nil
	case "gzip":
		started := time.Now()
		reader, err := gzip.NewReader(raw)
		if err != nil {
			raw.Close()
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				return err
			}
			return fmt.Errorf("invalid gzip request: %w", err)
		}
		timed := &timedCompressedReader{Reader: reader, elapsed: time.Since(started)}
		decoded := http.MaxBytesReader(w, timed, policy.MaxDecompressedBodyBytes)
		r.Body = &trafficRequestBody{Reader: decoded, decoded: decoded, raw: raw}
		r.ContentLength = -1
		r.Header.Del("Content-Encoding")
		return nil
	default:
		raw.Close()
		return errUnsupportedContentEncoding
	}
}

type trafficRequestBody struct {
	io.Reader
	decoded io.Closer
	raw     io.Closer
	once    sync.Once
}

func (b *trafficRequestBody) Close() error {
	var first error
	b.once.Do(func() {
		if b.decoded != nil {
			first = b.decoded.Close()
		}
		if b.raw != nil {
			if err := b.raw.Close(); first == nil {
				first = err
			}
		}
	})
	return first
}

type timedCompressedReader struct {
	io.Reader
	elapsed time.Duration
	once    sync.Once
}

func (r *timedCompressedReader) Read(p []byte) (int, error) {
	started := time.Now()
	n, err := r.Reader.Read(p)
	r.elapsed += time.Since(started)
	return n, err
}

func (r *timedCompressedReader) Close() error {
	r.once.Do(func() {
		recordBidPathLatency("compressed", r.elapsed)
	})
	if closer, ok := r.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type negotiatedResponseWriter struct {
	http.ResponseWriter
	acceptGzip bool
	writer     *gzip.Writer
	status     int
}

func newNegotiatedResponseWriter(w http.ResponseWriter, acceptEncoding string) *negotiatedResponseWriter {
	return &negotiatedResponseWriter{ResponseWriter: w, acceptGzip: acceptsGzip(acceptEncoding)}
}

func (w *negotiatedResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	appendVary(w.Header(), "Accept-Encoding")
	if w.acceptGzip && status == http.StatusOK && responseIsJSON(w.Header().Get("Content-Type")) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *negotiatedResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.writer != nil {
		n, err := w.writer.Write(data)
		if err != nil {
			return n, err
		}
		if err := w.writer.Flush(); err != nil {
			return n, err
		}
		return n, nil
	}
	return w.ResponseWriter.Write(data)
}

func (w *negotiatedResponseWriter) Close() error {
	if w.writer != nil {
		return w.writer.Close()
	}
	return nil
}

func responseIsJSON(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return contentType == "application/json" || contentType == "application/openrtb+json"
}

func acceptsGzip(value string) bool {
	accepted := false
	for _, part := range strings.Split(value, ",") {
		fields := strings.Split(part, ";")
		coding := strings.ToLower(strings.TrimSpace(fields[0]))
		if coding != "gzip" && coding != "*" {
			continue
		}
		quality := 1.0
		for _, parameter := range fields[1:] {
			name, raw, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(name), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				quality = 0
			} else {
				quality = parsed
			}
		}
		if quality > 0 {
			accepted = true
		}
		if coding == "gzip" {
			return quality > 0
		}
	}
	return accepted
}

func appendVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, token := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

type trafficStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *trafficStatusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *trafficStatusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *trafficStatusWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// ProtectedMetricsHandler restricts expvar to direct peers in the configured
// allowlist. The reverse proxy must also block /debug/vars from public routes.
func ProtectedMetricsHandler(c *Config, next http.Handler) http.Handler {
	networks, _ := parseMetricsAllowedCIDRs(nil)
	if c != nil {
		networks, _ = parseMetricsAllowedCIDRs(c.MetricsAllowedCIDRs)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(remoteAddrIP(r.RemoteAddr))
		for _, network := range networks {
			if ip != nil && network.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})
}

// MetricsHandler protects the metrics surface and refreshes bounded dependency
// health evidence only for an authorized scrape.
func (c *Controller) MetricsHandler(next http.Handler) http.Handler {
	return ProtectedMetricsHandler(c.C, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.refreshDependencyMetrics(r.Context())
		next.ServeHTTP(w, r)
	}))
}

func parseMetricsAllowedCIDRs(values []string) ([]*net.IPNet, error) {
	if len(values) == 0 {
		values = defaultMetricsAllowedCIDRs
	}
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if ip := net.ParseIP(value); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("metrics_allowed_cidrs entry %q is invalid", value)
		}
		networks = append(networks, network)
	}
	if len(networks) == 0 {
		return nil, fmt.Errorf("metrics_allowed_cidrs must contain at least one address or CIDR")
	}
	return networks, nil
}
