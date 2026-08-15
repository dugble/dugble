package domainreconciliation

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domainmodule "github.com/dugble/dugble/server/internal/modules/domain"
)

func TestNextVerificationDelayKeepsPendingChecksFast(t *testing.T) {
	id := uuid.MustParse("19a37bcd-5c35-49d0-84a4-2ade18222ddf")
	config := Config{PendingCheckInterval: 30 * time.Second, HealthCheckInterval: 24 * time.Hour}

	want := jitter(config.PendingCheckInterval, id)
	for _, attempt := range []int32{1, 2, 5, 20} {
		if got := nextVerificationDelay(domainmodule.StatusPending, nil, attempt, id, config); got != want {
			t.Fatalf("pending attempt %d delay = %s, want %s", attempt, got, want)
		}
	}
}

func TestNextVerificationDelayBacksOffOnlyOnErrors(t *testing.T) {
	id := uuid.MustParse("19a37bcd-5c35-49d0-84a4-2ade18222ddf")
	config := Config{PendingCheckInterval: 30 * time.Second, HealthCheckInterval: 24 * time.Hour}

	for _, attempt := range []int32{1, 2, 3, 6} {
		want := nextCheckDelay(max(attempt-1, 0), id)
		got := nextVerificationDelay(domainmodule.StatusPending, errors.New("provider unavailable"), attempt, id, config)
		if got != want {
			t.Fatalf("error attempt %d delay = %s, want %s", attempt, got, want)
		}
	}
}

func TestNextVerificationDelayMovesVerifiedDomainsToHealthCadence(t *testing.T) {
	id := uuid.MustParse("19a37bcd-5c35-49d0-84a4-2ade18222ddf")
	config := Config{PendingCheckInterval: 30 * time.Second, HealthCheckInterval: 24 * time.Hour}

	want := jitter(config.HealthCheckInterval, id)
	if got := nextVerificationDelay(domainmodule.StatusVerified, nil, 12, id, config); got != want {
		t.Fatalf("verified delay = %s, want %s", got, want)
	}
}
