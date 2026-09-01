package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/messaging/email/systemmail"
)

type resultQueueStub struct {
	result     FailureResult
	recipients []systemmail.Recipient
}

func (queue *resultQueueStub) MarkSucceeded(context.Context, uuid.UUID, string, int32, *string) error {
	return nil
}
func (queue *resultQueueStub) ScheduleRetry(context.Context, uuid.UUID, string, time.Time, *int32, *string, string) error {
	return nil
}
func (queue *resultQueueStub) MarkFailed(context.Context, uuid.UUID, string, *int32, *string, string) (FailureResult, error) {
	return queue.result, nil
}
func (queue *resultQueueStub) ReleaseClaim(context.Context, uuid.UUID, string) error { return nil }
func (queue *resultQueueStub) ListNotificationRecipients(context.Context, uuid.UUID) ([]systemmail.Recipient, error) {
	return queue.recipients, nil
}

type endpointNotifierStub struct {
	inputs []systemmail.SendWebhookEndpointDisabledInput
}

func (notifier *endpointNotifierStub) SendWebhookEndpointDisabled(_ context.Context, input systemmail.SendWebhookEndpointDisabledInput) error {
	notifier.inputs = append(notifier.inputs, input)
	return nil
}

func TestMarkFailedNotifiesOnlyWhenEndpointIsAutoDisabled(t *testing.T) {
	queue := &resultQueueStub{result: FailureResult{AutoDisabled: true, TeamID: uuid.New(), EndpointURL: "https://example.com/hooks", ConsecutiveFailures: 20}, recipients: []systemmail.Recipient{{Name: "Ada", Email: "ada@example.com"}}}
	notifier := &endpointNotifierStub{}
	processor := NewProcessor(queue, nil, RetryPolicy{}, "worker").WithNotifier(notifier)
	status := int32(503)
	if err := processor.markFailed(context.Background(), ClaimedDelivery{ID: uuid.New()}, &status, nil, errors.New("unavailable")); err != nil {
		t.Fatal(err)
	}
	if len(notifier.inputs) != 1 || notifier.inputs[0].FailureCount != 20 || notifier.inputs[0].ResponseStatus != "503" {
		t.Fatalf("notifications=%+v", notifier.inputs)
	}
	queue.result.AutoDisabled = false
	if err := processor.markFailed(context.Background(), ClaimedDelivery{ID: uuid.New()}, &status, nil, errors.New("unavailable")); err != nil {
		t.Fatal(err)
	}
	if len(notifier.inputs) != 1 {
		t.Fatalf("notifications=%d, want 1", len(notifier.inputs))
	}
}
