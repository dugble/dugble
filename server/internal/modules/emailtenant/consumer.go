package emailtenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	natsjs "github.com/nats-io/nats.go/jetstream"

	jetstreammessaging "github.com/dugble/dugble/server/internal/adapters/nats"
	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
)

var ErrProvisioningConsumerNotConfigured = errors.New("email tenant provisioning consumer is not fully configured")

type processedEventStore interface {
	IsProcessed(context.Context, string, uuid.UUID) (bool, error)
	MarkProcessed(context.Context, string, uuid.UUID, map[string]any) error
}

type consumerProvider interface {
	CreateOrUpdateConsumer(context.Context, string, natsjs.ConsumerConfig) (natsjs.Consumer, error)
}

type messagePublisher interface {
	Publish(context.Context, string, []byte, map[string]string, string) error
}

type provisioningCommandProcessor interface {
	Handle(context.Context, ProvisioningCommand) error
	HandleExhausted(context.Context, ProvisioningCommand, error) error
}

type ProvisioningConsumerConfig struct {
	Concurrency    int
	AckWait        time.Duration
	HandlerTimeout time.Duration
	MaxDeliver     int
	RetryBackOff   []time.Duration
}

type ProvisioningConsumer struct {
	provider  consumerProvider
	publisher messagePublisher
	processed processedEventStore
	processor provisioningCommandProcessor
	config    ProvisioningConsumerConfig
}

func NewProvisioningConsumer(client *jetstreammessaging.Client, processed processedEventStore, processor provisioningCommandProcessor, config ProvisioningConsumerConfig) *ProvisioningConsumer {
	if config.Concurrency <= 0 {
		config.Concurrency = 3
	}
	if config.AckWait <= 0 {
		config.AckWait = 2 * time.Minute
	}
	if config.HandlerTimeout <= 0 {
		config.HandlerTimeout = 60 * time.Second
	}
	if len(config.RetryBackOff) == 0 {
		config.RetryBackOff = DefaultProvisioningRetryBackOff()
	} else {
		config.RetryBackOff = append([]time.Duration(nil), config.RetryBackOff...)
	}
	if config.MaxDeliver <= 0 {
		config.MaxDeliver = len(config.RetryBackOff)
	}
	config.RetryBackOff = normalizeProvisioningRetryBackOff(config.RetryBackOff, config.MaxDeliver)
	return &ProvisioningConsumer{provider: client, publisher: client, processed: processed, processor: processor, config: config}
}

func (consumer *ProvisioningConsumer) Run(ctx context.Context) error {
	if consumer == nil || consumer.provider == nil || consumer.publisher == nil || consumer.processed == nil || consumer.processor == nil {
		return ErrProvisioningConsumerNotConfigured
	}
	streamConsumer, err := consumer.provider.CreateOrUpdateConsumer(ctx, jetstreammessaging.JobsStreamName, natsjs.ConsumerConfig{
		Name: ProvisionConsumer, Durable: ProvisionConsumer, Description: "Durable SES tenant provisioning jobs",
		DeliverPolicy: natsjs.DeliverAllPolicy, AckPolicy: natsjs.AckExplicitPolicy, AckWait: consumer.config.AckWait,
		MaxDeliver: consumer.config.MaxDeliver, BackOff: consumer.config.RetryBackOff,
		FilterSubject: ProvisionSubject, ReplayPolicy: natsjs.ReplayInstantPolicy,
		MaxAckPending: consumer.config.Concurrency * 4, MaxWaiting: consumer.config.Concurrency * 2, MaxRequestBatch: 1,
	})
	if err != nil {
		return fmt.Errorf("provision email tenant consumer: %w", err)
	}
	contexts := make([]natsjs.ConsumeContext, 0, consumer.config.Concurrency)
	for worker := range consumer.config.Concurrency {
		active, consumeErr := streamConsumer.Consume(func(message natsjs.Msg) { consumer.process(ctx, message) }, natsjs.PullMaxMessages(1))
		if consumeErr != nil {
			for _, item := range contexts {
				item.Stop()
			}
			return fmt.Errorf("start email tenant consumer worker %d: %w", worker, consumeErr)
		}
		contexts = append(contexts, active)
	}
	<-ctx.Done()
	for _, active := range contexts {
		active.Drain()
	}
	return nil
}

