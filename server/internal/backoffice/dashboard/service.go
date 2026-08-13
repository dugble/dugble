package dashboard

import "context"

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Operations(ctx context.Context) (Operations, error) {
	return s.repository.Operations(ctx)
}
