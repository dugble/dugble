package sms

import (
	"context"

	relaycore "github.com/dugble/relay"
)

type routeResult struct {
	providers []Provider
	capable   bool
	available bool
}

func (r *Relay) route(ctx context.Context, message Message) routeResult {
	result := routeResult{}
	if r == nil {
		return result
	}

	healthy := make([]Provider, 0, len(r.providers))
	degraded := make([]Provider, 0, len(r.providers))

	for _, provider := range r.providers {
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
		case relaycore.HealthUnavailable:
			r.observe(ctx, relaycore.Event{
				Kind:     relaycore.EventProviderSkipped,
				Channel:  relaycore.ChannelSMS,
				Provider: provider.Name(),
				Reason:   relaycore.ReasonProviderUnavailable,
			})
			continue
		case relaycore.HealthDegraded:
			result.available = true
			degraded = append(degraded, provider)
		default:
			result.available = true
			healthy = append(healthy, provider)
		}
	}

	result.providers = append(healthy, degraded...)
	return result
}
