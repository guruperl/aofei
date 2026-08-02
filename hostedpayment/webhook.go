package hostedpayment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
)

type WebhookResult struct {
	Duplicate   bool
	Reprocessed bool
	Retryable   bool
	Disposition string
	EventID     uint64
}

type stripeEvent struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Created    int64  `json:"created"`
	LiveMode   bool   `json:"livemode"`
	APIVersion string `json:"api_version"`
	Account    string `json:"account"`
	Data       struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeObject struct {
	ID               string            `json:"id"`
	Object           string            `json:"object"`
	Status           string            `json:"status"`
	PaymentStatus    string            `json:"payment_status"`
	Currency         string            `json:"currency"`
	Amount           int64             `json:"amount"`
	AmountTotal      int64             `json:"amount_total"`
	AmountReceived   int64             `json:"amount_received"`
	AmountRefunded   int64             `json:"amount_refunded"`
	AmountReversed   int64             `json:"amount_reversed"`
	Fee              int64             `json:"fee"`
	Net              int64             `json:"net"`
	PayoutsEnabled   bool              `json:"payouts_enabled"`
	DetailsSubmitted bool              `json:"details_submitted"`
	Capabilities     map[string]string `json:"capabilities"`
	PaymentIntent    json.RawMessage   `json:"payment_intent"`
	LatestCharge     json.RawMessage   `json:"latest_charge"`
	Charge           json.RawMessage   `json:"charge"`
	BalanceTxn       json.RawMessage   `json:"balance_transaction"`
	Source           json.RawMessage   `json:"source"`
	Metadata         map[string]string `json:"metadata"`
}

type eventPlan struct {
	disposition string
	failureCode string
	operation   *Operation
	binding     *Binding
	desired     OperationStatus
	object      stripeObject
	objectKind  string
	linked      map[string]string
	category    string
	amount      accounting.Money
	fee         accounting.Money
	net         accounting.Money
	reason      string
}

