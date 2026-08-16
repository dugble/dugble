package sms_test

import (
	"context"
	"errors"
	"testing"

	relaycore "github.com/dugble/relay"
	"github.com/dugble/relay/providers/mock"
	"github.com/dugble/relay/sms"
)

func TestRelayPrefersHealthyProviderOverEarlierDegradedProvider(t *testing.T) {
	degraded := mock.New("degraded", func(context.Context, sms.Message) (sms.SendResult, error) {
		t.Fatal("degraded provider should not be called while a healthy provider accepts")
		return sms.SendResult{}, nil
	})
	healthy := mock.New("healthy", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	})

	router, err := sms.NewRelay(degraded, healthy)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithHealth(relaycore.HealthFunc(func(_ context.Context, provider string) relaycore.HealthStatus {
		if provider == "degraded" {
			return relaycore.HealthDegraded
		}
		return relaycore.HealthHealthy
	}))

	result, err := router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "healthy" {
		t.Fatalf("provider = %q, want healthy", result.Provider)
	}
}

func TestRelayUsesDegradedProviderWhenNoHealthyProviderIsAvailable(t *testing.T) {
	degraded := mock.New("degraded", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	})
	router, err := sms.NewRelay(degraded)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithHealth(relaycore.HealthFunc(func(context.Context, string) relaycore.HealthStatus {
		return relaycore.HealthDegraded
	}))

	result, err := router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil || result.Provider != "degraded" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRelaySkipsUnavailableProvider(t *testing.T) {
	unavailable := mock.New("unavailable", func(context.Context, sms.Message) (sms.SendResult, error) {
		t.Fatal("unavailable provider must not be called")
		return sms.SendResult{}, nil
	})
	backup := mock.New("backup", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	})
	router, err := sms.NewRelay(unavailable, backup)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithHealth(relaycore.HealthFunc(func(_ context.Context, provider string) relaycore.HealthStatus {
		if provider == "unavailable" {
			return relaycore.HealthUnavailable
		}
		return relaycore.HealthHealthy
	}))

	result, err := router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil || result.Provider != "backup" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRelayReturnsNoAvailableProviders(t *testing.T) {
	provider := mock.New("provider", nil)
	router, err := sms.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithHealth(relaycore.HealthFunc(func(context.Context, string) relaycore.HealthStatus {
		return relaycore.HealthUnavailable
	}))

	_, err = router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if !errors.Is(err, sms.ErrNoAvailableProviders) {
		t.Fatalf("Send() error = %v, want ErrNoAvailableProviders", err)
	}
	if provider.Calls() != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.Calls())
	}
}
