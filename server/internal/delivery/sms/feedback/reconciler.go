package feedback

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	provider "github.com/dugble/dugble/server/internal/providers"
	relaysms "github.com/dugble/dugble/server/internal/relay/sms"
)

type pendingRepository interface {
	ListPending(context.Context, int32) ([]PendingMessage, error)
}

type providerLookup interface {
	Provider(string) relaysms.Provider
}

type ReconcilerConfig struct {
	BatchSize       int32
	Concurrency     int
	ProviderTimeout time.Duration
}

type Reconciler struct {
	repository pendingRepository
	providers  providerLookup
	processor  *Processor
	config     ReconcilerConfig
	now        func() time.Time
}

func NewReconciler(repository pendingRepository, providers providerLookup, processor *Processor, config ReconcilerConfig) *Reconciler {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.ProviderTimeout <= 0 {
		config.ProviderTimeout = 15 * time.Second
	}
	return &Reconciler{
		repository: repository,
		providers:  providers,
		processor:  processor,
		config:     config,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (reconciler *Reconciler) ReconcileBatch(ctx context.Context) (int, error) {
	if reconciler == nil || reconciler.repository == nil || reconciler.providers == nil || reconciler.processor == nil {
		return 0, ErrReconcilerNotConfigured
	}
	messages, err := reconciler.repository.ListPending(ctx, reconciler.config.BatchSize)
	if err != nil {
		return 0, err
	}
	if len(messages) == 0 {
		return 0, nil
	}

	semaphore := make(chan struct{}, reconciler.config.Concurrency)
	var wait sync.WaitGroup
	var mutex sync.Mutex
	var joined error
	processed := 0
	for _, message := range messages {
		message := message
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mutex.Lock()
				joined = errors.Join(joined, ctx.Err())
				mutex.Unlock()
				return
			}

			upstream := reconciler.providers.Provider(message.ProviderID)
			checker, ok := upstream.(provider.SMSStatusChecker)
			if !ok {
				mutex.Lock()
				joined = errors.Join(joined, fmt.Errorf("SMS provider %q does not support status reconciliation", message.ProviderID))
				mutex.Unlock()
				return
			}

			providerCtx, cancel := context.WithTimeout(ctx, reconciler.config.ProviderTimeout)
			response, checkErr := checker.CheckSMSStatus(providerCtx, provider.SMSStatusRequest{
				Reference:         message.ID.String(),
				ProviderMessageID: message.ProviderMessageID,
			})
			cancel()
			if checkErr != nil {
				mutex.Lock()
				joined = errors.Join(joined, fmt.Errorf("check %s SMS %s status: %w", message.ProviderID, message.ProviderMessageID, checkErr))
				mutex.Unlock()
				return
			}
			event, eventErr := statusEvent(message, response, reconciler.now())
			if eventErr != nil {
				mutex.Lock()
				joined = errors.Join(joined, eventErr)
				mutex.Unlock()
				return
			}
			if _, processErr := reconciler.processor.Handle(ctx, event); processErr != nil {
				mutex.Lock()
				joined = errors.Join(joined, processErr)
				mutex.Unlock()
				return
			}
			mutex.Lock()
			processed++
			mutex.Unlock()
		}()
	}
	wait.Wait()
	return processed, joined
}
