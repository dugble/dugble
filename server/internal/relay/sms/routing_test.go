package sms

import (
	"context"
	"errors"
	"reflect"
	"testing"

	relaycore "github.com/dugble/dugble/server/internal/relay"
)

type routingTestProvider struct {
	name     string
	state    SubmissionState
	attempts *[]string
}

func (p *routingTestProvider) Name() string { return p.name }

func (p *routingTestProvider) Send(_ context.Context, _ Message) (SendResult, error) {
	*p.attempts = append(*p.attempts, p.name)
	return SendResult{State: p.state}, nil
}

func TestRelayRoutesByCountryPriorityWithinHealthTier(t *testing.T) {
	attempts := []string{}
	mnotify := &routingTestProvider{name: "mnotify", state: SubmissionRejected, attempts: &attempts}
	moolre := &routingTestProvider{name: "moolre", state: SubmissionRejected, attempts: &attempts}
	sendexa := &routingTestProvider{name: "sendexa", state: SubmissionAccepted, attempts: &attempts}

	routes, err := relaycore.NewRouteTable([]relaycore.Route{
		{Provider: "mnotify", CountryCode: "GH", Priority: 1, Enabled: true},
		{Provider: "moolre", CountryCode: "GH", Priority: 2, Enabled: true},
		{Provider: "sendexa", CountryCode: "GH", Priority: 3, Enabled: true},
	})
	if err != nil {
		t.Fatalf("NewRouteTable() error = %v", err)
	}

	relay, err := NewRelay(sendexa, mnotify, moolre)
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}
	relay = relay.WithRoutes(routes).WithHealth(relaycore.HealthFunc(func(_ context.Context, provider string) relaycore.HealthStatus {
		if provider == "mnotify" {
			return relaycore.HealthDegraded
		}
		return relaycore.HealthHealthy
	}))

	_, err = relay.Send(context.Background(), Message{To: "+233200000000", From: "DUGBLE", Text: "hello", CountryCode: "GH"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	want := []string{"moolre", "sendexa"}
	if !reflect.DeepEqual(attempts, want) {
		t.Fatalf("attempts = %v, want %v", attempts, want)
	}
}

func TestRelayReturnsNoRouteForUnconfiguredCountry(t *testing.T) {
	attempts := []string{}
	provider := &routingTestProvider{name: "moolre", state: SubmissionAccepted, attempts: &attempts}
	routes, err := relaycore.NewRouteTable([]relaycore.Route{
		{Provider: "moolre", CountryCode: "GH", Priority: 1, Enabled: true},
	})
	if err != nil {
		t.Fatalf("NewRouteTable() error = %v", err)
	}
	relay, err := NewRelay(provider)
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}

	_, err = relay.WithRoutes(routes).Send(context.Background(), Message{To: "+254700000000", From: "DUGBLE", Text: "hello", CountryCode: "KE"})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("Send() error = %v, want %v", err, ErrNoRoute)
	}
	if len(attempts) != 0 {
		t.Fatalf("attempts = %v, want none", attempts)
	}
}
