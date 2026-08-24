package hostedpayment

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/guruperl/aofei/accounting"
)

type PartyType = accounting.PartyType

var ibanLike = regexp.MustCompile(`(?i)[A-Z]{2}[ -]?[0-9]{2}(?:[ -]?[A-Z0-9]){11,30}`)

const (
	PartyAdvertiser = accounting.PartyAdvertiser
	PartyPublisher  = accounting.PartyPublisher

	PermissionRead             = "payment.read"
	PermissionFundingBind      = "payment.funding.bind"
	PermissionPayoutBind       = "payment.payout.bind"
	PermissionBindingApprove   = "payment.binding.approve"
	PermissionCheckoutPropose  = "payment.checkout.propose"
	PermissionPayoutPropose    = "payment.payout.propose"
	PermissionRefundPropose    = "payment.refund.propose"
	PermissionOperationApprove = "payment.operation.approve"
	PermissionOperationExecute = "payment.operation.execute"
	PermissionOperationCancel  = "payment.operation.cancel"
	PermissionDisputeHandle    = "payment.dispute.handle"
	PermissionReconcile        = "payment.reconcile"
	PermissionSecretReadiness  = "payment.secret.readiness"
	PermissionRetentionPrune   = "payment.retention.prune"
)

type BindingKind string
type BindingStatus string
type OperationKind string
type OperationStatus string

const (
	BindingFundingCustomer BindingKind = "FundingCustomer"
	BindingPayoutAccount   BindingKind = "PayoutAccount"

	BindingProposed BindingStatus = "Proposed"
	BindingReady    BindingStatus = "Ready"
	BindingApproved BindingStatus = "Approved"
	BindingRevoked  BindingStatus = "Revoked"

	OperationFunding OperationKind = "Funding"
	OperationPayout  OperationKind = "Payout"
	OperationRefund  OperationKind = "Refund"

	OperationProposed          OperationStatus = "Proposed"
	OperationApproved          OperationStatus = "Approved"
	OperationSubmitting        OperationStatus = "Submitting"
	OperationSubmitted         OperationStatus = "Submitted"
	OperationCanceling         OperationStatus = "Canceling"
	OperationSucceeded         OperationStatus = "Succeeded"
	OperationFailed            OperationStatus = "Failed"
	OperationCanceled          OperationStatus = "Canceled"
	OperationDisputed          OperationStatus = "Disputed"
	OperationRefunded          OperationStatus = "Refunded"
	OperationPartiallyRefunded OperationStatus = "PartiallyRefunded"
)

var ErrConflict = errors.New("hosted payment state conflict")

const maxOperationAttempts = uint8(255)

const providerIdempotencyReplayWindow = 23 * time.Hour

type Scope struct {
	PartyType PartyType
	PartyID   uint64
}

type Actor struct {
	Role        string
	ID          string
	Scope       Scope
	Permissions map[string]bool
	RecentMFA   bool
}

type Binding struct {
	ID            uint64
	RequestKey    string
	Provider      string
	PartyType     PartyType
	PartyID       uint64
	Kind          BindingKind
	ProviderToken string
	Country       sql.NullString
	Status        BindingStatus
	ProviderReady bool
	Version       uint32
	CreatedBy     string
	ApprovedBy    sql.NullString
	RevokedBy     sql.NullString
	Reason        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Operation struct {
	ID                     uint64
	RequestKey             string
	Provider               string
	Kind                   OperationKind
	ParentOperationID      sql.NullInt64
	BindingID              sql.NullInt64
	StatementID            uint64
	PartyType              PartyType
	PartyID                uint64
	Amount                 accounting.Money
	Currency               string
	Status                 OperationStatus
	CurrentObjectToken     sql.NullString
	Version                uint32
	CreatedBy              string
	ApprovedBy             sql.NullString
	ExecutedBy             sql.NullString
	Reason                 string
	FailureCode            sql.NullString
	AttemptCount           uint8
	ProviderEventCreatedAt sql.NullTime
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ProposeOperationInput struct {
	RequestKey        string
	Kind              OperationKind
	ParentOperationID uint64
	StatementID       uint64
	PartyID           uint64
	Amount            accounting.Money
	Reason            string
}

type ExecuteResult struct {
	Operation Operation
	Redirect  *HostedRedirect
}

type Service struct {
	DB             *sql.DB
	Provider       Provider
	Config         Config
	WebhookSecret  []byte   `json:"-"`
	WebhookSecrets [][]byte `json:"-"`
	now            func() time.Time
}

// MaintenanceService exposes only aggregate health and bounded retention.
// Its underlying DB-only service is intentionally not exported.
type MaintenanceService struct{ service *Service }

func (*Service) String() string   { return "hostedpayment.Service{redacted}" }
func (*Service) GoString() string { return "hostedpayment.Service{redacted}" }

func NewService(config Config, db *sql.DB) (*Service, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("hosted payment database is nil")
	}
	apiKey := strings.TrimSpace(os.Getenv(config.APIKeyEnv))
	webhookSecret := strings.TrimSpace(os.Getenv(config.WebhookSecretEnv))
	previousWebhookSecret := ""
	if config.WebhookPreviousSecretEnv != "" {
		previousWebhookSecret = strings.TrimSpace(os.Getenv(config.WebhookPreviousSecretEnv))
	}
	if apiKey == "" || webhookSecret == "" {
		return nil, fmt.Errorf("hosted payment secret references must resolve to non-empty values")
	}
	if config.LiveMode && !apiKeyMatchesMode(apiKey, true) {
		return nil, fmt.Errorf("hosted payment live mode requires a live provider API key")
	}
	if !config.LiveMode && !apiKeyMatchesMode(apiKey, false) {
		return nil, fmt.Errorf("hosted payment sandbox mode requires a test provider API key")
	}
	provider, err := NewStripeProvider(config, apiKey, nil)
	if err != nil {
		return nil, err
	}
	service, err := NewServiceWithProvider(config, db, provider, []byte(webhookSecret))
	if err != nil {
		return nil, err
	}
	if previousWebhookSecret != "" {
		if len(previousWebhookSecret) < 16 {
			return nil, fmt.Errorf("previous hosted payment webhook secret is too short")
		}
		service.WebhookSecrets = append(service.WebhookSecrets, []byte(previousWebhookSecret))
	}
	return service, nil
}

func apiKeyMatchesMode(apiKey string, live bool) bool {
	mode := "test"
	if live {
		mode = "live"
	}
	for _, prefix := range []string{"sk_" + mode + "_", "rk_" + mode + "_"} {
		if len(apiKey) > len(prefix) && strings.HasPrefix(apiKey, prefix) {
			return true
		}
	}
	return false
}

func NewServiceWithProvider(config Config, db *sql.DB, provider Provider, webhookSecret []byte) (*Service, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil || provider == nil || len(webhookSecret) < 16 {
		return nil, fmt.Errorf("hosted payment database, provider, and webhook secret are required")
	}
	current := append([]byte(nil), webhookSecret...)
	return &Service{DB: db, Provider: provider, Config: config, WebhookSecret: current, WebhookSecrets: [][]byte{current}, now: time.Now}, nil
}

// NewMaintenanceService constructs the DB-only aggregate-health and retention
// boundary. It deliberately does not read or retain provider credentials and
// its returned type has no money-movement or webhook methods.
func NewMaintenanceService(config Config, db *sql.DB) (*MaintenanceService, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("hosted payment database is nil")
	}
	return &MaintenanceService{service: &Service{DB: db, Config: config, now: time.Now}}, nil
}

