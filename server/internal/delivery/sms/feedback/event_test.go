package feedback

import (
	"testing"
	"time"

	attempt "github.com/dugble/dugble/server/internal/delivery/attempt"
	provider "github.com/dugble/dugble/server/internal/providers"

	"github.com/google/uuid"
)

func TestStatusEventMapsProviderStates(t *testing.T) {
	t.Parallel()

	attemptID := uuid.New()
	pending := PendingMessage{
		AttemptID:         attemptID,
		ProviderID:        "moolre",
		ProviderMessageID: "provider-message",
		ReconcileAttempts: 2,
	}
	tests := []struct {
		providerStatus string
		status         provider.SMSStatus
		want           attempt.AttemptStatus
	}{
		{providerStatus: "queued", status: provider.SMSPending, want: attempt.StatusAccepted},
		{providerStatus: "submitted", status: provider.SMSPending, want: attempt.StatusSubmitted},
		{providerStatus: "sent", status: provider.SMSPending, want: attempt.StatusSent},
		{providerStatus: "delivered", status: provider.SMSDelivered, want: attempt.StatusDelivered},
		{providerStatus: "undelivered", status: provider.SMSFailed, want: attempt.StatusPermanentFailure},
		{providerStatus: "rejected", status: provider.SMSFailed, want: attempt.StatusRejected},
		{providerStatus: "expired", status: provider.SMSFailed, want: attempt.StatusExpired},
		{providerStatus: "unknown", status: provider.SMSUnknown, want: attempt.StatusUnknown},
	}
	for _, test := range tests {
		test := test
		t.Run(test.providerStatus, func(t *testing.T) {
			t.Parallel()
			event, err := statusEvent(pending, provider.SMSStatusResult{
				ProviderMessageID: pending.ProviderMessageID,
				ProviderStatus:    test.providerStatus,
				Status:            test.status,
			}, time.Now().UTC())
			if err != nil {
				t.Fatalf("statusEvent() error = %v", err)
			}
			if event.Status != test.want {
				t.Fatalf("statusEvent() status = %s, want %s", event.Status, test.want)
			}
			if event.AttemptID != attemptID || event.ProviderEventID != "poll:"+attemptID.String()+":3:"+test.providerStatus {
				t.Fatalf("statusEvent() identity = %+v", event)
			}
		})
	}
}
