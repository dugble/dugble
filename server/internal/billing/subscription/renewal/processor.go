package renewal

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Processor interface {
	ProcessTeam(context.Context, pgx.Tx, uuid.UUID) (Result, error)
}