func (s *Service) StartFundingCustomer(ctx context.Context, actor Actor, advertiserID uint64, requestKey, reason string) (Binding, error) {
	if err := authorize(actor, PermissionFundingBind, Scope{PartyType: PartyAdvertiser, PartyID: advertiserID}, true); err != nil {
		return Binding{}, err
	}
	if err := validateMutation(requestKey, reason); err != nil {
		return Binding{}, err
	}
	if existing, ok, err := s.bindingByRequest(ctx, requestKey); err != nil {
		return Binding{}, err
	} else if ok {
		if existing.PartyType != PartyAdvertiser || existing.PartyID != advertiserID || existing.Kind != BindingFundingCustomer ||
			existing.CreatedBy != actorName(actor) || existing.Reason != reason {
			return Binding{}, fmt.Errorf("%w: binding request key belongs to another operation", ErrConflict)
		}
		return existing, nil
	}
	if err := s.requireParty(ctx, PartyAdvertiser, advertiserID); err != nil {
		return Binding{}, err
	}
	token, err := s.Provider.CreateFundingCustomer(ctx, CustomerRequest{PartyID: advertiserID, IdempotencyKey: providerKey("binding", requestKey)})
	if err != nil {
		return Binding{}, err
	}
	return s.insertBinding(ctx, actor, requestKey, PartyAdvertiser, advertiserID, BindingFundingCustomer, token, "", true, reason)
}

func (s *Service) StartPayoutOnboarding(ctx context.Context, actor Actor, publisherID uint64, country, requestKey, reason string) (Binding, HostedRedirect, error) {
	scope := Scope{PartyType: PartyPublisher, PartyID: publisherID}
	if err := authorize(actor, PermissionPayoutBind, scope, true); err != nil {
		return Binding{}, HostedRedirect{}, err
	}
	if err := validateMutation(requestKey, reason); err != nil {
		return Binding{}, HostedRedirect{}, err
	}
	if !safeCountry.MatchString(country) {
		return Binding{}, HostedRedirect{}, fmt.Errorf("two-letter uppercase payout country is required")
	}
	binding, ok, err := s.bindingByRequest(ctx, requestKey)
	if err != nil {
		return Binding{}, HostedRedirect{}, err
	}
	if ok {
		if binding.PartyType != PartyPublisher || binding.PartyID != publisherID || binding.Kind != BindingPayoutAccount ||
			!binding.Country.Valid || binding.Country.String != country || binding.CreatedBy != actorName(actor) || binding.Reason != reason {
			return Binding{}, HostedRedirect{}, fmt.Errorf("%w: binding request key belongs to another operation", ErrConflict)
		}
	} else {
		if err := s.requireParty(ctx, PartyPublisher, publisherID); err != nil {
			return Binding{}, HostedRedirect{}, err
		}
		token, err := s.Provider.CreatePayoutAccount(ctx, PayoutAccountRequest{PartyID: publisherID, Country: country, IdempotencyKey: providerKey("binding", requestKey)})
		if err != nil {
			return Binding{}, HostedRedirect{}, err
		}
		binding, err = s.insertBinding(ctx, actor, requestKey, PartyPublisher, publisherID, BindingPayoutAccount, token, country, false, reason)
		if err != nil {
			return Binding{}, HostedRedirect{}, err
		}
	}
	redirect, err := s.payoutOnboardingLink(ctx, binding, requestKey)
	return binding, redirect, err
}

func (s *Service) payoutOnboardingLink(ctx context.Context, binding Binding, requestKey string) (HostedRedirect, error) {
	base := strings.TrimRight(s.Config.PublicBaseURL, "/")
	keyMaterial := requestKey + ":" + strconv.FormatUint(binding.ID, 10) + ":" + strconv.FormatUint(uint64(binding.Version), 10)
	return s.Provider.CreatePayoutOnboarding(ctx, OnboardingRequest{
		PartyID: binding.PartyID, AccountToken: binding.ProviderToken,
		RefreshURL:     base + "/goto/pub/g/hostedpayment?action=topics&onboarding=refresh",
		ReturnURL:      base + "/goto/pub/g/hostedpayment?action=topics",
		IdempotencyKey: providerKey("onboarding", keyMaterial),
	})
}

func (s *Service) RefreshPayoutOnboarding(ctx context.Context, actor Actor, bindingID uint64, expectedVersion uint32, requestKey, reason string) (HostedRedirect, error) {
	if bindingID == 0 || expectedVersion == 0 {
		return HostedRedirect{}, fmt.Errorf("binding and expected version are required")
	}
	if err := validateMutation(requestKey, reason); err != nil {
		return HostedRedirect{}, err
	}
	binding, err := s.Binding(ctx, bindingID)
	if err != nil {
		return HostedRedirect{}, err
	}
	if err := authorize(actor, PermissionPayoutBind, Scope{PartyType: binding.PartyType, PartyID: binding.PartyID}, true); err != nil {
		return HostedRedirect{}, err
	}
	refreshable := binding.Status == BindingProposed || binding.Status == BindingRevoked && binding.RevokedBy.Valid && binding.RevokedBy.String == "provider:stripe"
	if binding.Kind != BindingPayoutAccount || binding.Version != expectedVersion || !refreshable {
		return HostedRedirect{}, fmt.Errorf("%w: payout onboarding binding is not refreshable at the expected version", ErrConflict)
	}
	return s.payoutOnboardingLink(ctx, binding, requestKey)
}

