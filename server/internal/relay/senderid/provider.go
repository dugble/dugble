package senderid

import "context"

// Creator submits Sender IDs for provider approval.
type Creator interface {
	Name() string
	CreateSenderID(context.Context, CreateRequest) (CreateResult, error)
}

// StatusChecker reconciles Sender ID approval state with a provider.
type StatusChecker interface {
	Name() string
	CheckSenderIDStatus(context.Context, StatusRequest) (StatusResult, error)
}
