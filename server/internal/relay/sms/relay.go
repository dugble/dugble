package sms

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

var (
	ErrNoProviders          = relaycore.ErrNoProviders
	ErrNoCapableProviders   = relaycore.ErrNoCapableProviders
	ErrNoAvailableProviders = relaycore.ErrNoAvailableProviders
	ErrAllRejected          = relaycore.ErrAllRejected
	ErrSubmissionUnknown    = relaycore.ErrSubmissionUnknown
)

type Relay struct {
	providers []Provider
	health    relaycore.HealthSource
	observer  relaycore.Observer
}

func NewRelay(providers ...Provider) (*Relay, error) {
	filtered := make([]Provider, 0, len(providers))
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(provider.Name()))
		if name == "" {
			return nil, fmt.Errorf("SMS provider name is required")
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate SMS provider %q", name)
		}
		seen[name] = struct{}{}
		filtered = append(filtered, provider)
	}
	if len(filtered) == 0 {
		return nil, ErrNoProviders
	}
	return &Relay{providers: filtered}, nil
}

func (r *Relay) WithHealth(source relaycore.HealthSource) *Relay {
	if r == nil {
		return nil
	}
	clone := *r
	clone.health = source
	return &clone
}

func (r *Relay) WithObserver(observer relaycore.Observer) *Relay {
	if r == nil {
		return nil
	}
	clone := *r
	clone.observer = observer
	return &clone
}

func (r *Relay) ProviderIDs() []string {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.providers))
	for _, provider := range r.providers {
		ids = append(ids, strings.ToLower(strings.TrimSpace(provider.Name())))
	}
	sort.Strings(ids)
	return ids
}

func (r *Relay) Provider(name string) Provider {
	name = strings.ToLower(strings.TrimSpace(name))
	if r == nil || name == "" {
		return nil
	}
	for _, provider := range r.providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name()), name) {
			return provider
		}
	}
	return nil
}

// SendWithProvider submits through exactly one named provider. Durable delivery
// uses this after persisting the canonical provider choice, so one database
// attempt always represents one provider call.
func (r *Relay) SendWithProvider(ctx context.Context, providerName string, message Message) (SendResult, error) {
	if err := message.validate(); err != nil {
		return SendResult{State: SubmissionRejected}, err
	}
	provider := r.Provider(providerName)
	if provider == nil {
		return SendResult{State: SubmissionRejected}, fmt.Errorf("SMS provider %q is not configured", strings.TrimSpace(providerName))
	}
	if capable, ok := provider.(CapabilityProvider); ok && !capable.Capabilities().Supports(message) {
		return SendResult{Provider: provider.Name(), State: SubmissionRejected}, ErrNoCapableProviders
	}
	return r.sendProvider(ctx, provider, message)
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

	r.observe(ctx, relaycore.Event{Kind: relaycore.EventRouteSelected, Channel: relaycore.ChannelSMS, Providers: providerNames(route.providers)})
	for _, provider := range route.providers {
		result, err := r.sendProvider(ctx, provider, message)
		switch result.State {
		case SubmissionAccepted:
			return result, nil
		case SubmissionRejected:
			continue
		default:
			if err != nil {
				return result, fmt.Errorf("provider %s submission state unknown: %w", provider.Name(), err)
			}
			return result, fmt.Errorf("provider %s submission state unknown: %w", provider.Name(), ErrSubmissionUnknown)
		}
	}

	r.observe(ctx, relaycore.Event{Kind: relaycore.EventRouteExhausted, Channel: relaycore.ChannelSMS, Reason: relaycore.ReasonAllRejected})
	return SendResult{State: SubmissionRejected}, ErrAllRejected
}

func (r *Relay) sendProvider(ctx context.Context, provider Provider, message Message) (SendResult, error) {
	r.observe(ctx, relaycore.Event{Kind: relaycore.EventAttemptStarted, Channel: relaycore.ChannelSMS, Provider: provider.Name()})
	started := time.Now()
	result, err := provider.Send(ctx, message)
	result.Provider = provider.Name()
	result.State = result.State.Normalize()
	r.observe(ctx, relaycore.Event{
		Kind:              relaycore.EventAttemptFinished,
		Channel:           relaycore.ChannelSMS,
		Provider:          provider.Name(),
		Outcome:           result.State,
		ProviderMessageID: result.ProviderMessageID,
		Duration:          time.Since(started),
		HadError:          err != nil,
	})
	return result, err
}

func (r *Relay) observe(ctx context.Context, event relaycore.Event) {
	if r == nil || r.observer == nil {
		return
	}
	func() {
		defer func() { _ = recover() }()
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
