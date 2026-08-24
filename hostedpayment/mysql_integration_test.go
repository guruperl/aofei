package hostedpayment

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
)

type integrationProvider struct {
	mu            sync.Mutex
	customerCalls int
	accountCalls  int
	checkoutCalls int
	expireCalls   int
	checkoutOwner uint64
	transferCalls int
	refundCalls   int
}

func (p *integrationProvider) CreateFundingCustomer(context.Context, CustomerRequest) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.customerCalls++
	return "cus_a02_fixture", nil
}
func (p *integrationProvider) CreateFundingCheckout(_ context.Context, input CheckoutRequest) (HostedRedirect, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkoutCalls++
	if p.checkoutOwner == 0 {
		p.checkoutOwner = input.OperationID
	}
	token := "cs_a02_fixture"
	if input.OperationID != p.checkoutOwner {
		token = fmt.Sprintf("cs_a02_fixture_%d", input.OperationID)
	}
	return HostedRedirect{ObjectToken: token, URL: "https://checkout.stripe.com/c/pay/a02", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (p *integrationProvider) ExpireFundingCheckout(context.Context, string, string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireCalls++
	return nil
}
func (p *integrationProvider) CreatePayoutAccount(context.Context, PayoutAccountRequest) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accountCalls++
	if p.accountCalls == 1 {
		return "acct_a02_fixture", nil
	}
	return fmt.Sprintf("acct_a02_fixture_%d", p.accountCalls), nil
}
func (p *integrationProvider) CreatePayoutOnboarding(context.Context, OnboardingRequest) (HostedRedirect, error) {
	return HostedRedirect{ObjectToken: "acct_a02_fixture", URL: "https://connect.stripe.com/setup/s/a02", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (p *integrationProvider) CreateTransfer(_ context.Context, input TransferRequest) (ProviderObject, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transferCalls++
	token := "tr_a02_fixture"
	if p.transferCalls > 1 {
		token = fmt.Sprintf("tr_a02_fixture_%d", input.OperationID)
	}
	return ProviderObject{Token: token, Status: "submitted"}, nil
}
func (p *integrationProvider) CreateRefund(context.Context, RefundRequest) (ProviderObject, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refundCalls++
	return ProviderObject{Token: "re_a02_fixture", Status: "pending"}, nil
}
func (p *integrationProvider) RetrieveBalanceTransaction(_ context.Context, token string) (BalanceTransaction, error) {
	switch token {
	case "txn_a02_fixture":
		return BalanceTransaction{Token: token, SourceToken: "ch_a02_fixture", Currency: "USD", AmountCents: 1000, FeeCents: 30, NetCents: 970, Status: "available"}, nil
	case "txn_a02_payout":
		return BalanceTransaction{Token: token, SourceToken: "tr_a02_fixture", Currency: "USD", AmountCents: -1900, FeeCents: 0, NetCents: -1900, Status: "available"}, nil
	case "txn_a02_refund":
		return BalanceTransaction{Token: token, SourceToken: "re_a02_fixture", Currency: "USD", AmountCents: -200, FeeCents: 0, NetCents: -200, Status: "available"}, nil
	default:
		return BalanceTransaction{}, fmt.Errorf("unexpected balance transaction %q", token)
	}
}

func TestMySQLHostedPaymentLifecycle(t *testing.T) {
	dsn := os.Getenv("AOFEI_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("AOFEI_MYSQL_TEST_DSN is not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	const advertiserID = 990001
	const unboundAdvertiserID = 990002
	const publisherID = 990006
	if _, err := db.ExecContext(ctx, `INSERT INTO adv (adv_id,email,passwd,active,created) VALUES (?,?,?,'Yes',UTC_TIMESTAMP())`, advertiserID, "a02-adv@example.test", "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO adv (adv_id,email,passwd,active,created) VALUES (?,?,?,'Yes',UTC_TIMESTAMP())`, unboundAdvertiserID, "a02-unbound-adv@example.test", "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pub (pub_id,email,passwd,active,created) VALUES (?,?,?,'Yes',UTC_TIMESTAMP())`, publisherID, "a02-pub@example.test", "disabled"); err != nil {
		t.Fatal(err)
	}
	provider := new(integrationProvider)
	config := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: "http://127.0.0.1:4242",
		APIKeyEnv: "STRIPE_API_KEY", WebhookSecretEnv: "STRIPE_WEBHOOK_SECRET",
		PublicBaseURL: "http://127.0.0.1:8080", RequestTimeoutMS: 1000,
		MaxBodyBytes: 16 << 10, WebhookToleranceSeconds: 300, MaxAttempts: 1,
		RetryBaseMS: 25, EventRetentionDays: 400, ReconciliationMaxAgeDays: 90,
	}
	service, err := NewServiceWithProvider(config, db, provider, []byte("whsec_a02_fixture_long"))
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) }
	adv := Actor{Role: "adv", ID: fmt.Sprint(advertiserID), Scope: Scope{PartyType: PartyAdvertiser, PartyID: advertiserID}, Permissions: map[string]bool{
		PermissionRead: true, PermissionFundingBind: true, PermissionCheckoutPropose: true,
		PermissionOperationExecute: true, PermissionOperationCancel: true,
	}, RecentMFA: true}
	unboundAdv := Actor{Role: "adv", ID: fmt.Sprint(unboundAdvertiserID), Scope: Scope{PartyType: PartyAdvertiser, PartyID: unboundAdvertiserID}, Permissions: map[string]bool{
		PermissionRead: true, PermissionCheckoutPropose: true, PermissionOperationExecute: true,
	}, RecentMFA: true}
	pub := Actor{Role: "pub", ID: fmt.Sprint(publisherID), Scope: Scope{PartyType: PartyPublisher, PartyID: publisherID}, Permissions: map[string]bool{
		PermissionRead: true, PermissionPayoutBind: true,
	}, RecentMFA: true}
	maker := Actor{Role: "admin", ID: "9001", Permissions: map[string]bool{"*": true}, RecentMFA: true}
	checker := Actor{Role: "admin", ID: "9002", Permissions: map[string]bool{"*": true}, RecentMFA: true}
	settler := Actor{Role: "admin", ID: "9003", Permissions: map[string]bool{"*": true}, RecentMFA: true}

	fundingBinding, err := service.StartFundingCustomer(ctx, adv, advertiserID, "a02-funding-customer", "create hosted funding customer")
	if err != nil || fundingBinding.Status != BindingReady {
		t.Fatalf("funding binding=%#v err=%v", fundingBinding, err)
	}
	replayedBinding, err := service.StartFundingCustomer(ctx, adv, advertiserID, "a02-funding-customer", "create hosted funding customer")
	if err != nil || replayedBinding.ID != fundingBinding.ID || provider.customerCalls != 1 {
		t.Fatalf("idempotent funding binding=%#v err=%v provider calls=%d", replayedBinding, err, provider.customerCalls)
	}
	if err := service.ApproveBinding(ctx, checker, fundingBinding.ID, fundingBinding.Version, "independent customer-token review"); err != nil {
		t.Fatal(err)
	}
	unboundStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, unboundAdvertiserID, "1.000000", "a02-unbound-statement")
	unboundAmount, _ := accounting.ParseMoney("1.000000")
	unbound, err := service.ProposeOperation(ctx, unboundAdv, ProposeOperationInput{RequestKey: "a02-unbound-funding", Kind: OperationFunding, StatementID: unboundStatement, PartyID: unboundAdvertiserID, Amount: unboundAmount, Reason: "prove funding requires an approved hosted customer"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, unbound.ID, unbound.Version, "independently review unbound funding fixture"); err != nil {
		t.Fatal(err)
	}
	unbound, _ = service.Operation(ctx, unbound.ID)
	if _, err := service.ExecuteOperation(ctx, unboundAdv, unbound.ID, unbound.Version); err == nil || provider.checkoutCalls != 0 {
		t.Fatalf("funding without approved customer binding err=%v provider calls=%d", err, provider.checkoutCalls)
	}

	replayStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, advertiserID, "1.000000", "a02-expired-replay-statement")
	replayResult, err := db.ExecContext(ctx, `
INSERT INTO hosted_operation (request_key,provider,operation_kind,binding_id,statement_id,party_type,party_id,
 amount,currency,status,version,created_by,approved_by,executed_by,reason,failure_code,
 attempt_count,created_at,updated_at)
VALUES ('a02-expired-approved-replay','stripe','Funding',?,?,'advertiser',?,
 '1.000000','USD','Approved',2,'adv:990001','admin:9002','adv:990001',
 'exercise expired uncertain provider response','provider_unavailable',1,
 UTC_TIMESTAMP()-INTERVAL 24 HOUR,UTC_TIMESTAMP())`, fundingBinding.ID, replayStatement, advertiserID)
	if err != nil {
		t.Fatal(err)
	}
	replayIDRaw, _ := replayResult.LastInsertId()
	if _, err := db.ExecContext(ctx, `
INSERT INTO hosted_audit (object_type,object_id,actor,event,prior_state,new_state,reason,created_at)
VALUES ('Operation',?,'adv:990001','ProviderSubmissionStarted','Approved','Submitting',
 'exercise expired uncertain provider response',UTC_TIMESTAMP()-INTERVAL 24 HOUR)`, replayIDRaw); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecuteOperation(ctx, adv, uint64(replayIDRaw), 2); err == nil || provider.checkoutCalls != 0 {
		t.Fatalf("expired provider replay err=%v provider calls=%d", err, provider.checkoutCalls)
	}
	if err := service.CancelOperation(ctx, adv, uint64(replayIDRaw), 2, "must not release an uncertain provider movement"); err == nil {
		t.Fatal("uncertain provider submission was locally canceled")
	}
	replayOperation, _ := service.Operation(ctx, uint64(replayIDRaw))
	if replayOperation.Status != OperationApproved {
		t.Fatalf("uncertain provider submission released capacity=%#v", replayOperation)
	}
	payoutBinding, redirect, err := service.StartPayoutOnboarding(ctx, pub, publisherID, "US", "a02-payout-account", "start hosted publisher onboarding")
	if err != nil || !strings.HasPrefix(redirect.URL, "https://connect.stripe.com/") {
		t.Fatalf("payout onboarding=%#v %#v err=%v", payoutBinding, redirect, err)
	}
	if _, _, err := service.StartPayoutOnboarding(ctx, pub, publisherID, "CA", "a02-payout-account", "start hosted publisher onboarding"); err == nil || provider.accountCalls != 1 {
		t.Fatalf("changed-country binding replay err=%v provider calls=%d", err, provider.accountCalls)
	}
	pendingAccountEvent := fmt.Sprintf(`{"id":"evt_a02_account_pending","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture","data":{"object":{"id":"acct_a02_fixture","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"pending"}}}}`, service.now().Unix())
	if result, err := service.IngestWebhook(ctx, []byte(pendingAccountEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("pending account event=%#v err=%v", result, err)
	}
	payoutBinding, err = service.Binding(ctx, payoutBinding.ID)
	if err != nil || payoutBinding.Status != BindingProposed || payoutBinding.ProviderReady {
		t.Fatalf("prematurely ready payout binding=%#v err=%v", payoutBinding, err)
	}
	accountEvent := fmt.Sprintf(`{"id":"evt_a02_account","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture","data":{"object":{"id":"acct_a02_fixture","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"active"}}}}`, service.now().Add(time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(accountEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("account event=%#v err=%v", result, err)
	}
	payoutBinding, err = service.Binding(ctx, payoutBinding.ID)
	if err != nil || payoutBinding.Status != BindingReady {
		t.Fatalf("ready payout binding=%#v err=%v", payoutBinding, err)
	}
	staleAccountEvent := fmt.Sprintf(`{"id":"evt_a02_account_stale","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture","data":{"object":{"id":"acct_a02_fixture","object":"account","payouts_enabled":false,"details_submitted":true,"capabilities":{"transfers":"inactive"}}}}`, service.now().Unix())
	if result, err := service.IngestWebhook(ctx, []byte(staleAccountEvent)); err != nil || result.Disposition != "Ignored" {
		t.Fatalf("stale account event=%#v err=%v", result, err)
	}
	payoutBinding, err = service.Binding(ctx, payoutBinding.ID)
	if err != nil || payoutBinding.Status != BindingReady || !payoutBinding.ProviderReady {
		t.Fatalf("stale event regressed payout binding=%#v err=%v", payoutBinding, err)
	}
	if _, err := service.RefreshPayoutOnboarding(ctx, pub, payoutBinding.ID, payoutBinding.Version, "a02-ready-refresh", "ready binding must await review"); err == nil {
		t.Fatal("provider-ready binding incorrectly reopened onboarding")
	}
	if err := service.ApproveBinding(ctx, checker, payoutBinding.ID, payoutBinding.Version, "independent payout destination review"); err != nil {
		t.Fatal(err)
	}
	replacementBinding, _, err := service.StartPayoutOnboarding(ctx, pub, publisherID, "US", "a02-payout-account-replacement", "replace hosted publisher payout account")
	if err != nil {
		t.Fatal(err)
	}
	replacementEvent := fmt.Sprintf(`{"id":"evt_a02_account_replacement","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture_2","data":{"object":{"id":"acct_a02_fixture_2","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"active"}}}}`, service.now().Add(2*time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(replacementEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("replacement account event=%#v err=%v", result, err)
	}
	replacementBinding, _ = service.Binding(ctx, replacementBinding.ID)
	if err := service.ApproveBinding(ctx, checker, replacementBinding.ID, replacementBinding.Version, "independently approve replacement payout destination"); err != nil {
		t.Fatal(err)
	}
	payoutBinding, _ = service.Binding(ctx, payoutBinding.ID)
	if payoutBinding.Status != BindingRevoked || payoutBinding.RevokedBy.String != actorName(checker) {
		t.Fatalf("prior payout binding was not replaced=%#v", payoutBinding)
	}
	replayedOldAccount := fmt.Sprintf(`{"id":"evt_a02_account_old_ready","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture","data":{"object":{"id":"acct_a02_fixture","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"active"}}}}`, service.now().Add(3*time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(replayedOldAccount)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("old account readiness event=%#v err=%v", result, err)
	}
	payoutBinding, _ = service.Binding(ctx, payoutBinding.ID)
	if payoutBinding.Status != BindingRevoked {
		t.Fatalf("replacement-revoked payout binding was resurrected=%#v", payoutBinding)
	}
	if _, err := service.RefreshPayoutOnboarding(ctx, pub, payoutBinding.ID, payoutBinding.Version, "a02-replaced-refresh", "replaced binding must remain revoked"); err == nil {
		t.Fatal("human-replaced binding incorrectly reopened onboarding")
	}
	unknownPayoutEvent := fmt.Sprintf(`{"id":"evt_a02_unknown_payout","type":"payout.failed","created":%d,"livemode":false,"account":"acct_a02_unknown","data":{"object":{"id":"po_a02_unknown","object":"payout","currency":"usd","amount":500}}}`, service.now().Unix())
	if result, err := service.IngestWebhook(ctx, []byte(unknownPayoutEvent)); err != nil || result.Disposition != "Unresolved" {
		t.Fatalf("unknown payout event=%#v err=%v", result, err)
	}
	globalExceptions, err := service.ListReconciliations(ctx, settler, Scope{})
	if err != nil || len(globalExceptions) == 0 {
		t.Fatalf("global payout exception list=%#v err=%v", globalExceptions, err)
	}
	if err := service.ResolveReconciliation(ctx, checker, globalExceptions[0].ID, "unknown connected payout independently reviewed"); err != nil {
		t.Fatalf("resolve unowned payout exception: %v", err)
	}

	fundingStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, advertiserID, "10.000000", "a02-funding-statement")
	payoutStatement := insertConfirmedStatement(t, ctx, db, PartyPublisher, publisherID, "20.000000", "a02-payout-statement")
	fundingAmount, _ := accounting.ParseMoney("10.000000")
	funding, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-funding", Kind: OperationFunding, StatementID: fundingStatement, PartyID: advertiserID, Amount: fundingAmount, Reason: "fund confirmed statement"})
	if err != nil {
		t.Fatal(err)
	}
	replayedFunding, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-funding", Kind: OperationFunding, StatementID: fundingStatement, PartyID: advertiserID, Amount: fundingAmount, Reason: "fund confirmed statement"})
	if err != nil || replayedFunding.ID != funding.ID {
		t.Fatalf("idempotent funding replay=%#v err=%v", replayedFunding, err)
	}
	if err := service.ApproveOperation(ctx, checker, funding.ID, funding.Version, "independent funding review"); err != nil {
		t.Fatal(err)
	}
	funding, err = service.Operation(ctx, funding.ID)
	if err != nil {
		t.Fatal(err)
	}
	executed, err := service.ExecuteOperation(ctx, adv, funding.ID, funding.Version)
	if err != nil || executed.Redirect == nil || executed.Operation.Status != OperationSubmitted {
		t.Fatalf("funding execute=%#v err=%v", executed, err)
	}
	reopened, err := service.ExecuteOperation(ctx, adv, funding.ID, executed.Operation.Version)
	if err != nil || reopened.Redirect == nil || provider.checkoutCalls != 2 {
		t.Fatalf("reopen submitted checkout=%#v err=%v calls=%d", reopened, err, provider.checkoutCalls)
	}
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET status='Held',updated_at=UTC_TIMESTAMP() WHERE statement_id=?`, fundingStatement); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecuteOperation(ctx, adv, funding.ID, reopened.Operation.Version); err == nil || provider.checkoutCalls != 2 {
		t.Fatalf("Held checkout reopened err=%v calls=%d", err, provider.checkoutCalls)
	}
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET status='Confirmed',updated_at=UTC_TIMESTAMP() WHERE statement_id=?`, fundingStatement); err != nil {
		t.Fatal(err)
	}
	failedAttemptEvent := fmt.Sprintf(`{"id":"evt_a02_attempt_failed","type":"payment_intent.payment_failed","created":%d,"livemode":false,"data":{"object":{"id":"pi_a02_fixture","object":"payment_intent","status":"requires_payment_method","currency":"usd","amount":1000,"metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), funding.ID, fundingStatement, advertiserID)
	if result, err := service.IngestWebhook(ctx, []byte(failedAttemptEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("failed payment attempt=%#v err=%v", result, err)
	}
	funding, _ = service.Operation(ctx, funding.ID)
	if funding.Status != OperationSubmitted || funding.FailureCode.String != "payment_attempt_failed" {
		t.Fatalf("failed attempt released or hid checkout state=%#v", funding)
	}
	retryDollar, _ := accounting.ParseMoney("1.000000")
	if _, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-funding-after-attempt", Kind: OperationFunding, StatementID: fundingStatement, PartyID: advertiserID, Amount: retryDollar, Reason: "must remain reserved while checkout can retry"}); err == nil {
		t.Fatal("retryable payment attempt failure released statement capacity")
	}
	fundingEvent := fmt.Sprintf(`{"id":"evt_a02_funding","type":"payment_intent.succeeded","created":%d,"livemode":false,"data":{"object":{"id":"pi_a02_fixture","object":"payment_intent","status":"succeeded","currency":"usd","amount_received":1000,"metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), funding.ID, fundingStatement, advertiserID)
	if result, err := service.IngestWebhook(ctx, []byte(fundingEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("funding event=%#v err=%v", result, err)
	}
	if result, err := service.IngestWebhook(ctx, []byte(fundingEvent)); err != nil || !result.Duplicate {
		t.Fatalf("funding duplicate=%#v err=%v", result, err)
	}
	disputeEvent := fmt.Sprintf(`{"id":"evt_a02_dispute","type":"charge.dispute.created","created":%d,"livemode":false,"data":{"object":{"id":"du_a02_fixture","object":"dispute","status":"needs_response","currency":"usd","amount":1000,"charge":"ch_a02_fixture"}}}`, service.now().Add(2*time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(disputeEvent)); err != nil || result.Disposition != "Unresolved" || !result.Retryable {
		t.Fatalf("early dispute evidence=%#v err=%v", result, err)
	}
	chargeEvent := fmt.Sprintf(`{"id":"evt_a02_charge","type":"charge.succeeded","created":%d,"livemode":false,"data":{"object":{"id":"ch_a02_fixture","object":"charge","currency":"usd","amount":1000,"payment_intent":"pi_a02_fixture","balance_transaction":"txn_a02_fixture","metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Add(time.Second).Unix(), funding.ID, fundingStatement, advertiserID)
	if result, err := service.IngestWebhook(ctx, []byte(chargeEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("charge evidence=%#v err=%v", result, err)
	}
	if result, err := service.IngestWebhook(ctx, []byte(disputeEvent)); err != nil || result.Disposition != "Applied" || !result.Reprocessed {
		t.Fatalf("reprocessed dispute evidence=%#v err=%v", result, err)
	}
	if result, err := service.IngestWebhook(ctx, []byte(disputeEvent)); err != nil || !result.Duplicate {
		t.Fatalf("reprocessed dispute duplicate=%#v err=%v", result, err)
	}
	funding, _ = service.Operation(ctx, funding.ID)
	if funding.Status != OperationDisputed {
		t.Fatalf("dispute did not hold funding operation=%#v", funding)
	}
	disputeWonEvent := fmt.Sprintf(`{"id":"evt_a02_dispute_won","type":"charge.dispute.closed","created":%d,"livemode":false,"data":{"object":{"id":"du_a02_fixture","object":"dispute","status":"won","currency":"usd","amount":1000,"charge":"ch_a02_fixture"}}}`, service.now().Add(3*time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(disputeWonEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("won dispute evidence=%#v err=%v", result, err)
	}
	funding, _ = service.Operation(ctx, funding.ID)
	if funding.Status != OperationSucceeded {
		t.Fatalf("won dispute did not restore succeeded state=%#v", funding)
	}
	// A provider webhook can win the race with the request that initiated the
	// provider call. Recording the returned Checkout object must preserve the
	// terminal provider state instead of reporting a false ownership conflict.
	fastStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, advertiserID, "3.000000", "a02-fast-statement")
	fastAmount, _ := accounting.ParseMoney("3.000000")
	fast, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-fast", Kind: OperationFunding, StatementID: fastStatement, PartyID: advertiserID, Amount: fastAmount, Reason: "exercise webhook response race"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, fast.ID, fast.Version, "independent fast-path review"); err != nil {
		t.Fatal(err)
	}
	fast, _ = service.Operation(ctx, fast.ID)
	claimed, _, _, err := service.claimExecution(ctx, adv, fast.ID, fast.Version)
	if err != nil {
		t.Fatal(err)
	}
	fastEvent := fmt.Sprintf(`{"id":"evt_a02_fast","type":"payment_intent.succeeded","created":%d,"livemode":false,"data":{"object":{"id":"pi_a02_fast","object":"payment_intent","status":"succeeded","currency":"usd","amount_received":300,"latest_charge":"ch_a02_fast","metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), fast.ID, fastStatement, advertiserID)
	if result, err := service.IngestWebhook(ctx, []byte(fastEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("fast event=%#v err=%v", result, err)
	}
	if err := service.finishExecution(ctx, adv, claimed, ProviderObject{Token: "cs_a02_fast", Status: "submitted"}); err != nil {
		t.Fatalf("finish after fast webhook: %v", err)
	}
	fast, _ = service.Operation(ctx, fast.ID)
	if fast.Status != OperationSucceeded || fast.CurrentObjectToken.String != "cs_a02_fast" {
		t.Fatalf("fast operation=%#v", fast)
	}

	concurrentStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, advertiserID, "2.000000", "a02-concurrent-event-statement")
	concurrentAmount, _ := accounting.ParseMoney("2.000000")
	concurrentOperation, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-concurrent-event", Kind: OperationFunding, StatementID: concurrentStatement, PartyID: advertiserID, Amount: concurrentAmount, Reason: "exercise concurrent provider event claim"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, concurrentOperation.ID, concurrentOperation.Version, "independently review concurrent event fixture"); err != nil {
		t.Fatal(err)
	}
	concurrentOperation, _ = service.Operation(ctx, concurrentOperation.ID)
	if _, err := service.ExecuteOperation(ctx, adv, concurrentOperation.ID, concurrentOperation.Version); err != nil {
		t.Fatal(err)
	}
	concurrentPayload := []byte(fmt.Sprintf(`{"id":"evt_a02_concurrent","type":"payment_intent.succeeded","created":%d,"livemode":false,"data":{"object":{"id":"pi_a02_concurrent","object":"payment_intent","status":"succeeded","currency":"usd","amount_received":200,"metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), concurrentOperation.ID, concurrentStatement, advertiserID))
	type concurrentResult struct {
		result WebhookResult
		err    error
	}
	results := make(chan concurrentResult, 8)
	var eventWG sync.WaitGroup
	for i := 0; i < cap(results); i++ {
		eventWG.Add(1)
		go func() {
			defer eventWG.Done()
			result, err := service.IngestWebhook(ctx, concurrentPayload)
			results <- concurrentResult{result: result, err: err}
		}()
	}
	eventWG.Wait()
	close(results)
	var appliedEvents, duplicateEvents int
	for outcome := range results {
		if outcome.err != nil {
			t.Fatalf("concurrent event ingestion: %v", outcome.err)
		}
		if outcome.result.Duplicate {
			duplicateEvents++
		} else if outcome.result.Disposition == "Applied" {
			appliedEvents++
		}
	}
	if appliedEvents != 1 || duplicateEvents != 7 {
		t.Fatalf("concurrent event claims applied=%d duplicate=%d", appliedEvents, duplicateEvents)
	}

	payoutAmount, _ := accounting.ParseMoney("20.000000")
	payout, err := service.ProposeOperation(ctx, maker, ProposeOperationInput{RequestKey: "a02-payout", Kind: OperationPayout, StatementID: payoutStatement, PartyID: publisherID, Amount: payoutAmount, Reason: "schedule confirmed publisher payout"})
	if err != nil {
		t.Fatal(err)
	}
	oneDollar, _ := accounting.ParseMoney("1.000000")
	if _, err := service.ProposeOperation(ctx, maker, ProposeOperationInput{RequestKey: "a02-payout-over-capacity", Kind: OperationPayout, StatementID: payoutStatement, PartyID: publisherID, Amount: oneDollar, Reason: "must not exceed statement"}); err == nil {
		t.Fatal("aggregate payout operations exceeded the statement total")
	}
	if err := service.ApproveOperation(ctx, checker, payout.ID, payout.Version, "independent payout review"); err != nil {
		t.Fatal(err)
	}
	payout, _ = service.Operation(ctx, payout.ID)
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET status='Held',updated_at=UTC_TIMESTAMP() WHERE statement_id=?`, payoutStatement); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecuteOperation(ctx, maker, payout.ID, payout.Version); err == nil || provider.transferCalls != 0 {
		t.Fatalf("held payout error=%v transfer calls=%d", err, provider.transferCalls)
	}
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET status='Confirmed',updated_at=UTC_TIMESTAMP() WHERE statement_id=?`, payoutStatement); err != nil {
		t.Fatal(err)
	}
	executed, err = service.ExecuteOperation(ctx, maker, payout.ID, payout.Version)
	if err != nil || executed.Operation.Status != OperationSubmitted || provider.transferCalls != 1 {
		t.Fatalf("payout execute=%#v err=%v calls=%d", executed, err, provider.transferCalls)
	}
	if !executed.Operation.BindingID.Valid || uint64(executed.Operation.BindingID.Int64) != replacementBinding.ID {
		t.Fatalf("payout did not freeze its approved destination binding=%#v", executed.Operation)
	}
	payoutEvent := fmt.Sprintf(`{"id":"evt_a02_payout","type":"transfer.created","created":%d,"livemode":false,"data":{"object":{"id":"tr_a02_fixture","object":"transfer","currency":"usd","amount":2000,"balance_transaction":"txn_a02_payout","metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), payout.ID, payoutStatement, publisherID)
	if _, err := service.IngestWebhook(ctx, []byte(payoutEvent)); err != nil {
		t.Fatal(err)
	}
	olderFailure := fmt.Sprintf(`{"id":"evt_a02_payout_old","type":"transfer.reversed","created":%d,"livemode":false,"data":{"object":{"id":"tr_a02_fixture","object":"transfer","currency":"usd","amount":2000,"amount_reversed":2000,"metadata":{"aofei_operation_id":"%d"}}}}`, service.now().Add(-time.Second).Unix(), payout.ID)
	if result, err := service.IngestWebhook(ctx, []byte(olderFailure)); err != nil || result.Disposition != "Ignored" {
		t.Fatalf("reordered failure=%#v err=%v", result, err)
	}
	payout, _ = service.Operation(ctx, payout.ID)
	if payout.Status != OperationSucceeded {
		t.Fatalf("payout regressed to %s", payout.Status)
	}
	thirdBinding, _, err := service.StartPayoutOnboarding(ctx, pub, publisherID, "US", "a02-payout-account-third", "exercise immutable selected payout destination")
	if err != nil {
		t.Fatal(err)
	}
	thirdEvent := fmt.Sprintf(`{"id":"evt_a02_account_third","type":"account.updated","created":%d,"livemode":false,"account":"acct_a02_fixture_3","data":{"object":{"id":"acct_a02_fixture_3","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"active"}}}}`, service.now().Add(4*time.Second).Unix())
	if result, err := service.IngestWebhook(ctx, []byte(thirdEvent)); err != nil || result.Disposition != "Applied" {
		t.Fatalf("third account event=%#v err=%v", result, err)
	}
	thirdBinding, _ = service.Binding(ctx, thirdBinding.ID)
	if err := service.ApproveBinding(ctx, checker, thirdBinding.ID, thirdBinding.Version, "independently approve third payout destination"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	selectedID, selectedToken, err := executionBinding(ctx, tx, payout)
	_ = tx.Rollback()
	if err != nil || !selectedID.Valid || uint64(selectedID.Int64) != replacementBinding.ID || selectedToken != replacementBinding.ProviderToken {
		t.Fatalf("provider retry changed frozen destination id=%#v token=%q err=%v", selectedID, selectedToken, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE hosted_operation SET binding_id=? WHERE operation_id=?`, thirdBinding.ID, payout.ID); err == nil {
		t.Fatal("database allowed a selected provider binding to be replaced")
	}

	refundAmount, _ := accounting.ParseMoney("2.000000")
	refund, err := service.ProposeOperation(ctx, maker, ProposeOperationInput{RequestKey: "a02-refund", Kind: OperationRefund, ParentOperationID: funding.ID, StatementID: fundingStatement, PartyID: advertiserID, Amount: refundAmount, Reason: "approved advertiser refund"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, refund.ID, refund.Version, "independent refund review"); err != nil {
		t.Fatal(err)
	}
	refund, _ = service.Operation(ctx, refund.ID)
	if _, err := service.ExecuteOperation(ctx, maker, refund.ID, refund.Version); err != nil || provider.refundCalls != 1 {
		t.Fatalf("refund execute err=%v calls=%d", err, provider.refundCalls)
	}
	refundEvent := fmt.Sprintf(`{"id":"evt_a02_refund","type":"refund.updated","created":%d,"livemode":false,"data":{"object":{"id":"re_a02_fixture","object":"refund","status":"succeeded","currency":"usd","amount":200,"payment_intent":"pi_a02_fixture","balance_transaction":"txn_a02_refund","metadata":{"aofei_operation_id":"%d","aofei_statement_id":"%d","aofei_party_id":"%d"}}}}`, service.now().Unix(), refund.ID, fundingStatement, advertiserID)
	if _, err := service.IngestWebhook(ctx, []byte(refundEvent)); err != nil {
		t.Fatal(err)
	}

	summary, err := service.ReconcileOperation(ctx, settler, funding.ID, "daily provider reconciliation")
	if err != nil || !summary.Matched || summary.Amount != fundingAmount {
		t.Fatalf("reconcile=%#v err=%v", summary, err)
	}
	// Both the opened dispute and its terminal result remain explicit immutable
	// exceptions until a separately authorized operator reviews each fact.
	reconciliations, err := service.ListReconciliations(ctx, settler, Scope{})
	if err != nil {
		t.Fatal(err)
	}
	var disputeExceptions int
	for _, item := range reconciliations {
		if item.OperationID.Valid && uint64(item.OperationID.Int64) == funding.ID &&
			(item.Category == "Dispute" || item.Category == "Chargeback") && item.Status == "Unresolved" {
			if err := service.ResolveReconciliation(ctx, checker, item.ID, "provider dispute independently reviewed"); err != nil {
				t.Fatalf("resolve dispute reconciliation %d: %v", item.ID, err)
			}
			disputeExceptions++
		}
	}
	if disputeExceptions != 2 {
		t.Fatalf("resolved dispute exception count=%d, want 2", disputeExceptions)
	}
	refundSummary, err := service.ReconcileOperation(ctx, settler, refund.ID, "refund balance reconciliation")
	if err != nil || !refundSummary.Matched || absoluteMoney(refundSummary.Amount) != refundAmount {
		t.Fatalf("refund reconcile=%#v err=%v", refundSummary, err)
	}
	if err := (accounting.Service{DB: db}).Transition(ctx, accounting.TransitionInput{StatementID: fundingStatement, To: accounting.StatusSettled, Actor: "admin:9003", Reason: "provider funding reconciled", ExternalRef: "invoice:stripe-a02"}); err != nil {
		t.Fatal(err)
	}
	lateRefundAmount, _ := accounting.ParseMoney("1.000000")
	lateRefund, err := service.ProposeOperation(ctx, maker, ProposeOperationInput{RequestKey: "a02-late-refund", Kind: OperationRefund, ParentOperationID: funding.ID, StatementID: fundingStatement, PartyID: advertiserID, Amount: lateRefundAmount, Reason: "refund after invoice settlement"})
	if err != nil {
		t.Fatalf("post-settlement refund proposal: %v", err)
	}
	if err := service.CancelOperation(ctx, maker, lateRefund.ID, lateRefund.Version, "test-only refund proposal cleanup"); err != nil {
		t.Fatal(err)
	}

	payoutSummary, err := service.ReconcileOperation(ctx, settler, payout.ID, "payout mismatch reconciliation")
	if err != nil || payoutSummary.Matched || payoutSummary.Exceptions == 0 {
		t.Fatalf("payout mismatch reconcile=%#v err=%v", payoutSummary, err)
	}
	reconciliations, err = service.ListReconciliations(ctx, settler, Scope{})
	if err != nil {
		t.Fatal(err)
	}
	var unresolvedID uint64
	for _, item := range reconciliations {
		if item.OperationID.Valid && uint64(item.OperationID.Int64) == payout.ID && item.Status == "Unresolved" {
			unresolvedID = item.ID
			break
		}
	}
	if unresolvedID == 0 {
		t.Fatal("missing payout reconciliation exception")
	}
	if err := service.ResolveReconciliation(ctx, checker, unresolvedID, "provider dashboard evidence independently reviewed"); err != nil {
		t.Fatalf("resolve reconciliation: %v", err)
	}

	cancelStatement := insertConfirmedStatement(t, ctx, db, PartyAdvertiser, advertiserID, "5.000000", "a02-cancel-statement")
	cancelAmount, _ := accounting.ParseMoney("5.000000")
	cancelOp, err := service.ProposeOperation(ctx, adv, ProposeOperationInput{RequestKey: "a02-cancel", Kind: OperationFunding, StatementID: cancelStatement, PartyID: advertiserID, Amount: cancelAmount, Reason: "exercise cancel recovery"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, cancelOp.ID, cancelOp.Version, "independent cancellation review"); err != nil {
		t.Fatal(err)
	}
	cancelOp, _ = service.Operation(ctx, cancelOp.ID)
	if _, err := service.ExecuteOperation(ctx, adv, cancelOp.ID, cancelOp.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE hosted_operation SET status='Canceling',version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=?`, cancelOp.ID); err != nil {
		t.Fatal(err)
	}
	cancelOp, _ = service.Operation(ctx, cancelOp.ID)
	if err := service.CancelOperation(ctx, adv, cancelOp.ID, cancelOp.Version, "resume interrupted checkout cancellation"); err != nil {
		t.Fatalf("recover cancellation: %v", err)
	}
	cancelOp, _ = service.Operation(ctx, cancelOp.ID)
	if cancelOp.Status != OperationCanceled || provider.expireCalls != 1 {
		t.Fatalf("canceled operation=%#v expire calls=%d", cancelOp, provider.expireCalls)
	}

	recoveryStatement := insertConfirmedStatement(t, ctx, db, PartyPublisher, publisherID, "4.000000", "a02-recovery-statement")
	recoveryAmount, _ := accounting.ParseMoney("4.000000")
	recovery, err := service.ProposeOperation(ctx, maker, ProposeOperationInput{RequestKey: "a02-recovery", Kind: OperationPayout, StatementID: recoveryStatement, PartyID: publisherID, Amount: recoveryAmount, Reason: "exercise stale provider recovery"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveOperation(ctx, checker, recovery.ID, recovery.Version, "independent recovery review"); err != nil {
		t.Fatal(err)
	}
	recovery, _ = service.Operation(ctx, recovery.ID)
	if _, _, _, err := service.claimExecution(ctx, maker, recovery.ID, recovery.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE hosted_operation SET updated_at=? WHERE operation_id=?`, service.now().Add(-5*time.Minute), recovery.ID); err != nil {
		t.Fatal(err)
	}
	recovery, _ = service.Operation(ctx, recovery.ID)
	if _, err := service.ExecuteOperation(ctx, settler, recovery.ID, recovery.Version); err != nil {
		t.Fatalf("recover stale submission: %v", err)
	}

	// A size-one pool proves the restricted retention session is scrubbed before
	// its connection is returned for ordinary application work.
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, `
INSERT INTO hosted_event (provider,provider_event_token,event_type,object_kind,
 provider_object_token,provider_created_at,payload_sha256,disposition,received_at)
VALUES ('stripe','evt_a02_prune','future.object.changed','future.object','obj_a02_prune',
 UTC_TIMESTAMP()-INTERVAL 401 DAY,UNHEX(SHA2('a02-prune',256)),'Ignored',UTC_TIMESTAMP()-INTERVAL 401 DAY)`); err != nil {
		t.Fatal(err)
	}
	maintenance, err := NewMaintenanceService(config, db)
	if err != nil {
		t.Fatal(err)
	}
	maintenanceActor := Actor{Role: "admin", ID: "unix-uid:1001", Permissions: map[string]bool{PermissionRetentionPrune: true}}
	deleted, err := maintenance.PruneEvents(ctx, maintenanceActor, 100, "approved disposable retention schedule")
	if err != nil || deleted != 1 {
		t.Fatalf("prune hosted events deleted=%d err=%v", deleted, err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO hosted_event (provider,provider_event_token,event_type,object_kind,
 provider_object_token,provider_created_at,payload_sha256,disposition,received_at)
VALUES ('stripe','evt_a02_retention_guard','future.object.changed','future.object','obj_a02_retention_guard',
 UTC_TIMESTAMP()-INTERVAL 401 DAY,UNHEX(SHA2('a02-retention-guard',256)),'Ignored',UTC_TIMESTAMP()-INTERVAL 401 DAY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM hosted_event WHERE provider_event_token='evt_a02_retention_guard'`); err == nil {
		t.Fatal("ordinary pooled connection retained hosted-event deletion privilege")
	}
	var retentionGuard int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hosted_event WHERE provider_event_token='evt_a02_retention_guard'`).Scan(&retentionGuard); err != nil || retentionGuard != 1 {
		t.Fatalf("retention guard rows=%d err=%v", retentionGuard, err)
	}
	health, err := service.OperationalHealth(ctx)
	if err != nil || health.UnresolvedExceptions == 0 || health.WebhookEvents24Hours == 0 {
		t.Fatalf("hosted payment health=%#v err=%v", health, err)
	}

	bindings, err := service.ListBindings(ctx, pub, Scope{PartyType: PartyPublisher, PartyID: publisherID})
	if err != nil || len(bindings) != 3 || bindings[0].PartyID != publisherID || bindings[1].PartyID != publisherID || bindings[2].PartyID != publisherID {
		t.Fatalf("scoped bindings=%#v err=%v", bindings, err)
	}
	if _, err := service.ListOperations(ctx, adv, Scope{PartyType: PartyPublisher, PartyID: publisherID}); err == nil {
		t.Fatal("advertiser read publisher payment operations")
	}
}

func insertConfirmedStatement(t *testing.T, ctx context.Context, db *sql.DB, party PartyType, partyID uint64, amount, requestKey string) uint64 {
	t.Helper()
	result, err := db.ExecContext(ctx, `
INSERT INTO acct_statement (request_key,party_type,party_id,cadence,period_start,period_end,
 currency,source_amount,adjustment_amount,total_amount,status,created_by,confirmed_by,created_at,updated_at)
VALUES (?,?,?,'daily','2026-07-31','2026-07-31','USD',?,0,?,'Confirmed','admin:maker','admin:checker',UTC_TIMESTAMP(),UTC_TIMESTAMP())`, requestKey, party, partyID, amount, amount)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return uint64(id)
}
