package authn

import (
	"testing"
	"time"
)

func TestPrincipalRecentlyAuthenticated(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	principal := Principal{AssuranceLevel: AssuranceLevelTwo, AuthenticatedAt: now.Add(-5 * time.Minute)}
	if !principal.RecentlyAuthenticated(AssuranceLevelOne, 15*time.Minute, now) {
		t.Fatal("aal2 principal should satisfy a recent aal1 requirement")
	}
	if principal.RecentlyAuthenticated(AssuranceLevelThree, 15*time.Minute, now) {
		t.Fatal("aal2 principal should not satisfy an aal3 requirement")
	}
	if principal.RecentlyAuthenticated(AssuranceLevelOne, time.Minute, now) {
		t.Fatal("stale principal reported recently authenticated")
	}
}
