package smscampaign

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusDraft         = "draft"
	StatusScheduled     = "scheduled"
	StatusQueued        = "queued"
	StatusMaterializing = "materializing"
	StatusEstimating    = "estimating"
	StatusSending       = "sending"
	StatusSent          = "sent"
	StatusFailed        = "failed"
	StatusCanceled      = "canceled"
)

type Campaign struct {
	ID                         string     `json:"id"`
	TeamID                     string     `json:"team_id"`
	Name                       string     `json:"name"`
	Status                     string     `json:"status"`
	SegmentID                  string     `json:"segment_id"`
	SenderID                   string     `json:"sender_id"`
	Body                       string     `json:"body"`
	ScheduledAt                *time.Time `json:"scheduled_at,omitempty"`
	QueuedAt                   *time.Time `json:"queued_at,omitempty"`
	CanceledAt                 *time.Time `json:"canceled_at,omitempty"`
	MaterializedAt             *time.Time `json:"materialized_at,omitempty"`
	SentAt                     *time.Time `json:"sent_at,omitempty"`
	AudienceCount              int64      `json:"audience_count"`
	EligibleCount              int64      `json:"eligible_count"`
	ExcludedCount              int64      `json:"excluded_count"`
	FailedCount                int64      `json:"failed_count"`
	EstimatedSegments          int64      `json:"estimated_segments"`
	EstimatedCostUnits         int64      `json:"estimated_cost_units"`
	EstimatedBillableCostUnits int64      `json:"estimated_billable_cost_units"`
	PreflightAllowanceSegments int64      `json:"preflight_allowance_segments"`
	ActualSegments             int64      `json:"actual_segments"`
	ActualChargeUnits          int64      `json:"actual_charge_units"`
	Currency                   *string    `json:"currency,omitempty"`
	PreflightBalanceUnits      *int64     `json:"preflight_balance_units,omitempty"`
	PreflightAt                *time.Time `json:"preflight_at,omitempty"`
	RateLimitPerSecond         int32      `json:"rate_limit_per_second"`
	DailySendLimit             *int32     `json:"daily_send_limit,omitempty"`
	Revision                   int64      `json:"revision"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	Name               string `json:"name"`
	SegmentID          string `json:"segment_id"`
	SenderID           string `json:"sender_id"`
	Body               string `json:"body"`
	RateLimitPerSecond int32  `json:"rate_limit_per_second,omitempty"`
	DailySendLimit     *int32 `json:"daily_send_limit,omitempty"`
}

type UpdateRequest struct {
	Revision           int64   `json:"revision"`
	Name               *string `json:"name,omitempty"`
	SegmentID          *string `json:"segment_id,omitempty"`
	SenderID           *string `json:"sender_id,omitempty"`
	Body               *string `json:"body,omitempty"`
	RateLimitPerSecond *int32  `json:"rate_limit_per_second,omitempty"`
	DailySendLimit     *int32  `json:"daily_send_limit,omitempty"`
}

type RecordOptOutRequest struct {
	Phone    string  `json:"phone"`
	Source   string  `json:"source,omitempty"`
	SourceID *string `json:"source_id,omitempty"`
}

type ConsentEvent struct {
	ID         string    `json:"id"`
	ContactID  *string   `json:"contact_id,omitempty"`
	Phone      string    `json:"phone"`
	Action     string    `json:"action"`
	Source     string    `json:"source"`
	SourceID   *string   `json:"source_id,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

type ExclusionSummary struct {
	CampaignID string           `json:"campaign_id"`
	Total      int64            `json:"total"`
	Reasons    map[string]int64 `json:"reasons"`
}

type Analytics struct {
	CampaignID                 string  `json:"campaign_id"`
	Audience                   int64   `json:"audience"`
	Eligible                   int64   `json:"eligible"`
	Excluded                   int64   `json:"excluded"`
	Queued                     int64   `json:"queued"`
	Failed                     int64   `json:"failed"`
	Sent                       int64   `json:"sent"`
	Delivered                  int64   `json:"delivered"`
	DeliveryFailed             int64   `json:"delivery_failed"`
	EstimatedSegments          int64   `json:"estimated_segments"`
	EstimatedCostUnits         int64   `json:"estimated_cost_units"`
	EstimatedBillableCostUnits int64   `json:"estimated_billable_cost_units"`
	ActualSegments             int64   `json:"actual_segments"`
	ActualChargeUnits          int64   `json:"actual_charge_units"`
	Currency                   *string `json:"currency,omitempty"`
}

type DuplicateRequest struct {
	Name string `json:"name"`
}
type SendRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}
type ScheduleRequest struct {
	ScheduledAt time.Time `json:"scheduled_at"`
}
type ListRequest struct{ Limit, Offset int32 }

