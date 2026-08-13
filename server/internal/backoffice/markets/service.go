package markets

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

var marketCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

type repository interface {
	List(context.Context, int32, int32) ([]Market, error)
	Get(context.Context, string) (Market, error)
	GetCurrency(context.Context, string) (bool, error)
	Create(context.Context, CreateInput) (Market, error)
	SetEnabled(context.Context, string, bool) (Market, error)
}

type Service struct{ repository repository }

func NewService(repository repository) *Service { return &Service{repository: repository} }

func (service *Service) List(ctx context.Context, input ListInput) (Page, error) {
	limit, offset, err := validatePage(input.Limit, input.Offset)
	if err != nil {
		return Page{}, err
	}
	items, err := service.repository.List(ctx, limit+1, offset)
	if err != nil {
		return Page{}, apperrors.NewInternal("Unable to list markets", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	return Page{Data: items, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}

func (service *Service) Get(ctx context.Context, code string) (Market, error) {
	code, err := normalizeMarketCode(code)
	if err != nil {
		return Market{}, err
	}
	item, err := service.repository.Get(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return Market{}, apperrors.NewNotFound("Market not found")
	}
	if err != nil {
		return Market{}, apperrors.NewInternal("Unable to get market", err)
	}
	return item, nil
}

func (service *Service) Create(ctx context.Context, input CreateInput) (Market, error) {
	code, err := normalizeMarketCode(input.Code)
	if err != nil {
		return Market{}, err
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if !currencyCodePattern.MatchString(currency) {
		return Market{}, apperrors.NewBadRequest("Currency must be a three-letter ISO code")
	}
	enabled, err := service.repository.GetCurrency(ctx, currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return Market{}, apperrors.NewBadRequest("Currency is not configured")
	}
	if err != nil {
		return Market{}, apperrors.NewInternal("Unable to validate market currency", err)
	}
	requestedEnabled := input.IsEnabled == nil || *input.IsEnabled
	if requestedEnabled && !enabled {
		return Market{}, apperrors.NewBadRequest("Currency must be enabled before enabling the market")
	}
	input.Code, input.Currency = code, currency
	item, err := service.repository.Create(ctx, input)
	if postgresCode(err, "23505") {
		return Market{}, apperrors.NewConflict("Market already exists")
	}
	if postgresCode(err, "23503") {
		return Market{}, apperrors.NewBadRequest("Currency is not configured")
	}
	if err != nil {
		return Market{}, apperrors.NewInternal("Unable to create market", err)
	}
	return item, nil
}

func (service *Service) Update(ctx context.Context, code string, input UpdateInput) (Market, error) {
	code, err := normalizeMarketCode(code)
	if err != nil {
		return Market{}, err
	}
	if input.IsEnabled == nil {
		return Market{}, apperrors.NewBadRequest("is_enabled is required")
	}
	if !*input.IsEnabled && strings.TrimSpace(input.Reason) == "" {
		return Market{}, apperrors.NewBadRequest("reason is required when disabling a market")
	}
	if *input.IsEnabled {
		market, err := service.repository.Get(ctx, code)
		if errors.Is(err, pgx.ErrNoRows) {
			return Market{}, apperrors.NewNotFound("Market not found")
		}
		if err != nil {
			return Market{}, apperrors.NewInternal("Unable to get market", err)
		}
		enabled, err := service.repository.GetCurrency(ctx, market.Currency)
		if err != nil {
			return Market{}, apperrors.NewInternal("Unable to validate market currency", err)
		}
		if !enabled {
			return Market{}, apperrors.NewBadRequest("Currency must be enabled before enabling the market")
		}
	}
	item, err := service.repository.SetEnabled(ctx, code, *input.IsEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return Market{}, apperrors.NewNotFound("Market not found")
	}
	if err != nil {
		return Market{}, apperrors.NewInternal("Unable to update market", err)
	}
	return item, nil
}

func normalizeMarketCode(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !marketCodePattern.MatchString(value) {
		return "", apperrors.NewBadRequest("Market code must be a two-letter ISO country code")
	}
	return value, nil
}

func validatePage(limit, offset int32) (int32, int32, error) {
	if limit < 0 || limit > MaxPageLimit {
		return 0, 0, apperrors.NewBadRequest("Limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, apperrors.NewBadRequest("Offset must not be negative")
	}
	if limit == 0 {
		limit = DefaultPageLimit
	}
	return limit, offset, nil
}

func postgresCode(err error, code string) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == code
}
