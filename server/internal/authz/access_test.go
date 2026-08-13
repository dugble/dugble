package authz

import (
	"testing"

	"github.com/google/uuid"
)

func TestAuthorizedTeamIDRequiresEstablishedAccess(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	valid := Access{
		Actor: Actor{Type: ActorTypeUser, UserID: uuid.New()},
		Scope: TeamScope{TeamID: teamID, Status: StatusActive},
	}.AuthorizedTeamID()
	got, err := valid.UUID()
	if err != nil || got != teamID {
		t.Fatalf("AuthorizedTeamID().UUID() = (%v, %v)", got, err)
	}

	tests := []Access{
		{},
		{Actor: Actor{Type: ActorTypeUser, UserID: uuid.New()}, Scope: TeamScope{TeamID: teamID, Status: StatusSuspended}},
		{Scope: TeamScope{TeamID: teamID, Status: StatusActive}},
	}
	for _, access := range tests {
		if id := access.AuthorizedTeamID(); id.Valid() {
			t.Fatal("invalid access produced a repository-safe team id")
		}
	}
}