func (consumer *ProvisioningConsumer) process(parent context.Context, message natsjs.Msg) {
	metadata, err := message.Metadata()
	if err != nil {
		_ = message.Nak()
		return
	}
	command, err := decodeProvisioningCommand(message)
	if err != nil {
		consumer.deadLetter(parent, message, metadata, uuid.Nil, err)
		return
	}
	processed, err := consumer.processed.IsProcessed(parent, ProvisionConsumer, command.EventID)
	if err != nil {
		_ = message.Nak()
		return
	}
	if processed {
		_ = message.Ack()
		return
	}

	processorCtx, cancel := context.WithTimeout(parent, consumer.config.HandlerTimeout)
	err = consumer.processor.Handle(processorCtx, command)
	cancel()
	if err != nil {
		if parent.Err() != nil {
			return
		}
		if int(metadata.NumDelivered) >= consumer.config.MaxDeliver {
			finalizeCtx, finalizeCancel := context.WithTimeout(parent, consumer.config.HandlerTimeout)
			finalizeErr := consumer.processor.HandleExhausted(finalizeCtx, command, err)
			finalizeCancel()
			if finalizeErr != nil {
				_ = message.NakWithDelay(time.Minute)
				return
			}
			consumer.deadLetter(parent, message, metadata, command.EventID, err)
			return
		}
		_ = message.NakWithDelay(provisioningRetryDelay(consumer.config.RetryBackOff, metadata.NumDelivered))
		return
	}

	if err := consumer.processed.MarkProcessed(parent, ProvisionConsumer, command.EventID, map[string]any{
		"subject": message.Subject(), "stream_sequence": metadata.Sequence.Stream, "deliveries": metadata.NumDelivered,
	}); err != nil {
		_ = message.Nak()
		return
	}
	_ = message.Ack()
}

func decodeProvisioningCommand(message natsjs.Msg) (ProvisioningCommand, error) {
	var command ProvisioningCommand
	if err := json.Unmarshal(message.Data(), &command); err != nil {
		return ProvisioningCommand{}, fmt.Errorf("decode email tenant provisioning command: %w", err)
	}
	if err := ValidateProvisioningCommand(command); err != nil {
		return ProvisioningCommand{}, err
	}
	headerID, err := uuid.Parse(strings.TrimSpace(message.Headers().Get("Dugble-Event-Id")))
	if err != nil || headerID != command.EventID {
		return ProvisioningCommand{}, errors.New("email tenant event ID does not match outbox header")
	}
	return command, nil
}

func (consumer *ProvisioningConsumer) deadLetter(ctx context.Context, message natsjs.Msg, metadata *natsjs.MsgMetadata, eventID uuid.UUID, cause error) {
	headers := map[string]string{
		"Dugble-Original-Subject":   message.Subject(),
		"Dugble-Dead-Letter-Reason": provisioningFailureReason(cause),
		"Dugble-Delivery-Count":     strconv.FormatUint(metadata.NumDelivered, 10),
	}
	messageID := eventID.String() + "-dlq"
	if eventID == uuid.Nil {
		messageID = fmt.Sprintf("%s-%d-dlq", ProvisionConsumer, metadata.Sequence.Stream)
	}
	if err := consumer.publisher.Publish(ctx, ProvisionDLQSubject, message.Data(), headers, messageID); err != nil {
		_ = message.NakWithDelay(time.Minute)
		return
	}
	if err := message.TermWithReason(provisioningFailureReason(cause)); err != nil {
		sentrymonitoring.Error("failed to terminate dead-lettered tenant provisioning command", "event_id", eventID, "error", err)
	}
}

var defaultProvisioningRetryBackOff = []time.Duration{
	time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
}

func DefaultProvisioningRetryBackOff() []time.Duration {
	return append([]time.Duration(nil), defaultProvisioningRetryBackOff...)
}

func normalizeProvisioningRetryBackOff(delays []time.Duration, maxDeliver int) []time.Duration {
	if maxDeliver <= 0 || len(delays) == 0 {
		return nil
	}
	result := make([]time.Duration, maxDeliver)
	for index := range result {
		source := index
		if source >= len(delays) {
			source = len(delays) - 1
		}
		result[index] = delays[source]
	}
	return result
}

func provisioningRetryDelay(delays []time.Duration, delivered uint64) time.Duration {
	if len(delays) == 0 {
		return 0
	}
	index := int(delivered) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(delays) {
		index = len(delays) - 1
	}
	return delays[index]
}

func provisioningFailureReason(err error) string {
	if err == nil {
		return "unknown email tenant provisioning failure"
	}
	reason := strings.TrimSpace(err.Error())
	if len(reason) > 512 {
		return reason[:512]
	}
	if reason == "" {
		return "unknown email tenant provisioning failure"
	}
	return reason
}
