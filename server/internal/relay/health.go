package relay

import "context"

// HealthStatus describes whether a provider should currently receive traffic.
type HealthStatus string

const (
	HealthHealthy     HealthStatus = "healthy"
	HealthDegraded    HealthStatus = "degraded"
	HealthUnavailable HealthStatus = "unavailable"
)

// HealthSource reports provider health to routing. Configuration state such as
// disabled/enabled intentionally does not belong here.
type HealthSource interface {
	Status(context.Context, string) HealthStatus
}

// HealthFunc adapts a function into a HealthSource.
type HealthFunc func(context.Context, string) HealthStatus

func (fn HealthFunc) Status(ctx context.Context, provider string) HealthStatus {
	if fn == nil {
		return HealthHealthy
	}
	return fn(ctx, provider)
}
