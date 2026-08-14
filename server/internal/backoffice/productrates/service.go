package productrates

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	defaultPageLimit int32 = 50
	maximumPageLimit int32 = 100
)

var marketCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)
var identifierPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) List(ctx context.Context, input ListInput) (Page, error) {
	limit, offset, err := validatePage(input.Limit, input.Offset)
	if err != nil {
		return Page{}, err
	}
	items, err := service.repository.List(ctx, limit+1, offset)
	if err != nil {
		return Page{}, apperrors.NewInternal("Unable to list product rates", err)
	}
	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	return Page{Data: items, Limit: limit, Offset: offset, HasMore: hasMore}, nil
}

func (service *Service) Get(ctx context.Context, id string) (ProductRate, error) {
	rateID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return ProductRate{}, apperrors.NewBadRequest("Invalid product rate ID")
	}
	item, err := service.repository.Get(ctx, rateID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProductRate{}, apperrors.NewNotFound("Product rate not found")
		}
		return ProductRate{}, apperrors.NewInternal("Unable to get product rate", err)
	}
	return item, nil
}

func (service *Service) Create(ctx context.Context, input CreateInput) (ProductRate, error) {
	input.Product = strings.ToLower(strings.TrimSpace(input.Product))
	input.Meter = strings.ToLower(strings.TrimSpace(input.Meter))
	input.BillingMarket = strings.ToUpper(strings.TrimSpace(input.BillingMarket))
	input.Tier = strings.ToLower(strings.TrimSpace(input.Tier))
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if !identifierPattern.MatchString(input.Product) || !identifierPattern.MatchString(input.Meter) {
		return ProductRate{}, apperrors.NewBadRequest("Product and meter must use lowercase letters, numbers, or underscores")
	}
	if !marketCodePattern.MatchString(input.BillingMarket) {
		return ProductRate{}, apperrors.NewBadRequest("Billing market must be a two-letter ISO country code")
	}
	if !validTier(input.Tier) {
		return ProductRate{}, apperrors.NewBadRequest("Tier must be growth, scale, or enterprise")
	}
	if !currencyCodePattern.MatchString(input.Currency) {
		return ProductRate{}, apperrors.NewBadRequest("Currency must be a three-letter ISO code")
	}
	if input.CostUnits <= 0 {
		return ProductRate{}, apperrors.NewBadRequest("Cost units must be greater than zero")
	}
	if input.EffectiveFrom.IsZero() {
		return ProductRate{}, apperrors.NewBadRequest("Effective from is required")
	}
	if input.EffectiveUntil != nil && !input.EffectiveUntil.After(input.EffectiveFrom) {
		return ProductRate{}, apperrors.NewBadRequest("Effective until must be after effective from")
	}
	item, err := service.repository.Create(ctx, input)
	if err != nil {
		switch {
		case postgresCode(err, "23P01"):
			return ProductRate{}, apperrors.NewConflict("Product rate overlaps an existing rate")
		case postgresCode(err, "23503"):
			return ProductRate{}, apperrors.NewBadRequest("Billing market and currency are not configured together")
		default:
			return ProductRate{}, apperrors.NewInternal("Unable to create product rate", err)
		}
	}
	return item, nil
}

func (service *Service) Close(ctx context.Context, id string, input CloseInput) (ProductRate, error) {
	if strings.TrimSpace(input.Reason) == "" {
		return ProductRate{}, apperrors.NewBadRequest("Reason is required")
	}
	rateID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return ProductRate{}, apperrors.NewBadRequest("Invalid product rate ID")
	}
	if input.EffectiveUntil.IsZero() {
		return ProductRate{}, apperrors.NewBadRequest("Effective until is required")
	}
	item, err := service.repository.Close(ctx, rateID, input.EffectiveUntil)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProductRate{}, apperrors.NewInternal("Unable to close product rate", err)
	}
	if _, getErr := service.repository.Get(ctx, rateID); errors.Is(getErr, pgx.ErrNoRows) {
		return ProductRate{}, apperrors.NewNotFound("Product rate not found")
	}
	return ProductRate{}, apperrors.NewConflict("Effective until must shorten the active rate period")
}

func validTier(value string) bool {
	return value == "growth" || value == "scale" || value == "enterprise"
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
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == code
}

func (service *Service) Replace(ctx context.Context, id string, input ReplaceInput) (ProductRate, error) {
	if strings.TrimSpace(input.Reason) == "" {
		return ProductRate{}, apperrors.NewBadRequest("Reason is required")
	}
	current, err := service.Get(ctx, id)
	if err != nil {
		return ProductRate{}, err
	}
	if input.Replacement.EffectiveFrom.IsZero() {
		return ProductRate{}, apperrors.NewBadRequest("Replacement effective from is required")
	}
	if input.Replacement.CostUnits <= 0 {
		return ProductRate{}, apperrors.NewBadRequest("Cost units must be greater than zero")
	}
	if input.Replacement.EffectiveUntil != nil && !input.Replacement.EffectiveUntil.After(input.Replacement.EffectiveFrom) {
		return ProductRate{}, apperrors.NewBadRequest("Effective until must be after effective from")
	}
	input.Replacement.Product, input.Replacement.Meter, input.Replacement.BillingMarket, input.Replacement.Tier, input.Replacement.Currency = current.Product, current.Meter, current.BillingMarket, current.Tier, current.Currency
	rateID, _ := uuid.Parse(strings.TrimSpace(id))
	item, err := service.repository.Replace(ctx, rateID, input.Replacement)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductRate{}, apperrors.NewConflict("Replacement effective from must shorten the active rate period")
	}
	if postgresCode(err, "23P01") {
		return ProductRate{}, apperrors.NewConflict("Product rate overlaps an existing rate")
	}
	if err != nil {
		return ProductRate{}, apperrors.NewInternal("Unable to replace product rate", err)
	}
	return item, nil
}
