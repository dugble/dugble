package renewal

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewWorkerValidatesDependencies(t *testing.T) {
	if _, err := NewWorker(nil, nil, DefaultConfig()); err == nil {
		t.Fatal("expected missing database error")
	}
}

func TestBatchResultCountsFailuresAsProcessed(t *testing.T) {
	var result BatchResult
	teamID := uuid.New()
	wantErr := errors.New("invalid subscription state")

	result.AddFailure(Failure{TeamID: teamID, Err: wantErr})

	if result.Processed != 1 || len(result.Failures) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Failures[0].TeamID != teamID || !errors.Is(result.Failures[0].Err, wantErr) {
		t.Fatalf("failure = %+v", result.Failures[0])
	}
}

func TestBatchResultCountsOutcomes(t *testing.T) {
	var result BatchResult
	for _, outcome := range []Outcome{OutcomeRenewed, OutcomePlanChanged, OutcomeCanceled, OutcomePastDue, OutcomePriceUnavailable} {
		result.Add(outcome)
	}
	if result.Processed != 5 || result.Renewed != 1 || result.PlanChanged != 1 || result.Canceled != 1 || result.PastDue != 1 || result.PriceUnavailable != 1 {
		t.Fatalf("result = %+v", result)
	}
}
