package senderid

import "context"

// Status is Dugble's provider-neutral Sender ID approval state.
type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusSuspended Status = "suspended"
	StatusUnknown   Status = "unknown"

	// StatusActive is retained as a semantic alias for provider integrations
	// that describe an approved Sender ID as active.
	StatusActive = StatusApproved
)

// CreateRequest is the provider-neutral Sender ID registration request.
type CreateRequest struct {
	Name        string
	CountryCode string
	Purpose     string
}

// CreateResult describes a provider registration attempt.
type CreateResult struct {
	Provider          string
	Name              string
	ProviderReference string
	Status            Status
	ProviderStatus    string
	ProviderCode      string
}

// StatusRequest identifies a previously submitted provider registration.
type StatusRequest struct {
	Name              string
	ProviderReference string
}

// StatusResult is the normalized provider approval state.
type StatusResult struct {
	Provider          string
	Name              string
	ProviderReference string
	Status            Status
	ProviderStatus    string
	ProviderCode      string
}

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
