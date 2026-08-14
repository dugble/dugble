package email

import (
	"context"
	"strings"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/google/uuid"
)

type Service struct{ repository *Repository }

func NewService(r *Repository) *Service { return &Service{repository: r} }
func (s *Service) List(ctx context.Context, f Filter) ([]Row, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	return s.repository.List(ctx, f)
}
func (s *Service) Detail(ctx context.Context, id string) (Detail, error) {
	uid, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return Detail{}, apperrors.NewBadRequest("Invalid email message ID")
	}
	return s.repository.Detail(ctx, uid)
}
