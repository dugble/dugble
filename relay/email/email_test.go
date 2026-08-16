package email_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	relaycore "github.com/dugble/relay"
	"github.com/dugble/relay/email"
)

func TestMessageValidation(t *testing.T) {
	valid := email.Message{
		From: email.Address{Email: "sender@example.com", Name: "Sender"},
		To:   []email.Address{{Email: "recipient@example.com"}},
		Text: "hello",
	}

	provider := newProvider("provider", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{State: email.SubmissionAccepted}, nil
	})
	router, err := email.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		message email.Message
		wantErr bool
	}{
		{name: "valid text", message: valid},
		{name: "valid html", message: email.Message{From: valid.From, To: valid.To, HTML: "<p>hello</p>"}},
		{name: "missing from", message: email.Message{To: valid.To, Text: "hello"}, wantErr: true},
		{name: "missing recipients", message: email.Message{From: valid.From, Text: "hello"}, wantErr: true},
		{name: "invalid recipient", message: email.Message{From: valid.From, To: []email.Address{{Email: "not-an-email"}}, Text: "hello"}, wantErr: true},
		{name: "missing body", message: email.Message{From: valid.From, To: valid.To}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := router.Send(context.Background(), tc.message)
			if tc.wantErr && !errors.Is(err, relaycore.ErrInvalidMessage) {
				t.Fatalf("Send() error = %v, want ErrInvalidMessage", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Send() error = %v", err)
			}
		})
	}
}

func TestRelayFallsBackOnlyAfterRejected(t *testing.T) {
	primary := newProvider("primary", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{State: email.SubmissionRejected}, errors.New("definite rejection")
	})
	secondary := newProvider("secondary", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{State: email.SubmissionAccepted, ProviderMessageID: "msg-123"}, nil
	})
	router, _ := email.NewRelay(primary, secondary)

	result, err := router.Send(context.Background(), message())
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "secondary" || result.ProviderMessageID != "msg-123" || result.State != email.SubmissionAccepted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if primary.calls != 1 || secondary.calls != 1 {
		t.Fatalf("calls primary=%d secondary=%d", primary.calls, secondary.calls)
	}
}

func TestRelayStopsAfterUnknown(t *testing.T) {
	primary := newProvider("primary", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{}, errors.New("timeout")
	})
	secondary := newProvider("secondary", func(context.Context, email.Message) (email.SendResult, error) {
		t.Fatal("secondary must not be called after unknown submission")
		return email.SendResult{}, nil
	})
	router, _ := email.NewRelay(primary, secondary)

	result, err := router.Send(context.Background(), message())
	if err == nil || result.State != email.SubmissionUnknown || result.Provider != "primary" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if secondary.calls != 0 {
		t.Fatalf("secondary calls = %d, want 0", secondary.calls)
	}
}

func TestCapabilitiesSkipUnsupportedProvider(t *testing.T) {
	incapable := providerWithCapabilities{
		provider: newProvider("text-only", func(context.Context, email.Message) (email.SendResult, error) {
			t.Fatal("incapable provider must not be called")
			return email.SendResult{}, nil
		}),
		capabilities: email.Capabilities{},
	}
	capable := providerWithCapabilities{
		provider: newProvider("html", func(context.Context, email.Message) (email.SendResult, error) {
			return email.SendResult{State: email.SubmissionAccepted}, nil
		}),
		capabilities: email.Capabilities{HTML: true},
	}
	router, _ := email.NewRelay(incapable, capable)
	msg := message()
	msg.HTML = "<p>hello</p>"
	msg.Text = ""

	result, err := router.Send(context.Background(), msg)
	if err != nil || result.Provider != "html" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRelayPrefersHealthyOverDegraded(t *testing.T) {
	degraded := newProvider("degraded", func(context.Context, email.Message) (email.SendResult, error) {
		t.Fatal("degraded provider should not run while healthy provider accepts")
		return email.SendResult{}, nil
	})
	healthy := newProvider("healthy", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{State: email.SubmissionAccepted}, nil
	})
	router, _ := email.NewRelay(degraded, healthy)
	router = router.WithHealth(relaycore.HealthFunc(func(_ context.Context, provider string) relaycore.HealthStatus {
		if provider == "degraded" {
			return relaycore.HealthDegraded
		}
		return relaycore.HealthHealthy
	}))

	result, err := router.Send(context.Background(), message())
	if err != nil || result.Provider != "healthy" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestObserverUsesEmailChannelAndAttemptOrder(t *testing.T) {
	provider := newProvider("provider", func(context.Context, email.Message) (email.SendResult, error) {
		return email.SendResult{State: email.SubmissionAccepted, ProviderMessageID: "msg-1"}, nil
	})
	router, _ := email.NewRelay(provider)
	var events []relaycore.Event
	router = router.WithObserver(relaycore.ObserverFunc(func(_ context.Context, event relaycore.Event) {
		events = append(events, event)
	}))

	_, err := router.Send(context.Background(), message())
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]relaycore.EventKind, 0, len(events))
	for _, event := range events {
		if event.Channel != relaycore.ChannelEmail {
			t.Fatalf("channel = %q, want email", event.Channel)
		}
		kinds = append(kinds, event.Kind)
	}
	want := []relaycore.EventKind{relaycore.EventRouteSelected, relaycore.EventAttemptStarted, relaycore.EventAttemptFinished}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("event kinds = %v, want %v", kinds, want)
	}
	if events[2].Outcome != relaycore.SubmissionAccepted || events[2].ProviderMessageID != "msg-1" {
		t.Fatalf("finished event = %+v", events[2])
	}
}

func TestCapabilities(t *testing.T) {
	base := message()
	if !((email.Capabilities{}).Supports(base)) {
		t.Fatal("plain text single-recipient message should be supported")
	}

	html := base
	html.HTML = "<p>hello</p>"
	if (email.Capabilities{}).Supports(html) {
		t.Fatal("HTML should require HTML capability")
	}

	reply := base
	reply.ReplyTo = &email.Address{Email: "reply@example.com"}
	if (email.Capabilities{}).Supports(reply) {
		t.Fatal("Reply-To should require ReplyTo capability")
	}

	multiple := base
	multiple.To = append(multiple.To, email.Address{Email: "second@example.com"})
	if (email.Capabilities{}).Supports(multiple) {
		t.Fatal("multiple recipients should require MultipleRecipients capability")
	}
}

func message() email.Message {
	return email.Message{
		From:    email.Address{Email: "sender@example.com", Name: "Sender"},
		To:      []email.Address{{Email: "recipient@example.com", Name: "Recipient"}},
		Subject: "Hello",
		Text:    "hello",
	}
}

type provider struct {
	name  string
	send  func(context.Context, email.Message) (email.SendResult, error)
	calls int
}

func newProvider(name string, send func(context.Context, email.Message) (email.SendResult, error)) *provider {
	return &provider{name: name, send: send}
}

func (p *provider) Name() string { return p.name }

func (p *provider) Send(ctx context.Context, message email.Message) (email.SendResult, error) {
	p.calls++
	return p.send(ctx, message)
}

type providerWithCapabilities struct {
	provider     *provider
	capabilities email.Capabilities
}

func (p providerWithCapabilities) Name() string { return p.provider.Name() }
func (p providerWithCapabilities) Send(ctx context.Context, message email.Message) (email.SendResult, error) {
	return p.provider.Send(ctx, message)
}
func (p providerWithCapabilities) Capabilities() email.Capabilities { return p.capabilities }
