package sms

import (
	"context"
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

type routeResult struct {
	providers []Provider
	routed    bool
	capable   bool
	available bool
}

func (r *Relay) route(ctx context.Context, message Message) routeResult {
	result := routeResult{}
	if r == nil {
		return result
	}

	candidates := r.providers
	if r.routes != nil {
		names := r.routes.ProviderNames(message.CountryCode)
		if len(names) == 0 {
			return result
		}
		result.routed = true
		candidates = providersByName(r.providers, names)
	} else {
		result.routed = true
	}

	healthy := make([]Provider, 0, len(candidates))
	degraded := make([]Provider, 0, len(candidates))

	for _, provider := range candidates {
		if providerWithCapabilities, ok := provider.(CapabilityProvider); ok {
			if !providerWithCapabilities.Capabilities().Supports(message) {
				r.observe(ctx, relaycore.Event{
					Kind:     relaycore.EventProviderSkipped,
					Channel:  relaycore.ChannelSMS,
					Provider: provider.Name(),
					Reason:   relaycore.ReasonUnsupportedCapability,
				})
				continue
			}
		}
		result.capable = true

		status := relaycore.HealthHealthy
		if r.health != nil {
			status = r.health.Status(ctx, provider.Name())
		}

		switch status {
		case relaycore.HealthHealthy:
			result.available = true
			healthy = append(healthy, provider)
		case relaycore.HealthDegraded:
			result.available = true
			degraded = append(degraded, provider)
		case relaycore.HealthUnavailable:
			r.observe(ctx, relaycore.Event{
				Kind:     relaycore.EventProviderSkipped,
				Channel:  relaycore.ChannelSMS,
				Provider: provider.Name(),
				Reason:   relaycore.ReasonProviderUnavailable,
			})
		default:
			r.observe(ctx, relaycore.Event{
				Kind:     relaycore.EventProviderSkipped,
				Channel:  relaycore.ChannelSMS,
				Provider: provider.Name(),
				Reason:   relaycore.ReasonProviderUnavailable,
			})
		}
	}

	result.providers = append(healthy, degraded...)
	return result
}

func providersByName(providers []Provider, names []string) []Provider {
	ordered := make([]Provider, 0, len(names))
	for _, name := range names {
		for _, provider := range providers {
			if strings.EqualFold(strings.TrimSpace(provider.Name()), strings.TrimSpace(name)) {
				ordered = append(ordered, provider)
				break
			}
		}
	}
	return ordered
}
