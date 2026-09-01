package broadcast

import (
	"context"
	"errors"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/integrations/monitoring/sentry"
)

var ErrJobNotConfigured = errors.New("broadcast execution job is not configured")

type jobProcessor interface {
	ProcessBatch(context.Context, int) error
}

type JobConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

type Job struct {
	processor jobProcessor
	config    JobConfig
}

func NewJob(processor jobProcessor, config JobConfig) *Job {
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	return &Job{processor: processor, config: config}
}

func (j *Job) Run(ctx context.Context) error {
	if j == nil || j.processor == nil {
		return ErrJobNotConfigured
	}

	for {
		if err := j.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
			sentrymonitoring.Error("broadcast execution poll failed", "error", err)
		}

		timer := time.NewTimer(j.config.PollInterval)
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

func (j *Job) poll(ctx context.Context) error {
	if j == nil || j.processor == nil {
		return ErrJobNotConfigured
	}
	return j.processor.ProcessBatch(ctx, j.config.BatchSize)
}
