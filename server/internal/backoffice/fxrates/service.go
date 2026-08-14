package fxrates

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultPageLimit int32 = 50
	maximumPageLimit int32 = 100
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)
var positiveRatePattern = regexp.MustCompile(`^(?:0*[1-9][0-9]*|0*)\.[0-9]*[1-9][0-9]*$|^[1-9][0-9]*(?:\.0*)?$`)

type Service struct{ repository *Repository }

func NewService(r *Repository) *Service { return &Service{repository: r} }

func (s *Service) List(ctx context.Context, in ListInput) (Page, error) {
	limit, offset, err := validatePage(in.Limit, in.Offset)
	if err != nil {
		return Page{}, err
	}
	items, err := s.repository.List(ctx, limit+1, offset)
	if err != nil {
		return Page{}, apperrors.NewInternal("Unable to list FX rates", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	return Page{Data: items, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}

func (s *Service) Get(ctx context.Context, id string) (FXRate, error) {
	rateID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return FXRate{}, apperrors.NewBadRequest("Invalid FX rate ID")
	}
	item, err := s.repository.Get(ctx, rateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return FXRate{}, apperrors.NewNotFound("FX rate not found")
	}
	if err != nil {
		return FXRate{}, apperrors.NewInternal("Unable to get FX rate", err)
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (FXRate, error) {
	if err := normalizeAndValidate(&in.BaseCurrency, &in.QuoteCurrency, &in.Rate, in.EffectiveFrom); err != nil {
		return FXRate{}, err
	}
	if in.EffectiveUntil != nil && !in.EffectiveUntil.After(in.EffectiveFrom) {
		return FXRate{}, apperrors.NewBadRequest("Effective until must be after effective from")
	}
	item, err := s.repository.Create(ctx, in)
	return handleWriteResult(item, err, "create")
}

func (s *Service) Close(ctx context.Context, id string, in CloseInput) (FXRate, error) {
	rateID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return FXRate{}, apperrors.NewBadRequest("Invalid FX rate ID")
	}
	if in.EffectiveUntil.IsZero() {
		return FXRate{}, apperrors.NewBadRequest("Effective until is required")
	}
	if strings.TrimSpace(in.Reason) == "" {
		return FXRate{}, apperrors.NewBadRequest("Reason is required")
	}
	item, err := s.repository.Close(ctx, rateID, in.EffectiveUntil)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FXRate{}, apperrors.NewInternal("Unable to close FX rate", err)
	}
	if _, getErr := s.repository.Get(ctx, rateID); errors.Is(getErr, pgx.ErrNoRows) {
		return FXRate{}, apperrors.NewNotFound("FX rate not found")
	}
	return FXRate{}, apperrors.NewConflict("Effective until must shorten the active rate period")
}

func (s *Service) Replace(ctx context.Context, id string, in ReplaceInput) (FXRate, error) {
	rateID, parseErr := uuid.Parse(strings.TrimSpace(id))
	if parseErr != nil {
		return FXRate{}, apperrors.NewBadRequest("Invalid FX rate ID")
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return FXRate{}, err
	}
	if strings.TrimSpace(in.Reason) == "" {
		return FXRate{}, apperrors.NewBadRequest("Reason is required")
	}
	in.BaseCurrency, in.QuoteCurrency = current.BaseCurrency, current.QuoteCurrency
	if err := normalizeAndValidate(&in.BaseCurrency, &in.QuoteCurrency, &in.Rate, in.EffectiveFrom); err != nil {
		return FXRate{}, err
	}
	item, err := s.repository.Replace(ctx, rateID, in)
	if errors.Is(err, pgx.ErrNoRows) {
		return FXRate{}, apperrors.NewConflict("Replacement effective from must shorten the active rate period")
	}
	return handleWriteResult(item, err, "replace")
}

func normalizeAndValidate(baseCurrency, quoteCurrency, rate *string, effectiveFrom time.Time) error {
	*baseCurrency = strings.ToUpper(strings.TrimSpace(*baseCurrency))
	*quoteCurrency = strings.ToUpper(strings.TrimSpace(*quoteCurrency))
	*rate = strings.TrimSpace(*rate)
	if !currencyCodePattern.MatchString(*baseCurrency) || !currencyCodePattern.MatchString(*quoteCurrency) {
		return apperrors.NewBadRequest("Currencies must be three-letter ISO codes")
	}
	if *baseCurrency == *quoteCurrency {
		return apperrors.NewBadRequest("Base and quote currencies must differ")
	}
	if !positiveRatePattern.MatchString(*rate) {
		return apperrors.NewBadRequest("Rate must be greater than zero")
	}
	if effectiveFrom.IsZero() {
		return apperrors.NewBadRequest("Effective from is required")
	}
	return nil
}

func handleWriteResult(item FXRate, err error, operation string) (FXRate, error) {
	if err == nil {
		return item, nil
	}
	if postgresCode(err, "23P01") || postgresCode(err, "23505") {
		return FXRate{}, apperrors.NewConflict("FX rate overlaps an existing rate")
	}
	if postgresCode(err, "23503") {
		return FXRate{}, apperrors.NewBadRequest("Base and quote currencies must be configured")
	}
	return FXRate{}, apperrors.NewInternal("Unable to "+operation+" FX rate", err)
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

func postgresCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
