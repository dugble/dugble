package broadcast

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	StatusDraft     = "draft"
	StatusScheduled = "scheduled"
	StatusQueued    = "queued"
	StatusSent      = "sent"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"
)

type Broadcast struct {
	ID                string         `json:"id"`
	TeamID            string         `json:"team_id"`
	Name              string         `json:"name"`
	Status            string         `json:"status"`
	SegmentID         string         `json:"segment_id"`
	TopicID           *string        `json:"topic_id,omitempty"`
	TemplateID        string         `json:"template_id"`
	TemplateVersionID *string        `json:"template_version_id,omitempty"`
	VariableBindings  map[string]any `json:"variable_bindings"`
	ScheduledAt       *time.Time     `json:"scheduled_at,omitempty"`
	QueuedAt          *time.Time     `json:"queued_at,omitempty"`
	SentAt            *time.Time     `json:"sent_at,omitempty"`
	CanceledAt        *time.Time     `json:"canceled_at,omitempty"`
	AudienceCount     int64          `json:"audience_count"`
	EligibleCount     int64          `json:"eligible_count"`
	SuppressedCount   int64          `json:"suppressed_count"`
	QueuedCount       int64          `json:"queued_count"`
	FailedCount       int64          `json:"failed_count"`
	Revision          int64          `json:"revision"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type FanoutRecipient struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	BroadcastID       uuid.UUID
	ContactID         *uuid.UUID
	Email             string
	FirstName         *string
	LastName          *string
	ContactSnapshot   map[string]any
	TemplateID        uuid.UUID
	TemplateVersionID uuid.UUID
	VariableBindings  map[string]any
	AttemptCount      int32
}

type CreateRequest struct {
	Name             string         `json:"name"`
	SegmentID        string         `json:"segment_id"`
	TopicID          *string        `json:"topic_id,omitempty"`
	Template         string         `json:"template"`
	VariableBindings map[string]any `json:"variable_bindings,omitempty"`
}

type UpdateRequest struct {
	Revision         int64           `json:"revision"`
	Name             *string         `json:"name,omitempty"`
	SegmentID        *string         `json:"segment_id,omitempty"`
	TopicID          **string        `json:"topic_id,omitempty"`
	Template         *string         `json:"template,omitempty"`
	VariableBindings *map[string]any `json:"variable_bindings,omitempty"`
}

func (r *UpdateRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateRequest

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	*r = UpdateRequest(decoded)
	if raw, ok := fields["topic_id"]; ok {
		var topicID *string
		if err := json.Unmarshal(raw, &topicID); err != nil {
			return err
		}
		r.TopicID = &topicID
	}
	return nil
}

type DuplicateRequest struct {
	Name string `json:"name"`
}

type SendRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

type PreviewRequest struct {
	Variables map[string]any `json:"variables,omitempty"`
}

type ListRequest struct {
	Limit  int32
	Offset int32
}

type Recipient struct {
	ID              string         `json:"id"`
	BroadcastID     string         `json:"broadcast_id"`
	ContactID       *string        `json:"contact_id,omitempty"`
	Email           string         `json:"email"`
	FirstName       *string        `json:"first_name,omitempty"`
	LastName        *string        `json:"last_name,omitempty"`
	ContactSnapshot map[string]any `json:"contact_snapshot"`
	Status          string         `json:"status"`
	ExclusionReason *string        `json:"exclusion_reason,omitempty"`
	EmailMessageID  *string        `json:"email_message_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	QueuedAt        *time.Time     `json:"queued_at,omitempty"`
}

type ExclusionSummary struct {
	Object      string           `json:"object"`
	BroadcastID string           `json:"broadcast_id"`
	Total       int64            `json:"total"`
	Reasons     map[string]int64 `json:"reasons"`
}

type Analytics struct {
	Object      string `json:"object"`
	BroadcastID string `json:"broadcast_id"`
	Audience    int64  `json:"audience"`
	Eligible    int64  `json:"eligible"`
	Excluded    int64  `json:"excluded"`
	Queued      int64  `json:"queued"`
	Delivered   int64  `json:"delivered"`
	Bounced     int64  `json:"bounced"`
	Complained  int64  `json:"complained"`
	Failed      int64  `json:"failed"`
	Opened      int64  `json:"opened"`
	Clicked     int64  `json:"clicked"`
}
