package providers

import (
	"context"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

// Sender submits SMS messages through a communications provider.
type Sender interface {
	sms.Provider
}

// SenderIDCreator submits sender IDs for provider approval.
type SenderIDCreator interface {
	Name() string
	CreateSenderID(context.Context, CreateSenderIDRequest) (CreateSenderIDResult, error)
}

// SMSStatusChecker reconciles the delivery state of a submitted SMS.
type SMSStatusChecker interface {
	Name() string
	CheckSMSStatus(context.Context, SMSStatusRequest) (SMSStatusResult, error)
}

// SenderIDStatusChecker reconciles the approval state of a submitted sender ID.
type SenderIDStatusChecker interface {
	Name() string
	CheckSenderIDStatus(context.Context, SenderIDStatusRequest) (SenderIDStatusResult, error)
}

