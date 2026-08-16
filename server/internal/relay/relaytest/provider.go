package relaytest

import (
	"context"
	"strings"
	"testing"

	relay "github.com/dugble/dugble/server/internal/relay"
)

// Provider is the channel-shaped provider surface exercised by Contract.
type Provider[M any] interface {
	relay.Provider
	Send(context.Context, M) (relay.Result, error)
}

// Factory constructs a fresh provider and representative message for one scenario.
type Factory[M any] func(t *testing.T) (Provider[M], M)

// RunRelay composes a primary and fallback provider through a channel Relay.
type RunRelay[M any] func(context.Context, Provider[M], Provider[M], M) (relay.Result, error)

// Contract verifies provider submission semantics. Rejected is optional because
// adapters must not invent a safe rejection classification for test coverage.
type Contract[M any] struct {
	Name     string
	Accepted Factory[M]
	Rejected Factory[M]
	Unknown  Factory[M]
	RunRelay RunRelay[M]
}

// Run executes direct state checks and optional fallback behavior checks.
func (c Contract[M]) Run(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(c.Name) == "" {
		t.Fatal("relaytest.Contract Name is required")
	}
	if c.Accepted == nil {
		t.Fatal("relaytest.Contract Accepted factory is required")
	}
	if c.Unknown == nil {
		t.Fatal("relaytest.Contract Unknown factory is required")
	}

	c.runState(t, "accepted", c.Accepted, relay.SubmissionAccepted)
	if c.Rejected != nil {
		c.runState(t, "rejected", c.Rejected, relay.SubmissionRejected)
	}
	c.runState(t, "unknown", c.Unknown, relay.SubmissionUnknown)

	if c.RunRelay == nil {
		return
	}
	c.runFallback(t, "accepted_stops", c.Accepted, false, relay.SubmissionAccepted)
	if c.Rejected != nil {
		c.runFallback(t, "rejected_falls_back", c.Rejected, true, relay.SubmissionAccepted)
	}
	c.runFallback(t, "unknown_stops", c.Unknown, false, relay.SubmissionUnknown)
}

func (c Contract[M]) runState(t *testing.T, name string, factory Factory[M], want relay.SubmissionState) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		provider, message := factory(t)
		assertProviderName(t, provider, c.Name)
		result, _ := provider.Send(context.Background(), message)
		if result.State.Normalize() != want {
			t.Fatalf("submission state = %q, want %q", result.State.Normalize(), want)
		}
	})
}

func (c Contract[M]) runFallback(t *testing.T, name string, factory Factory[M], wantFallback bool, want relay.SubmissionState) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		primary, message := factory(t)
		assertProviderName(t, primary, c.Name)
		fallback := &acceptingProvider[M]{name: "relaytest-fallback"}
		result, _ := c.RunRelay(context.Background(), primary, fallback, message)
		if result.State.Normalize() != want {
			t.Fatalf("relay result state = %q, want %q", result.State.Normalize(), want)
		}
		if wantFallback && fallback.calls != 1 {
			t.Fatalf("fallback calls = %d, want 1", fallback.calls)
		}
		if !wantFallback && fallback.calls != 0 {
			t.Fatalf("fallback calls = %d, want 0", fallback.calls)
		}
	})
}

func assertProviderName[M any](t *testing.T, provider Provider[M], want string) {
	t.Helper()
	if provider == nil {
		t.Fatal("factory returned nil provider")
	}
	if provider.Name() != want {
		t.Fatalf("provider name = %q, want %q", provider.Name(), want)
	}
}

type acceptingProvider[M any] struct {
	name  string
	calls int
}

func (p *acceptingProvider[M]) Name() string { return p.name }

func (p *acceptingProvider[M]) Send(context.Context, M) (relay.Result, error) {
	p.calls++
	return relay.Result{State: relay.SubmissionAccepted}, nil
}