func (s *Service) insertBinding(ctx context.Context, actor Actor, requestKey string, party PartyType, partyID uint64, kind BindingKind, token, country string, ready bool, reason string) (Binding, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Binding{}, err
	}
	defer tx.Rollback()
	status := BindingProposed
	if ready {
		status = BindingReady
	}
	existing, err := scanBinding(tx.QueryRowContext(ctx, bindingSelect+` WHERE request_key=? FOR UPDATE`, requestKey))
	if err == nil {
		if existing.PartyType != party || existing.PartyID != partyID || existing.Kind != kind || existing.ProviderToken != token || !bindingCountryMatches(existing, country) ||
			existing.CreatedBy != actorName(actor) || existing.Reason != reason {
			return Binding{}, fmt.Errorf("%w: binding request key belongs to another operation", ErrConflict)
		}
		if err := tx.Commit(); err != nil {
			return Binding{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Binding{}, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO hosted_binding (request_key, provider, party_type, party_id, binding_kind,
 provider_token, country, status, provider_ready, version, created_by, reason, created_at, updated_at)
VALUES (?,'stripe',?,?,?,?,NULLIF(?,''),?,?,1,?,?,UTC_TIMESTAMP(),UTC_TIMESTAMP())
`, requestKey, party, partyID, kind, token, country, status, ready, actorName(actor), reason)
	if err != nil {
		// A concurrent caller can pass the initial lookup while the provider is
		// completing the same idempotent request. Resolve the unique-key race to
		// the exact durable binding instead of surfacing a spurious failure.
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return Binding{}, errors.Join(err, rollbackErr)
		}
		existing, ok, lookupErr := s.bindingByRequest(ctx, requestKey)
		if lookupErr == nil && ok && existing.PartyType == party && existing.PartyID == partyID &&
			existing.Kind == kind && existing.ProviderToken == token && bindingCountryMatches(existing, country) && existing.CreatedBy == actorName(actor) && existing.Reason == reason {
			return existing, nil
		}
		return Binding{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Binding{}, err
	}
	if err := mapBindingObject(ctx, tx, uint64(id), providerObjectKindForBinding(kind), token); err != nil {
		return Binding{}, err
	}
	if err := insertAudit(ctx, tx, "Binding", uint64(id), actorName(actor), "BindingProposed", "Absent", string(status), reason, token); err != nil {
		return Binding{}, err
	}
	if err := tx.Commit(); err != nil {
		return Binding{}, err
	}
	return s.bindingByID(ctx, uint64(id))
}

func (s *Service) ApproveBinding(ctx context.Context, actor Actor, bindingID uint64, expectedVersion uint32, reason string) error {
	if bindingID == 0 || expectedVersion == 0 || validateReason(reason) != nil {
		return fmt.Errorf("binding, expected version, and bounded reason are required")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	binding, err := bindingByIDTx(ctx, tx, bindingID, true)
	if err != nil {
		return err
	}
	if err := authorize(actor, PermissionBindingApprove, Scope{PartyType: binding.PartyType, PartyID: binding.PartyID}, true); err != nil {
		return err
	}
	if binding.Version != expectedVersion || binding.Status != BindingReady || !binding.ProviderReady {
		return fmt.Errorf("%w: binding is not ready at the expected version", ErrConflict)
	}
	if binding.CreatedBy == actorName(actor) {
		return fmt.Errorf("%w: binding maker cannot approve the same binding", ErrConflict)
	}
	rows, err := tx.QueryContext(ctx, `
SELECT binding_id,provider_token,status FROM hosted_binding
WHERE party_type=? AND party_id=? AND binding_kind=? AND status='Approved' FOR UPDATE`, binding.PartyType, binding.PartyID, binding.Kind)
	if err != nil {
		return err
	}
	type priorBinding struct {
		id     uint64
		token  string
		status BindingStatus
	}
	var priors []priorBinding
	for rows.Next() {
		var prior priorBinding
		if err := rows.Scan(&prior.id, &prior.token, &prior.status); err != nil {
			rows.Close()
			return err
		}
		priors = append(priors, prior)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, prior := range priors {
		if _, err := tx.ExecContext(ctx, `UPDATE hosted_binding SET status='Revoked',revoked_by=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE binding_id=? AND status='Approved'`, actorName(actor), prior.id); err != nil {
			return err
		}
		if err := insertAudit(ctx, tx, "Binding", prior.id, actorName(actor), "BindingReplaced", "Approved", "Revoked", reason, prior.token); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE hosted_binding SET status='Approved',approved_by=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE binding_id=? AND status='Ready' AND version=?`, actorName(actor), bindingID, expectedVersion)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Binding", bindingID, actorName(actor), "BindingApproved", "Ready", "Approved", reason, binding.ProviderToken); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ProposeOperation(ctx context.Context, actor Actor, input ProposeOperationInput) (Operation, error) {
	party, permission, err := operationPartyPermission(input.Kind)
	if err != nil {
		return Operation{}, err
	}
	scope := Scope{PartyType: party, PartyID: input.PartyID}
	if err := authorize(actor, permission, scope, true); err != nil {
		return Operation{}, err
	}
	if err := validateMutation(input.RequestKey, input.Reason); err != nil {
		return Operation{}, err
	}
	if input.StatementID == 0 || input.PartyID == 0 {
		return Operation{}, fmt.Errorf("statement and party ids are required")
	}
	if _, err := moneyToCents(input.Amount); err != nil {
		return Operation{}, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Operation{}, err
	}
	defer tx.Rollback()
	existing, err := scanOperation(tx.QueryRowContext(ctx, operationSelect+` WHERE request_key=? FOR UPDATE`, input.RequestKey))
	if err == nil {
		parentMatches := input.ParentOperationID == 0 && !existing.ParentOperationID.Valid ||
			input.ParentOperationID != 0 && existing.ParentOperationID.Valid && uint64(existing.ParentOperationID.Int64) == input.ParentOperationID
		if existing.Kind != input.Kind || existing.StatementID != input.StatementID || existing.PartyID != input.PartyID ||
			existing.Amount != input.Amount || existing.CreatedBy != actorName(actor) || existing.Reason != input.Reason || !parentMatches {
			return Operation{}, fmt.Errorf("%w: operation request key belongs to another operation", ErrConflict)
		}
		if err := tx.Commit(); err != nil {
			return Operation{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Operation{}, err
	}
	if input.Kind == OperationRefund {
		if err := validateRefundParent(ctx, tx, input.ParentOperationID, input.StatementID, input.PartyID, input.Amount, input.RequestKey); err != nil {
			return Operation{}, err
		}
	} else {
		if input.ParentOperationID != 0 {
			return Operation{}, fmt.Errorf("only refunds may reference a parent operation")
		}
		if err := validateStatementForMovement(ctx, tx, input.StatementID, party, input.PartyID, input.Amount); err != nil {
			return Operation{}, err
		}
		if err := validateStatementCapacity(ctx, tx, input.StatementID, input.Kind, input.Amount, input.RequestKey); err != nil {
			return Operation{}, err
		}
	}
	var parent any
	if input.ParentOperationID != 0 {
		parent = input.ParentOperationID
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO hosted_operation (request_key,provider,operation_kind,parent_operation_id,
 statement_id,party_type,party_id,amount,currency,status,version,created_by,reason,created_at,updated_at)
VALUES (?,'stripe',?,?,?,?,?,?,'USD','Proposed',1,?,?,UTC_TIMESTAMP(),UTC_TIMESTAMP())
`, input.RequestKey, input.Kind, parent, input.StatementID, party, input.PartyID, input.Amount.String(), actorName(actor), input.Reason)
	if err != nil {
		return Operation{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Operation{}, err
	}
	operation, err := operationByIDTx(ctx, tx, uint64(id), true)
	if err != nil {
		return Operation{}, err
	}
	if operation.Kind != input.Kind || operation.StatementID != input.StatementID || operation.PartyID != input.PartyID || operation.Amount != input.Amount || operation.CreatedBy != actorName(actor) {
		return Operation{}, fmt.Errorf("%w: operation request key belongs to another operation", ErrConflict)
	}
	if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "OperationProposed", "Absent", "Proposed", input.Reason, ""); err != nil {
		return Operation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Operation{}, err
	}
	return s.Operation(ctx, operation.ID)
}

func (s *Service) ApproveOperation(ctx context.Context, actor Actor, operationID uint64, expectedVersion uint32, reason string) error {
	if operationID == 0 || expectedVersion == 0 || validateReason(reason) != nil {
		return fmt.Errorf("operation, expected version, and bounded reason are required")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	operation, err := operationByIDTx(ctx, tx, operationID, true)
	if err != nil {
		return err
	}
	if err := authorize(actor, PermissionOperationApprove, Scope{PartyType: operation.PartyType, PartyID: operation.PartyID}, true); err != nil {
		return err
	}
	if operation.Status != OperationProposed || operation.Version != expectedVersion {
		return fmt.Errorf("%w: operation is not proposed at the expected version", ErrConflict)
	}
	if operation.CreatedBy == actorName(actor) {
		return fmt.Errorf("%w: operation maker cannot approve the same operation", ErrConflict)
	}
	if operation.Kind == OperationRefund {
		if err := validateRefundParent(ctx, tx, uint64(operation.ParentOperationID.Int64), operation.StatementID, operation.PartyID, operation.Amount, operation.RequestKey); err != nil {
			return err
		}
	} else if err := validateStatementForMovement(ctx, tx, operation.StatementID, operation.PartyType, operation.PartyID, operation.Amount); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Approved',approved_by=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Proposed' AND version=?`, actorName(actor), operationID, expectedVersion)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Operation", operationID, actorName(actor), "OperationApproved", "Proposed", "Approved", reason, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ExecuteOperation(ctx context.Context, actor Actor, operationID uint64, expectedVersion uint32) (ExecuteResult, error) {
	operation, err := s.Operation(ctx, operationID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := authorize(actor, PermissionOperationExecute, Scope{PartyType: operation.PartyType, PartyID: operation.PartyID}, true); err != nil {
		return ExecuteResult{}, err
	}
	if operation.ApprovedBy.Valid && operation.ApprovedBy.String == actorName(actor) {
		return ExecuteResult{}, fmt.Errorf("%w: operation checker cannot execute the same operation", ErrConflict)
	}
	claimed, binding, paymentIntent, err := s.claimExecution(ctx, actor, operationID, expectedVersion)
	if err != nil {
		return ExecuteResult{}, err
	}
	idempotencyKey := providerKey("operation", claimed.RequestKey)
	var object ProviderObject
	var redirect *HostedRedirect
	switch claimed.Kind {
	case OperationFunding:
		created, callErr := s.Provider.CreateFundingCheckout(ctx, CheckoutRequest{
			OperationID: claimed.ID, StatementID: claimed.StatementID, PartyID: claimed.PartyID,
			Amount: claimed.Amount, CustomerToken: binding, IdempotencyKey: idempotencyKey,
			SuccessURL: strings.TrimRight(s.Config.PublicBaseURL, "/") + "/goto/adv/g/hostedpayment?action=topics&checkout=success",
			CancelURL:  strings.TrimRight(s.Config.PublicBaseURL, "/") + "/goto/adv/g/hostedpayment?action=topics&checkout=cancel",
		})
		err = callErr
		if err == nil {
			redirect = &created
			object = ProviderObject{Token: created.ObjectToken, Status: "submitted"}
		}
	case OperationPayout:
		object, err = s.Provider.CreateTransfer(ctx, TransferRequest{OperationID: claimed.ID, StatementID: claimed.StatementID, PartyID: claimed.PartyID, Amount: claimed.Amount, AccountToken: binding, IdempotencyKey: idempotencyKey})
	case OperationRefund:
		object, err = s.Provider.CreateRefund(ctx, RefundRequest{OperationID: claimed.ID, StatementID: claimed.StatementID, PartyID: claimed.PartyID, Amount: claimed.Amount, PaymentIntentToken: paymentIntent, IdempotencyKey: idempotencyKey})
	default:
		err = fmt.Errorf("operation kind is invalid")
	}
	if err != nil {
		s.recordExecutionFailure(ctx, claimed.ID, err)
		return ExecuteResult{}, err
	}
	if err := s.finishExecution(ctx, actor, claimed, object); err != nil {
		return ExecuteResult{}, err
	}
	final, err := s.Operation(ctx, claimed.ID)
	return ExecuteResult{Operation: final, Redirect: redirect}, err
}

func (s *Service) claimExecution(ctx context.Context, actor Actor, operationID uint64, expectedVersion uint32) (Operation, string, string, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Operation{}, "", "", err
	}
	defer tx.Rollback()
	operation, err := operationByIDTx(ctx, tx, operationID, true)
	if err != nil {
		return Operation{}, "", "", err
	}
	bindingID, binding, err := executionBinding(ctx, tx, operation)
	if err != nil {
		return Operation{}, "", "", err
	}
	paymentIntent, err := refundPaymentIntent(ctx, tx, operation)
	if err != nil {
		return Operation{}, "", "", err
	}
	providerInputToken := binding
	if paymentIntent != "" {
		providerInputToken = paymentIntent
	}
	if operation.Status == OperationSubmitting {
		if operation.Version != expectedVersion {
			return Operation{}, "", "", fmt.Errorf("%w: submitting operation is not at the expected version", ErrConflict)
		}
		if !operation.ExecutedBy.Valid {
			return Operation{}, "", "", fmt.Errorf("%w: provider submission owner is missing", ErrConflict)
		}
		if operation.AttemptCount == maxOperationAttempts {
			return Operation{}, "", "", fmt.Errorf("%w: provider submission attempt limit reached", ErrConflict)
		}
		if err := validateProviderReplayWindow(ctx, tx, operation.ID); err != nil {
			return Operation{}, "", "", err
		}
		priorActor := operation.ExecutedBy.String
		event := "ProviderSubmissionRetried"
		auditReason := "retry exact provider idempotency key after incomplete response"
		if operation.ExecutedBy.String != actorName(actor) {
			var staleSeconds int64
			if err := tx.QueryRowContext(ctx, `SELECT TIMESTAMPDIFF(SECOND,updated_at,UTC_TIMESTAMP()) FROM hosted_operation WHERE operation_id=?`, operation.ID).Scan(&staleSeconds); err != nil {
				return Operation{}, "", "", err
			}
			if actor.Role != "admin" || !actor.Permissions[PermissionReconcile] && !actor.Permissions["*"] || staleSeconds < 120 {
				return Operation{}, "", "", fmt.Errorf("%w: another actor owns the active provider request", ErrConflict)
			}
			event = "StuckSubmissionRecovered"
			auditReason = "retry exact provider idempotency key after stale submission by " + priorActor
		}
		result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET executed_by=?,attempt_count=attempt_count+1,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Submitting' AND version=?`, actorName(actor), operation.ID, operation.Version)
		if err != nil {
			return Operation{}, "", "", err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return Operation{}, "", "", ErrConflict
		}
		if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), event, "Submitting", "Submitting", auditReason, providerInputToken); err != nil {
			return Operation{}, "", "", err
		}
		operation.ExecutedBy = sql.NullString{String: actorName(actor), Valid: true}
		operation.AttemptCount++
		operation.Version++
		if operation.Kind == OperationRefund {
			if err := validateRefundParent(ctx, tx, uint64(operation.ParentOperationID.Int64), operation.StatementID, operation.PartyID, operation.Amount, operation.RequestKey); err != nil {
				return Operation{}, "", "", err
			}
		} else if err := validateStatementForMovement(ctx, tx, operation.StatementID, operation.PartyType, operation.PartyID, operation.Amount); err != nil {
			return Operation{}, "", "", err
		}
	} else if operation.Status == OperationSubmitted && operation.Kind == OperationFunding {
		if operation.Version != expectedVersion || !operation.ExecutedBy.Valid || operation.ExecutedBy.String != actorName(actor) {
			return Operation{}, "", "", fmt.Errorf("%w: submitted checkout is not retryable at the expected version", ErrConflict)
		}
		if err := validateStatementForMovement(ctx, tx, operation.StatementID, operation.PartyType, operation.PartyID, operation.Amount); err != nil {
			return Operation{}, "", "", err
		}
		if operation.AttemptCount == maxOperationAttempts {
			return Operation{}, "", "", fmt.Errorf("%w: provider submission attempt limit reached", ErrConflict)
		}
		if err := validateProviderReplayWindow(ctx, tx, operation.ID); err != nil {
			return Operation{}, "", "", err
		}
		result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET attempt_count=attempt_count+1,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Submitted' AND version=?`, operation.ID, operation.Version)
		if err != nil {
			return Operation{}, "", "", err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return Operation{}, "", "", ErrConflict
		}
		if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "HostedCheckoutReopened", "Submitted", "Submitted", "reopen existing hosted checkout with its exact provider idempotency key", operation.CurrentObjectToken.String); err != nil {
			return Operation{}, "", "", err
		}
		operation.AttemptCount++
		operation.Version++
	} else {
		if operation.Status != OperationApproved || operation.Version != expectedVersion {
			return Operation{}, "", "", fmt.Errorf("%w: operation is not approved at the expected version", ErrConflict)
		}
		// An Approved operation with a prior attempt represents an uncertain
		// provider response that was returned to a recoverable local state. It is
		// still a replay of the original Stripe idempotency key, not a new first
		// submission, and must stop before Stripe may prune that key.
		if operation.AttemptCount != 0 {
			if err := validateProviderReplayWindow(ctx, tx, operation.ID); err != nil {
				return Operation{}, "", "", err
			}
		}
		if operation.Kind == OperationRefund {
			if err := validateRefundParent(ctx, tx, uint64(operation.ParentOperationID.Int64), operation.StatementID, operation.PartyID, operation.Amount, operation.RequestKey); err != nil {
				return Operation{}, "", "", err
			}
		} else if err := validateStatementForMovement(ctx, tx, operation.StatementID, operation.PartyType, operation.PartyID, operation.Amount); err != nil {
			return Operation{}, "", "", err
		}
		if operation.AttemptCount == maxOperationAttempts {
			return Operation{}, "", "", fmt.Errorf("%w: provider submission attempt limit reached", ErrConflict)
		}
		result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Submitting',binding_id=COALESCE(binding_id,?),executed_by=?,attempt_count=attempt_count+1,version=version+1,failure_code=NULL,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Approved' AND version=?`, nullableExecutionBindingID(bindingID), actorName(actor), operation.ID, expectedVersion)
		if err != nil {
			return Operation{}, "", "", err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return Operation{}, "", "", ErrConflict
		}
		if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "ProviderSubmissionStarted", "Approved", "Submitting", operation.Reason, providerInputToken); err != nil {
			return Operation{}, "", "", err
		}
		operation.Status = OperationSubmitting
		operation.Version++
		operation.ExecutedBy = sql.NullString{String: actorName(actor), Valid: true}
		operation.BindingID = bindingID
	}
	if err := tx.Commit(); err != nil {
		return Operation{}, "", "", err
	}
	return operation, binding, paymentIntent, nil
}

func (s *Service) finishExecution(ctx context.Context, actor Actor, operation Operation, object ProviderObject) error {
	if err := ValidateOpaqueToken(object.Token, prefixesForOperation(operation.Kind)...); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := operationByIDTx(ctx, tx, operation.ID, true)
	if err != nil {
		return err
	}
	if !current.ExecutedBy.Valid || current.ExecutedBy.String != actorName(actor) {
		return fmt.Errorf("%w: operation provider submission ownership changed", ErrConflict)
	}
	if err := mapOperationObject(ctx, tx, operation.ID, providerObjectKindForOperation(operation.Kind), object.Token); err != nil {
		return err
	}
	if current.Status != OperationSubmitting {
		switch current.Status {
		case OperationSubmitted, OperationSucceeded, OperationFailed, OperationCanceled,
			OperationDisputed, OperationPartiallyRefunded, OperationRefunded:
		default:
			return fmt.Errorf("%w: operation provider submission ownership changed", ErrConflict)
		}
		if current.CurrentObjectToken.Valid && current.CurrentObjectToken.String != object.Token {
			return fmt.Errorf("%w: operation provider submission object changed", ErrConflict)
		}
		if !current.CurrentObjectToken.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET current_object_token=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND current_object_token IS NULL`, object.Token, operation.ID); err != nil {
				return err
			}
			if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "ProviderSubmissionConfirmedAfterEvent", string(current.Status), string(current.Status), operation.Reason, object.Token); err != nil {
				return err
			}
		}
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Submitted',current_object_token=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Submitting'`, object.Token, operation.ID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "ProviderSubmissionAccepted", "Submitting", "Submitted", operation.Reason, object.Token); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) recordExecutionFailure(ctx context.Context, operationID uint64, providerErr error) {
	status := OperationFailed
	code := "provider_rejected"
	if errors.Is(providerErr, ErrProviderUnavailable) || errors.Is(providerErr, context.Canceled) || errors.Is(providerErr, context.DeadlineExceeded) {
		status = OperationApproved
		code = "provider_unavailable"
	}
	var typed *ProviderError
	if errors.As(providerErr, &typed) && typed.Code != "" {
		code = typed.Code
	}
	if !safeErrorCode(code) {
		code = "provider_error"
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(cleanupCtx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return
	}
	defer tx.Rollback()
	operation, err := operationByIDTx(cleanupCtx, tx, operationID, true)
	if err != nil || operation.Status != OperationSubmitting {
		return
	}
	if _, err := tx.ExecContext(cleanupCtx, `UPDATE hosted_operation SET status=?,failure_code=?,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Submitting'`, status, code, operationID); err != nil {
		return
	}
	if err := insertAudit(cleanupCtx, tx, "Operation", operationID, "system:provider", "ProviderSubmissionFailed", "Submitting", string(status), code, ""); err != nil {
		return
	}
	_ = tx.Commit()
}

