package subscription

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidPeriod = errors.New("subscription charge period is invalid")

type charger interface {
	ChargePeriod(context.Context, pgx.Tx, Input) (Result, error)
}
type Service struct{ repository charger }

func NewService(repository charger) *Service { return &Service{repository: repository} }
func (s *Service) ChargePeriod(ctx context.Context, tx pgx.Tx, input Input) (Result, error) {
	if input.SubscriptionID == uuid.Nil || input.TeamID == uuid.Nil || input.PlanCode == "" || input.PeriodStart.IsZero() || !input.PeriodEnd.After(input.PeriodStart) {
		return Result{}, ErrInvalidPeriod
	}
	return s.repository.ChargePeriod(ctx, tx, input)
}
