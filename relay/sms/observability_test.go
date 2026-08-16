package sms_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	relaycore "github.com/dugble/relay"
	"github.com/dugble/relay/providers/mock"
	"github.com/dugble/relay/sms"
)

func TestObserverReceivesRouteAndAttemptLifecycle(t *testing.T) {
	primary := mock.New("primary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionRejected}, errors.New("definite rejection with provider detail")
	})
	secondary := mock.New("secondary", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted, ProviderMessageID: "msg-123"}, nil
	})

	router, err := sms.NewRelay(primary, secondary)
	if err != nil {
		t.Fatal(err)
	}
	var events []relaycore.Event
	router = router.WithObserver(relaycore.ObserverFunc(func(_ context.Context, event relaycore.Event) {
		events = append(events, event)
	}))

	result, err := router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "secondary" || result.State != sms.SubmissionAccepted {
		t.Fatalf("unexpected result: %+v", result)
	}

	if len(events) != 5 {
		t.Fatalf("events = %d, want 5: %+v", len(events), events)
	}
	if events[0].Kind != relaycore.EventRouteSelected || !reflect.DeepEqual(events[0].Providers, []string{"primary", "secondary"}) {
		t.Fatalf("route event = %+v", events[0])
	}
	if events[1].Kind != relaycore.EventAttemptStarted || events[1].Provider != "primary" {
		t.Fatalf("first attempt start = %+v", events[1])
	}
	if events[2].Kind != relaycore.EventAttemptFinished || events[2].Provider != "primary" || events[2].Outcome != relaycore.OutcomeRejected || !events[2].HadError {
		t.Fatalf("first attempt finish = %+v", events[2])
	}
	if events[3].Kind != relaycore.EventAttemptStarted || events[3].Provider != "secondary" {
		t.Fatalf("second attempt start = %+v", events[3])
	}
	if events[4].Kind != relaycore.EventAttemptFinished || events[4].Provider != "secondary" || events[4].Outcome != relaycore.OutcomeAccepted || events[4].ProviderMessageID != "msg-123" || events[4].HadError {
		t.Fatalf("second attempt finish = %+v", events[4])
	}
	for _, event := range events {
		if event.Channel != relaycore.ChannelSMS {
			t.Fatalf("event channel = %q, want sms", event.Channel)
		}
	}
}

func TestObserverReceivesSkipAndExhaustionReasons(t *testing.T) {
	incapable := providerWithCapabilities{
		Provider: mock.New("incapable", nil),
		capabilities: sms.Capabilities{
			RequiresE164Recipient: true,
		},
	}
	unavailable := mock.New("unavailable", nil)

	router, err := sms.NewRelay(incapable, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithHealth(relaycore.HealthFunc(func(_ context.Context, provider string) relaycore.HealthStatus {
		if provider == "unavailable" {
			return relaycore.HealthUnavailable
		}
		return relaycore.HealthHealthy
	}))
	var events []relaycore.Event
	router = router.WithObserver(relaycore.ObserverFunc(func(_ context.Context, event relaycore.Event) {
		events = append(events, event)
	}))

	_, err = router.Send(context.Background(), sms.Message{To: "0200000000", From: "Acme", Text: "hello"})
	if !errors.Is(err, sms.ErrNoAvailableProviders) {
		t.Fatalf("Send() error = %v, want ErrNoAvailableProviders", err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3: %+v", len(events), events)
	}
	if events[0].Kind != relaycore.EventProviderSkipped || events[0].Provider != "incapable" || events[0].Reason != relaycore.ReasonUnsupportedCapability {
		t.Fatalf("capability skip = %+v", events[0])
	}
	if events[1].Kind != relaycore.EventProviderSkipped || events[1].Provider != "unavailable" || events[1].Reason != relaycore.ReasonProviderUnavailable {
		t.Fatalf("health skip = %+v", events[1])
	}
	if events[2].Kind != relaycore.EventRouteExhausted || events[2].Reason != relaycore.ReasonNoAvailableProviders {
		t.Fatalf("exhaustion event = %+v", events[2])
	}
}

func TestObserverReceivesAllRejectedExhaustion(t *testing.T) {
	provider := mock.New("provider", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionRejected}, errors.New("rejected")
	})
	router, err := sms.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}
	var events []relaycore.Event
	router = router.WithObserver(relaycore.ObserverFunc(func(_ context.Context, event relaycore.Event) {
		events = append(events, event)
	}))

	_, err = router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if !errors.Is(err, sms.ErrAllRejected) {
		t.Fatalf("Send() error = %v, want ErrAllRejected", err)
	}
	last := events[len(events)-1]
	if last.Kind != relaycore.EventRouteExhausted || last.Reason != relaycore.ReasonAllRejected {
		t.Fatalf("last event = %+v", last)
	}
}

func TestObserverPanicDoesNotAbortDelivery(t *testing.T) {
	provider := mock.New("provider", func(context.Context, sms.Message) (sms.SendResult, error) {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	})
	router, err := sms.NewRelay(provider)
	if err != nil {
		t.Fatal(err)
	}
	router = router.WithObserver(relaycore.ObserverFunc(func(context.Context, relaycore.Event) {
		panic("observer failed")
	}))

	result, err := router.Send(context.Background(), sms.Message{To: "+233200000000", From: "Acme", Text: "hello"})
	if err != nil || result.State != sms.SubmissionAccepted || provider.Calls() != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, provider.Calls())
	}
}

type providerWithCapabilities struct {
	*mock.Provider
	capabilities sms.Capabilities
}

func (p providerWithCapabilities) Capabilities() sms.Capabilities {
	return p.capabilities
}