func (s *Service) CancelOperation(ctx context.Context, actor Actor, operationID uint64, expectedVersion uint32, reason string) error {
	operation, err := s.Operation(ctx, operationID)
	if err != nil {
		return err
	}
	if err := authorize(actor, PermissionOperationCancel, Scope{PartyType: operation.PartyType, PartyID: operation.PartyID}, true); err != nil {
		return err
	}
	if validateReason(reason) != nil {
		return fmt.Errorf("bounded cancellation reason is required")
	}
	if operation.Status == OperationProposed || operation.Status == OperationApproved {
		if operation.AttemptCount != 0 {
			return fmt.Errorf("%w: an uncertain provider submission must remain reserved and be reconciled", ErrConflict)
		}
		return s.localCancel(ctx, actor, operation, expectedVersion, reason)
	}
	if operation.Kind != OperationFunding || !operation.CurrentObjectToken.Valid ||
		(operation.Status != OperationSubmitted && operation.Status != OperationCanceling) || operation.Version != expectedVersion {
		return fmt.Errorf("%w: this submitted operation cannot be canceled", ErrConflict)
	}
	if operation.Status == OperationSubmitted {
		tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return err
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Canceling',version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Submitted' AND version=?`, operationID, expectedVersion)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return ErrConflict
		}
		if err := insertAudit(ctx, tx, "Operation", operationID, actorName(actor), "CheckoutCancellationStarted", "Submitted", "Canceling", reason, operation.CurrentObjectToken.String); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if err := s.Provider.ExpireFundingCheckout(ctx, operation.CurrentObjectToken.String, providerKey("cancel", operation.RequestKey)); err != nil {
		if s.recordCancellationFailure(operationID) {
			return nil
		}
		return err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := operationByIDTx(ctx, tx, operationID, true)
	if err != nil {
		return err
	}
	if current.Status == OperationCanceled {
		return tx.Commit()
	}
	if current.Status != OperationCanceling {
		return ErrConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Canceled',failure_code=NULL,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Canceling'`, operationID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Operation", operationID, actorName(actor), "CheckoutCanceled", string(current.Status), "Canceled", reason, operation.CurrentObjectToken.String); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) recordCancellationFailure(operationID uint64) bool {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(cleanupCtx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false
	}
	defer tx.Rollback()
	operation, err := operationByIDTx(cleanupCtx, tx, operationID, true)
	if err != nil {
		return false
	}
	if operation.Status == OperationCanceled {
		return true
	}
	if operation.Status != OperationCanceling {
		return false
	}
	result, err := tx.ExecContext(cleanupCtx, `UPDATE hosted_operation SET status='Submitted',version=version+1,failure_code='cancel_failed',updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status='Canceling'`, operationID)
	if err != nil {
		return false
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return false
	}
	if err := insertAudit(cleanupCtx, tx, "Operation", operationID, "system:provider", "CheckoutCancellationFailed", "Canceling", "Submitted", "cancel_failed", operation.CurrentObjectToken.String); err != nil {
		return false
	}
	_ = tx.Commit()
	return false
}

