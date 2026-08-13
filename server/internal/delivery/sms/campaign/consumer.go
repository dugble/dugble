package campaign

import (
	"context"
	"errors"
	"time"

	sentrymonitoring "github.com/coffeyvidzro/dugble/server/internal/adapters/monitoring/sentry"
)

type Config struct {
	PollInterval time.Duration
	BatchSize    int
}
type Consumer struct {
	processor *Processor
	config    Config
}

func NewConsumer(processor *Processor, config Config) *Consumer {
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	return &Consumer{processor: processor, config: config}
}
func (c *Consumer) Run(ctx context.Context) error {
	for {
		if err := c.processor.ProcessBatch(ctx, c.config.BatchSize); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("SMS campaign execution poll failed", "error", err)
		}
		timer := time.NewTimer(c.config.PollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}
