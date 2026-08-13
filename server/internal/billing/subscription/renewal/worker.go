package renewal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const claimNextDueSQL = `
SELECT team_id
FROM team_subscriptions
WHERE status IN ('active', 'past_due')
  AND current_period_end <= now()
  AND NOT (team_id = ANY($1::uuid[]))
ORDER BY current_period_end, team_id
FOR UPDATE SKIP LOCKED
LIMIT 1
`

type Config struct {
	PollInterval time.Duration
	BatchSize    int
	OnFailure    func(context.Context, Failure)
}

func DefaultConfig() Config {
	return Config{PollInterval: time.Minute, BatchSize: 100}
}

type Worker struct {
	db      *pgxpool.Pool
	service Processor
	config  Config
}

type teamProcessingError struct {
	teamID uuid.UUID
	err    error
}

func (err *teamProcessingError) Error() string {
	return fmt.Sprintf("process subscription renewal for team %s: %v", err.teamID, err.err)
}

func (err *teamProcessingError) Unwrap() error {
	return err.err
}

func NewWorker(db *pgxpool.Pool, service Processor, config Config) (*Worker, error) {
	if db == nil {
		return nil, errors.New("subscription renewal database is required")
	}
	if service == nil {
		return nil, errors.New("subscription renewal processor is required")
	}
	if config.PollInterval <= 0 {
		return nil, errors.New("subscription renewal poll interval must be positive")
	}
	if config.BatchSize <= 0 {
		return nil, errors.New("subscription renewal batch size must be positive")
	}
	return &Worker{db: db, service: service, config: config}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("subscription renewal context is required")
	}
	if _, err := w.ProcessBatch(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := w.ProcessBatch(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) ProcessBatch(ctx context.Context) (BatchResult, error) {
	processedTeams := make([]uuid.UUID, 0, w.config.BatchSize)
	result := BatchResult{}
	for len(processedTeams) < w.config.BatchSize {
		teamID, renewal, found, err := w.processNext(ctx, processedTeams)
		if err != nil {
			var processingErr *teamProcessingError
			if !errors.As(err, &processingErr) {
				return result, err
			}
			processedTeams = append(processedTeams, processingErr.teamID)
			failure := Failure{TeamID: processingErr.teamID, Err: processingErr.err}
			result.AddFailure(failure)
			if w.config.OnFailure != nil {
				w.config.OnFailure(ctx, failure)
			}
			continue
		}
		if !found {
			return result, nil
		}
		processedTeams = append(processedTeams, teamID)
		result.Add(renewal.Outcome)
	}
	return result, nil
}

func (w *Worker) processNext(ctx context.Context, excluded []uuid.UUID) (uuid.UUID, Result, bool, error) {
	tx, err := w.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, Result{}, false, fmt.Errorf("begin subscription renewal: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var teamID uuid.UUID
	if err := tx.QueryRow(ctx, claimNextDueSQL, excluded).Scan(&teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, Result{}, false, nil
		}
		return uuid.Nil, Result{}, false, fmt.Errorf("claim due subscription: %w", err)
	}
	renewal, err := w.service.ProcessTeam(ctx, tx, teamID)
	if err != nil {
		return teamID, Result{}, true, &teamProcessingError{teamID: teamID, err: err}
	}
	if err := tx.Commit(ctx); err != nil {
		return teamID, Result{}, true, fmt.Errorf("commit subscription renewal for team %s: %w", teamID, err)
	}
	return teamID, renewal, true, nil
}
