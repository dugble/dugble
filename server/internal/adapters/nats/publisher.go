package nats

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	natsgo "github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"
	"github.com/newrelic/go-agent/v3/integrations/nrnats"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Publisher publishes a deduplicated message to JetStream.
type Publisher interface {
	Publish(context.Context, string, []byte, map[string]string, string) error
}

type publishError struct {
	err       error
	code      string
	retryable bool
	permanent bool
}

func (err publishError) Error() string       { return err.err.Error() }
func (err publishError) Unwrap() error       { return err.err }
func (err publishError) Retryable() bool     { return err.retryable }
func (err publishError) Permanent() bool     { return err.permanent }
func (err publishError) FailureCode() string { return err.code }

func (client *Client) Publish(
	ctx context.Context,
	subject string,
	payload []byte,
	headers map[string]string,
	messageID string,
) error {
	if ctx == nil {
		return publishError{err: fmt.Errorf("JetStream publish context is required"), code: "invalid_context", permanent: true}
	}
	if client == nil || client.jetStream == nil || client.connection == nil {
		return publishError{err: ErrClientUnavailable, code: "client_unavailable", retryable: true}
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return publishError{err: fmt.Errorf("JetStream subject is required"), code: "invalid_subject", permanent: true}
	}
	if len(payload) > maxMessageSize {
		return publishError{
			err:       fmt.Errorf("JetStream message payload is %d bytes; maximum is %d", len(payload), maxMessageSize),
			code:      "message_too_large",
			permanent: true,
		}
	}

	message := &natsgo.Msg{Subject: subject, Data: payload, Header: natsgo.Header{}}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		if key != "" {
			message.Header.Set(key, value)
		}
	}

	transaction := newrelic.FromContext(ctx)
	ownsTransaction := false
	if transaction == nil && client.monitoring != nil {
		transaction = client.monitoring.StartTransaction("NATS publish " + subject)
		ctx = newrelic.NewContext(ctx, transaction)
		ownsTransaction = true
	}
	if ownsTransaction {
		defer transaction.End()
	}
	if transaction != nil {
		transaction.AddAttribute("messaging.system", "nats")
		transaction.AddAttribute("messaging.destination", subject)
		transaction.InsertDistributedTraceHeaders(http.Header(message.Header))
		defer nrnats.StartPublishSegment(transaction, client.connection, subject).End()
	}

	options := make([]natsjs.PublishOpt, 0, 1)
	if messageID = strings.TrimSpace(messageID); messageID != "" {
		options = append(options, natsjs.WithMsgID(messageID))
	}
	if _, err := client.jetStream.PublishMsg(ctx, message, options...); err != nil {
		wrapped := fmt.Errorf("publish JetStream message to %s: %w", subject, err)
		if transaction != nil {
			transaction.NoticeError(wrapped)
		}
		return publishError{err: wrapped, code: "jetstream_publish_failed", retryable: true}
	}
	return nil
}
