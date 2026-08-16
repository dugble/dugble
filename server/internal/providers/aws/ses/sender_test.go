package ses

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/smithy-go"
)

type sendClientStub struct {
	retryer aws.Retryer
}

func (stub *sendClientStub) SendEmail(
	_ context.Context,
	_ *sesv2.SendEmailInput,
	options ...func(*sesv2.Options),
) (*sesv2.SendEmailOutput, error) {
	configured := sesv2.Options{}
	for _, configure := range options {
		configure(&configured)
	}
	stub.retryer = configured.Retryer
	return &sesv2.SendEmailOutput{MessageId: aws.String("provider-message")}, nil
}

func TestSendDisablesSDKRetries(t *testing.T) {
	stub := &sendClientStub{}
	client := &Client{
		defaultRegion: "eu-north-1",
		defaultFrom:   "sender@example.com",
		v2SendingClients: map[string]sesV2SendAPI{
			"eu-north-1": stub,
		},
	}

	result, err := client.Send(context.Background(), Message{
		Region:           "eu-north-1",
		Stream:           "transactional",
		ConfigurationSet: "dugble-transactional",
		SESTenantName:    "dugble-system",
		From:             Address{Email: "sender@example.com"},
		To:               []Address{{Email: "recipient@example.com"}},
		Subject:          "Retry safety",
		Text:             "Test message",
	})
	if err != nil {
		t.Fatalf("send email: %v", err)
	}
	if result.Provider != ProviderSES || result.MessageID != "provider-message" {
		t.Fatalf("unexpected send result: %#v", result)
	}
	if stub.retryer == nil {
		t.Fatal("expected an explicit SES retryer")
	}
	if attempts := stub.retryer.MaxAttempts(); attempts != 1 {
		t.Fatalf("expected one SDK attempt, got %d", attempts)
	}
}

func TestRequestTimeoutIsSubmissionUnknown(t *testing.T) {
	err := classifySESFailure(&smithy.GenericAPIError{
		Code:    "RequestTimeoutException",
		Message: "response outcome unknown",
		Fault:   smithy.FaultServer,
	})
	if !IsSubmissionUnknown(err) {
		t.Fatalf("expected submission-unknown SES error, got %v", err)
	}
	if IsRetryable(err) {
		t.Fatal("submission-unknown errors must not be classified as retryable")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected classified provider error, not a raw context deadline")
	}
}
