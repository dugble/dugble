package plan

import (
	"context"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/security/authz"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type store interface {
	List(context.Context, uuid.UUID) ([]Plan, error)
}

type Service struct{ repository store }

func NewService(repository store) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context) ([]Plan, error) {
	access, decision := authz.ResolveAccess(ctx, authz.PermissionWalletRead)
	if !decision.Allowed {
		return nil, apperrors.NewForbidden(decision.Reason)
	}
	plans, err := s.repository.List(ctx, access.Scope.TeamID)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list billing plans", err)
	}
	return plans, nil
}
