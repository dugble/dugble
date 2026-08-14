package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	hubteladapter "github.com/dugble/dugble/server/internal/adapters/hubtel"
	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/billing/payment"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type store interface {
	Get(context.Context, uuid.UUID) (Wallet, error)
	ListLedger(context.Context, uuid.UUID, int32, int32) ([]LedgerEntry, error)
	Credit(context.Context, uuid.UUID, int64, string) (Wallet, error)
}

type PaymentProvider interface {
	InitiateCheckout(context.Context, hubteladapter.InitiateCheckoutRequest) (hubteladapter.InitiateCheckoutResponse, error)
	VerifyTransaction(context.Context, string) (hubteladapter.PaymentStatus, error)
	MapCallback(hubteladapter.CallbackPayload) (hubteladapter.PaymentStatus, error)
}

type paymentService interface {
	Create(context.Context, payment.CreateInput) (payment.Transaction, error)
	Complete(context.Context, payment.CompleteInput) (payment.Transaction, error)
}

type Service struct {
	repository  store
	hubtel      PaymentProvider
	payments    paymentService
	frontendURL string
	backendURL  string
}

type ServiceConfig struct {
	FrontendURL string
	BackendURL  string
}

func NewService(repository store, cfg ServiceConfig, hubtel PaymentProvider, payments paymentService) *Service {
	return &Service{
		repository:  repository,
		hubtel:      hubtel,
		payments:    payments,
		frontendURL: strings.TrimRight(strings.TrimSpace(cfg.FrontendURL), "/"),
		backendURL:  strings.TrimRight(strings.TrimSpace(cfg.BackendURL), "/"),
	}
}

func (s *Service) Get(ctx context.Context) (Wallet, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionWalletRead)
	if !decision.Allowed {
		return Wallet{}, apperrors.NewForbidden(decision.Reason)
	}
	wallet, err := s.repository.Get(ctx, access.Scope.TeamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Wallet{}, apperrors.NewNotFound("Wallet not found")
		}
		return Wallet{}, apperrors.NewInternal("Unable to get wallet", err)
	}
	return wallet, nil
}

func (s *Service) ListLedger(ctx context.Context, limit int32, offset int32) (LedgerPage, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionWalletLedgerRead)
	if !decision.Allowed {
		return LedgerPage{}, apperrors.NewForbidden(decision.Reason)
	}
	limit, offset, err := validateLedgerPage(limit, offset)
	if err != nil {
		return LedgerPage{}, err
	}
	entries, err := s.repository.ListLedger(ctx, access.Scope.TeamID, limit, offset)
	if err != nil {
		return LedgerPage{}, apperrors.NewInternal("Unable to list wallet ledger", err)
	}
	return LedgerPage{Entries: entries, Limit: limit, Offset: offset}, nil
}

func (s *Service) TopUp(ctx context.Context, req TopUpRequest) (TopUpResponse, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionTeamUpdate)
	if !decision.Allowed {
		return TopUpResponse{}, apperrors.NewForbidden(decision.Reason)
	}
	if s.hubtel == nil || s.payments == nil {
		return TopUpResponse{}, apperrors.NewServiceUnavailable("Wallet top-ups are not configured", nil)
	}
	if req.AmountUnits <= 0 {
		return TopUpResponse{}, apperrors.NewBadRequest("Top-up amount must be greater than zero")
	}

	clientReference := "dgb-" + uuid.NewString()
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "Dugble wallet top-up"
	}

	transaction, err := s.payments.Create(ctx, payment.CreateInput{
		TeamID:          access.Scope.TeamID.String(),
		Provider:        payment.ProviderHubtel,
		ClientReference: clientReference,
		Currency:        payment.CurrencyGHS,
		AmountUnits:     req.AmountUnits,
	})
	if err != nil {
		return TopUpResponse{}, err
	}

	checkout, err := s.hubtel.InitiateCheckout(ctx, hubteladapter.InitiateCheckoutRequest{
		TotalAmount:     float64(req.AmountUnits) / 100,
		Description:     description,
		CallbackURL:     s.backendURL + "/wallet/webhook/hubtel",
		ReturnURL:       s.frontendURL + "/dashboard/billing/transactions",
		CancellationURL: s.frontendURL + "/dashboard/billing/transactions",
		ClientReference: clientReference,
	})
	if err != nil {
		return TopUpResponse{}, apperrors.NewServiceUnavailable("Unable to initiate Hubtel checkout", err)
	}
	if checkout.ResponseCode != "0000" || !strings.EqualFold(strings.TrimSpace(checkout.Status), "Success") {
		return TopUpResponse{}, apperrors.NewBadRequest("Hubtel checkout was not accepted")
	}

	return TopUpResponse{
		TransactionID:     transaction.ID,
		ClientReference:   clientReference,
		CheckoutID:        checkout.Data.CheckoutID,
		CheckoutURL:       checkout.Data.CheckoutURL,
		CheckoutDirectURL: checkout.Data.CheckoutDirectURL,
	}, nil
}

