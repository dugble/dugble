package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
)

type job struct {
	name string
	run  func(context.Context) error
}

type result struct {
	name string
	err  error
}

type Worker struct {
	jobs []job
}

func newJobs(modules modules) []job {
	return []job{
		{name: "subscription renewal worker", run: modules.subscriptionRenewal},
		{name: "outbox relay", run: modules.outboxRelay},
		{name: "transactional email delivery consumer", run: modules.transactionalEmail},
		{name: "marketing email delivery consumer", run: modules.marketingEmail},
		{name: "system email consumer", run: modules.systemEmail},
		{name: "email tenant provisioning consumer", run: modules.emailTenantProvisioning},
		{name: "email feedback consumer", run: modules.emailFeedback},
		{name: "email feedback reconciler", run: modules.emailFeedbackReconciler},
		{name: "email feedback metrics collector", run: modules.emailFeedbackMetrics},
		{name: "SMS delivery consumer", run: modules.smsDelivery},
		{name: "SMS feedback reconciler", run: modules.smsFeedback},
		{name: "SMS campaign execution consumer", run: modules.smsCampaign},
		{name: "webhook delivery consumer", run: modules.webhookDelivery},
		{name: "sender domain reconciliation consumer", run: modules.domainReconciliation},
		{name: "broadcast execution consumer", run: modules.broadcastExecution},
		{name: "Sender ID reconciliation", run: modules.senderIDReconciliation},
	}
}

func newWorker(jobs ...job) (*Worker, error) {
	if len(jobs) == 0 {
		return nil, errors.New("worker requires at least one job")
	}
	seen := make(map[string]struct{}, len(jobs))
	for _, configured := range jobs {
		configured.name = strings.TrimSpace(configured.name)
		if configured.name == "" {
			return nil, errors.New("worker job name is required")
		}
		if configured.run == nil {
			return nil, fmt.Errorf("worker job %q has no run function", configured.name)
		}
		if _, exists := seen[configured.name]; exists {
			return nil, fmt.Errorf("duplicate worker job %q", configured.name)
		}
		seen[configured.name] = struct{}{}
	}
	return &Worker{jobs: jobs}, nil
}

func (worker *Worker) Run(ctx context.Context) error {
	if worker == nil || len(worker.jobs) == 0 {
		return errors.New("worker is not configured")
	}
	if ctx == nil {
		return errors.New("worker context is required")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan result, len(worker.jobs))
	for _, configured := range worker.jobs {
		configured := configured
		go func() {
			results <- result{name: configured.name, err: configured.run(runCtx)}
		}()
	}

	sentrymonitoring.Info("worker starting", "jobs", len(worker.jobs))
	completed := 0
	var runErr error
	contextDone := ctx.Done()
	for completed < len(worker.jobs) {
		select {
		case <-contextDone:
			contextDone = nil
			cancel()
		case completedJob := <-results:
			completed++
			if completedJob.err != nil && !errors.Is(completedJob.err, context.Canceled) {
				runErr = errors.Join(runErr, fmt.Errorf("run %s: %w", completedJob.name, completedJob.err))
				cancel()
				continue
			}
			if completedJob.err == nil && runCtx.Err() == nil {
				runErr = errors.Join(runErr, fmt.Errorf("run %s: job stopped unexpectedly", completedJob.name))
				cancel()
			}
		}
	}
	sentrymonitoring.Info("worker stopped")
	return runErr
}
