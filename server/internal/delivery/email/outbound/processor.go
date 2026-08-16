package emaildelivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	awsses "github.com/dugble/dugble/server/internal/providers/aws/ses"
)

const defaultStaleProcessingAfter = 2 * time.Minute

type deliveryRepository interface {
	Claim(context.Context, uuid.UUID, uuid.UUID) (DeliveryMessage, error)
	RecoverStale(context.Context