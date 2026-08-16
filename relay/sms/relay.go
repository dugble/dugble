package sms

import (
	"context"
	"fmt"
	"time"

	relaycore "github.com/dugble/relay"
)

// Routing errors are retained in sms for source compatibility. New code may
// use the channel-neutral relay errors directly.
var (
	ErrNoProviders          = relaycore.ErrNoProviders
	ErrNoCapableProviders   = relaycore.ErrNoCapableProviders
	ErrNoAvailableProviders = relaycore.ErrNoAvailableProviders
	ErrAllRejected          = relaycore.ErrAllRejected
)

// Relay executes providers in configured order and only falls back after a
// definitive rejection.
type Relay struct {
	providers []Provider
	health    relaycore.HealthSource
	observer  relaycore.Observer
}

func NewRelay(providers ...Provider) (*Relay, error) {
	filtered := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		filtered = append(filtered, provider)
	}
	if len(filtered) == 0 {
		return nil, ErrNoProviders
	}
	return &Relay{providers: filtered}, nil
}

// WithHealth returns a copy of Relay that uses source when ordering providers.
// Unknown health values are treated as healthy for backward compatibility.
func (r *Relay) WithHealth(source relaycore.HealthSource) *Relay {
	if r == nil {
		return nil
	}
	clone := *r
	clone.health = source
	return &clone
}

// WithObserver returns a copy of Relay that emits lifecycle events to observer.
// Observer callbacks run synchronously; panics are isolated from delivery.
func (r *Relay) WithObserver(observer relaycore.Observer) *Relay {
	if r == nil {
		return nil
	}
	clone := *r
	clone.observer = observer
	return &clone
}

// Send submits an SMS using routed providers. Accepted and unknown states stop
// execution immediately; only rejected allows Relay to try the next provider.
func (r *Relay) Send(ctx context.Context, message Message) (SendResult, error) {
	if err := message.validate(); err != nil {
		return SendResult{}, err
	}
	if r == nil || len(r.providers) == 0 {
		return SendResult{}, ErrNoProviders
	}

	route := r.route(ctx, message)
	if len(route.providers) == 0 {
		switch {
		case !route.capable:
			r.observe(ctx, relaycore.Event{Kind: relaycore.EventRouteExhausted, Channel: relaycore.ChannelSMS, Reason: relaycore.ReasonNoCapableProviders})
			return SendResult{State: SubmissionRejected}, ErrNoCapableProviders
		case !route.available:
			r.observe(ctx, relaycore.Event{Kind: relaycore.EventRouteExhausted, Channel: relaycore.ChannelSMS, Reason: relaycore.ReasonNoAvailableProviders})
			return SendResult{State: SubmissionRejected}, ErrNoAvailableProviders
		default:
			return SendResult{State: SubmissionRejected}, ErrNoProviders
		}
	}

	r.observe(ctx, relaycore.Event{
		Kind:      relaycore.EventRouteSelected,
		Channel:   relaycore.ChannelSMS,
		Providers: providerNames(route.providers),
	})

	for _, provider := range route.providers {
		r.observe(ctx, relaycore.Event{Kind: relaycore.EventAttemptStarted, Channel: relaycore.ChannelSMS, Provider: provider.Name()})
		started := time.Now()
		result, err := provider.Send(ctx, message)
		duration := time.Since(started)
		result.Provider = provider.Name()
		result.State = result.State.Normalize()

		r.observe(ctx, relaycore.Event{
			Kind:              relaycore.EventAttemptFinished,
			Channel:           relaycore.ChannelSMS,
			Provider:          provider.Name(),
			Outcome:           result.State,
			ProviderMessageID: result.ProviderMessageID,
			Duration:          duration,
			HadError:          err != nil,
		})

		switch result.State {
		case SubmissionAccepted:
			return result, nil
		case SubmissionRejected:
			continue
		default:
			if err != nil {
				return result, fmt.Errorf("provider %s submission state unknown: %w", provider.Name(), err)
			}
			return result, nil
		}
	}

	r.observe(ctx, relaycore.Event{Kind: relaycore.EventRouteExhausted, Channel: relaycore.ChannelSMS, Reason: relaycore.ReasonAllRejected})
	return SendResult{State: SubmissionRejected}, ErrAllRejected
}

func (r *Relay) observe(ctx context.Context, event relaycore.Event) {
	if r == nil || r.observer == nil {
		return
	}
	func() {
		defer func() {
			_ = recover()
		}()
		r.observer.Observe(ctx, event)
	}()
}

func providerNames(providers []Provider) []string {
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		names = append(names, provider.Name())
	}
	return names
}