func (s *Service) HandleHubtelCallback(ctx context.Context, payload hubteladapter.CallbackPayload) (*payment.Transaction, error) {
	if s.hubtel == nil || s.payments == nil {
		return nil, apperrors.NewServiceUnavailable("Wallet top-ups are not configured", nil)
	}
	if payload.ResponseCode != "0000" || !strings.EqualFold(strings.TrimSpace(payload.Status), "Success") || !hubteladapter.IsPaidStatus(payload.Data.Status) {
		return nil, nil
	}

	callbackStatus, err := s.hubtel.MapCallback(payload)
	if err != nil {
		return nil, apperrors.NewBadRequest("Invalid Hubtel callback payload")
	}
	callbackStatus.ClientReference = strings.TrimSpace(callbackStatus.ClientReference)
	if callbackStatus.ClientReference == "" {
		return nil, apperrors.NewBadRequest("Hubtel callback client reference is required")
	}

	verifiedStatus, err := s.hubtel.VerifyTransaction(ctx, callbackStatus.ClientReference)
	if err != nil {
		return nil, apperrors.NewServiceUnavailable("Unable to verify Hubtel payment", err)
	}
	if !hubteladapter.IsPaidStatus(verifiedStatus.Status) {
		return nil, nil
	}
	verifiedStatus.ClientReference = strings.TrimSpace(verifiedStatus.ClientReference)
	if verifiedStatus.ClientReference != callbackStatus.ClientReference {
		return nil, apperrors.NewBadRequest("Hubtel transaction reference does not match callback")
	}

	var verified hubteladapter.TransactionStatusResponse
	if err := json.Unmarshal(verifiedStatus.Raw, &verified); err != nil {
		return nil, apperrors.NewBadRequest("Invalid Hubtel transaction status")
	}
	amount, err := amountUnits(verified.Data.Amount)
	if err != nil {
		return nil, apperrors.NewBadRequest("Invalid Hubtel payment amount")
	}
	providerTransactionID := strings.TrimSpace(verified.Data.TransactionID)
	if providerTransactionID == "" {
		providerTransactionID = strings.TrimSpace(verified.Data.ExternalTransactionID)
	}
	if providerTransactionID == "" {
		return nil, apperrors.NewBadRequest("Hubtel transaction id is required")
	}

	transaction, err := s.payments.Complete(ctx, payment.CompleteInput{
		Provider:              payment.ProviderHubtel,
		ClientReference:       verifiedStatus.ClientReference,
		ProviderTransactionID: providerTransactionID,
		AmountUnits:           amount,
	})
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// Credit is intentionally not exposed by the public wallet routes. A future
// payment provider integration should call it only after verifying a payment.
func (s *Service) Credit(ctx context.Context, input CreditInput) (Wallet, error) {
	teamID, amountUnits, referenceID, err := validateCredit(input)
	if err != nil {
		return Wallet{}, err
	}
	wallet, err := s.repository.Credit(ctx, teamID, amountUnits, referenceID)
	if err != nil {
		return Wallet{}, apperrors.NewInternal("Unable to credit wallet", err)
	}
	return wallet, nil
}

func amountUnits(amount float64) (int64, error) {
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("invalid amount")
	}
	units := amount * 100
	rounded := math.Round(units)
	if math.Abs(units-rounded) > 1e-6 || rounded > math.MaxInt64 {
		return 0, fmt.Errorf("amount has unsupported precision")
	}
	return int64(rounded), nil
}
