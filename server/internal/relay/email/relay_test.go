package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/email"
)

type provider struct {
	name  string
	calls int
	send  func(context.Context, email.Message) (email.SendResult, error)
}

func (p *provider) Name() string { return p.name }

func (p *provider) Send(ctx context.Context, message email.Message) (email.SendResult, error) {
	p.calls++
	return p.send(ctx, message)
}

func message() email.Message {
	return email.Message{
		From:    email.Address{Email: "sender@example.com"},
		To:      []email.Address{{Email: "recipient@example.com"}},
		Subject: "hello",
		Text:    "hello",
	}
}

func TestRelaySubmissionSafety(t *testing.T) {
	t.Run("accepted stops", func(t *testing.T) {
		primary := &provider{name: "primary", send: func(context.Context, email.Message) (email.SendResult, error) {
			return email.SendResult{State: email.SubmissionAccepted}, nil
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, email.Message) (email.SendResult, error) {
			t.Fatal("fallback must not run after accepted")
			return email.SendResult{}, nil
		}}
		router, err := email.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if err != nil {
			t.Fatal(err)
		}
		if result.State != email.SubmissionAccepted || fallback.calls != 0 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})

	t.Run("rejected falls back", func(t *testing.T) {
		primary := &provider{name: "primary", send: func(context.Context, email.Message) (email.SendResult, error) {
			return email.SendResult{State: email.SubmissionRejected}, errors.New("definite rejection")
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, email.Message) (email.SendResult, error) {
			return email.SendResult{State: email.SubmissionAccepted}, nil
		}}
		router, err := email.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if err != nil {
			t.Fatal(err)
		}
		if result.Provider != "fallback" || result.State != email.SubmissionAccepted || fallback.calls != 1 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})

	t.Run("unknown stops", func(t *testing.T) {
		ambiguous := errors.New("request timed out")
		primary := &provider{name: "primary", send: func(context.Context, email.Message) (email.SendResult, error) {
			return email.SendResult{State: email.SubmissionUnknown}, ambiguous
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, email.Message) (email.SendResult, error) {
			t.Fatal("fallback must not run after unknown")
			return email.SendResult{}, nil
		}}
		router, err := email.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if !errors.Is(err, ambiguous) {
			t.Fatalf("error=%v, want wrapped ambiguous error", err)
		}
		if result.State != email.SubmissionUnknown || fallback.calls != 0 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})
}
