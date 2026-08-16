package sms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dugble/relay/providers/mock"
	"github.com/dugble/relay/sms"
)

func message() sms.Message {
	return sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}
}

func TestRelayStopsAfterAccepted(t *testing.T) {
	primary := mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{ProviderMessageID: "msg-1", State: sms.SubmissionAccepted}, nil
	})
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		t.Fatal("secondary provider must not be called")
		return sms.SendResult{}, nil
	})

	relay, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Provider != "primary" || result.State != sms.SubmissionAccepted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if secondary.Calls() != 0 {
		t.Fatalf("secondary calls = %d, want 0", secondary.Calls())
	}
}

func TestRelayFallsBackAfterDefiniteRejection(t *testing.T) {
	primary := mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionRejected}, errors.New("provider rejected request")
	})
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{ProviderMessageID: "msg-2", State: sms.SubmissionAccepted}, nil
	})

	relay, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Provider != "secondary" || result.State != sms.SubmissionAccepted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if primary.Calls() != 1 || secondary.Calls() != 1 {
		t.Fatalf("calls primary=%d secondary=%d, want 1/1", primary.Calls(), secondary.Calls())
	}
}

func TestRelayNeverFallsBackAfterUnknownAcceptance(t *testing.T) {
	timeout := errors.New("request timed out")
	primary := mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionUnknown}, timeout
	})
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		t.Fatal("secondary provider must not be called after ambiguous acceptance")
		return sms.SendResult{}, nil
	})

	relay, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if err == nil || !errors.Is(err, timeout) {
		t.Fatalf("Send() error = %v, want wrapped timeout", err)
	}
	if result.Provider != "primary" || result.State != sms.SubmissionUnknown {
		t.Fatalf("unexpected result: %+v", result)
	}
	if secondary.Calls() != 0 {
		t.Fatalf("secondary calls = %d, want 0", secondary.Calls())
	}
}

func TestRelayTreatsMissingStateAsUnknown(t *testing.T) {
	primary := mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{}, nil
	})
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		t.Fatal("secondary provider must not be called")
		return sms.SendResult{}, nil
	})

	relay, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.State != sms.SubmissionUnknown {
		t.Fatalf("state = %q, want unknown", result.State)
	}
}

func TestRelayReturnsAllRejected(t *testing.T) {
	first := mock.New("first", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionRejected}, nil
	})
	second := mock.New("second", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionRejected}, nil
	})

	relay, err := sms.NewRelay(first, second)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if !errors.Is(err, sms.ErrAllRejected) {
		t.Fatalf("Send() error = %v, want ErrAllRejected", err)
	}
	if result.State != sms.SubmissionRejected {
		t.Fatalf("state = %q, want rejected", result.State)
	}
}

func TestRelayValidatesMessage(t *testing.T) {
	provider := mock.New("provider", nil)
	relay, err := sms.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = relay.Send(context.Background(), sms.Message{})
	if !errors.Is(err, sms.ErrInvalidMessage) {
		t.Fatalf("Send() error = %v, want ErrInvalidMessage", err)
	}
	if provider.Calls() != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.Calls())
	}
}

func TestRelaySkipsProviderThatCannotSupportMessage(t *testing.T) {
	primary := &capabilityProvider{
		Provider: mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
			t.Fatal("incompatible provider must not be called")
			return sms.SendResult{}, nil
		}),
		capabilities: sms.Capabilities{MaxSenderIDLength: 3},
	}
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	})

	relay, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := relay.Send(context.Background(), message())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Provider != "secondary" {
		t.Fatalf("provider = %q, want secondary", result.Provider)
	}
}

func TestRelayReturnsNoCapableProviders(t *testing.T) {
	provider := &capabilityProvider{
		Provider:     mock.New("provider", nil),
		capabilities: sms.Capabilities{RequiresE164Recipient: true},
	}
	relay, err := sms.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}
	_, err = relay.Send(context.Background(), sms.Message{To: "0200000000", From: "Acme", Text: "hello"})
	if !errors.Is(err, sms.ErrNoCapableProviders) {
		t.Fatalf("Send() error = %v, want ErrNoCapableProviders", err)
	}
	if provider.Calls() != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.Calls())
	}
}

type capabilityProvider struct {
	*mock.Provider
	capabilities sms.Capabilities
}

func (p *capabilityProvider) Capabilities() sms.Capabilities { return p.capabilities }
