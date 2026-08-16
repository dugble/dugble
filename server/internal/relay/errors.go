package relay

import "errors"

var (
	ErrInvalidMessage       = errors.New("invalid message")
	ErrNoProviders          = errors.New("no providers configured")
	ErrNoCapableProviders   = errors.New("no providers support the message")
	ErrNoAvailableProviders = errors.New("no providers are currently available")
	ErrAllRejected          = errors.New("all providers rejected the message")
)
