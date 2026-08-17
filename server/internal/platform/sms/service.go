package sms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Route struct {
	ProviderID         string
	DestinationCountry string
	Priority           int
	Enabled            bool
}

type RoutingConfig struct { Routes []Route }

func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{Routes: []Route{
		{ProviderID: "moolre", DestinationCountry: "GH", Priority: 1, Enabled: true},
		{ProviderID: "sendexa", DestinationCountry: "GH", Priority: 2, Enabled: true},
	}}
}

type RoutingService struct {
	routes map[string][]string
	providers []Provider
}

var _ Router = (*RoutingService)(nil)

func NewRoutingService(config RoutingConfig, providers ...Provider) (*RoutingService, error) {
	registry, normalizedProviders, err := providerRegistry(providers)
	if err != nil { return nil, err }
	if len(config.Routes) == 0 { return nil, ErrNoProviderAvailable }
	type orderedRoute struct { id string; priority int }
	byCountry := make(map[string][]orderedRoute)
	seenProvider := make(map[string]struct{})
	seenPriority := make(map[string]struct{})
	for _, route := range config.Routes {
		if !route.Enabled { continue }
		providerID := normalizeProviderID(route.ProviderID)
		country := NormalizeCountryCode(route.DestinationCountry)
		if providerID == "" || !IsCountryCode(country) || route.Priority < 1 { return nil, ErrInvalidProviderID }
		if _, exists := registry[providerID]; !exists { return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, providerID) }
		providerKey := country + "\x00" + providerID
		if _, exists := seenProvider[providerKey]; exists { return nil, fmt.Errorf("%w: %s", ErrDuplicateProvider, providerID) }
		seenProvider[providerKey] = struct{}{}
		priorityKey := fmt.Sprintf("%s\x00%d", country, route.Priority)
		if _, exists := seenPriority[priorityKey]; exists { return nil, fmt.Errorf("duplicate SMS route priority %d for %s", route.Priority, country) }
		seenPriority[priorityKey] = struct{}{}
		byCountry[country] = append(byCountry[country], orderedRoute{id: providerID, priority: route.Priority})
	}
	routes := make(map[string][]string, len(byCountry))
	for country, values := range byCountry {
		sort.SliceStable(values, func(i, j int) bool { return values[i].priority < values[j].priority })
		for _, value := range values { routes[country] = append(routes[country], value.id) }
	}
	return &RoutingService{routes: routes, providers: normalizedProviders}, nil
}

