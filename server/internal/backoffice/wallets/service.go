package wallets

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	defaultPageLimit int32 = 50
	maximumPageLimit int32 = 100
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, input ListInput) (WalletPage, error) {
	limit, offset, err := validatePage(input.Limit, input.Offset)
	if err != nil {
		return WalletPage{}, err
	}
	items, err := s.repository.List(ctx, limit+1, offset)
	if err != nil {
		return WalletPage{}, apperrors.NewInternal("Unable to list wallets", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	return WalletPage{Data: items, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}
func (s *Service) Get(ctx context.Context, id string) (Wallet, error) {
	teamID, err := parseTeamID(id)
	if err != nil {
		return Wallet{}, err
	}
	item, err := s.repository.Get(ctx, teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperrors.NewNotFound("Wallet not found")
	}
	if err != nil {
		return Wallet{}, apperrors.NewInternal("Unable to get wallet", err)
	}
	return item, nil
}
func (s *Service) ListTransactions(ctx context.Context, input TransactionListInput) (TransactionPage, error) {
	limit, offset, err := validatePage(input.Limit, input.Offset)
	if err != nil {
		return TransactionPage{}, err
	}
	var teamID *uuid.UUID
	if strings.TrimSpace(input.TeamID) != "" {
		parsed, parseErr := parseTeamID(input.TeamID)
		if parseErr != nil {
			return TransactionPage{}, parseErr
		}
		teamID = &parsed
	}
	items, err := s.repository.ListTransactions(ctx, teamID, limit+1, offset)
	if err != nil {
		return TransactionPage{}, apperrors.NewInternal("Unable to list wallet transactions", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	return TransactionPage{Data: items, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}
func (s *Service) GetTransaction(ctx context.Context, teamValue, transactionValue string) (Transaction, error) {
	teamID, err := parseTeamID(teamValue)
	if err != nil {
		return Transaction{}, err
	}
	transactionID, err := uuid.Parse(strings.TrimSpace(transactionValue))
	if err != nil {
		return Transaction{}, apperrors.NewBadRequest("Invalid transaction ID")
	}
	item, err := s.repository.GetTransaction(ctx, teamID, transactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, apperrors.NewNotFound("Wallet transaction not found")
	}
	if err != nil {
		return Transaction{}, apperrors.NewInternal("Unable to get wallet transaction", err)
	}
	return item, nil
}
func (s *Service) Adjust(ctx context.Context, id string, input AdjustmentInput) (Wallet, error) {
	input.ReferenceID = strings.TrimSpace(input.ReferenceID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.AmountUnits == 0 {
		return Wallet{}, apperrors.NewBadRequest("Amount units must not be zero")
	}
	if input.ReferenceID == "" {
		return Wallet{}, apperrors.NewBadRequest("Reference ID is required")
	}
	if input.Reason == "" {
		return Wallet{}, apperrors.NewBadRequest("Reason is required")
	}
	actorID, err := uuid.Parse(input.ActorUserID)
	if err != nil {
		return Wallet{}, apperrors.NewBadRequest("Authenticated administrator is invalid")
	}
	teamID, err := parseTeamID(id)
	if err != nil {
		return Wallet{}, err
	}
	if err = s.repository.Adjust(ctx, teamID, input.AmountUnits, input.ReferenceID, input.Reason, actorID, input.SessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, getErr := s.repository.Get(ctx, teamID); errors.Is(getErr, pgx.ErrNoRows) {
				return Wallet{}, apperrors.NewNotFound("Wallet not found")
			}
			return Wallet{}, apperrors.NewConflict("Adjustment would overdraw the wallet or its reference ID was already used")
		}
		return Wallet{}, apperrors.NewInternal("Unable to adjust wallet", err)
	}
	return s.Get(ctx, teamID.String())
}
func parseTeamID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid team ID")
	}
	return id, nil
}
func validatePage(limit, offset int32) (int32, int32, error) {
	if limit < 0 || limit > maximumPageLimit {
		return 0, 0, apperrors.NewBadRequest("Limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, apperrors.NewBadRequest("Offset must not be negative")
	}
	if limit == 0 {
		limit = defaultPageLimit
	}
	return limit, offset, nil
}
