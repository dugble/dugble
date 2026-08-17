package sms

import (
	"context"
	"fmt"
	"sort"
	"strings"

	provider "github.com/dugble/dugble/server/internal/providers"
	relaycore "github.com/dugble/dugble/server/internal/relay"
	relaysms "github.com/dugble/dugble/server/internal/relay/sms"
)

// RelayService adapts the provider-neutral SMS relay to the application's
// existing SMS request/response contract while platform/sms is migrated.
type RelayService struct {
	relay     *relaysms.Relay
	providers map[string]relaysms.Provider
}

func NewRelayService(routes *relaycore.RouteTable, providers ...relaysms.Provider) (*RelayService, error) {
	relay, err := relaysms.NewRelay(providers...)
	if err != nil {
		return nil, err
	}
	if routes != nil {
		relay = relay.WithRoutes(routes)
	}

	registry := make(map[string]relaysms.Provider, len(providers))
	for _, upstream := range providers {
		if upstream == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(upstream.Name()))
		if name == "" {
			return nil, ErrInvalidProviderID
		}
		if _, exists := registry[name]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateProvider, name)
		}
		registry[name] = upstream
	}
	return &RelayService{relay: relay, providers: registry}, nil
}

func (service *RelayService) Send(ctx context.Context, request SendRequest) (*SendResponse, error) {
	if service == nil || service.relay == nil {
		return nil, ErrRouterRequired
	}
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return nil, err
	}

	result, err := service.relay.Send(ctx, relayMessage(request))
	if err != nil {
		return nil, err
	}
	return sendResponse(result), nil
}

// SendWithProvider submits through one already-selected provider. Durable SMS
// delivery uses this when the approved Sender ID registration pins a provider.
func (service *RelayService) SendWithProvider(ctx context.Context, providerID string, request SendRequest) (*SendResponse, error) {
	if service == nil {
		return nil, ErrRouterRequired
	}
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return nil, err
	}

	providerID = strings.ToLower(strings.TrimSpace(providerID))
	upstream := service.providers[providerID]
	if upstream == nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerID)
	}

	result, err := upstream.Send(ctx, relayMessage(request))
	result.Provider = upstream.Name()
	result.State = result.State.Normalize()
	if result.State == relaysms.SubmissionAccepted {
		return sendResponse(result), nil
	}
	if err == nil {
		err = fmt.Errorf("provider %s returned submission state %s", upstream.Name(), result.State)
	}
	return nil, &SubmissionError{
		Provider: upstream.Name(),
		State:    result.State,
		Err:      err,
	}
}

func (service *RelayService) ProviderIDs() []string {
	if service == nil {
		return nil
	}
	result := make([]string, 0, len(service.providers))
	for name := range service.providers {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (service *RelayService) CheckStatus(ctx context.Context, providerID, providerMessageID string) (*StatusResponse, error) {
	if service == nil {
		return nil, ErrRouterRequired
	}
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	providerMessageID = strings.TrimSpace(providerMessageID)
	upstream := service.providers[providerID]
	if upstream == nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerID)
	}

	checker, ok := upstream.(provider.SMSStatusChecker)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support SMS status checks", providerID)
	}
	result, err := checker.CheckSMSStatus(ctx, provider.SMSStatusRequest{
		Reference:         providerMessageID,
		ProviderMessageID: providerMessageID,
	})
	if err != nil {
		return nil, err
	}
	return &StatusResponse{
		ProviderID:     providerID,
		ProviderMsgID:  result.ProviderMessageID,
		Status:         platformStatus(result.Status),
		ProviderStatus: result.ProviderStatus,
	}, nil
}

// SubmissionError retains the old definitive-rejection signal for durable
// delivery while using Relay's explicit submission state internally.
type SubmissionError struct {
	Provider string
	State    relaysms.SubmissionState
	Err      error
}

func (err *SubmissionError) Error() string {
	if err == nil {
		return "SMS submission failed"
	}
	if err.Err != nil {
		return fmt.Sprintf("SMS submission via %s failed: %v", err.Provider, err.Err)
	}
	return fmt.Sprintf("SMS submission via %s failed", err.Provider)
}

func (err *SubmissionError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

func (err *SubmissionError) SafeToFallback() bool {
	return err != nil && err.State == relaysms.SubmissionRejected
}

func relayMessage(request SendRequest) relaysms.Message {
	return relaysms.Message{
		Reference:   request.Reference,
		To:          request.To,
		From:        request.From,
		Text:        request.Message,
		CountryCode: request.DestinationCountry,
		Purpose:     relaysms.PurposeTransactional,
	}
}

func sendResponse(result relaysms.SendResult) *SendResponse {
	return &SendResponse{
		ProviderID:    strings.ToLower(strings.TrimSpace(result.Provider)),
		ProviderMsgID: strings.TrimSpace(result.ProviderMessageID),
		Status:        StatusSubmitted,
	}
}

func platformStatus(status provider.SMSStatus) string {
	switch status {
	case provider.SMSPending:
		return StatusSubmitted
	case provider.SMSDelivered:
		return StatusDelivered
	case provider.SMSFailed:
		return StatusFailed
	default:
		return StatusUnknown
	}
}
