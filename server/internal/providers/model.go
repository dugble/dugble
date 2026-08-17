package providers

type SenderIDStatus string

const (
	SenderIDPending  SenderIDStatus = "pending"
	SenderIDActive   SenderIDStatus = "active"
	SenderIDRejected SenderIDStatus = "rejected"
	SenderIDUnknown  SenderIDStatus = "unknown"
)

type SMSStatus string

const (
	SMSPending   SMSStatus = "pending"
	SMSDelivered SMSStatus = "delivered"
	SMSFailed    SMSStatus = "failed"
	SMSUnknown   SMSStatus = "unknown"
)

type CreateSenderIDRequest struct {
	SenderID string
	UseCase  string
}

type CreateSenderIDResult struct {
	SenderID          string
	ProviderReference string
	Status            SenderIDStatus
	ProviderCode      string
}

type SMSStatusRequest struct {
	Reference         string
	ProviderMessageID string
}

type SMSStatusResult struct {
	Reference         string
	ProviderMessageID string
	Status            SMSStatus
	ProviderStatus    string
	ProviderCode      string
}

type SenderIDStatusRequest struct {
	SenderID          string
	ProviderReference string
}

type SenderIDStatusResult struct {
	SenderID          string
	ProviderReference string
	Status            SenderIDStatus
	ProviderStatus    string
	ProviderCode      string
}