func (s *Service) WebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricWebhookRequests.Add(1)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if r.Method != http.MethodPost || mediaErr != nil || strings.ToLower(mediaType) != "application/json" {
			metricWebhookInvalid.Add(1)
			http.Error(w, "invalid webhook request", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, s.Config.MaxBodyBytes+1))
		if err != nil || int64(len(body)) > s.Config.MaxBodyBytes {
			metricWebhookInvalid.Add(1)
			http.Error(w, "invalid webhook request", http.StatusBadRequest)
			return
		}
		if err := s.VerifyWebhookSignature(body, r.Header.Get("Stripe-Signature"), s.currentTime()); err != nil {
			metricWebhookInvalid.Add(1)
			http.Error(w, "invalid webhook signature", http.StatusBadRequest)
			return
		}
		result, err := s.IngestWebhook(r.Context(), body)
		if err != nil {
			metricWebhookErrors.Add(1)
			http.Error(w, "webhook temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		recordWebhookDisposition(result)
		if result.Retryable {
			metricWebhookErrors.Add(1)
			http.Error(w, "webhook dependency temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (s *Service) VerifyWebhookSignature(payload []byte, header string, now time.Time) error {
	if s == nil || len(s.WebhookSecret) < 16 {
		return fmt.Errorf("webhook verifier is unavailable")
	}
	if len(header) == 0 || len(header) > 4096 {
		return fmt.Errorf("webhook signature header is invalid")
	}
	var timestamp int64
	var signatures [][]byte
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed <= 0 {
				return fmt.Errorf("webhook timestamp is invalid")
			}
			timestamp = parsed
		case "v1":
			decoded, err := hex.DecodeString(value)
			if err == nil && len(decoded) == sha256.Size {
				signatures = append(signatures, decoded)
				if len(signatures) > 16 {
					return fmt.Errorf("webhook signature header has too many signatures")
				}
			}
		}
	}
	if timestamp == 0 || len(signatures) == 0 {
		return fmt.Errorf("webhook signature fields are missing")
	}
	delta := now.Unix() - timestamp
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(s.Config.WebhookToleranceSeconds) {
		return fmt.Errorf("webhook timestamp is outside the accepted window")
	}
	matched := 0
	secrets := s.WebhookSecrets
	if len(secrets) == 0 {
		secrets = [][]byte{s.WebhookSecret}
	}
	for _, secret := range secrets {
		mac := hmac.New(sha256.New, secret)
		_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
		_, _ = mac.Write([]byte("."))
		_, _ = mac.Write(payload)
		expected := mac.Sum(nil)
		for _, signature := range signatures {
			matched |= subtle.ConstantTimeCompare(expected, signature)
		}
	}
	if matched != 1 {
		return fmt.Errorf("webhook signature does not match")
	}
	return nil
}

func (s *Service) IngestWebhook(ctx context.Context, payload []byte) (WebhookResult, error) {
	if int64(len(payload)) > s.Config.MaxBodyBytes {
		return WebhookResult{}, fmt.Errorf("webhook payload exceeds configured limit")
	}
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return WebhookResult{}, fmt.Errorf("webhook JSON is invalid")
	}
	if err := ValidateOpaqueToken(event.ID, "evt_"); err != nil {
		return WebhookResult{}, err
	}
	if event.Type == "" || len(event.Type) > 96 || event.Created <= 0 || event.LiveMode != s.Config.LiveMode || len(event.Data.Object) == 0 {
		return WebhookResult{}, fmt.Errorf("webhook event envelope is invalid")
	}
	if event.APIVersion != stripeAPIVersion && !(event.APIVersion == "" && s.Config.APIBaseURL != "https://api.stripe.com") {
		return WebhookResult{}, fmt.Errorf("webhook event API version is invalid")
	}
	if event.Account != "" {
		if err := ValidateOpaqueToken(event.Account, "acct_"); err != nil {
			return WebhookResult{}, fmt.Errorf("webhook connected account is invalid")
		}
	}
	createdAt := time.Unix(event.Created, 0).UTC()
	now := s.currentTime()
	if createdAt.After(now.Add(time.Minute)) || createdAt.Before(now.AddDate(0, 0, -s.Config.EventRetentionDays)) {
		return WebhookResult{}, fmt.Errorf("webhook event creation time is invalid")
	}
	var object stripeObject
	if err := json.Unmarshal(event.Data.Object, &object); err != nil || object.ID == "" || object.Object == "" || len(object.Object) > 32 {
		return WebhookResult{}, fmt.Errorf("webhook object is invalid")
	}
	prefix := prefixForStripeObject(object.Object)
	var tokenErr error
	if prefix == "" {
		tokenErr = ValidateOpaqueToken(object.ID)
	} else {
		tokenErr = ValidateOpaqueToken(object.ID, prefix)
	}
	if tokenErr != nil {
		return WebhookResult{}, tokenErr
	}
	payloadHash := sha256.Sum256(payload)
	// Exact operation/binding FOR UPDATE locks serialize state transitions, and
	// the provider event/object unique keys serialize identity. READ COMMITTED
	// avoids taking serializable gap locks on a new provider token before its
	// owner row is locked, which can deadlock concurrent delivery of one event.
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return WebhookResult{}, err
	}
	defer tx.Rollback()
	plan, err := s.planEvent(ctx, tx, event, object, createdAt)
	if err != nil {
		return WebhookResult{}, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO hosted_event (provider,provider_event_token,event_type,object_kind,
 provider_object_token,operation_id,binding_id,provider_created_at,payload_sha256,
 disposition,failure_code,received_at)
VALUES ('stripe',?,?,?,?,?,?,?,?,?,NULLIF(?,''),UTC_TIMESTAMP())`, event.ID, event.Type,
		object.Object, object.ID, nullableOperationID(plan.operation), nullableBindingID(plan.binding),
		createdAt, payloadHash[:], plan.disposition, plan.failureCode)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
			return WebhookResult{}, err
		}
		var existingID uint64
		var existingHash []byte
		var existingDisposition string
		if err := tx.QueryRowContext(ctx, `SELECT event_id,payload_sha256,disposition FROM hosted_event WHERE provider='stripe' AND provider_event_token=? FOR UPDATE`, event.ID).Scan(&existingID, &existingHash, &existingDisposition); err != nil {
			return WebhookResult{}, err
		}
		if len(existingHash) != sha256.Size || subtle.ConstantTimeCompare(existingHash, payloadHash[:]) != 1 {
			return WebhookResult{}, fmt.Errorf("%w: provider event token was reused with different content", ErrConflict)
		}
		if existingDisposition == "Unresolved" && (plan.disposition == "Applied" || plan.disposition == "Ignored") {
			if plan.disposition == "Applied" {
				if err := s.applyEventPlan(ctx, tx, existingID, event, createdAt, plan); err != nil {
					return WebhookResult{}, err
				}
			}
			result, err := tx.ExecContext(ctx, `UPDATE hosted_event SET operation_id=?,binding_id=?,disposition=?,failure_code=NULLIF(?, '') WHERE event_id=? AND disposition='Unresolved'`, nullableOperationID(plan.operation), nullableBindingID(plan.binding), plan.disposition, plan.failureCode, existingID)
			if err != nil {
				return WebhookResult{}, err
			}
			if affected, _ := result.RowsAffected(); affected != 1 {
				return WebhookResult{}, ErrConflict
			}
			if err := tx.Commit(); err != nil {
				return WebhookResult{}, err
			}
			return WebhookResult{Reprocessed: true, Disposition: plan.disposition, EventID: existingID}, nil
		}
		if err := tx.Commit(); err != nil {
			return WebhookResult{}, err
		}
		return WebhookResult{Duplicate: true, Retryable: retryableEventPlan(plan), Disposition: "Duplicate", EventID: existingID}, nil
	}
	eventIDRaw, err := result.LastInsertId()
	if err != nil {
		return WebhookResult{}, err
	}
	eventID := uint64(eventIDRaw)
	if err := s.applyEventPlan(ctx, tx, eventID, event, createdAt, plan); err != nil {
		return WebhookResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return WebhookResult{}, err
	}
	return WebhookResult{Retryable: retryableEventPlan(plan), Disposition: plan.disposition, EventID: eventID}, nil
}

func retryableEventPlan(plan eventPlan) bool {
	if plan.disposition != "Unresolved" {
		return false
	}
	switch plan.failureCode {
	case "operation_not_found", "binding_not_found":
		return true
	default:
		return false
	}
}

func (s *Service) planEvent(ctx context.Context, tx *sql.Tx, event stripeEvent, object stripeObject, createdAt time.Time) (eventPlan, error) {
	plan := eventPlan{disposition: "Ignored", object: object, objectKind: stripeObjectKind(object.Object), linked: make(map[string]string)}
	connectedBindingEvent := event.Type == "account.updated" || event.Type == "payout.failed"
	if event.Account != "" && !connectedBindingEvent {
		// Objects owned by connected accounts are outside the platform funding,
		// refund, and transfer namespace. In particular, never let metadata on a
		// publisher's direct charge claim an advertiser operation.
		plan.failureCode = "connected_account_event_out_of_scope"
		return plan, nil
	}
	if connectedBindingEvent && event.Account == "" {
		plan.disposition, plan.failureCode = "Unresolved", "connected_account_missing"
		plan.category, plan.reason = "ProviderMismatch", "connected-account event has no exact Stripe account owner"
		return plan, nil
	}
	if event.Type == "account.updated" && event.Account != object.ID {
		plan.disposition, plan.failureCode = "Unresolved", "connected_account_mismatch"
		plan.category, plan.reason = "ProviderMismatch", "account update object differs from its connected-account owner"
		return plan, nil
	}
	metadataOperationID := metadataUint(object.Metadata, "aofei_operation_id")
	operationID := metadataOperationID
	mappedOperationID, err := operationForToken(ctx, tx, object.ID)
	if err != nil {
		return plan, err
	}
	if operationID != 0 && mappedOperationID != 0 && operationID != mappedOperationID {
		operation, lookupErr := operationByIDTx(ctx, tx, mappedOperationID, true)
		if lookupErr != nil {
			return plan, lookupErr
		}
		plan.operation = &operation
		plan.disposition, plan.failureCode = "Unresolved", "provider_owner_mismatch"
		plan.category, plan.reason = "ProviderMismatch", "provider object metadata conflicts with its immutable Aofei owner"
		return plan, nil
	}
	if operationID == 0 {
		operationID = mappedOperationID
	}
	if operationID == 0 {
		for _, raw := range []json.RawMessage{object.PaymentIntent, object.LatestCharge, object.Charge, object.Source} {
			if token := tokenFromJSON(raw); token != "" {
				operationID, err = operationForToken(ctx, tx, token)
				if err != nil {
					return plan, err
				}
				if operationID != 0 {
					break
				}
			}
		}
	}
	if operationID != 0 {
		operation, err := operationByIDTx(ctx, tx, operationID, true)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return plan, err
		}
		if err == nil {
			plan.operation = &operation
		}
	}
	if plan.operation != nil && metadataOperationID != 0 {
		statementID := metadataUint(object.Metadata, "aofei_statement_id")
		partyID := metadataUint(object.Metadata, "aofei_party_id")
		missingFastOwnership := mappedOperationID == 0 && (statementID == 0 || partyID == 0)
		if missingFastOwnership || statementID != 0 && statementID != plan.operation.StatementID ||
			partyID != 0 && partyID != plan.operation.PartyID {
			plan.disposition, plan.failureCode = "Unresolved", "provider_metadata_mismatch"
			plan.category, plan.reason = "ProviderMismatch", "provider object metadata does not match the exact Aofei statement and party"
			return plan, nil
		}
	}
	if plan.operation != nil && !plan.operation.ExecutedBy.Valid {
		plan.disposition, plan.failureCode = "Unresolved", "operation_not_submitted"
		plan.category, plan.reason = "ProviderMismatch", "provider event targets an Aofei operation that was never submitted"
		return plan, nil
	}
	bindingToken := object.ID
	if event.Type == "payout.failed" && event.Account != "" {
		bindingToken = event.Account
	}
	if strings.HasPrefix(bindingToken, "acct_") {
		binding, err := bindingForToken(ctx, tx, bindingToken)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return plan, err
		}
		if err == nil {
			plan.binding = &binding
		}
	}

	for kind, raw := range map[string]json.RawMessage{
		"PaymentIntent":      object.PaymentIntent,
		"Charge":             firstRaw(object.LatestCharge, object.Charge),
		"BalanceTransaction": object.BalanceTxn,
	} {
		if token := tokenFromJSON(raw); token != "" {
			plan.linked[kind] = token
		}
	}

	switch event.Type {
	case "account.updated":
		if plan.binding == nil || plan.binding.Kind != BindingPayoutAccount {
			plan.disposition, plan.failureCode = "Unresolved", "binding_not_found"
			plan.category, plan.reason = "ProviderMismatch", "provider account update has no exact publisher binding"
			return plan, nil
		}
		var latest sql.NullTime
		if err := tx.QueryRowContext(ctx, `SELECT MAX(provider_created_at) FROM hosted_event WHERE binding_id=? AND event_type='account.updated' AND disposition='Applied'`, plan.binding.ID).Scan(&latest); err != nil {
			return plan, err
		}
		incomingReady := payoutAccountReady(object)
		if latest.Valid && (createdAt.Before(latest.Time) || createdAt.Equal(latest.Time) && (!incomingReady || plan.binding.ProviderReady)) {
			plan.disposition, plan.failureCode = "Ignored", "stale_or_equal_binding_state"
			return plan, nil
		}
		plan.disposition = "Applied"
		return plan, nil
	case "payout.failed":
		if plan.binding == nil || plan.binding.Kind != BindingPayoutAccount {
			plan.disposition, plan.failureCode = "Unresolved", "binding_not_found"
			plan.category, plan.reason = "ProviderMismatch", "failed connected-account payout has no exact publisher binding"
			return plan, nil
		}
		plan.amount = centsMoney(object.Amount)
		if strings.ToUpper(object.Currency) != "USD" || plan.amount <= 0 || plan.amount == accounting.Money(math.MinInt64) {
			plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
			plan.category, plan.reason = "ProviderMismatch", "failed connected-account payout is outside the supported USD contract"
			plan.amount = 0
			return plan, nil
		}
		plan.disposition = "Applied"
		plan.category, plan.reason = "PayoutFailure", "provider reported a failed connected-account payout; exact operation requires manual reconciliation"
		return plan, nil
	case "checkout.session.completed":
		if object.PaymentStatus == "paid" {
			plan.desired = OperationSucceeded
		} else {
			plan.desired = OperationSubmitted
		}
	case "checkout.session.expired":
		plan.desired = OperationCanceled
	case "checkout.session.async_payment_succeeded":
		plan.desired = OperationSucceeded
	case "checkout.session.async_payment_failed":
		plan.desired = OperationFailed
	case "payment_intent.succeeded":
		plan.desired = OperationSucceeded
	case "charge.succeeded":
		plan.desired = OperationSucceeded
	case "payment_intent.payment_failed", "charge.failed":
		// A Checkout Session can remain open for another payment attempt. Keep
		// its capacity reserved until a signed success, expiry, or cancellation.
		plan.desired = OperationSubmitted
		plan.failureCode = "payment_attempt_failed"
	case "refund.failed":
		plan.desired = OperationFailed
	case "transfer.created":
		plan.desired = OperationSucceeded
	case "transfer.reversed":
		plan.desired = OperationDisputed
		plan.category, plan.reason = "PayoutFailure", "provider transfer was reversed"
		plan.amount = centsMoney(object.AmountReversed)
	case "refund.created", "refund.updated", "charge.refund.updated":
		if object.Status == "succeeded" {
			plan.desired = OperationSucceeded
		} else if object.Status == "failed" || object.Status == "canceled" {
			plan.desired = OperationFailed
		} else {
			plan.desired = OperationSubmitted
		}
	case "charge.refunded":
		if object.Amount > 0 && object.AmountRefunded >= object.Amount {
			plan.desired = OperationRefunded
		} else {
			plan.desired = OperationPartiallyRefunded
		}
		plan.category, plan.reason = "Refund", "provider reported refunded funds"
		plan.amount = centsMoney(object.AmountRefunded)
	case "charge.dispute.created", "charge.dispute.updated":
		plan.desired = OperationDisputed
		plan.category, plan.reason = "Dispute", "provider dispute requires operator review"
		plan.amount = centsMoney(object.Amount)
	case "charge.dispute.closed":
		if object.Status == "won" {
			plan.desired = OperationSucceeded
		} else {
			plan.desired = OperationDisputed
		}
		plan.category, plan.reason = "Chargeback", "provider dispute reached a terminal result"
		plan.amount = centsMoney(object.Amount)
	default:
		return plan, nil
	}
	if (plan.category == "Refund" || plan.category == "Dispute" || plan.category == "Chargeback") &&
		(strings.ToUpper(object.Currency) != "USD" || plan.amount <= 0 || plan.amount == accounting.Money(math.MinInt64)) {
		plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
		plan.category, plan.reason = "ProviderMismatch", "provider refund or dispute amount is outside the supported USD contract"
		plan.amount = 0
		return plan, nil
	}
	if plan.operation == nil {
		plan.disposition, plan.failureCode = "Unresolved", "operation_not_found"
		plan.category, plan.reason = "MissingEvent", "supported provider event has no exact Aofei operation"
		return plan, nil
	}
	if !eventMatchesOperation(event.Type, plan.operation.Kind) {
		plan.disposition, plan.failureCode = "Unresolved", "operation_kind_mismatch"
		plan.category, plan.reason = "ProviderMismatch", "provider event type does not match operation kind"
		return plan, nil
	}
	if event.Type == "transfer.reversed" {
		original := centsMoney(object.Amount)
		if strings.ToUpper(object.Currency) != "USD" || original != plan.operation.Amount || plan.amount <= 0 ||
			plan.amount == accounting.Money(math.MinInt64) || plan.amount > plan.operation.Amount {
			plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
			plan.category, plan.reason = "ProviderMismatch", "provider transfer reversal is outside the approved payout amount"
			plan.amount = 0
			return plan, nil
		}
	}
	if event.Type == "charge.refunded" {
		original := centsMoney(object.Amount)
		if original != plan.operation.Amount || plan.amount > original {
			plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
			plan.category, plan.reason = "ProviderMismatch", "provider refund summary exceeds or differs from the approved funding amount"
			plan.amount = 0
			return plan, nil
		}
	}
	if (strings.HasPrefix(event.Type, "charge.dispute.")) && plan.amount > plan.operation.Amount {
		plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
		plan.category, plan.reason = "ProviderMismatch", "provider dispute exceeds the approved funding amount"
		plan.amount = 0
		return plan, nil
	}
	if needsAmountMatch(event.Type, plan.desired) {
		cents := objectEventCents(event.Type, object)
		amount := centsMoney(cents)
		if strings.ToUpper(object.Currency) != "USD" || amount == accounting.Money(math.MinInt64) || amount != plan.operation.Amount {
			plan.disposition, plan.failureCode = "Unresolved", "currency_or_amount_mismatch"
			plan.category, plan.reason = "ProviderMismatch", "provider event amount or currency differs from the approved operation"
			return plan, nil
		}
	}
	if !eventShouldApply(*plan.operation, event.Type, plan.desired, createdAt) {
		plan.disposition, plan.failureCode = "Ignored", "stale_or_weaker_state"
		return plan, nil
	}
	plan.disposition = "Applied"
	return plan, nil
}

func (s *Service) applyEventPlan(ctx context.Context, tx *sql.Tx, eventID uint64, event stripeEvent, createdAt time.Time, plan eventPlan) error {
	if plan.binding != nil && plan.disposition == "Applied" && (plan.objectKind == "PayoutAccount" || plan.objectKind == "Payout") {
		if err := mapBindingObject(ctx, tx, plan.binding.ID, plan.objectKind, plan.object.ID); err != nil {
			return err
		}
	}
	if plan.binding != nil && event.Type == "account.updated" && plan.disposition == "Applied" {
		ready := payoutAccountReady(plan.object)
		if err := applyAccountUpdate(ctx, tx, *plan.binding, ready, event.ID); err != nil {
			return err
		}
	}
	if plan.operation != nil && plan.disposition == "Applied" {
		if plan.objectKind != "" {
			if err := mapOperationObject(ctx, tx, plan.operation.ID, plan.objectKind, plan.object.ID); err != nil {
				return err
			}
		}
		for kind, token := range plan.linked {
			// A refund references its parent funding PaymentIntent/Charge. Those
			// immutable provider objects remain owned by the funding operation;
			// only the re_ object belongs to the refund operation.
			if plan.operation.Kind == OperationRefund && (kind == "PaymentIntent" || kind == "Charge") {
				continue
			}
			if err := mapOperationObject(ctx, tx, plan.operation.ID, kind, token); err != nil {
				return err
			}
		}
		if plan.desired != "" {
			prior := plan.operation.Status
			if _, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status=?,provider_event_created_at=?,version=version+1,failure_code=NULLIF(?,''),updated_at=UTC_TIMESTAMP() WHERE operation_id=?`, plan.desired, createdAt, plan.failureCode, plan.operation.ID); err != nil {
				return err
			}
			if err := insertAudit(ctx, tx, "Operation", plan.operation.ID, "provider:stripe", "ProviderEventApplied", string(prior), string(plan.desired), event.Type, plan.object.ID); err != nil {
				return err
			}
		}
	}
	if plan.category != "" {
		status := "Unresolved"
		if plan.disposition == "Applied" && plan.category == "Settlement" {
			status = "Matched"
		}
		if err := insertReconciliation(ctx, tx, eventID, plan, status); err != nil {
			return err
		}
	}
	return nil
}