func (s *Service) localCancel(ctx context.Context, actor Actor, operation Operation, expectedVersion uint32, reason string) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE hosted_operation SET status='Canceled',failure_code=NULL,version=version+1,updated_at=UTC_TIMESTAMP() WHERE operation_id=? AND status=? AND version=?`, operation.ID, operation.Status, expectedVersion)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Operation", operation.ID, actorName(actor), "OperationCanceled", string(operation.Status), "Canceled", reason, ""); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) Binding(ctx context.Context, id uint64) (Binding, error) {
	return s.bindingByID(ctx, id)
}

func (s *Service) bindingByID(ctx context.Context, id uint64) (Binding, error) {
	row := s.DB.QueryRowContext(ctx, bindingSelect+` WHERE binding_id=?`, id)
	return scanBinding(row)
}

func (s *Service) bindingByRequest(ctx context.Context, requestKey string) (Binding, bool, error) {
	row := s.DB.QueryRowContext(ctx, bindingSelect+` WHERE request_key=?`, requestKey)
	binding, err := scanBinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Binding{}, false, nil
	}
	return binding, err == nil, err
}

func (s *Service) ListBindings(ctx context.Context, actor Actor, scope Scope) ([]Binding, error) {
	if err := authorize(actor, PermissionRead, scope, false); err != nil {
		return nil, err
	}
	query := bindingSelect
	var args []any
	if scope.PartyType != "" {
		query += ` WHERE party_type=? AND party_id=?`
		args = append(args, scope.PartyType, scope.PartyID)
	}
	query += ` ORDER BY binding_id DESC LIMIT 200`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Binding
	for rows.Next() {
		binding, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, binding)
	}
	return result, rows.Err()
}

func (s *Service) Operation(ctx context.Context, id uint64) (Operation, error) {
	row := s.DB.QueryRowContext(ctx, operationSelect+` WHERE operation_id=?`, id)
	return scanOperation(row)
}

func (s *Service) ListOperations(ctx context.Context, actor Actor, scope Scope) ([]Operation, error) {
	if err := authorize(actor, PermissionRead, scope, false); err != nil {
		return nil, err
	}
	query := operationSelect
	var args []any
	if scope.PartyType != "" {
		query += ` WHERE party_type=? AND party_id=?`
		args = append(args, scope.PartyType, scope.PartyID)
	}
	query += ` ORDER BY operation_id DESC LIMIT 200`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Operation
	for rows.Next() {
		operation, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, operation)
	}
	return result, rows.Err()
}

const bindingSelect = `SELECT binding_id,request_key,provider,party_type,party_id,binding_kind,
provider_token,country,status,provider_ready,version,created_by,approved_by,revoked_by,reason,created_at,updated_at
FROM hosted_binding`

const operationSelect = `SELECT operation_id,request_key,provider,operation_kind,parent_operation_id,
binding_id,statement_id,party_type,party_id,CAST(amount AS CHAR),currency,status,current_object_token,
version,created_by,approved_by,executed_by,reason,failure_code,attempt_count,
provider_event_created_at,created_at,updated_at FROM hosted_operation`

type scanner interface{ Scan(...any) error }

func scanBinding(row scanner) (Binding, error) {
	var binding Binding
	err := row.Scan(&binding.ID, &binding.RequestKey, &binding.Provider, &binding.PartyType, &binding.PartyID,
		&binding.Kind, &binding.ProviderToken, &binding.Country, &binding.Status, &binding.ProviderReady, &binding.Version,
		&binding.CreatedBy, &binding.ApprovedBy, &binding.RevokedBy, &binding.Reason, &binding.CreatedAt, &binding.UpdatedAt)
	return binding, err
}

func bindingCountryMatches(binding Binding, country string) bool {
	if country == "" {
		return !binding.Country.Valid
	}
	return binding.Country.Valid && binding.Country.String == country
}

func scanOperation(row scanner) (Operation, error) {
	var operation Operation
	var amount string
	err := row.Scan(&operation.ID, &operation.RequestKey, &operation.Provider, &operation.Kind,
		&operation.ParentOperationID, &operation.BindingID, &operation.StatementID, &operation.PartyType, &operation.PartyID,
		&amount, &operation.Currency, &operation.Status, &operation.CurrentObjectToken, &operation.Version,
		&operation.CreatedBy, &operation.ApprovedBy, &operation.ExecutedBy, &operation.Reason,
		&operation.FailureCode, &operation.AttemptCount, &operation.ProviderEventCreatedAt,
		&operation.CreatedAt, &operation.UpdatedAt)
	if err == nil {
		operation.Amount, err = accounting.ParseMoney(amount)
	}
	return operation, err
}

func bindingByIDTx(ctx context.Context, tx *sql.Tx, id uint64, lock bool) (Binding, error) {
	query := bindingSelect + ` WHERE binding_id=?`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanBinding(tx.QueryRowContext(ctx, query, id))
}

func operationByIDTx(ctx context.Context, tx *sql.Tx, id uint64, lock bool) (Operation, error) {
	query := operationSelect + ` WHERE operation_id=?`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanOperation(tx.QueryRowContext(ctx, query, id))
}

func executionBinding(ctx context.Context, tx *sql.Tx, operation Operation) (sql.NullInt64, string, error) {
	if operation.Kind != OperationFunding && operation.Kind != OperationPayout {
		return sql.NullInt64{}, "", nil
	}
	kind := BindingPayoutAccount
	if operation.Kind == OperationFunding {
		kind = BindingFundingCustomer
	}
	if operation.BindingID.Valid {
		binding, err := bindingByIDTx(ctx, tx, uint64(operation.BindingID.Int64), true)
		if err != nil {
			return sql.NullInt64{}, "", fmt.Errorf("selected provider binding: %w", err)
		}
		if binding.PartyType != operation.PartyType || binding.PartyID != operation.PartyID || binding.Kind != kind {
			return sql.NullInt64{}, "", fmt.Errorf("%w: selected provider binding belongs to another operation", ErrConflict)
		}
		return operation.BindingID, binding.ProviderToken, nil
	}
	if operation.AttemptCount != 0 {
		return sql.NullInt64{}, "", fmt.Errorf("%w: attempted provider operation has no immutable binding", ErrConflict)
	}
	var id int64
	var token string
	err := tx.QueryRowContext(ctx, `SELECT binding_id,provider_token FROM hosted_binding WHERE party_type=? AND party_id=? AND binding_kind=? AND status='Approved' AND provider_ready=1 ORDER BY binding_id DESC LIMIT 1 FOR UPDATE`, operation.PartyType, operation.PartyID, kind).Scan(&id, &token)
	if err != nil {
		return sql.NullInt64{}, "", fmt.Errorf("approved provider binding: %w", err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, token, nil
}

func refundPaymentIntent(ctx context.Context, tx *sql.Tx, operation Operation) (string, error) {
	if operation.Kind != OperationRefund {
		return "", nil
	}
	if !operation.ParentOperationID.Valid {
		return "", fmt.Errorf("refund parent is missing")
	}
	var paymentIntent string
	err := tx.QueryRowContext(ctx, `SELECT provider_token FROM hosted_provider_object WHERE operation_id=? AND object_kind='PaymentIntent' ORDER BY object_id DESC LIMIT 1`, operation.ParentOperationID.Int64).Scan(&paymentIntent)
	if err != nil {
		return "", fmt.Errorf("refund payment intent: %w", err)
	}
	return paymentIntent, nil
}

func nullableExecutionBindingID(bindingID sql.NullInt64) any {
	if !bindingID.Valid {
		return nil
	}
	return bindingID.Int64
}

func (s *Service) requireParty(ctx context.Context, party PartyType, id uint64) error {
	if id == 0 {
		return fmt.Errorf("party id is required")
	}
	table := "adv"
	column := "adv_id"
	if party == PartyPublisher {
		table, column = "pub", "pub_id"
	} else if party != PartyAdvertiser {
		return fmt.Errorf("party type is invalid")
	}
	var one int
	err := s.DB.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE `+column+`=?`, id).Scan(&one)
	return err
}

