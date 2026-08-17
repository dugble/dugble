package senderid

import (
	"context"
	"fmt"
	"strings"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

// Relay coordinates Sender ID registration and status reconciliation across
// provider implementations.
type Relay struct {
	providers []Provider
	routes    *relaycore.RouteTable
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
		return nil, relaycore.ErrNoProviders
	}
	return &Relay{providers: filtered}, nil
}

// WithRoutes returns a shallow copy that uses the supplied country/provider
// route table for registration fan-out.
func (r *Relay) WithRoutes(routes *relaycore.RouteTable) *Relay {
	if r == nil {
		return nil
	}
	clone := *r
	clone.routes = routes
	return &clone
}

// Create registers a Sender ID with every provider selected for the request's
// country. All selected providers are attempted so a partial provider failure
// does not prevent the remaining registrations from being submitted.
func (r *Relay) Create(ctx context.Context, request CreateRequest) ([]CreateResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("sender ID context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return nil, err
	}

	providers, err := r.route(request.CountryCode)
	if err != nil {
		return nil, err
	}

	results := make([]CreateResult, 0, len(providers))
	failures := make([]string, 0)
	for _, provider := range providers {
		result, createErr := provider.CreateSenderID(ctx, request)
		result.Provider = provider.Name()
		if strings.TrimSpace(result.Name) == "" {
			result.Name = request.Name
		}
		results = append(results, result)
		if createErr != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), createErr))
		}
	}
	if len(failures) != 0 {
		return results, fmt.Errorf("sender ID registration failed for %d provider(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return results, nil
}

// CheckStatus reconciles a registration only with the provider that owns it.
func (r *Relay) CheckStatus(ctx context.Context, request StatusRequest) (StatusResult, error) {
	if ctx == nil {
		return StatusResult{}, fmt.Errorf("sender ID context is required")
	}
	if err := ctx.Err(); err != nil {
		return StatusResult{}, err
	}
	if err := ValidateName(request.Name); err != nil {
		return StatusResult{}, err
	}

	provider, err := r.provider(request.Provider)
	if err != nil {
		return StatusResult{}, err
	}
	result, err := provider.CheckSenderIDStatus(ctx, request)
	result.Provider = provider.Name()
	if strings.TrimSpace(result.Name) == "" {
		result.Name = NormalizeName(request.Name)
	}
	return result, err
}
