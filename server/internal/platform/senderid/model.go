package senderid

import (
	"context"

	relaysenderid "github.com/dugble/dugble/server/internal/relay/senderid"
)

const (
	ProviderLeamout = "leamout"
	ProviderMNotify = "mnotify"
	ProviderMoolre  = "moolre"
	ProviderRunnage = "runnage"

	// Deprecated: use relay/senderid status constants.
	StatusPending   = string(relaysenderid.StatusPending)
	StatusApproved  = string(relaysenderid.StatusApproved)
	StatusRejected  = string(relaysenderid.StatusRejected)
	StatusSuspended = string(relaysenderid.StatusSuspended)
	StatusUnknown   = string(relaysenderid.StatusUnknown)
)

// CreateRequest is retained for legacy adapter compatibility. New code should
// use relay/senderid.CreateRequest.
type CreateRequest struct {
	SenderID string
	Purpose  string
}

type CreateResponse struct {
	ProviderID string
	SenderID   string
	Status     string
}

type StatusResponse struct {
	ProviderID     string
	SenderID       string
	Status         string
	ProviderStatus string
	Whitelisted    bool
}

// Provider is retained for legacy adapter compatibility while adapters migrate
// to relay/senderid contracts.
type Provider interface {
	ID() string
	Create(context.Context, CreateRequest) (*CreateResponse, error)
	CheckStatus(context.Context, string) (*StatusResponse, error)
}