func validateStatementForMovement(ctx context.Context, tx *sql.Tx, statementID uint64, party PartyType, partyID uint64, amount accounting.Money) error {
	var actualParty PartyType
	var actualPartyID uint64
	var status accounting.Status
	var totalRaw string
	if err := tx.QueryRowContext(ctx, `SELECT party_type,party_id,status,CAST(total_amount AS CHAR) FROM acct_statement WHERE statement_id=? FOR UPDATE`, statementID).Scan(&actualParty, &actualPartyID, &status, &totalRaw); err != nil {
		return err
	}
	total, err := accounting.ParseMoney(totalRaw)
	if err != nil {
		return err
	}
	if actualParty != party || actualPartyID != partyID {
		return fmt.Errorf("%w: statement belongs to another party", ErrConflict)
	}
	if status != accounting.StatusConfirmed {
		return fmt.Errorf("%w: statement must be Confirmed and not Held before provider movement", ErrConflict)
	}
	if amount > total {
		return fmt.Errorf("%w: provider movement exceeds statement total", ErrConflict)
	}
	return nil
}

func validateRefundParent(ctx context.Context, tx *sql.Tx, parentID, statementID, partyID uint64, amount accounting.Money, requestKey string) error {
	if parentID == 0 {
		return fmt.Errorf("refund parent operation is required")
	}
	parent, err := operationByIDTx(ctx, tx, parentID, true)
	if err != nil {
		return err
	}
	if parent.Kind != OperationFunding || parent.StatementID != statementID || parent.PartyID != partyID || (parent.Status != OperationSucceeded && parent.Status != OperationPartiallyRefunded) {
		return fmt.Errorf("%w: refund parent is not a succeeded funding operation", ErrConflict)
	}
	var committedRaw string
	if err := tx.QueryRowContext(ctx, `SELECT CAST(COALESCE(SUM(amount),0) AS CHAR) FROM hosted_operation WHERE parent_operation_id=? AND operation_kind='Refund' AND status NOT IN ('Failed','Canceled') AND request_key<>?`, parentID, requestKey).Scan(&committedRaw); err != nil {
		return err
	}
	committed, err := accounting.ParseMoney(committedRaw)
	if err != nil {
		return err
	}
	want, err := committed.Add(amount)
	if err != nil || want > parent.Amount {
		return fmt.Errorf("%w: refund exceeds remaining funded amount", ErrConflict)
	}
	return nil
}

