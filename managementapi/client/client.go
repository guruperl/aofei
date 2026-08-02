// Code generated from docs/management-api-openapi.yaml; DO NOT EDIT.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	api "github.com/guruperl/aofei/managementapi"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type Meta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	Limit      int    `json:"limit"`
	Order      string `json:"order"`
	Timezone   string `json:"timezone"`
	Currency   string `json:"currency,omitempty"`
	Freshness  string `json:"freshness,omitempty"`
	Source     string `json:"source,omitempty"`
	Partial    bool   `json:"partial,omitempty"`
}

type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}
type One[T any] struct {
	Data T `json:"data"`
}
type Accepted[T any] struct {
	Data struct {
		Resource  T             `json:"resource"`
		Operation api.Operation `json:"operation"`
	} `json:"data"`
}

type Error struct {
	Status                   int
	Code, Message, RequestID string
}

func (e *Error) Error() string {
	return fmt.Sprintf("management API %d %s: %s", e.Status, e.Code, e.Message)
}

func New(baseURL, token string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("management API base URL is invalid")
	}
	if token == "" {
		return nil, fmt.Errorf("management API token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: parsed.String(), token: token, httpClient: httpClient}, nil
}

func (c *Client) Advertiser(ctx context.Context) (map[string]any, error) {
	var response One[map[string]any]
	err := c.do(ctx, http.MethodGet, "/api/v1/advertiser", "", "", nil, &response)
	return response.Data, err
}

func (c *Client) ListCampaigns(ctx context.Context, cursor string, limit int) ([]api.Campaign, Meta, error) {
	var response Page[api.Campaign]
	err := c.do(ctx, http.MethodGet, pagePath("/api/v1/campaigns", cursor, limit), "", "", nil, &response)
	return response.Data, response.Meta, err
}

func (c *Client) GetCampaign(ctx context.Context, id uint64) (api.Campaign, error) {
	var response One[api.Campaign]
	err := c.do(ctx, http.MethodGet, idPath("/api/v1/campaigns", id), "", "", nil, &response)
	return response.Data, err
}

func (c *Client) CreateCampaign(ctx context.Context, idempotencyKey string, input api.CampaignInput) (Accepted[api.Campaign], error) {
	var response Accepted[api.Campaign]
	err := c.do(ctx, http.MethodPost, "/api/v1/campaigns", idempotencyKey, "", input, &response)
	return response, err
}

func (c *Client) UpdateCampaign(ctx context.Context, id, version uint64, idempotencyKey string, input api.CampaignInput) (Accepted[api.Campaign], error) {
	var response Accepted[api.Campaign]
	err := c.do(ctx, http.MethodPatch, idPath("/api/v1/campaigns", id), idempotencyKey, versionTag(version), input, &response)
	return response, err
}

func (c *Client) ListItems(ctx context.Context, campaignID uint64, cursor string, limit int) ([]api.Item, Meta, error) {
	var response Page[api.Item]
	err := c.do(ctx, http.MethodGet, pagePath(idPath("/api/v1/campaigns", campaignID)+"/items", cursor, limit), "", "", nil, &response)
	return response.Data, response.Meta, err
}

func (c *Client) GetItem(ctx context.Context, id uint64) (api.Item, error) {
	var response One[api.Item]
	err := c.do(ctx, http.MethodGet, idPath("/api/v1/items", id), "", "", nil, &response)
	return response.Data, err
}

func (c *Client) CreateItem(ctx context.Context, campaignID uint64, idempotencyKey string, input api.ItemInput) (Accepted[api.Item], error) {
	var response Accepted[api.Item]
	err := c.do(ctx, http.MethodPost, idPath("/api/v1/campaigns", campaignID)+"/items", idempotencyKey, "", input, &response)
	return response, err
}

func (c *Client) UpdateItem(ctx context.Context, id, version uint64, idempotencyKey string, input api.ItemInput) (Accepted[api.Item], error) {
	var response Accepted[api.Item]
	err := c.do(ctx, http.MethodPatch, idPath("/api/v1/items", id), idempotencyKey, versionTag(version), input, &response)
	return response, err
}

func (c *Client) ListCreatives(ctx context.Context, itemID uint64, cursor string, limit int) ([]api.Creative, Meta, error) {
	var response Page[api.Creative]
	err := c.do(ctx, http.MethodGet, pagePath(idPath("/api/v1/items", itemID)+"/creatives", cursor, limit), "", "", nil, &response)
	return response.Data, response.Meta, err
}

func (c *Client) GetCreative(ctx context.Context, id uint64) (api.Creative, error) {
	var response One[api.Creative]
	err := c.do(ctx, http.MethodGet, idPath("/api/v1/creatives", id), "", "", nil, &response)
	return response.Data, err
}

func (c *Client) CreateCreative(ctx context.Context, itemID uint64, idempotencyKey string, input api.CreativeInput) (Accepted[api.Creative], error) {
	var response Accepted[api.Creative]
	err := c.do(ctx, http.MethodPost, idPath("/api/v1/items", itemID)+"/creatives", idempotencyKey, "", input, &response)
	return response, err
}

func (c *Client) UpdateCreative(ctx context.Context, id, version uint64, idempotencyKey string, input api.CreativeInput) (Accepted[api.Creative], error) {
	var response Accepted[api.Creative]
	err := c.do(ctx, http.MethodPatch, idPath("/api/v1/creatives", id), idempotencyKey, versionTag(version), input, &response)
	return response, err
}

func (c *Client) GetTargeting(ctx context.Context, itemID uint64) (api.Targeting, error) {
	var response One[api.Targeting]
	err := c.do(ctx, http.MethodGet, idPath("/api/v1/items", itemID)+"/targeting", "", "", nil, &response)
	return response.Data, err
}

func (c *Client) UpdateTargeting(ctx context.Context, itemID, version uint64, idempotencyKey string, input api.TargetingInput) (Accepted[api.Targeting], error) {
	var response Accepted[api.Targeting]
	err := c.do(ctx, http.MethodPatch, idPath("/api/v1/items", itemID)+"/targeting", idempotencyKey, versionTag(version), input, &response)
	return response, err
}

func (c *Client) GetOperation(ctx context.Context, id string) (api.Operation, error) {
	var response One[api.Operation]
	err := c.do(ctx, http.MethodGet, "/api/v1/operations/"+url.PathEscape(id), "", "", nil, &response)
	return response.Data, err
}

func (c *Client) DeliveryReports(ctx context.Context, from, to time.Time, cursor string, limit int) ([]api.DeliveryReport, Meta, error) {
	values := url.Values{"from": {from.UTC().Format(time.RFC3339)}, "to": {to.UTC().Format(time.RFC3339)}}
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	var response Page[api.DeliveryReport]
	err := c.do(ctx, http.MethodGet, "/api/v1/reports/delivery?"+values.Encode(), "", "", nil, &response)
	return response.Data, response.Meta, err
}

func (c *Client) do(ctx context.Context, method, path, idempotencyKey, ifMatch string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(response.Body, 2<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				RequestID string `json:"request_id"`
			} `json:"error"`
		}
		if err := decoder.Decode(&failure); err != nil {
			return fmt.Errorf("management API status %d", response.StatusCode)
		}
		return &Error{Status: response.StatusCode, Code: failure.Error.Code, Message: failure.Error.Message, RequestID: failure.Error.RequestID}
	}
	return decoder.Decode(output)
}

func idPath(base string, id uint64) string { return base + "/" + strconv.FormatUint(id, 10) }
func versionTag(version uint64) string     { return `"v` + strconv.FormatUint(version, 10) + `"` }
func pagePath(base, cursor string, limit int) string {
	values := url.Values{}
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}
