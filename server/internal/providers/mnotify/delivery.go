package mnotify

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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

	response, err := p.client.send(ctx, sendPayload{
		Recipient:    []string{normalizeRecipient(message.To)},
		Sender:       strings.TrimSpace(message.From),
		Message:      message.Text,
		IsSchedule:   false,
		ScheduleDate: "",
	})
	if err != nil {
		return sms.SendResult{State: sms.SubmissionUnknown}, err
	}

	code := responseCode(response.body.Code)
	result := sms.SendResult{ProviderMessageID: response.body.Summary.MessageID}
	if response.statusCode == http.StatusOK && strings.EqualFold(response.body.Status, "success") && code == "2000" {
		result.State = sms.SubmissionAccepted
		return result, nil
	}

	result.State = sms.SubmissionUnknown
	return result, &APIError{StatusCode: response.statusCode, Code: code, Message: response.body.Message}
}

func normalizeRecipient(recipient string) string {
	value := strings.TrimSpace(recipient)
	return strings.TrimPrefix(value, "+")
}

func responseCode(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return number.String()
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if _, err := strconv.Atoi(text); err == nil {
			return text
		}
		return text
	}
	return ""
}