func validateStatementCapacity(ctx context.Context, tx *sql.Tx, statementID uint64, kind OperationKind, amount accounting.Money, requestKey string) error {
	var totalRaw, committedRaw string
	if err := tx.QueryRowContext(ctx, `SELECT CAST(total_amount AS CHAR) FROM acct_statement WHERE statement_id=?`, statementID).Scan(&totalRaw); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT CAST(COALESCE(SUM(amount),0) AS CHAR) FROM hosted_operation WHERE statement_id=? AND operation_kind=? AND status NOT IN ('Failed','Canceled') AND request_key<>?`, statementID, kind, requestKey).Scan(&committedRaw); err != nil {
		return err
	}
	total, err := accounting.ParseMoney(totalRaw)
	if err != nil {
		return err
	}
	committed, err := accounting.ParseMoney(committedRaw)
	if err != nil {
		return err
	}
	want, err := committed.Add(amount)
	if err != nil || want > total {
		return fmt.Errorf("%w: active provider operations exceed the statement total", ErrConflict)
	}
	return nil
}

func validateProviderReplayWindow(ctx context.Context, tx *sql.Tx, operationID uint64) error {
	var ageSeconds sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT TIMESTAMPDIFF(SECOND,MIN(created_at),UTC_TIMESTAMP())
FROM hosted_audit
WHERE object_type='Operation' AND object_id=? AND event='ProviderSubmissionStarted'`, operationID).Scan(&ageSeconds); err != nil {
		return err
	}
	if !ageSeconds.Valid || ageSeconds.Int64 < 0 || ageSeconds.Int64 >= int64(providerIdempotencyReplayWindow/time.Second) {
		return fmt.Errorf("%w: provider idempotency replay window is unavailable or expired; reconcile without resubmitting", ErrConflict)
	}
	return nil
}

