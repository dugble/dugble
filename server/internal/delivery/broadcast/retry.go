package broadcastexecution

import (
	"strings"
	"time"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const maxFanoutFailureMessageRunes = 255

type RetryPolicy struct {
	MaxAttempts int32
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 8,
		BaseDelay:   time.Minute,
		MaxDelay:    30 * time.Minute,
	}
}

func (policy RetryPolicy) normalized() RetryPolicy {
	defaults := DefaultRetryPolicy()
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = defaults.MaxAttempts
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = defaults.BaseDelay
	}
	if policy.MaxDelay < policy.BaseDelay {
		policy.MaxDelay = defaults.MaxDelay
	}
	return policy
}

func (policy RetryPolicy) delay(attemptCount int32) time.Duration {
	policy = policy.normalized()
	delay := policy.BaseDelay
	for attempt := int32(0); attempt < attemptCount; attempt++ {
		if delay >= policy.MaxDelay/2 {
			return policy.MaxDelay
		}
		delay *= 2
	}
	if delay > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay
}

type fanoutFailure struct {
	code      string
	message   string
	retryable bool
	cause     error
}

func classifyRenderFailure(err error) fanoutFailure {
	return fanoutFailure{
		code:      apperrors.CodeBadRequest,
		message:   safeFailureMessage(err.Error()),
		retryable: false,
		cause:     err,
	}
}

func classifyFanoutFailure(err error) fanoutFailure {
	code := apperrors.CodeOf(err)
	message := safeFailureMessage(apperrors.MessageOf(err))
	retryable := true
	switch code {
	case apperrors.CodeBadRequest,
		apperrors.CodeForbidden,
		apperrors.CodeNotFound,
		apperrors.CodePayloadTooLarge,
		apperrors.CodeUnprocessableEntity:
		retryable = false
	case apperrors.CodePaymentRequired,
		apperrors.CodeConflict,
		apperrors.CodeTooManyRequests,
		apperrors.CodeInternal,
		apperrors.CodeServiceUnavailable:
		retryable = true
	}
	return fanoutFailure{code: code, message: message, retryable: retryable, cause: err}
}

func safeFailureMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Recipient fanout failed"
	}
	runes := []rune(message)
	if len(runes) > maxFanoutFailureMessageRunes {
		message = string(runes[:maxFanoutFailureMessageRunes])
	}
	return message
}
