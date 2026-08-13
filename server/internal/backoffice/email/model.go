package email

import "time"

type Filter struct{ Query, Status string }
type Row struct {
	ID, TeamName, FromEmail, ToEmail, Subject, Status, Provider, ErrorMessage string
	CreatedAt                                                                 time.Time
}
type Recipient struct {
	Email, Type, Status, LastEvent, ErrorCode, ErrorMessage string
	LastEventAt, DeliveredAt, FailedAt                      *time.Time
}
type Detail struct {
	ID, TeamID, TeamName, SenderDomainID, MessageType                                         string
	FromEmail, FromName, ReplyToEmail, ToEmail, ToName, Subject                               string
	HTMLBody, TextBody, Status, DeliveryProvider, ProviderRegion, Provider, ProviderMessageID string
	ErrorCode, ErrorMessage, Metadata, RecipientsJSON, Headers, Attachments, Tags             string
	ScheduledAt, ProcessingAt, SubmittedAt, DeliveredAt, FailedAt                             *time.Time
	QueuedAt, CreatedAt, UpdatedAt                                                            time.Time
	Recipients                                                                                []Recipient
}