func operationPartyPermission(kind OperationKind) (PartyType, string, error) {
	switch kind {
	case OperationFunding:
		return PartyAdvertiser, PermissionCheckoutPropose, nil
	case OperationPayout:
		return PartyPublisher, PermissionPayoutPropose, nil
	case OperationRefund:
		return PartyAdvertiser, PermissionRefundPropose, nil
	default:
		return "", "", fmt.Errorf("operation kind is invalid")
	}
}

func authorize(actor Actor, permission string, scope Scope, recentMFA bool) error {
	if actor.Role == "" || actor.ID == "" || !validActorPart(actor.Role) || !validActorPart(actor.ID) || len(actor.Role)+1+len(actor.ID) > 128 {
		return fmt.Errorf("authenticated hosted-payment actor is invalid")
	}
	if strings.HasPrefix(actor.ID, "unix-uid:") {
		return fmt.Errorf("offline maintenance principal cannot authorize hosted-payment actions")
	}
	if actor.Permissions == nil || (!actor.Permissions[permission] && !actor.Permissions["*"]) {
		return fmt.Errorf("hosted-payment permission %s is required", permission)
	}
	if recentMFA && !actor.RecentMFA {
		return fmt.Errorf("recent MFA is required")
	}
	if scope.PartyType != PartyAdvertiser && scope.PartyType != PartyPublisher {
		if actor.Scope.PartyType != "" {
			return fmt.Errorf("global scope is not authorized")
		}
		return nil
	}
	if scope.PartyID == 0 {
		return fmt.Errorf("exact party scope is required")
	}
	if actor.Scope.PartyType != "" && (actor.Scope.PartyType != scope.PartyType || actor.Scope.PartyID != scope.PartyID) {
		return fmt.Errorf("cross-account hosted-payment access is denied")
	}
	return nil
}

func authorizeMaintenance(actor Actor, permission string) error {
	if permission != PermissionRetentionPrune || actor.Role != "admin" || !validUnixActorID(actor.ID) ||
		len(actor.Role)+1+len(actor.ID) > 128 || actor.Scope != (Scope{}) || actor.RecentMFA ||
		len(actor.Permissions) != 1 || !actor.Permissions[permission] {
		return fmt.Errorf("authenticated hosted-payment maintenance principal is invalid")
	}
	return nil
}

func validUnixActorID(value string) bool {
	if !strings.HasPrefix(value, "unix-uid:") || len(value) == len("unix-uid:") {
		return false
	}
	raw := strings.TrimPrefix(value, "unix-uid:")
	uid, err := strconv.ParseUint(raw, 10, 64)
	return err == nil && strconv.FormatUint(uid, 10) == raw
}

func validActorPart(value string) bool {
	if len(value) < 1 || len(value) > 96 {
		return false
	}
	for _, r := range value {
		if r != '@' && r != '.' && r != '_' && r != '-' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func actorName(actor Actor) string { return actor.Role + ":" + actor.ID }

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func validateMutation(requestKey, reason string) error {
	if !safeIdempotencyKey.MatchString(requestKey) || containsSensitivePaymentMaterial(requestKey) {
		return fmt.Errorf("bounded request key is required")
	}
	return validateReason(reason)
}

func validateReason(reason string) error {
	if !utf8.ValidString(reason) || strings.TrimSpace(reason) == "" || utf8.RuneCountInString(reason) > 500 || containsSensitivePaymentMaterial(reason) {
		return fmt.Errorf("bounded reason is required")
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return fmt.Errorf("bounded reason must not contain control characters")
		}
	}
	return nil
}

func containsSensitivePaymentMaterial(value string) bool {
	lower := strings.ToLower(value)
	for _, prefix := range []string{"sk_live_", "sk_test_", "rk_live_", "rk_test_", "whsec_"} {
		if strings.Contains(lower, prefix) {
			return true
		}
	}
	if ibanLike.MatchString(value) {
		return true
	}
	digits := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
			if digits >= 9 {
				return true
			}
		case r == ' ' || r == '-' || r == '.' || r == '/' || r == '(' || r == ')':
			// Card, routing, and account numbers commonly use these separators.
		default:
			digits = 0
		}
	}
	return false
}

func providerKey(namespace, requestKey string) string {
	digest := sha256.Sum256([]byte(namespace + ":" + requestKey))
	return "aofei:" + namespace + ":" + fmt.Sprintf("%x", digest[:16])
}

func insertAudit(ctx context.Context, tx *sql.Tx, objectType string, objectID uint64, actor, event, prior, next, reason, token string) error {
	var digest any
	if token != "" {
		hash := sha256.Sum256([]byte(token))
		digest = hash[:]
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO hosted_audit (object_type,object_id,actor,event,prior_state,new_state,reason,provider_token_sha256,created_at) VALUES (?,?,?,?,NULLIF(?,''),NULLIF(?,''),?,?,UTC_TIMESTAMP())`, objectType, objectID, actor, event, prior, next, reason, digest)
	return err
}

func providerObjectKindForBinding(kind BindingKind) string {
	if kind == BindingPayoutAccount {
		return "PayoutAccount"
	}
	return "Customer"
}

func providerObjectKindForOperation(kind OperationKind) string {
	switch kind {
	case OperationPayout:
		return "Transfer"
	case OperationRefund:
		return "Refund"
	default:
		return "Checkout"
	}
}

func prefixesForOperation(kind OperationKind) []string {
	switch kind {
	case OperationPayout:
		return []string{"tr_"}
	case OperationRefund:
		return []string{"re_"}
	default:
		return []string{"cs_"}
	}
}