func (service *RoutingService) Route(ctx context.Context, destinationCountry string) ([]string, error) {
	if service == nil { return nil, ErrRouterRequired }
	if ctx == nil { return nil, errors.New("SMS routing context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	country := NormalizeCountryCode(destinationCountry)
	result := service.routes[country]
	if len(result) == 0 { return nil, ErrNoProviderAvailable }
	return append([]string(nil), result...), nil
}
func (service *RoutingService) ShouldFallback(ctx context.Context, providerID string, err error) bool {
	if service == nil || ctx == nil || ctx.Err() != nil || err == nil { return false }
	var fallback interface { SafeToFallback() bool }
	return errors.As(err, &fallback) && fallback.SafeToFallback()
}
func (service *RoutingService) ProviderIDs() []string {
	if service == nil { return nil }
	result := make([]string, 0, len(service.providers))
	for _, provider := range service.providers { result = append(result, normalizeProviderID(provider.ID())) }
	return result
}
func (service *RoutingService) Providers() []Provider { if service == nil { return nil }; return append([]Provider(nil), service.providers...) }

type Service struct { router Router; providers map[string]Provider }
type routedProviderSource interface { ProviderIDs() []string }
type providerSource interface { Providers() []Provider }

func NewService(router Router, providers ...Provider) (*Service, error) {
	if router == nil { return nil, ErrRouterRequired }
	if len(providers) == 0 { if source, ok := router.(providerSource); ok { providers = source.Providers() } }
	registry, _, err := providerRegistry(providers)
	if err != nil { return nil, err }
	if source, ok := router.(routedProviderSource); ok {
		for _, providerID := range source.ProviderIDs() { providerID = normalizeProviderID(providerID); if _, exists := registry[providerID]; !exists { return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, providerID) } }
	}
	return &Service{router: router, providers: registry}, nil
}

func (service *Service) Send(ctx context.Context, request SendRequest) (*SendResponse, error) {
	if service == nil || service.router == nil { return nil, ErrRouterRequired }
	if ctx == nil { return nil, errors.New("SMS send context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	request = request.Normalize(); if err := request.Validate(); err != nil { return nil, err }
	providerIDs, err := service.router.Route(ctx, request.DestinationCountry)
	if err != nil { return nil, fmt.Errorf("route SMS request: %w", err) }
	attempts := make([]ProviderAttempt, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		response, attemptErr := service.SendWithProvider(ctx, providerID, request)
		if attemptErr == nil { return response, nil }
		providerID = normalizeProviderID(providerID); attempts = append(attempts, ProviderAttempt{ProviderID: providerID, Err: attemptErr})
		if !service.ShouldFallback(ctx, providerID, attemptErr) { break }
	}
	return nil, &SendError{Attempts: attempts}
}

func (service *Service) SendWithProvider(ctx context.Context, providerID string, request SendRequest) (*SendResponse, error) {
	if service == nil || service.router == nil { return nil, ErrRouterRequired }
	if ctx == nil { return nil, errors.New("SMS send context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	request = request.Normalize(); if err := request.Validate(); err != nil { return nil, err }
	providerID = normalizeProviderID(providerID); upstream := service.providers[providerID]
	if upstream == nil { return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerID) }
	response, err := upstream.Send(ctx, request); if err != nil { return nil, err }
	if err := validateSendResponse(providerID, response); err != nil { return nil, err }
	return response, nil
}
func (service *Service) ShouldFallback(ctx context.Context, providerID string, err error) bool { if service == nil || service.router == nil { return false }; return service.router.ShouldFallback(ctx, providerID, err) }
func (service *Service) ProviderIDs() []string { if service == nil { return nil }; result := make([]string,0,len(service.providers)); for id := range service.providers { result=append(result,id) }; sort.Strings(result); return result }
func (service *Service) CheckStatus(ctx context.Context, providerID, providerMessageID string) (*StatusResponse, error) {
	if service == nil || service.router == nil { return nil, ErrRouterRequired }
	if ctx == nil { return nil, errors.New("SMS status context is required") }
	if err := ctx.Err(); err != nil { return nil, err }
	providerID = normalizeProviderID(providerID); providerMessageID = strings.TrimSpace(providerMessageID)
	if providerID == "" { return nil, &ValidationError{Field:"provider_id",Reason:"provider ID is required"} }
	if providerMessageID == "" { return nil, &ValidationError{Field:"provider_message_id",Reason:"provider message ID is required"} }
	upstream := service.providers[providerID]; if upstream == nil { return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerID) }
	response, err := upstream.CheckStatus(ctx, providerMessageID); if err != nil { return nil, fmt.Errorf("check %s SMS status: %w", providerID, err) }
	if err := validateStatusResponse(providerID, providerMessageID, response); err != nil { return nil, err }
	return response, nil
}

func providerRegistry(providers []Provider) (map[string]Provider, []Provider, error) {
	if len(providers) == 0 { return nil, nil, ErrProviderRequired }
	registry := make(map[string]Provider,len(providers)); normalized := make([]Provider,0,len(providers))
	for _, upstream := range providers {
		if upstream == nil { return nil,nil,ErrProviderRequired }
		id := normalizeProviderID(upstream.ID()); if id == "" { return nil,nil,ErrInvalidProviderID }
		if _, exists := registry[id]; exists { return nil,nil,fmt.Errorf("%w: %s",ErrDuplicateProvider,id) }
		registry[id]=upstream; normalized=append(normalized,upstream)
	}
	return registry,normalized,nil
}
func validateSendResponse(expected string, response *SendResponse) error {
	if response == nil { return fmt.Errorf("%w: send response is nil",ErrInvalidProviderReply) }
	response.ProviderID=normalizeProviderID(response.ProviderID); response.ProviderMsgID=strings.TrimSpace(response.ProviderMsgID); response.Status=strings.ToLower(strings.TrimSpace(response.Status))
	if response.ProviderID != expected || response.ProviderMsgID == "" || !IsKnownStatus(response.Status) { return fmt.Errorf("%w: invalid send response from %s",ErrInvalidProviderReply,expected) }
	return nil
}
func validateStatusResponse(expected, messageID string, response *StatusResponse) error {
	if response == nil { return fmt.Errorf("%w: status response is nil",ErrInvalidProviderReply) }
	response.ProviderID=normalizeProviderID(response.ProviderID); response.ProviderMsgID=strings.TrimSpace(response.ProviderMsgID); response.Status=strings.ToLower(strings.TrimSpace(response.Status))
	if response.ProviderID != expected || response.ProviderMsgID != messageID || !IsKnownStatus(response.Status) { return fmt.Errorf("%w: invalid status response from %s",ErrInvalidProviderReply,expected) }
	return nil
}
func normalizeProviderID(providerID string) string { return strings.ToLower(strings.TrimSpace(providerID)) }
