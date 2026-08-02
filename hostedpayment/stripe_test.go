package hostedpayment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guruperl/aofei/accounting"
)

func stripeFixture(t *testing.T, handler http.HandlerFunc) (*StripeProvider, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	config := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: server.URL,
		PublicBaseURL: server.URL, APIKeyEnv: "STRIPE_API_KEY",
		WebhookSecretEnv: "STRIPE_WEBHOOK_SECRET", RequestTimeoutMS: 1000,
		MaxBodyBytes: 4096, WebhookToleranceSeconds: 300, MaxAttempts: 1,
		RetryBaseMS: 25, EventRetentionDays: 400, ReconciliationMaxAgeDays: 90,
	}
	provider, err := NewStripeProvider(config, "sk_test_fixture", server.Client())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return provider, server
}

func TestCreateFundingCheckoutSendsOpaqueMetadataAndExactCents(t *testing.T) {
	provider, server := stripeFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if user, _, ok := r.BasicAuth(); !ok || user != "sk_test_fixture" {
			t.Errorf("provider authentication missing")
		}
		if got := r.Header.Get("Stripe-Version"); got != stripeAPIVersion {
			t.Errorf("Stripe-Version = %q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "funding:44:execute" {
			t.Errorf("idempotency key = %q", got)
		}
		_ = r.ParseForm()
		for key, want := range map[string]string{
			"line_items[0][price_data][unit_amount]":            "1234",
			"metadata[aofei_operation_id]":                      "9",
			"metadata[aofei_statement_id]":                      "44",
			"payment_intent_data[metadata][aofei_statement_id]": "44",
			"payment_intent_data[metadata][aofei_party_id]":     "7",
			"customer": "cus_fixture",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "cs_fixture", "url": "https://checkout.stripe.com/c/pay/test", "expires_at": time.Now().Add(time.Hour).Unix(),
		})
	})
	defer server.Close()
	amount, _ := accounting.ParseMoney("12.340000")
	redirect, err := provider.CreateFundingCheckout(context.Background(), CheckoutRequest{
		OperationID: 9, StatementID: 44, PartyID: 7, Amount: amount,
		CustomerToken: "cus_fixture", SuccessURL: "https://www.w8m.com/funding/success",
		CancelURL: "https://www.w8m.com/funding/cancel", IdempotencyKey: "funding:44:execute",
	})
	if err != nil || redirect.ObjectToken != "cs_fixture" {
		t.Fatalf("CreateFundingCheckout = %#v, %v", redirect, err)
	}
}

func TestCreateFundingCheckoutRequiresApprovedCustomerTokenBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	provider, server := stripeFixture(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
	defer server.Close()
	amount, _ := accounting.ParseMoney("12.340000")
	_, err := provider.CreateFundingCheckout(context.Background(), CheckoutRequest{
		OperationID: 9, StatementID: 44, PartyID: 7, Amount: amount,
		SuccessURL: "https://www.w8m.com/funding/success",
		CancelURL:  "https://www.w8m.com/funding/cancel", IdempotencyKey: "funding:44:execute",
	})
	if err == nil || calls.Load() != 0 {
		t.Fatalf("missing customer binding error=%v calls=%d", err, calls.Load())
	}
}

func TestProviderRejectsFractionalCentBeforeNetwork(t *testing.T) {
	var calls atomic.Int32
	provider, server := stripeFixture(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
	defer server.Close()
	amount, _ := accounting.ParseMoney("0.000001")
	_, err := provider.CreateTransfer(context.Background(), TransferRequest{
		OperationID: 1, StatementID: 2, PartyID: 3, Amount: amount,
		AccountToken: "acct_fixture", IdempotencyKey: "payout:2:execute",
	})
	if err == nil || calls.Load() != 0 {
		t.Fatalf("fractional-cent transfer error=%v calls=%d", err, calls.Load())
	}
}

func TestCreateRefundCarriesExactOwnershipMetadata(t *testing.T) {
	provider, server := stripeFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		for key, want := range map[string]string{
			"amount":                       "250",
			"payment_intent":               "pi_fixture",
			"metadata[aofei_operation_id]": "17",
			"metadata[aofei_statement_id]": "44",
			"metadata[aofei_party_id]":     "7",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		_, _ = w.Write([]byte(`{"id":"re_fixture","status":"pending"}`))
	})
	defer server.Close()
	amount, _ := accounting.ParseMoney("2.500000")
	object, err := provider.CreateRefund(context.Background(), RefundRequest{
		OperationID: 17, StatementID: 44, PartyID: 7, Amount: amount,
		PaymentIntentToken: "pi_fixture", IdempotencyKey: "refund:17:create",
	})
	if err != nil || object.Token != "re_fixture" {
		t.Fatalf("CreateRefund = %#v, %v", object, err)
	}
}

func TestProviderRetriesOnlySanitizedRetryableErrorsWithSameKey(t *testing.T) {
	var calls atomic.Int32
	provider, server := stripeFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Idempotency-Key") != "customer:7:create" {
			t.Errorf("idempotency key changed")
		}
		if calls.Load() == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":"temporary_failure","message":"sensitive provider detail"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"cus_fixture"}`))
	})
	defer server.Close()
	provider.maxAttempts = 2
	provider.retryBase = time.Millisecond
	token, err := provider.CreateFundingCustomer(context.Background(), CustomerRequest{PartyID: 7, IdempotencyKey: "customer:7:create"})
	if err != nil || token != "cus_fixture" || calls.Load() != 2 {
		t.Fatalf("customer = %q, %v, calls=%d", token, err, calls.Load())
	}
}

func TestCreatePayoutOnboardingUsesHostedStripeURL(t *testing.T) {
	provider, server := stripeFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("account") != "acct_fixture" || r.Form.Get("type") != "account_onboarding" {
			t.Errorf("onboarding form = %v", url.Values(r.Form))
		}
		_, _ = w.Write([]byte(`{"object":"account_link","url":"https://connect.stripe.com/setup/s/test","expires_at":1785600000}`))
	})
	defer server.Close()
	redirect, err := provider.CreatePayoutOnboarding(context.Background(), OnboardingRequest{
		PartyID: 6, AccountToken: "acct_fixture", RefreshURL: "https://www.w8m.com/payout/refresh",
		ReturnURL: "https://www.w8m.com/payout/return", IdempotencyKey: "onboard:6:create",
	})
	if err != nil || !strings.HasPrefix(redirect.URL, "https://connect.stripe.com/") {
		t.Fatalf("onboarding = %#v, %v", redirect, err)
	}
}

func TestProviderErrorDoesNotExposeResponseMessage(t *testing.T) {
	provider, server := stripeFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_request","message":"card 4242424242424242"}}`))
	})
	defer server.Close()
	_, err := provider.CreateFundingCustomer(context.Background(), CustomerRequest{PartyID: 7, IdempotencyKey: "customer:7:create"})
	if err == nil || strings.Contains(err.Error(), "4242") || !strings.Contains(err.Error(), "invalid_request") {
		t.Fatalf("provider error = %v", err)
	}
}

func TestAcceptedButInvalidProviderResponseRemainsRetryable(t *testing.T) {
	provider, server := stripeFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"unexpected_object"}`))
	})
	defer server.Close()
	_, err := provider.CreateFundingCustomer(context.Background(), CustomerRequest{PartyID: 7, IdempotencyKey: "customer:7:create"})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("invalid accepted response error=%v", err)
	}
}
