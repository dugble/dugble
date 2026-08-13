package nats

import (
	"context"
	"errors"
	"strings"
	"testing"

	natsgo "github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

func TestCreateOrUpdateConsumerValidatesDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client *Client
		ctx    context.Context
		stream string
		config natsjs.ConsumerConfig
		want   string
	}{
		{name: "context", client: &Client{}, stream: "JOBS", config: natsjs.ConsumerConfig{Durable: "worker"}, want: "context is required"},
		{name: "client", ctx: context.Background(), stream: "JOBS", config: natsjs.ConsumerConfig{Durable: "worker"}, want: ErrClientUnavailable.Error()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := test.client.CreateOrUpdateConsumer(test.ctx, test.stream, test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CreateOrUpdateConsumer() error = %v, want %q", err, test.want)
			}
			if test.name == "client" && !errors.Is(err, ErrClientUnavailable) {
				t.Fatalf("CreateOrUpdateConsumer() error does not wrap ErrClientUnavailable: %v", err)
			}
		})
	}
}

func TestConnectionOptionsRetryUntilNATSRecovers(t *testing.T) {
	t.Parallel()

	options := natsgo.GetDefaultOptions()
	for _, apply := range connectionOptions("worker") {
		if err := apply(&options); err != nil {
			t.Fatalf("apply connection option: %v", err)
		}
	}
	if options.MaxReconnect != -1 {
		t.Fatalf("MaxReconnect = %d, want -1", options.MaxReconnect)
	}
	if options.ReconnectWait != defaultReconnectWait {
		t.Fatalf("ReconnectWait = %v, want %v", options.ReconnectWait, defaultReconnectWait)
	}
}

func TestConsumerNamePrefersDurable(t *testing.T) {
	t.Parallel()

	config := natsjs.ConsumerConfig{Durable: " durable ", Name: "name"}
	if got := consumerName(config); got != "durable" {
		t.Fatalf("consumerName() = %q, want durable", got)
	}
}
