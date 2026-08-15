package broadcast

import (
	"context"
	"errors"
	"testing"
)

type fakeJobProcessor struct {
	calls int
	err   error
}

func (p *fakeJobProcessor) ProcessBatch(context.Context, int) error {
	p.calls++
	return p.err
}

func TestJobPollUsesConfiguredBatchSize(t *testing.T) {
	processor := &fakeJobProcessor{}
	job := NewJob(processor, JobConfig{BatchSize: 25})

	if err := job.poll(context.Background()); err != nil {
		t.Fatalf("poll returned error: %v", err)
	}
	if processor.calls != 1 {
		t.Fatalf("expected one processor call, got %d", processor.calls)
	}
}

func TestJobPollReturnsProcessorError(t *testing.T) {
	expected := errors.New("processor failed")
	processor := &fakeJobProcessor{err: expected}
	job := NewJob(processor, JobConfig{BatchSize: 10})

	if err := job.poll(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestJobPollRequiresProcessor(t *testing.T) {
	var job *Job
	if err := job.poll(context.Background()); !errors.Is(err, ErrJobNotConfigured) {
		t.Fatalf("expected job configuration error, got %v", err)
	}
}
