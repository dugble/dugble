package moolre

import (
	"context"
	"net/http"
	"strings"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	if p == nil || p.client == nil {
		return sms.SendResult{State: sms.SubmissionRejected}, ErrInvalidConfig
	}
	if strings.TrimSpace(message.From) == "" || !p.Capabilities().Supports(message) {
		return sms.SendResult{State: sms.SubmissionRejected}, sms.ErrInvalidMessage
	}

	response, requestErr := p.client.send(ctx, sendPayload{
		Type:     1,
		SenderID: strings.TrimSpace(message.From),
		Messages: []messagePayload{{
			Recipient: strings.TrimSpace(message.To),
			Message:   message.Text,
		}},
	})

	if response.statusCode == http.StatusOK && response.body.Status == 1 && strings.EqualFold(strings.TrimSpace(response.body.Code), "SMS01") {
		return sms.SendResult{State: sms.SubmissionAccepted}, nil
	}

	if isSafeToFallbackStatus(response.statusCode) {
		return sms.SendResult{State: sms.SubmissionRejected}, &APIError{
			StatusCode: response.statusCode,
			Code:       response.body.Code,
			Message:    response.body.Message,
		}
	}

	if requestErr != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, requestErr
	}

	return sms.SendResult{State: sms.SubmissionUnknown}, &APIError{
		StatusCode: response.statusCode,
		Code:       response.body.Code,
		Message:    response.body.Message,
	}
}

func isSafeToFallbackStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return true
	default:
		return false
	}
}
