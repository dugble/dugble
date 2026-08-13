package contact

import "time"

type Contact struct {
	ID                  string         `json:"id"`
	TeamID              string         `json:"team_id"`
	Email               string         `json:"email"`
	Phone               *string        `json:"phone,omitempty"`
	NormalizedPhone     *string        `json:"normalized_phone,omitempty"`
	PhoneCountry        *string        `json:"phone_country,omitempty"`
	SMSConsentStatus    string         `json:"sms_consent_status"`
	SMSConsentUpdatedAt *time.Time     `json:"sms_consent_updated_at,omitempty"`
	SMSConsentSource    *string        `json:"sms_consent_source,omitempty"`
	FirstName           *string        `json:"first_name,omitempty"`
	LastName            *string        `json:"last_name,omitempty"`
	Unsubscribed        bool           `json:"unsubscribed"`
	Properties          map[string]any `json:"properties"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type SegmentMembership struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	AssignedAt time.Time `json:"assigned_at"`
}

type CreateRequest struct {
	Email            string         `json:"email"`
	Phone            *string        `json:"phone,omitempty"`
	SMSConsentStatus string         `json:"sms_consent_status,omitempty"`
	SMSConsentSource *string        `json:"sms_consent_source,omitempty"`
	NormalizedPhone  *string        `json:"-"`
	PhoneCountry     *string        `json:"-"`
	FirstName        *string        `json:"first_name,omitempty"`
	LastName         *string        `json:"last_name,omitempty"`
	Unsubscribed     bool           `json:"unsubscribed"`
	Properties       map[string]any `json:"properties,omitempty"`
}

type UpdateRequest struct {
	Email            *string         `json:"email,omitempty"`
	Phone            *string         `json:"phone,omitempty"`
	SMSConsentStatus *string         `json:"sms_consent_status,omitempty"`
	SMSConsentSource *string         `json:"sms_consent_source,omitempty"`
	FirstName        *string         `json:"first_name,omitempty"`
	LastName         *string         `json:"last_name,omitempty"`
	Unsubscribed     *bool           `json:"unsubscribed,omitempty"`
	Properties       *map[string]any `json:"properties,omitempty"`
}

const (
	SMSConsentUnknown  = "unknown"
	SMSConsentOptedIn  = "opted_in"
	SMSConsentOptedOut = "opted_out"
)

type ListRequest struct {
	Limit  int32
	Offset int32
}

const (
	ObjectList          = "list"
	SubscriptionOptIn   = "opt_in"
	SubscriptionOptOut  = "opt_out"
	maxContactTopicPage = 100
)

type ContactTopic struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Subscription string  `json:"subscription"`
}

type ListContactTopicsRequest struct {
	Limit  int32
	After  string
	Before string
}

type UpdateContactTopic struct {
	ID           string `json:"id"`
	Subscription string `json:"subscription"`
}

type UpdateContactTopicsRequest []UpdateContactTopic

type ContactTopicListResponse struct {
	Object  string         `json:"object"`
	HasMore bool           `json:"has_more"`
	Data    []ContactTopic `json:"data"`
}

type UpdateContactTopicsResponse struct {
	ID string `json:"id"`
}
