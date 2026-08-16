package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	attempt "github.com/dugble/dugble/server/internal/delivery/attempt"
	feedback "github.com/dugble/dugble/server/internal/delivery/feedback"
	smsmodule "github.com/dugble/dugble/server/internal/modules/sms"
	provider "github.com/dugble/dugble/server/internal/providers"
)

const providerStatusEventType = "delivery_status"

func statusEvent(
	pending PendingMessage,
	response provider.SMSStatusResult,
	observedAt time.Time,
) (feedback.Event, error) {
	providerID := strings.ToLower(strings.TrimSpace(pending.ProviderID))
	providerMessageID := strings.TrimSpace(response.ProviderMessageID)
	if providerID == "" {
		return feedback.Event{}, ErrProviderRequired
	}
	if providerMessageID == "" {
		return feedback.Event{}, ErrProviderMessageRequired
	}
	if providerMessageID != strings.TrimSpace(pending.ProviderMessageID) {
		return feedback.Event{}, errors.New("SMS provider status response does not match the pending attempt")
	}
	providerStatus := strings.ToLower(strings.TrimSpace(response.ProviderStatus))
	attemptStatus, normalizedStatus, err := deliveryAttemptStatus(response.Status, providerStatus)
	if err != nil {
		return feedback.Event{}, err
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	} else {
		observedAt = observedAt.UTC()
	}
	if providerStatus == "" {
		providerStatus = normalizedStatus
	}
	metadata, err := json.Marshal(map[string]any{
		"source":            "poll",
		"normalized_status": normalizedStatus,
		"provider_status":   providerStatus,
	})
	if err != nil {
		return feedback.Event{}, fmt.Errorf("encode SMS feedback metadata: %w", err)
	}
	errorCode, errorMessage := smsFeedbackError(normalizedStatus, providerStatus)
	event := feedback.Event{
		AttemptID:         pending.AttemptID,
		Provider:          providerID,
		ProviderEventID:   fmt.Sprintf("poll:%s:%d:%s", pending.AttemptID, pending.ReconcileAttempts+1, normalizedStatus),
		ProviderMessageID: providerMessageID,
		EventType:         providerStatusEventType + "." + normalizedStatus,
		Channel:           attempt.ChannelSMS,
		Status:            attemptStatus,
		ProviderStatus:    providerStatus,
		ErrorCode:         errorCode,
		ErrorMessage:      errorMessage,
		OccurredAt:        observedAt,
		ReceivedAt:        observedAt,
		Metadata:          metadata,
	}
	if err := event.Validate(); err != nil {
		return feedback.Event{}, err
	}
	return event, nil
}

func deliveryAttemptStatus(status provider.SMSStatus, providerStatus string) (attempt.AttemptStatus, string, error) {
	native := strings.ToLower(strings.TrimSpace(providerStatus))
	switch native {
	case smsmodule.StatusQueued:
		return attempt.StatusAccepted, smsmodule.StatusQueued, nil
	case smsmodule.StatusSubmitted:
		return attempt.StatusSubmitted, smsmodule.StatusSubmitted, nil
	case smsmodule.StatusSent:
		return attempt.StatusSent, smsmodule.StatusSent, nil
	case smsmodule.StatusDelivered:
		return attempt.StatusDelivered, smsmodule.StatusDelivered, nil
	case smsmodule.StatusUndelivered, smsmodule.StatusFailed:
		return attempt.StatusPermanentFailure, native, nil
	case smsmodule.StatusRejected:
		return attempt.StatusRejected, smsmodule.StatusRejected, nil
	case smsmodule.StatusExpired:
		return attempt.StatusExpired, smsmodule.StatusExpired, nil
	case smsmodule.StatusUnknown:
		return attempt.StatusUnknown, smsmodule.StatusUnknown, nil
	}

	switch status {
	case provider.SMSPending:
		return attempt.StatusSubmitted, smsmodule.StatusSubmitted, nil
	case provider.SMSDelivered:
		return attempt.StatusDelivered, smsmodule.StatusDelivered, nil
	case provider.SMSFailed:
		return attempt.StatusPermanentFailure, smsmodule.StatusFailed, nil
	case provider.SMSUnknown:
		return attempt.StatusUnknown, smsmodule.StatusUnknown, nil
	default:
		return "", "", ErrUnsupportedStatus
	}
}

func smsFeedbackError(status, providerStatus string) (string, string) {
	switch status {
	case smsmodule.StatusUndelivered:
		return "sms_undelivered", providerStatus
	case smsmodule.StatusRejected:
		return "sms_rejected", providerStatus
	case smsmodule.StatusFailed:
		return "sms_failed", providerStatus
	case smsmodule.StatusExpired:
		return "sms_expired", providerStatus
	case smsmodule.StatusUnknown:
		return "sms_unknown", providerStatus
	default:
		return "", ""
	}
}
