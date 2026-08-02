package hostedpayment

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/guruperl/aofei/accounting"
)

type Provider interface {
	CreateFundingCustomer(context.Context, CustomerRequest) (string, error)
	CreateFundingCheckout(context.Context, CheckoutRequest) (HostedRedirect, error)
	ExpireFundingCheckout(context.Context, string, string) error
	CreatePayoutAccount(context.Context, PayoutAccountRequest) (string, error)
	CreatePayoutOnboarding(context.Context, OnboardingRequest) (HostedRedirect, error)
	CreateTransfer(context.Context, TransferRequest) (ProviderObject, error)
	CreateRefund(context.Context, RefundRequest) (ProviderObject, error)
	RetrieveBalanceTransaction(context.Context, string) (BalanceTransaction, error)
}

type CustomerRequest struct {
	PartyID        uint64
	IdempotencyKey string
}

type CheckoutRequest struct {
	OperationID    uint64
	StatementID    uint64
	PartyID        uint64
	Amount         accounting.Money
	CustomerToken  string
	SuccessURL     string
	CancelURL      string
	IdempotencyKey string
}

type PayoutAccountRequest struct {
	PartyID        uint64
	Country        string
	IdempotencyKey string
}

type OnboardingRequest struct {
	PartyID        uint64
	AccountToken   string
	RefreshURL     string
	ReturnURL      string
	IdempotencyKey string
}

type TransferRequest struct {
	OperationID    uint64
	StatementID    uint64
	PartyID        uint64
	Amount         accounting.Money
	AccountToken   string
	IdempotencyKey string
}

type RefundRequest struct {
	OperationID        uint64
	StatementID        uint64
	PartyID            uint64
	Amount             accounting.Money
	PaymentIntentToken string
	IdempotencyKey     string
}

type HostedRedirect struct {
	ObjectToken string
	URL         string
	ExpiresAt   time.Time
}

type ProviderObject struct {
	Token  string
	Status string
}

type BalanceTransaction struct {
	Token       string
	SourceToken string
	Currency    string
	AmountCents int64
	FeeCents    int64
	NetCents    int64
	Status      string
}

var (
	safeIdempotencyKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	safeProviderToken  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,127}$`)
	safeCountry        = regexp.MustCompile(`^[A-Z]{2}$`)
)

func ValidateOpaqueToken(token string, prefixes ...string) error {
	if !safeProviderToken.MatchString(token) {
		return fmt.Errorf("provider token is invalid")
	}
	if len(prefixes) == 0 {
		return nil
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(token, prefix) {
			return nil
		}
	}
	return fmt.Errorf("provider token has an unexpected object type")
}

func moneyToCents(amount accounting.Money) (int64, error) {
	const microDollarsPerCent = int64(10_000)
	value := int64(amount)
	if value <= 0 || value%microDollarsPerCent != 0 {
		return 0, fmt.Errorf("provider movement amount must be a positive exact USD cent")
	}
	return value / microDollarsPerCent, nil
}

func validateIdempotencyKey(key string) error {
	if !safeIdempotencyKey.MatchString(key) {
		return fmt.Errorf("provider idempotency key is invalid")
	}
	return nil
}

func validateHostedRedirect(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("provider returned an invalid hosted URL")
	}
	host := strings.ToLower(u.Hostname())
	if host != "stripe.com" && !strings.HasSuffix(host, ".stripe.com") {
		return fmt.Errorf("provider returned a hosted URL outside stripe.com")
	}
	return nil
}
