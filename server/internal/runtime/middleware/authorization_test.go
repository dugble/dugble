package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authn"
	"github.com/dugble/dugble/server/internal/security/authz"
)

func TestChainBuildsPipelineInDeclaredOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	middleware := func(name string) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				calls = append(calls, name+":before")
				err := next(c)
				calls = append(calls, name+":after")
				return err
			}
		}
	}
	handler := Chain(middleware("authenticate"), middleware("select"), middleware("authorize"))(func(*echo.Context) error {
		calls = append(calls, "handler")
		return nil
	})
	router := echo.New()
	ctx := router.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), httptest.NewRecorder())
	if err := handler(ctx); err != nil {
		t.Fatalf("handler error = %v", err)
	}
	want := []string{"authenticate:before", "select:before", "authorize:before", "handler", "authorize:after", "select:after", "authenticate:after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestSelectTeamAccessRejectsMismatchedTeamToken(t *testing.T) {
	t.Parallel()

	teamID, otherTeamID, tokenID := uuid.New(), uuid.New(), uuid.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set(defaultTenantHeader, otherTeamID.String())
	ctx := echo.New().NewContext(request, httptest.NewRecorder())
	principal := authn.Principal{Kind: authn.PrincipalTeamToken, TeamID: &teamID, TokenID: &tokenID, Scopes: authz.NewScopeSet(authz.ScopeSMSRead)}

	if _, err := selectTeamAccess(ctx, nil, principal, defaultTenantParam, defaultTenantHeader); err == nil {
		t.Fatal("selectTeamAccess() allowed a token to select another team")
	}
}

type membershipRepository struct{ membership authz.Membership }

func (repository membershipRepository) GetTenantMembership(context.Context, uuid.UUID, uuid.UUID) (authz.Membership, error) {
	return repository.membership, nil
}

func TestSelectTeamAccessUsesPrincipalScopes(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set(defaultTenantHeader, teamID.String())
	ctx := echo.New().NewContext(request, httptest.NewRecorder())
	principal := authn.Principal{Kind: authn.PrincipalUserSession, UserID: userID, Scopes: authz.NewScopeSet(authz.ScopeContactsRead)}
	repository := membershipRepository{membership: authz.Membership{TeamID: teamID, UserID: userID, Role: authz.RoleMember, Status: authz.StatusActive}}

	access, err := selectTeamAccess(ctx, repository, principal, defaultTenantParam, defaultTenantHeader)
	if err != nil {
		t.Fatalf("selectTeamAccess() error = %v", err)
	}
	if !access.Scope.Scopes.Has(authz.ScopeContactsRead) || access.Scope.Scopes.Has(authz.ScopeContactsWrite) {
		t.Fatalf("selectTeamAccess() scopes = %v", access.Scope.Scopes.Scopes())
	}
}
