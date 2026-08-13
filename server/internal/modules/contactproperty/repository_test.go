package contactproperty

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/coffeyvidzro/dugble/server/internal/authz"
)

func TestRepositoryRejectsUnscopedPropertyAccess(t *testing.T) {
	t.Parallel()

	repository := &Repository{}
	if _, err := repository.Get(context.Background(), uuid.New(), authz.AccessibleTeamID{}); err == nil {
		t.Fatal("Get() accepted an unscoped team identifier")
	}
	if _, err := repository.Create(context.Background(), authz.AccessibleTeamID{}, CreateRequest{}); err == nil {
		t.Fatal("Create() accepted an unscoped team identifier")
	}
}
