package suppression

import (
	"time"

	"github.com/google/uuid"
)

const (
	ChannelEmail = "email"
	ChannelSMS   = "sms"
)

// ChannelSuppression is the channel-neutral suppression record used by
// delivery and feedback flows. The existing Suppression type remains the
// email-facing API representation.
type ChannelSuppression struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	Channel           string
	Address           string
	NormalizedAddress string
	Reason            string
	Origin            string
	SourceID          *string
	CreatedAt         time.Time
}

type CreateChannelParams struct {
	TeamID            uuid.UUID
	Channel           string
	Address           string
	NormalizedAddress string
	Reason            string
	Origin            string
	SourceID          *string
}
