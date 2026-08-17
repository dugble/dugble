package senderid

import (
	"fmt"
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

func (r *Relay) route(countryCode string) ([]Provider, error) {
	if r == nil || len(r.providers) == 0 {
		return nil, relaycore.ErrNoProviders
	}

	if r.routes == nil {
		providers := make([]Provider, len(r.providers))
		copy(providers, r.providers)
		return providers, nil
	}

	names := r.routes.ProviderNames(countryCode)
	if len(names) == 0 {
		return nil, relaycore.ErrNoRoute
	}

	byName := make(map[string]Provider, len(r.providers))
	for _, provider := range r.providers {
		byName[strings.ToLower(strings.TrimSpace(provider.Name()))] = provider
	}

	providers := make([]Provider, 0, len(names))
	for _, name := range names {
		provider, ok := byName[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("sender ID route references unregistered provider %q", name)
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func (r *Relay) provider(name string) (Provider, error) {
	if r == nil || len(r.providers) == 0 {
		return nil, relaycore.ErrNoProviders
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("sender ID provider is required")
	}
	for _, provider := range r.providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name()), name) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("sender ID provider %q is not registered", name)
}