func payoutAccountReady(object stripeObject) bool {
	return object.PayoutsEnabled && object.DetailsSubmitted && object.Capabilities["transfers"] == "active"
}

func applyAccountUpdate(ctx context.Context, tx *sql.Tx, binding Binding, ready bool, eventToken string) error {
	prior, next := binding.Status, binding.Status
	approvedBy := any(binding.ApprovedBy)
	revokedBy := any(binding.RevokedBy)
	if ready {
		if binding.Status == BindingProposed || binding.Status == BindingRevoked && binding.RevokedBy.Valid && binding.RevokedBy.String == "provider:stripe" {
			next = BindingReady
			approvedBy = nil
			revokedBy = nil
		}
	} else {
		switch binding.Status {
		case BindingApproved:
			next = BindingRevoked
			revokedBy = "provider:stripe"
		case BindingReady:
			next = BindingProposed
		}
	}
	if next == prior && binding.ProviderReady == ready {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE hosted_binding SET status=?,provider_ready=?,approved_by=?,revoked_by=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE binding_id=?`, next, ready, approvedBy, revokedBy, binding.ID); err != nil {
		return err
	}
	return insertAudit(ctx, tx, "Binding", binding.ID, "provider:stripe", "PayoutReadinessChanged", string(prior), string(next), "authenticated account.updated event", eventToken)
}

func insertReconciliation(ctx context.Context, tx *sql.Tx, eventID uint64, plan eventPlan, status string) error {
	var operationID any
	if plan.operation != nil {
		operationID = plan.operation.ID
	}
	requestKey := fmt.Sprintf("event:%d:%s", eventID, plan.category)
	_, err := tx.ExecContext(ctx, `
INSERT INTO hosted_reconciliation (request_key,operation_id,binding_id,event_id,category,
 provider_object_token,currency,amount,fee,net,status,reason,created_by,created_at)
VALUES (?,?,?,?,?,?,'USD',?,?,?,?,?,'provider:stripe',UTC_TIMESTAMP())`, requestKey, operationID, nullableBindingID(plan.binding), eventID,
		plan.category, nullableString(plan.object.ID), plan.amount.String(), plan.fee.String(), plan.net.String(), status, plan.reason)
	return err
}

func mapOperationObject(ctx context.Context, tx *sql.Tx, operationID uint64, kind, token string) error {
	if operationID == 0 || kind == "" {
		return nil
	}
	if err := ValidateOpaqueToken(token, prefixForObjectKind(kind)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO hosted_provider_object (provider,object_kind,provider_token,operation_id,created_at)
VALUES ('stripe',?,?,?,UTC_TIMESTAMP())`, kind, token, operationID); err != nil {
		return err
	}
	var actualOperation uint64
	var actualKind string
	var bindingID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT operation_id,object_kind,binding_id FROM hosted_provider_object WHERE provider='stripe' AND provider_token=? FOR UPDATE`, token).Scan(&actualOperation, &actualKind, &bindingID); err != nil {
		return err
	}
	if actualOperation != operationID || actualKind != kind || bindingID.Valid {
		return fmt.Errorf("%w: provider object token belongs to another owner", ErrConflict)
	}
	return nil
}

func mapBindingObject(ctx context.Context, tx *sql.Tx, bindingID uint64, kind, token string) error {
	if bindingID == 0 || kind == "" {
		return nil
	}
	if err := ValidateOpaqueToken(token, prefixForObjectKind(kind)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO hosted_provider_object (provider,object_kind,provider_token,binding_id,created_at)
VALUES ('stripe',?,?,?,UTC_TIMESTAMP())`, kind, token, bindingID); err != nil {
		return err
	}
	var actualBinding uint64
	var actualKind string
	var operationID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT binding_id,object_kind,operation_id FROM hosted_provider_object WHERE provider='stripe' AND provider_token=? FOR UPDATE`, token).Scan(&actualBinding, &actualKind, &operationID); err != nil {
		return err
	}
	if actualBinding != bindingID || actualKind != kind || operationID.Valid {
		return fmt.Errorf("%w: provider object token belongs to another owner", ErrConflict)
	}
	return nil
}

func eventShouldApply(operation Operation, eventType string, desired OperationStatus, createdAt time.Time) bool {
	if operation.ProviderEventCreatedAt.Valid && createdAt.Before(operation.ProviderEventCreatedAt.Time) {
		return false
	}
	if strings.HasPrefix(eventType, "charge.dispute.") &&
		(operation.Status == OperationPartiallyRefunded || operation.Status == OperationRefunded) {
		// Refund state is a terminal money fact. The separate unresolved dispute
		// reconciliation row preserves the later dispute without erasing it.
		return false
	}
	// These provider events are legitimate later transitions that do not fit a
	// single monotonically increasing status rank.
	switch eventType {
	case "transfer.reversed", "charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed", "charge.refunded":
		return !operation.ProviderEventCreatedAt.Valid || createdAt.After(operation.ProviderEventCreatedAt.Time) || desired != operation.Status
	}
	if !operation.ProviderEventCreatedAt.Valid {
		return true
	}
	// A later event can confirm the same state while supplying immutable linked
	// objects (most importantly a Charge's balance_transaction). Persist that
	// evidence even though the visible operation state does not change.
	if desired == operation.Status {
		return true
	}
	return operationStateRank(desired) > operationStateRank(operation.Status)
}

func operationStateRank(status OperationStatus) int {
	switch status {
	case OperationProposed:
		return 1
	case OperationApproved, OperationSubmitting:
		return 2
	case OperationSubmitted, OperationCanceling:
		return 3
	case OperationFailed, OperationCanceled:
		return 4
	case OperationSucceeded:
		return 5
	case OperationDisputed:
		return 6
	case OperationPartiallyRefunded:
		return 7
	case OperationRefunded:
		return 8
	default:
		return 0
	}
}

func eventMatchesOperation(eventType string, kind OperationKind) bool {
	if eventType == "charge.refund.updated" {
		return kind == OperationRefund
	}
	if strings.HasPrefix(eventType, "checkout.session.") || strings.HasPrefix(eventType, "payment_intent.") || strings.HasPrefix(eventType, "charge.") {
		return kind == OperationFunding
	}
	if strings.HasPrefix(eventType, "transfer.") {
		return kind == OperationPayout
	}
	if strings.HasPrefix(eventType, "refund.") {
		return kind == OperationRefund
	}
	return strings.HasPrefix(eventType, "balance_transaction.")
}

func needsAmountMatch(eventType string, desired OperationStatus) bool {
	if desired != OperationSucceeded {
		return false
	}
	return eventType == "checkout.session.completed" || eventType == "checkout.session.async_payment_succeeded" ||
		eventType == "payment_intent.succeeded" || eventType == "charge.succeeded" || eventType == "transfer.created" ||
		strings.HasPrefix(eventType, "refund.") || eventType == "charge.refund.updated"
}

func objectEventCents(eventType string, object stripeObject) int64 {
	switch eventType {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		return object.AmountTotal
	case "payment_intent.succeeded":
		if object.AmountReceived != 0 {
			return object.AmountReceived
		}
		return object.Amount
	default:
		return object.Amount
	}
}

func centsMoney(cents int64) accounting.Money {
	if cents > math.MaxInt64/10_000 || cents < math.MinInt64/10_000 {
		return accounting.Money(math.MinInt64)
	}
	return accounting.Money(cents * 10_000)
}

func subtractOverflows(left, right int64) bool {
	return right > 0 && left < math.MinInt64+right || right < 0 && left > math.MaxInt64+right
}

func operationForToken(ctx context.Context, tx *sql.Tx, token string) (uint64, error) {
	var id uint64
	err := tx.QueryRowContext(ctx, `SELECT operation_id FROM hosted_provider_object WHERE provider='stripe' AND provider_token=? AND operation_id IS NOT NULL`, token).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

func bindingForToken(ctx context.Context, tx *sql.Tx, token string) (Binding, error) {
	return scanBinding(tx.QueryRowContext(ctx, bindingSelect+` WHERE provider='stripe' AND provider_token=? FOR UPDATE`, token))
}

func metadataUint(metadata map[string]string, key string) uint64 {
	value, err := strconv.ParseUint(metadata[key], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func tokenFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var token string
	if json.Unmarshal(raw, &token) == nil {
		return token
	}
	var object struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &object) == nil {
		return object.ID
	}
	return ""
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) != 0 && string(value) != "null" && tokenFromJSON(value) != "" {
			return value
		}
	}
	return nil
}

func stripeObjectKind(object string) string {
	switch object {
	case "checkout.session":
		return "Checkout"
	case "payment_intent":
		return "PaymentIntent"
	case "charge":
		return "Charge"
	case "account":
		return "PayoutAccount"
	case "transfer":
		return "Transfer"
	case "refund":
		return "Refund"
	case "balance_transaction":
		return "BalanceTransaction"
	case "payout":
		return "Payout"
	case "dispute":
		return "Dispute"
	default:
		return ""
	}
}

func prefixForStripeObject(object string) string {
	kind := stripeObjectKind(object)
	if kind == "" {
		return ""
	}
	return prefixForObjectKind(kind)
}

func prefixForObjectKind(kind string) string {
	switch kind {
	case "Customer":
		return "cus_"
	case "Checkout":
		return "cs_"
	case "PaymentIntent":
		return "pi_"
	case "Charge":
		return "ch_"
	case "PayoutAccount":
		return "acct_"
	case "Transfer":
		return "tr_"
	case "Refund":
		return "re_"
	case "BalanceTransaction":
		return "txn_"
	case "Payout":
		return "po_"
	case "Dispute":
		return "du_"
	default:
		return "invalid_"
	}
}

func nullableOperationID(operation *Operation) any {
	if operation == nil {
		return nil
	}
	return operation.ID
}

func nullableBindingID(binding *Binding) any {
	if binding == nil {
		return nil
	}
	return binding.ID
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
