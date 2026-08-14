package contact

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authz"
)

func TestRepositoryRejectsUnscopedContactAccess(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	if _, err := repository.Get(context.Background(), uuid.New(), authz.AccessibleTeamID{}); err == nil {
		t.Fatal("Get() accepted an unscoped team identifier")
	}
	if _, err := repository.Create(context.Background(), authz.AccessibleTeamID{}, CreateRequest{}); err == nil {
		t.Fatal("Create() accepted an unscoped team identifier")
	}
}