type Preview struct {
	Body       string `json:"body"`
	Encoding   string `json:"encoding"`
	Characters int    `json:"characters"`
	Segments   int32  `json:"segments"`
}

type Recipient struct {
	ID                     string         `json:"id"`
	CampaignID             string         `json:"campaign_id"`
	ContactID              *string        `json:"contact_id,omitempty"`
	Phone                  *string        `json:"phone,omitempty"`
	PhoneCountry           *string        `json:"phone_country,omitempty"`
	ContactSnapshot        map[string]any `json:"contact_snapshot"`
	Status                 string         `json:"status"`
	DeliveryStatus         *string        `json:"delivery_status,omitempty"`
	DeliveredAt            *time.Time     `json:"delivered_at,omitempty"`
	ExclusionReason        *string        `json:"exclusion_reason,omitempty"`
	SMSMessageID           *string        `json:"sms_message_id,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	QueuedAt               *time.Time     `json:"queued_at,omitempty"`
	RenderedBody           *string        `json:"rendered_body,omitempty"`
	AttemptCount           int32          `json:"attempt_count"`
	FailureCode            *string        `json:"failure_code,omitempty"`
	FailureMessage         *string        `json:"failure_message,omitempty"`
	Encoding               *string        `json:"encoding,omitempty"`
	EstimatedSegments      *int32         `json:"estimated_segments,omitempty"`
	EstimatedUnitCostUnits *int64         `json:"estimated_unit_cost_units,omitempty"`
	EstimatedCostUnits     *int64         `json:"estimated_cost_units,omitempty"`
	ActualSegments         *int32         `json:"actual_segments,omitempty"`
	ActualChargeUnits      *int64         `json:"actual_charge_units,omitempty"`
}

type CostEstimate struct {
	CampaignID                 string     `json:"campaign_id"`
	Currency                   *string    `json:"currency,omitempty"`
	Recipients                 int64      `json:"recipients"`
	EstimatedSegments          int64      `json:"estimated_segments"`
	EstimatedCostUnits         int64      `json:"estimated_cost_units"`
	EstimatedBillableCostUnits int64      `json:"estimated_billable_cost_units"`
	PreflightAllowanceSegments int64      `json:"preflight_allowance_segments"`
	MinimumRecipientCostUnits  int64      `json:"minimum_recipient_cost_units"`
	MaximumRecipientCostUnits  int64      `json:"maximum_recipient_cost_units"`
	ActualSegments             int64      `json:"actual_segments"`
	ActualChargeUnits          int64      `json:"actual_charge_units"`
	PreflightBalanceUnits      *int64     `json:"preflight_balance_units,omitempty"`
	PreflightAt                *time.Time `json:"preflight_at,omitempty"`
}

type FanoutRecipient struct {
	ID              uuid.UUID
	TeamID          uuid.UUID
	CampaignID      uuid.UUID
	ContactID       *uuid.UUID
	Phone           string
	PhoneCountry    string
	ContactSnapshot map[string]any
	CampaignBody    string
	RenderedBody    string
	SenderID        uuid.UUID
	SenderName      string
	AttemptCount    int32
}
