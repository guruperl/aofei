package hostedpayment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/guruperl/aofei/accounting"
)

func TestMaintenanceBoundaryDoesNotRequireOrExposeProviderSecrets(t *testing.T) {
	config := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: "https://api.stripe.com",
		PublicBaseURL: "https://www.w8m.com", APIKeyEnv: "A02_MISSING_API_KEY",
		WebhookSecretEnv: "A02_MISSING_WEBHOOK_SECRET",
	}.WithDefaults()
	maintenance, err := NewMaintenanceService(config, new(sql.DB))
	if err != nil || maintenance == nil {
		t.Fatalf("maintenance service=%#v err=%v", maintenance, err)
	}
	encoded, err := json.Marshal(&Service{WebhookSecret: []byte("whsec_must_not_serialize"), WebhookSecrets: [][]byte{[]byte("previous_must_not_serialize")}})
	if err != nil || strings.Contains(string(encoded), "serialize") {
		t.Fatalf("service JSON exposed webhook material: %s, %v", encoded, err)
	}
	formatted := fmt.Sprintf("%#v", &Service{WebhookSecret: []byte("whsec_must_not_format")})
	if strings.Contains(formatted, "must_not_format") || !strings.Contains(formatted, "redacted") {
		t.Fatalf("service formatter exposed webhook material: %s", formatted)
	}
}

func TestProviderReplayStopsBeforeStripeCanPruneTheIdempotencyKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT TIMESTAMPDIFF").WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"age"}).AddRow(int64(providerIdempotencyReplayWindow / time.Second)))
	if err := validateProviderReplayWindow(context.Background(), tx, 9); err == nil {
		t.Fatal("provider replay remained enabled at the safety-window boundary")
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProviderKeyModeAcceptsLeastPrivilegeRestrictedKeys(t *testing.T) {
	for _, key := range []string{"sk_test_fixture", "rk_test_fixture"} {
		if !apiKeyMatchesMode(key, false) {
			t.Errorf("sandbox key %q was rejected", key)
		}
	}
	for _, key := range []string{"sk_live_fixture", "rk_live_fixture"} {
		if !apiKeyMatchesMode(key, true) {
			t.Errorf("live key %q was rejected", key)
		}
	}
	if apiKeyMatchesMode("pk_test_publishable", false) || apiKeyMatchesMode("rk_live_wrong_mode", false) {
		t.Fatal("publishable or wrong-mode provider key was accepted")
	}
	config := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: "https://api.stripe.com",
		PublicBaseURL: "https://www.w8m.com", APIKeyEnv: "STRIPE_API_KEY",
		WebhookSecretEnv: "STRIPE_WEBHOOK_SECRET",
	}.WithDefaults()
	if _, err := NewStripeProvider(config, "rk_live_wrong_mode", nil); err == nil {
		t.Fatal("direct provider construction bypassed configured key mode")
	}
}

type countingProvider struct{ calls int }

func (p *countingProvider) CreateFundingCustomer(context.Context, CustomerRequest) (string, error) {
	p.calls++
	return "cus_fixture", nil
}
func (p *countingProvider) CreateFundingCheckout(context.Context, CheckoutRequest) (HostedRedirect, error) {
	p.calls++
	return HostedRedirect{}, nil
}
func (p *countingProvider) ExpireFundingCheckout(context.Context, string, string) error {
	p.calls++
	return nil
}
func (p *countingProvider) CreatePayoutAccount(context.Context, PayoutAccountRequest) (string, error) {
	p.calls++
	return "acct_fixture", nil
}
func (p *countingProvider) CreatePayoutOnboarding(context.Context, OnboardingRequest) (HostedRedirect, error) {
	p.calls++
	return HostedRedirect{}, nil
}
func (p *countingProvider) CreateTransfer(context.Context, TransferRequest) (ProviderObject, error) {
	p.calls++
	return ProviderObject{}, nil
}
func (p *countingProvider) CreateRefund(context.Context, RefundRequest) (ProviderObject, error) {
	p.calls++
	return ProviderObject{}, nil
}
func (p *countingProvider) RetrieveBalanceTransaction(context.Context, string) (BalanceTransaction, error) {
	p.calls++
	return BalanceTransaction{}, nil
}

func TestCrossAccountAndMissingMFADenialsOccurBeforeProviderOrDatabase(t *testing.T) {
	provider := new(countingProvider)
	service := &Service{Provider: provider}
	actor := Actor{
		Role: "adv", ID: "7", Scope: Scope{PartyType: PartyAdvertiser, PartyID: 7},
		Permissions: map[string]bool{PermissionFundingBind: true}, RecentMFA: true,
	}
	if _, err := service.StartFundingCustomer(context.Background(), actor, 8, "bind-8", "wrong account"); err == nil {
		t.Fatal("cross-account binding succeeded")
	}
	actor.RecentMFA = false
	if _, err := service.StartFundingCustomer(context.Background(), actor, 7, "bind-7", "missing MFA"); err == nil {
		t.Fatal("binding without recent MFA succeeded")
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls after authorization denials = %d", provider.calls)
	}
}

func TestProviderMovementRequiresExactCent(t *testing.T) {
	amount, _ := accounting.ParseMoney("1.000001")
	if _, err := moneyToCents(amount); err == nil {
		t.Fatal("sub-cent movement succeeded")
	}
	amount, _ = accounting.ParseMoney("1.010000")
	if cents, err := moneyToCents(amount); err != nil || cents != 101 {
		t.Fatalf("exact cent conversion=%d,%v", cents, err)
	}
}

func TestAllFinancialMutationPermissionsRequireRecentMFA(t *testing.T) {
	actor := Actor{Role: "admin", ID: "2", Permissions: map[string]bool{"*": true}}
	for _, permission := range []string{
		PermissionFundingBind, PermissionPayoutBind, PermissionBindingApprove,
		PermissionCheckoutPropose, PermissionPayoutPropose, PermissionRefundPropose,
		PermissionOperationApprove, PermissionOperationExecute, PermissionOperationCancel,
		PermissionDisputeHandle, PermissionReconcile, PermissionSecretReadiness,
	} {
		if err := authorize(actor, permission, Scope{PartyType: PartyAdvertiser, PartyID: 7}, true); err == nil {
			t.Errorf("permission %s succeeded without recent MFA", permission)
		}
	}
}

func TestFinancialAuditReasonsAreBoundedHumanText(t *testing.T) {
	if err := validateReason("独立复核通过"); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"", "line one\nline two", string([]byte{0xff}), "card 4242 4242 4242 4242", "copy whsec_example_secret_here"} {
		if err := validateReason(reason); err == nil {
			t.Errorf("unsafe reason %q succeeded", reason)
		}
	}
	if err := validateMutation("4242-4242-4242-4242", "safe operator reason"); err == nil {
		t.Fatal("card-like idempotency key succeeded")
	}
}
