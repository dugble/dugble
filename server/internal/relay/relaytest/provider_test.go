package relaytest_test

import (
	"context"
	"errors"
	"testing"

	relay "github.com/dugble/dugble/server/internal/relay"
	"github.com/dugble/dugble/server/internal/relay/relaytest"
)

func TestContract(t *testing.T) {
	contract := relaytest.Contract[string]{
		Name:     "primary",
		Accepted: scriptedFactory("primary", relay.SubmissionAccepted),
		Rejected: scriptedFactory("primary", relay.SubmissionRejected),
		Unknown:  scriptedFactory("primary", relay.SubmissionUnknown),
		RunRelay: func(ctx context.Context, primary, fallback relaytest.Provider[string], message string) (relay.Result, error) {
			result, err := primary.Send(ctx, message)
			result.Provider = primary.Name()
			result.State = result.State.Normalize()
			switch result.State {
			case relay.SubmissionAccepted:
				return result, nil
			case relay.SubmissionRejected:
				fallbackResult, fallbackErr := fallback.Send(ctx, message)
				fallbackResult.Provider = fallback.Name()
				fallbackResult.State = fallbackResult.State.Normalize()
				return fallbackResult, fallbackErr
			default:
				return result, err
			}
		},
	}
	contract.Run(t)
}

func TestContractAllowsNoRejectedScenario(t *testing.T) {
	contract := relaytest.Contract[string]{
		Name:     "primary",
		Accepted: scriptedFactory("primary", relay.SubmissionAccepted),
		Unknown:  scriptedFactory("primary", relay.SubmissionUnknown),
	}
	contract.Run(t)
}

func scriptedFactory(name string, state relay.SubmissionState) relaytest.Factory[string] {
	return func(t *testing.T) (relaytest.Provider[string], string) {
		t.Helper()
		return scriptedProvider{name: name, state: state}, "message"
	}
}

type scriptedProvider struct {
	name  string
	state relay.SubmissionState
}

func (p scriptedProvider) Name() string { return p.name }

func (p scriptedProvider) Send(context.Context, string) (relay.Result, error) {
	if p.state == relay.SubmissionAccepted {
		return relay.Result{State: p.state}, nil
	}
	return relay.Result{State: p.state}, errors.New("provider result")
}
