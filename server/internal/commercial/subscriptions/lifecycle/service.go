package lifecycle

import "time"

type Service struct{}

func NewService() *Service                                           { return &Service{} }
func (*Service) Decide(state State, now time.Time) (Decision, error) { return Decide(state, now) }
