package hostedpayment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrProviderUnavailable = errors.New("hosted payment provider unavailable")

const stripeAPIVersion = "2024-06-20"

type ProviderError struct {
	StatusCode int
	Code       string
	Retryable  bool
}

func (e *ProviderError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("provider request failed with HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("provider request failed with HTTP %d (%s)", e.StatusCode, e.Code)
}

func (e *ProviderError) Unwrap() error {
	if e.Retryable {
		return ErrProviderUnavailable
	}
	return nil
}

type StripeProvider struct {
	baseURL     *url.URL
	apiKey      string
	client      *http.Client
	maxBody     int64
	maxAttempts int
	retryBase   time.Duration
}

func (*StripeProvider) String() string   { return "hostedpayment.StripeProvider{redacted}" }
func (*StripeProvider) GoString() string { return "hostedpayment.StripeProvider{redacted}" }

func NewStripeProvider(config Config, apiKey string, client *http.Client) (*StripeProvider, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("hosted payment API key is empty")
	}
	if !apiKeyMatchesMode(apiKey, config.LiveMode) {
		return nil, fmt.Errorf("hosted payment API key does not match the configured provider mode")
	}
	base, err := url.Parse(config.APIBaseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{
			Timeout: config.requestTimeout(),
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &StripeProvider{
		baseURL: base, apiKey: apiKey, client: client, maxBody: config.MaxBodyBytes,
		maxAttempts: config.MaxAttempts, retryBase: config.retryBase(),
	}, nil
}

func (p *StripeProvider) CreateFundingCustomer(ctx context.Context, input CustomerRequest) (string, error) {
	if input.PartyID == 0 {
		return "", fmt.Errorf("advertiser party id is required")
	}
	values := url.Values{
		"metadata[aofei_party_type]": {"advertiser"},
		"metadata[aofei_party_id]":   {strconv.FormatUint(input.PartyID, 10)},
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := p.postForm(ctx, "/v1/customers", values, input.IdempotencyKey, &response); err != nil {
		return "", err
	}
	if err := ValidateOpaqueToken(response.ID, "cus_"); err != nil {
		return "", invalidProviderResponse()
	}
	return response.ID, nil
}

func (p *StripeProvider) CreateFundingCheckout(ctx context.Context, input CheckoutRequest) (HostedRedirect, error) {
	if input.OperationID == 0 || input.StatementID == 0 || input.PartyID == 0 {
		return HostedRedirect{}, fmt.Errorf("operation, statement, and advertiser ids are required")
	}
	cents, err := moneyToCents(input.Amount)
	if err != nil {
		return HostedRedirect{}, err
	}
	if err := validateCallbackURL(input.SuccessURL); err != nil {
		return HostedRedirect{}, fmt.Errorf("success URL: %w", err)
	}
	if err := validateCallbackURL(input.CancelURL); err != nil {
		return HostedRedirect{}, fmt.Errorf("cancel URL: %w", err)
	}
	if err := ValidateOpaqueToken(input.CustomerToken, "cus_"); err != nil {
		return HostedRedirect{}, fmt.Errorf("funding customer: %w", err)
	}
	values := url.Values{
		"mode":                                   {"payment"},
		"client_reference_id":                    {strconv.FormatUint(input.OperationID, 10)},
		"success_url":                            {input.SuccessURL},
		"cancel_url":                             {input.CancelURL},
		"line_items[0][quantity]":                {"1"},
		"line_items[0][price_data][currency]":    {"usd"},
		"line_items[0][price_data][unit_amount]": {strconv.FormatInt(cents, 10)},
		"line_items[0][price_data][product_data][name]":     {"W8M advertiser funding"},
		"metadata[aofei_operation_id]":                      {strconv.FormatUint(input.OperationID, 10)},
		"metadata[aofei_statement_id]":                      {strconv.FormatUint(input.StatementID, 10)},
		"metadata[aofei_party_id]":                          {strconv.FormatUint(input.PartyID, 10)},
		"payment_intent_data[metadata][aofei_operation_id]": {strconv.FormatUint(input.OperationID, 10)},
		"payment_intent_data[metadata][aofei_statement_id]": {strconv.FormatUint(input.StatementID, 10)},
		"payment_intent_data[metadata][aofei_party_id]":     {strconv.FormatUint(input.PartyID, 10)},
	}
	values.Set("customer", input.CustomerToken)
	var response struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := p.postForm(ctx, "/v1/checkout/sessions", values, input.IdempotencyKey, &response); err != nil {
		return HostedRedirect{}, err
	}
	if err := ValidateOpaqueToken(response.ID, "cs_"); err != nil {
		return HostedRedirect{}, invalidProviderResponse()
	}
	if err := validateHostedRedirect(response.URL); err != nil {
		return HostedRedirect{}, invalidProviderResponse()
	}
	return HostedRedirect{ObjectToken: response.ID, URL: response.URL, ExpiresAt: time.Unix(response.ExpiresAt, 0).UTC()}, nil
}

func (p *StripeProvider) ExpireFundingCheckout(ctx context.Context, token, idempotencyKey string) error {
	if err := ValidateOpaqueToken(token, "cs_"); err != nil {
		return err
	}
	return p.postForm(ctx, "/v1/checkout/sessions/"+url.PathEscape(token)+"/expire", nil, idempotencyKey, &struct{}{})
}

func (p *StripeProvider) CreatePayoutAccount(ctx context.Context, input PayoutAccountRequest) (string, error) {
	if input.PartyID == 0 || !safeCountry.MatchString(input.Country) {
		return "", fmt.Errorf("publisher id and two-letter uppercase country are required")
	}
	values := url.Values{
		"type":                               {"express"},
		"country":                            {input.Country},
		"capabilities[transfers][requested]": {"true"},
		"metadata[aofei_party_type]":         {"publisher"},
		"metadata[aofei_party_id]":           {strconv.FormatUint(input.PartyID, 10)},
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := p.postForm(ctx, "/v1/accounts", values, input.IdempotencyKey, &response); err != nil {
		return "", err
	}
	if err := ValidateOpaqueToken(response.ID, "acct_"); err != nil {
		return "", invalidProviderResponse()
	}
	return response.ID, nil
}

func (p *StripeProvider) CreatePayoutOnboarding(ctx context.Context, input OnboardingRequest) (HostedRedirect, error) {
	if input.PartyID == 0 {
		return HostedRedirect{}, fmt.Errorf("publisher id is required")
	}
	if err := ValidateOpaqueToken(input.AccountToken, "acct_"); err != nil {
		return HostedRedirect{}, err
	}
	if err := validateCallbackURL(input.RefreshURL); err != nil {
		return HostedRedirect{}, fmt.Errorf("refresh URL: %w", err)
	}
	if err := validateCallbackURL(input.ReturnURL); err != nil {
		return HostedRedirect{}, fmt.Errorf("return URL: %w", err)
	}
	values := url.Values{
		"account":     {input.AccountToken},
		"refresh_url": {input.RefreshURL},
		"return_url":  {input.ReturnURL},
		"type":        {"account_onboarding"},
	}
	var response struct {
		Object    string `json:"object"`
		URL       string `json:"url"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := p.postForm(ctx, "/v1/account_links", values, input.IdempotencyKey, &response); err != nil {
		return HostedRedirect{}, err
	}
	if response.Object != "account_link" {
		return HostedRedirect{}, invalidProviderResponse()
	}
	if err := validateHostedRedirect(response.URL); err != nil {
		return HostedRedirect{}, invalidProviderResponse()
	}
	return HostedRedirect{ObjectToken: input.AccountToken, URL: response.URL, ExpiresAt: time.Unix(response.ExpiresAt, 0).UTC()}, nil
}

func (p *StripeProvider) CreateTransfer(ctx context.Context, input TransferRequest) (ProviderObject, error) {
	if input.OperationID == 0 || input.StatementID == 0 || input.PartyID == 0 {
		return ProviderObject{}, fmt.Errorf("operation, statement, and publisher ids are required")
	}
	if err := ValidateOpaqueToken(input.AccountToken, "acct_"); err != nil {
		return ProviderObject{}, err
	}
	cents, err := moneyToCents(input.Amount)
	if err != nil {
		return ProviderObject{}, err
	}
	values := url.Values{
		"amount":                       {strconv.FormatInt(cents, 10)},
		"currency":                     {"usd"},
		"destination":                  {input.AccountToken},
		"transfer_group":               {fmt.Sprintf("aofei_statement_%d", input.StatementID)},
		"metadata[aofei_operation_id]": {strconv.FormatUint(input.OperationID, 10)},
		"metadata[aofei_statement_id]": {strconv.FormatUint(input.StatementID, 10)},
		"metadata[aofei_party_id]":     {strconv.FormatUint(input.PartyID, 10)},
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := p.postForm(ctx, "/v1/transfers", values, input.IdempotencyKey, &response); err != nil {
		return ProviderObject{}, err
	}
	if err := ValidateOpaqueToken(response.ID, "tr_"); err != nil {
		return ProviderObject{}, invalidProviderResponse()
	}
	return ProviderObject{Token: response.ID, Status: "submitted"}, nil
}

func (p *StripeProvider) CreateRefund(ctx context.Context, input RefundRequest) (ProviderObject, error) {
	if input.OperationID == 0 || input.StatementID == 0 || input.PartyID == 0 {
		return ProviderObject{}, fmt.Errorf("operation, statement, and advertiser ids are required")
	}
	if err := ValidateOpaqueToken(input.PaymentIntentToken, "pi_"); err != nil {
		return ProviderObject{}, err
	}
	cents, err := moneyToCents(input.Amount)
	if err != nil {
		return ProviderObject{}, err
	}
	values := url.Values{
		"amount":                       {strconv.FormatInt(cents, 10)},
		"payment_intent":               {input.PaymentIntentToken},
		"reason":                       {"requested_by_customer"},
		"metadata[aofei_operation_id]": {strconv.FormatUint(input.OperationID, 10)},
		"metadata[aofei_statement_id]": {strconv.FormatUint(input.StatementID, 10)},
		"metadata[aofei_party_id]":     {strconv.FormatUint(input.PartyID, 10)},
	}
	var response struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := p.postForm(ctx, "/v1/refunds", values, input.IdempotencyKey, &response); err != nil {
		return ProviderObject{}, err
	}
	if err := ValidateOpaqueToken(response.ID, "re_"); err != nil {
		return ProviderObject{}, invalidProviderResponse()
	}
	return ProviderObject{Token: response.ID, Status: response.Status}, nil
}

func (p *StripeProvider) RetrieveBalanceTransaction(ctx context.Context, token string) (BalanceTransaction, error) {
	if err := ValidateOpaqueToken(token, "txn_"); err != nil {
		return BalanceTransaction{}, err
	}
	var response struct {
		ID       string `json:"id"`
		Source   string `json:"source"`
		Currency string `json:"currency"`
		Amount   int64  `json:"amount"`
		Fee      int64  `json:"fee"`
		Net      int64  `json:"net"`
		Status   string `json:"status"`
	}
	if err := p.get(ctx, "/v1/balance_transactions/"+url.PathEscape(token), &response); err != nil {
		return BalanceTransaction{}, err
	}
	if err := ValidateOpaqueToken(response.ID, "txn_"); err != nil {
		return BalanceTransaction{}, invalidProviderResponse()
	}
	return BalanceTransaction{Token: response.ID, SourceToken: response.Source, Currency: strings.ToUpper(response.Currency), AmountCents: response.Amount, FeeCents: response.Fee, NetCents: response.Net, Status: response.Status}, nil
}

func (p *StripeProvider) postForm(ctx context.Context, path string, values url.Values, idempotencyKey string, output any) error {
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return err
	}
	return p.do(ctx, http.MethodPost, path, values, idempotencyKey, output)
}

func (p *StripeProvider) get(ctx context.Context, path string, output any) error {
	return p.do(ctx, http.MethodGet, path, nil, "", output)
}

func (p *StripeProvider) do(ctx context.Context, method, path string, values url.Values, idempotencyKey string, output any) error {
	if p == nil || p.baseURL == nil || p.client == nil {
		return fmt.Errorf("hosted payment provider is not initialized")
	}
	endpoint := *p.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	var body []byte
	if values != nil {
		body = []byte(values.Encode())
	}
	var last error
	for attempt := 1; attempt <= p.maxAttempts; attempt++ {
		metricProviderRequests.Add(1)
		req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.SetBasicAuth(p.apiKey, "")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Stripe-Version", stripeAPIVersion)
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		response, err := p.client.Do(req)
		if err == nil {
			last = p.decodeResponse(response, output)
		} else {
			last = fmt.Errorf("%w: provider transport failed", ErrProviderUnavailable)
		}
		if !errors.Is(last, ErrProviderUnavailable) || attempt == p.maxAttempts {
			if last != nil {
				metricProviderErrors.Add(1)
			}
			return last
		}
		timer := time.NewTimer(p.retryBase * time.Duration(1<<(attempt-1)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

func (p *StripeProvider) decodeResponse(response *http.Response, output any) error {
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, p.maxBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("%w: provider response read failed", ErrProviderUnavailable)
	}
	if int64(len(body)) > p.maxBody {
		return fmt.Errorf("%w: provider response exceeds configured limit", ErrProviderUnavailable)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &envelope)
		code := envelope.Error.Code
		if !safeErrorCode(code) {
			code = ""
		}
		retryable := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusConflict ||
			response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return &ProviderError{StatusCode: response.StatusCode, Code: code, Retryable: retryable}
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("%w: provider returned invalid JSON", ErrProviderUnavailable)
	}
	return nil
}

func invalidProviderResponse() error {
	return fmt.Errorf("%w: provider response failed validation", ErrProviderUnavailable)
}

func safeErrorCode(code string) bool {
	if code == "" || len(code) > 64 {
		return false
	}
	for _, r := range code {
		if r != '_' && r != '-' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func validateCallbackURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("callback URL is invalid")
	}
	if u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	if u.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
		return nil
	}
	return fmt.Errorf("callback URL must use HTTPS outside loopback")
}
