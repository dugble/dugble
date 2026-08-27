package messagetemplate

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
)

// NOTE: preserve the remainder of this file from the branch; only category type
// conversions are required here.
