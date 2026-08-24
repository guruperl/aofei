package managementapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v4"
)

var managementAPIRequests = expvar.NewMap("aofei_management_api_requests_total")

type handler struct{ service *Service }

func newHandler(service *Service) http.Handler { return &handler{service: service} }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Request-ID", requestID)
	if h == nil || h.service == nil {
		writeError(w, requestID, http.StatusNotFound, "not_found", "resource not found", nil)
		return
	}
	if r.URL.Path == "/api/v1" || r.URL.Path == "/api/v1/" {
		writeError(w, requestID, http.StatusNotFound, "not_found", "resource not found", nil)
		return
	}
	if r.ContentLength > h.service.config.MaxBodyBytes {
		writeError(w, requestID, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the configured limit", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.service.config.requestTimeout())
	defer cancel()
	r = r.WithContext(ctx)
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") || strings.Contains(strings.TrimPrefix(authorization, "Bearer "), " ") {
		w.Header().Set("WWW-Authenticate", `Bearer realm="w8m-management-api"`)
		managementAPIRequests.Add("unauthorized", 1)
		writeError(w, requestID, http.StatusUnauthorized, "unauthorized", "invalid service credential", nil)
		return
	}
	principal, err := h.service.Authenticate(ctx, strings.TrimPrefix(authorization, "Bearer "))
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="w8m-management-api"`)
			managementAPIRequests.Add("unauthorized", 1)
			writeError(w, requestID, http.StatusUnauthorized, "unauthorized", "invalid service credential", nil)
		} else {
			w.Header().Set("Retry-After", "1")
			managementAPIRequests.Add("dependency", 1)
			writeError(w, requestID, http.StatusServiceUnavailable, "dependency_unavailable", "credential verification is temporarily unavailable", nil)
		}
		return
	}
	allowed, err := h.service.allowQuota(ctx, principal)
	if err != nil {
		managementAPIRequests.Add("dependency", 1)
		w.Header().Set("Retry-After", "1")
		writeError(w, requestID, http.StatusServiceUnavailable, "dependency_unavailable", "request admission is temporarily unavailable", nil)
		return
	}
	if !allowed {
		managementAPIRequests.Add("quota", 1)
		w.Header().Set("Retry-After", "60")
		writeError(w, requestID, http.StatusTooManyRequests, "rate_limited", "request quota exceeded", nil)
		return
	}
	status, err := h.dispatch(w, r, requestID, principal)
	if err != nil {
		managementAPIRequests.Add("request_error", 1)
		h.writeServiceError(w, requestID, err)
		return
	}
	managementAPIRequests.Add(strconv.Itoa(status), 1)
}

func (h *handler) dispatch(w http.ResponseWriter, r *http.Request, requestID string, p Principal) (int, error) {
	pathValue := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1"), "/")
	parts := strings.Split(pathValue, "/")
	if len(parts) == 1 && parts[0] == "advertiser" && r.Method == http.MethodGet {
		if err := requireScope(p, ScopeCampaignRead); err != nil {
			return 0, err
		}
		data, err := h.service.advertiser(r.Context(), p.AdvertiserID)
		return writeData(w, http.StatusOK, data, nil, err)
	}
	if len(parts) == 1 && parts[0] == "campaigns" {
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCampaignRead); err != nil {
				return 0, err
			}
			cursor, limit, err := parsePage(r)
			if err != nil {
				return 0, err
			}
			items, next, err := h.service.campaigns(r.Context(), p.AdvertiserID, cursor, limit)
			meta := &pageMeta{NextCursor: next, Limit: limit, Order: "id", Timezone: "UTC", Currency: "USD"}
			return writeData(w, http.StatusOK, items, meta, err)
		case http.MethodPost:
			if err := requireScope(p, ScopeCampaignWrite); err != nil {
				return 0, err
			}
			var input campaignWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateCampaignWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "campaign.create", Value: input})
		}
	}
	if len(parts) == 2 && parts[0] == "campaigns" {
		id, err := parseUint(parts[1], "campaign id")
		if err != nil {
			return 0, clientError{status: 404, code: "not_found", err: ErrNotFound}
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCampaignRead); err != nil {
				return 0, err
			}
			item, err := h.service.campaign(r.Context(), h.service.db, p.AdvertiserID, id)
			if err == nil {
				w.Header().Set("ETag", versionETag(item.Version))
			}
			return writeData(w, http.StatusOK, item, nil, err)
		case http.MethodPatch:
			if err := requireScope(p, ScopeCampaignWrite); err != nil {
				return 0, err
			}
			if _, err := h.service.campaign(r.Context(), h.service.db, p.AdvertiserID, id); err != nil {
				return 0, err
			}
			version, err := parseIfMatch(r)
			if err != nil {
				return 0, err
			}
			var input campaignWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateCampaignWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "campaign.update", ResourceID: id, Version: version, Value: input})
		}
	}
	if len(parts) == 3 && parts[0] == "campaigns" && parts[2] == "items" {
		campaignID, err := parseUint(parts[1], "campaign id")
		if err != nil {
			return 0, ErrNotFound
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCampaignRead); err != nil {
				return 0, err
			}
		case http.MethodPost:
			if err := requireScope(p, ScopeCampaignWrite); err != nil {
				return 0, err
			}
		}
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			if _, err := h.service.campaign(r.Context(), h.service.db, p.AdvertiserID, campaignID); err != nil {
				return 0, err
			}
		}
		switch r.Method {
		case http.MethodGet:
			cursor, limit, err := parsePage(r)
			if err != nil {
				return 0, err
			}
			items, next, err := h.service.items(r.Context(), p.AdvertiserID, campaignID, cursor, limit)
			return writeData(w, 200, items, &pageMeta{NextCursor: next, Limit: limit, Order: "id", Timezone: "UTC", Currency: "USD"}, err)
		case http.MethodPost:
			var input itemWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateItemWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "item.create", ParentID: campaignID, Value: input})
		}
	}
	if len(parts) == 2 && parts[0] == "items" {
		id, err := parseUint(parts[1], "item id")
		if err != nil {
			return 0, ErrNotFound
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCampaignRead); err != nil {
				return 0, err
			}
			item, err := h.service.item(r.Context(), h.service.db, p.AdvertiserID, id)
			if err == nil {
				w.Header().Set("ETag", versionETag(item.Version))
			}
			return writeData(w, 200, item, nil, err)
		case http.MethodPatch:
			if err := requireScope(p, ScopeCampaignWrite); err != nil {
				return 0, err
			}
			if _, err := h.service.item(r.Context(), h.service.db, p.AdvertiserID, id); err != nil {
				return 0, err
			}
			version, err := parseIfMatch(r)
			if err != nil {
				return 0, err
			}
			var input itemWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateItemWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "item.update", ResourceID: id, Version: version, Value: input})
		}
	}
	if len(parts) == 3 && parts[0] == "items" && parts[2] == "creatives" {
		itemID, err := parseUint(parts[1], "item id")
		if err != nil {
			return 0, ErrNotFound
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCreativeRead); err != nil {
				return 0, err
			}
		case http.MethodPost:
			if err := requireScope(p, ScopeCreativeWrite); err != nil {
				return 0, err
			}
		}
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			if _, err := h.service.item(r.Context(), h.service.db, p.AdvertiserID, itemID); err != nil {
				return 0, err
			}
		}
		switch r.Method {
		case http.MethodGet:
			cursor, limit, err := parsePage(r)
			if err != nil {
				return 0, err
			}
			items, next, err := h.service.creatives(r.Context(), p.AdvertiserID, itemID, cursor, limit)
			return writeData(w, 200, items, &pageMeta{NextCursor: next, Limit: limit, Order: "id", Timezone: "UTC"}, err)
		case http.MethodPost:
			var input creativeWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateCreativeWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "creative.create", ParentID: itemID, Value: input})
		}
	}
	if len(parts) == 2 && parts[0] == "creatives" {
		id, err := parseUint(parts[1], "creative id")
		if err != nil {
			return 0, ErrNotFound
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeCreativeRead); err != nil {
				return 0, err
			}
			item, err := h.service.creative(r.Context(), h.service.db, p.AdvertiserID, id)
			if err == nil {
				w.Header().Set("ETag", versionETag(item.Version))
			}
			return writeData(w, 200, item, nil, err)
		case http.MethodPatch:
			if err := requireScope(p, ScopeCreativeWrite); err != nil {
				return 0, err
			}
			if _, err := h.service.creative(r.Context(), h.service.db, p.AdvertiserID, id); err != nil {
				return 0, err
			}
			version, err := parseIfMatch(r)
			if err != nil {
				return 0, err
			}
			var input creativeWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateCreativeWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "creative.update", ResourceID: id, Version: version, Value: input})
		}
	}
	if len(parts) == 3 && parts[0] == "items" && parts[2] == "targeting" {
		itemID, err := parseUint(parts[1], "item id")
		if err != nil {
			return 0, ErrNotFound
		}
		switch r.Method {
		case http.MethodGet:
			if err := requireScope(p, ScopeTargetingRead); err != nil {
				return 0, err
			}
			item, err := h.service.targeting(r.Context(), h.service.db, p.AdvertiserID, itemID)
			if err == nil {
				w.Header().Set("ETag", versionETag(item.Version))
			}
			return writeData(w, 200, item, nil, err)
		case http.MethodPatch:
			if err := requireScope(p, ScopeTargetingWrite); err != nil {
				return 0, err
			}
			if _, err := h.service.targeting(r.Context(), h.service.db, p.AdvertiserID, itemID); err != nil {
				return 0, err
			}
			version, err := parseIfMatch(r)
			if err != nil {
				return 0, err
			}
			var input targetingWrite
			body, err := decodeBody(w, r, h.service.config.MaxBodyBytes, &input)
			if err != nil {
				return 0, err
			}
			if err := validateTargetingWrite(&input); err != nil {
				return 0, clientError{status: 400, code: "invalid_request", err: err}
			}
			return h.writeMutation(w, r, p, body, mutation{Kind: "targeting.update", ResourceID: itemID, Version: version, Value: input})
		}
	}
	if len(parts) == 2 && parts[0] == "operations" && r.Method == http.MethodGet {
		if !p.Has(ScopeCampaignRead) && !p.Has(ScopeCreativeRead) && !p.Has(ScopeTargetingRead) {
			return 0, ErrForbidden
		}
		if len(parts[1]) != 32 {
			return 0, ErrNotFound
		}
		if _, err := hex.DecodeString(parts[1]); err != nil {
			return 0, ErrNotFound
		}
		item, err := h.service.operation(r.Context(), p.AdvertiserID, parts[1])
		return writeData(w, 200, item, nil, err)
	}
	if len(parts) == 2 && parts[0] == "reports" && parts[1] == "delivery" && r.Method == http.MethodGet {
		if err := requireScope(p, ScopeReportRead); err != nil {
			return 0, err
		}
		cursor, limit, err := parsePage(r)
		if err != nil {
			return 0, err
		}
		from, to, err := parseReportRange(r, h.service.now().UTC())
		if err != nil {
			return 0, err
		}
		items, next, partial, err := h.service.reports(r.Context(), p.AdvertiserID, cursor, limit, from, to)
		meta := &pageMeta{NextCursor: next, Limit: limit, Order: "id", Timezone: "UTC", Currency: "USD", Freshness: "30m", Source: "report_delivery", Partial: partial}
		return writeData(w, 200, items, meta, err)
	}
	if knownPath(parts) {
		w.Header().Set("Allow", allowedMethods(parts))
		return 0, clientError{status: http.StatusMethodNotAllowed, code: "method_not_allowed", err: fmt.Errorf("method not allowed")}
	}
	return 0, ErrNotFound
}

func (h *handler) writeMutation(w http.ResponseWriter, r *http.Request, p Principal, body []byte, mutation mutation) (int, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(key) < 8 || len(key) > 128 || hasControl(key) {
		return 0, clientError{status: 400, code: "invalid_idempotency_key", err: fmt.Errorf("Idempotency-Key must contain 8 to 128 visible characters")}
	}
	requestHash := h.service.digest("management-api-request-v1", r.Method+"\n"+r.URL.EscapedPath()+"\n"+r.Header.Get("If-Match")+"\n"+string(body))
	keyHash := h.service.digest("management-api-idempotency-v1", key)
	result, err := h.service.mutate(r.Context(), p, keyHash, requestHash, mutation)
	if err != nil {
		return 0, err
	}
	if result.Replay {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	w.WriteHeader(result.Status)
	_, err = w.Write(result.Body)
	return result.Status, err
}

func (s *Service) allowQuota(ctx context.Context, p Principal) (bool, error) {
	minute := s.now().UTC().Unix() / 60
	credentialKey := "aofei:management-api:quota:credential:" + strconv.FormatUint(p.CredentialID, 10) + ":" + strconv.FormatInt(minute, 10)
	accountKey := "aofei:management-api:quota:account:" + strconv.FormatUint(p.AdvertiserID, 10) + ":" + strconv.FormatInt(minute, 10)
	const script = `
local c = redis.call('INCR', KEYS[1])
if c == 1 then redis.call('EXPIRE', KEYS[1], 120) end
if c > tonumber(ARGV[1]) then return 0 end
local a = redis.call('INCR', KEYS[2])
if a == 1 then redis.call('EXPIRE', KEYS[2], 120) end
if a > tonumber(ARGV[2]) then return 0 end
return 1`
	var allowed int
	err := s.redis.Do(ctx, radix.Cmd(&allowed, "EVAL", script, "2", credentialKey, accountKey,
		strconv.Itoa(s.config.CredentialRequestsMinute), strconv.Itoa(s.config.AccountRequestsMinute)))
	return allowed == 1, err
}

type clientError struct {
	status int
	code   string
	err    error
}

func (e clientError) Error() string { return e.err.Error() }
func (e clientError) Unwrap() error { return e.err }

func (h *handler) writeServiceError(w http.ResponseWriter, requestID string, err error) {
	var client clientError
	switch {
	case errors.As(err, &client):
		writeError(w, requestID, client.status, client.code, client.err.Error(), nil)
	case errors.Is(err, ErrUnauthorized):
		writeError(w, requestID, 401, "unauthorized", "invalid service credential", nil)
	case errors.Is(err, ErrForbidden):
		writeError(w, requestID, 403, "forbidden", "credential scope does not permit this operation", nil)
	case errors.Is(err, ErrNotFound):
		writeError(w, requestID, 404, "not_found", "resource not found", nil)
	case errors.Is(err, ErrConflict):
		writeError(w, requestID, 409, "version_conflict", "resource version does not match If-Match", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		writeError(w, requestID, 409, "idempotency_conflict", "Idempotency-Key was used for a different request", nil)
	case errors.Is(err, ErrIdempotencyPending):
		w.Header().Set("Retry-After", "1")
		writeError(w, requestID, 409, "idempotency_pending", "the original request is still being processed", nil)
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		w.Header().Set("Retry-After", "1")
		writeError(w, requestID, 503, "request_unavailable", "request did not complete; retry idempotent writes with the same key", nil)
	default:
		writeError(w, requestID, 503, "dependency_unavailable", "a required dependency is temporarily unavailable", nil)
	}
}

func requireScope(p Principal, scope string) error {
	if !p.Has(scope) {
		return ErrForbidden
	}
	return nil
}

func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, target any) ([]byte, error) {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if mediaType != "application/json" {
		return nil, clientError{status: 415, code: "unsupported_media_type", err: fmt.Errorf("Content-Type must be application/json")}
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, clientError{status: 413, code: "body_too_large", err: fmt.Errorf("request body exceeds the configured limit")}
		}
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, ErrMoneyStringRequired) {
			return nil, clientError{status: 400, code: "money_string_required", err: ErrMoneyStringRequired}
		}
		return nil, clientError{status: 400, code: "invalid_json", err: fmt.Errorf("request body is not valid contract JSON")}
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, clientError{status: 400, code: "invalid_json", err: fmt.Errorf("request body contains trailing data")}
	}
	return body, nil
}

func parsePage(r *http.Request) (uint64, int, error) {
	limit := defaultPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxPageSize {
			return 0, 0, clientError{status: 400, code: "invalid_page", err: fmt.Errorf("limit must be between 1 and 100")}
		}
		limit = value
	}
	cursor := uint64(0)
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, 0, clientError{status: 400, code: "invalid_page", err: fmt.Errorf("cursor is invalid")}
		}
		cursor = value
	}
	return cursor, limit, nil
}

func parseReportRange(r *http.Request, now time.Time) (time.Time, time.Time, error) {
	to := now.Truncate(time.Minute)
	from := to.Add(-24 * time.Hour)
	for _, field := range []struct {
		name   string
		target *time.Time
	}{{"from", &from}, {"to", &to}} {
		if raw := r.URL.Query().Get(field.name); raw != "" {
			value, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return time.Time{}, time.Time{}, clientError{status: 400, code: "invalid_range", err: fmt.Errorf("%s must be RFC3339", field.name)}
			}
			*field.target = value.UTC()
		}
	}
	if !from.Before(to) || to.Sub(from) > 31*24*time.Hour || to.After(now.Add(5*time.Minute)) {
		return time.Time{}, time.Time{}, clientError{status: 400, code: "invalid_range", err: fmt.Errorf("report range must be ordered, no longer than 31 days, and not in the future")}
	}
	return from, to, nil
}

func parseIfMatch(r *http.Request) (uint64, error) {
	raw := r.Header.Get("If-Match")
	if raw == "" {
		return 0, clientError{status: 428, code: "precondition_required", err: fmt.Errorf("If-Match with the current version ETag is required")}
	}
	if len(raw) < 4 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return 0, clientError{status: 400, code: "invalid_precondition", err: fmt.Errorf("If-Match version is invalid")}
	}
	value := raw[1 : len(raw)-1]
	if !strings.HasPrefix(value, "v") || len(value) < 2 || value[1] == '0' {
		return 0, clientError{status: 400, code: "invalid_precondition", err: fmt.Errorf("If-Match version is invalid")}
	}
	version, err := strconv.ParseUint(value[1:], 10, 64)
	if err != nil || version == 0 {
		return 0, clientError{status: 400, code: "invalid_precondition", err: fmt.Errorf("If-Match version is invalid")}
	}
	return version, nil
}

func versionETag(version uint64) string { return `"v` + strconv.FormatUint(version, 10) + `"` }

func writeData(w http.ResponseWriter, status int, data any, meta *pageMeta, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	w.WriteHeader(status)
	return status, json.NewEncoder(w).Encode(envelope{Data: data, Meta: meta})
}

func writeError(w http.ResponseWriter, requestID string, status int, code, message string, fields map[string]string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorDetail{Code: code, Message: message, RequestID: requestID, Fields: fields}})
}

func newRequestID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(value)
}

func knownPath(parts []string) bool {
	if len(parts) == 1 {
		return parts[0] == "advertiser" || parts[0] == "campaigns"
	}
	if len(parts) == 2 {
		return parts[0] == "campaigns" || parts[0] == "items" || parts[0] == "creatives" || parts[0] == "operations" || (parts[0] == "reports" && parts[1] == "delivery")
	}
	return len(parts) == 3 && ((parts[0] == "campaigns" && parts[2] == "items") || (parts[0] == "items" && (parts[2] == "creatives" || parts[2] == "targeting")))
}

func allowedMethods(parts []string) string {
	if len(parts) == 1 && parts[0] == "advertiser" {
		return "GET"
	}
	if len(parts) == 2 && (parts[0] == "operations" || parts[0] == "reports") {
		return "GET"
	}
	if len(parts) == 3 && parts[2] == "targeting" {
		return "GET, PATCH"
	}
	if len(parts) == 1 || len(parts) == 3 {
		return "GET, POST"
	}
	return "GET, PATCH"
}
