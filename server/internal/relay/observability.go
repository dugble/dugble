package relay

import (
	"context"
	"time"
)

// Channel identifies the communications channel that emitted an event.
type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
)

// EventKind identifies a delivery lifecycle event emitted by Relay.
type EventKind string

const (
	EventRouteSelected   EventKind = "route_selected"
	EventProviderSkipped EventKind = "provider_skipped"
	EventAttemptStarted  EventKind = "attempt_started"
	EventAttemptFinished EventKind = "attempt_finished"
	EventRouteExhausted  EventKind = "route_exhausted"
)

// Outcome is kept as an alias for observability compatibility. Provider
// outcomes use the same channel-neutral submission state as delivery results.
type Outcome = SubmissionState

const (
	OutcomeAccepted = SubmissionAccepted
	OutcomeRejected = SubmissionRejected
	OutcomeUnknown  = SubmissionUnknown
)

// EventReason provides stable machine-readable context for skip and exhaustion
// events.
type EventReason string

const (
	ReasonUnsupportedCapability EventReason = "unsupported_capability"
	ReasonProviderUnavailable   EventReason = "provider_unavailable"
	ReasonNoCapableProviders    EventReason = "no_capable_providers"
	ReasonNoAvailableProviders  EventReason = "no_available_providers"
	ReasonAllRejected           EventReason = "all_rejected"
)

// Event is a channel-neutral observability signal. It intentionally excludes
// message content, destinations, sender values, and raw provider errors so
// observers do not receive communication or provider-response payloads by
// default.
type Event struct {
	Kind              EventKind
	Channel           Channel
	Provider          string
	Providers         []string
	Outcome           Outcome
	Reason            EventReason
	ProviderMessageID string
	Duration          time.Duration
	HadError          bool
}

// Observer receives delivery lifecycle events synchronously. Implementations
// should return quickly and move expensive work to their own queue or worker.
// Observer errors are intentionally not part of this contract.
type Observer interface {
	Observe(context.Context, Event)
}

// ObserverFunc adapts a function into an Observer.
type ObserverFunc func(context.Context, Event)

func (fn ObserverFunc) Observe(ctx context.Context, event Event) {
	if fn != nil {
		fn(ctx, event)
	}
}
