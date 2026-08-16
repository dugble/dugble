package sms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

type provider struct {
	name  string
	calls int
	send  func(context.Context, sms.Message) (sms.SendResult, error)
}

func (p *provider) Name() string { return p.name }

func (p *provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	p.calls++
	return p.send(ctx, message)
}

func message() sms.Message {
	return sms.Message{To: "+233200000000", From: "Acme", Text: "hello"}
}

func TestRelaySubmissionSafety(t *testing.T) {
	t.Run("accepted stops", func(t *testing.T) {
		primary := &provider{name: "primary", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			return sms.SendResult{State: sms.SubmissionAccepted}, nil
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			t.Fatal("fallback must not run after accepted")
			return sms.SendResult{}, nil
		}}
		router, err := sms.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if err != nil {
			t.Fatal(err)
		}
		if result.State != sms.SubmissionAccepted || fallback.calls != 0 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})

	t.Run("rejected falls back", func(t *testing.T) {
		primary := &provider{name: "primary", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			return sms.SendResult{State: sms.SubmissionRejected}, errors.New("definite rejection")
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			return sms.SendResult{State: sms.SubmissionAccepted}, nil
		}}
		router, err := sms.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if err != nil {
			t.Fatal(err)
		}
		if result.Provider != "fallback" || result.State != sms.SubmissionAccepted || fallback.calls != 1 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})

	t.Run("unknown stops", func(t *testing.T) {
		ambiguous := errors.New("request timed out")
		primary := &provider{name: "primary", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			return sms.SendResult{State: sms.SubmissionUnknown}, ambiguous
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			t.Fatal("fallback must not run after unknown")
			return sms.SendResult{}, nil
		}}
		router, err := sms.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if !errors.Is(err, ambiguous) {
			t.Fatalf("error=%v, want wrapped ambiguous error", err)
		}
		if result.State != sms.SubmissionUnknown || fallback.calls != 0 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})

	t.Run("missing state is unknown", func(t *testing.T) {
		primary := &provider{name: "primary", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			return sms.SendResult{}, nil
		}}
		fallback := &provider{name: "fallback", send: func(context.Context, sms.Message) (sms.SendResult, error) {
			t.Fatal("fallback must not run for unrecognized state")
			return sms.SendResult{}, nil
		}}
		router, err := sms.NewRelay(primary, fallback)
		if err != nil {
			t.Fatal(err)
		}
		result, err := router.Send(context.Background(), message())
		if err != nil {
			t.Fatal(err)
		}
		if result.State != sms.SubmissionUnknown || fallback.calls != 0 {
			t.Fatalf("result=%+v fallback calls=%d", result, fallback.calls)
		}
	})
}
