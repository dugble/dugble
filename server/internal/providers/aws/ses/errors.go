package ses

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/aws/smithy-go"
)

var (
	ErrMessageTooLarge           = errors.New("email exceeds the SES raw message size limit")
	ErrReservedHeader            = errors.New("email header is reserved by the SES integration")
	ErrUnsupportedAttachmentPath = errors.New("attachment paths are not supported by the SES integration")
)

// SendError classifies an SES submission failure for Dugble's delivery state
// machine without discarding the underlying AWS error.
type SendError struct {
	Code              string
	Retryable         bool
	SubmissionUnknown bool
	Err               error
}

func (e *SendError) Error() string {
	if e == nil || e.Err == nil {
		return "email send failed"
	}
	return e.Err.Error()
}

func (e *SendError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewSendError(code string, retryable bool, err error) error {
	return &SendError{Code: normalizeCode(code), Retryable: retryable, Err: err}
}

func NewSubmissionUnknownError(code string, err error) error {
	return &SendError{Code: normalizeCode(code), SubmissionUnknown: true, Err: err}
}

func IsSubmissionUnknown(err error) bool {
	if err == nil {
		return false
	}
	var sendError *SendError
	if errors.As(err, &sendError) {
		return sendError.SubmissionUnknown
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError)
}

func IsRetryable(err error) bool {
	if err == nil || IsSubmissionUnknown(err) {
		return false
	}
	var sendError *SendError
	return errors.As(err, &sendError) && sendError.Retryable
}

func FailureCode(err error) string {
	var sendError *SendError
	if errors.As(err, &sendError) && sendError.Code != "" {
		return sendError.Code
	}
	return "provider_rejected"
}

func classifySESFailure(err error) error {
	var apiError smithy.APIError
	if !errors.As(err, &apiError) {
		return NewSubmissionUnknownError("ses_submission_unknown", err)
	}
	code := strings.ToLower(strings.TrimSpace(apiError.ErrorCode()))
	switch code {
	case "requesttimeout", "requesttimeoutexception":
		return NewSubmissionUnknownError(code, err)
	case "throttling", "throttlingexception", "toomanyrequestsexception", "serviceunavailable", "internalfailure", "internalservererror":
		return NewSendError(code, true, err)
	default:
		return NewSendError(code, false, err)
	}
}

func isAlreadyExists(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && strings.EqualFold(strings.TrimSpace(apiError.ErrorCode()), "AlreadyExistsException")
}

func normalizeCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "_")
	code = strings.ReplaceAll(code, " ", "_")
	return code
}
