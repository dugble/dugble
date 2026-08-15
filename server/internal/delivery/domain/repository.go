package domainreconciliation

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainmodule "github.com/dugble/dugble/server/internal/modules/domain"
	"github.com/dugble/dugble/server/internal/platform/systemmail"
)

type repository interface {
	ClaimPendingReconciliations(context.Context, string, int32, time.Time) ([]domainmodule.ReconciliationClaim, error)
	CompleteReconciliation(context.Context, uuid.UUID, string, string, []domainmodule.VerificationRecord, time.Time) (domainmodule.SenderDomain, error)
	ResetReconciliationAttempts(context.Context, uuid.UUID) error
	RecordReconciliationFailure(context.Context, uuid.UUID, string, error, time.Time) (domainmodule.SenderDomain, error)
	CompleteHealthCheck(context.Context, uuid.UUID, string, time.Time) (domainmodule.SenderDomain, error)
	RecordHealthFailure(context.Context, uuid.UUID, string, error, int32, time.Time) (domainmodule.SenderDomain, error)
	ListNotificationRecipients(context.Context, uuid.UUID) ([]systemmail.Recipient, error)
}

type checker interface {
	Check(context.Context, domainmodule.SenderDomain) (domainmodule.ReconciliationResult, error)
}
