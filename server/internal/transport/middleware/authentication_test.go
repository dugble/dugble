package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authn"
	"github.com/coffeyvidzro/dugble/server/internal/modules/teamtoken"
)

type staticResolver struct {
	principal  authn.Principal
	credential authn.Credential
}

func (resolver *staticResolver) Resolve(_ context.Context, credential authn.Credential) (authn.Principal, error) {
	resolver.credential = credential
	return resolver.principal, nil
}

func TestAuthenticateAppliesCSRFOnlyToSessionPrincipal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		principal authn.Principal
		prepare   func(*http.Request)
		wantCSRF  bool
	}{
		{
			name:      "session",
			principal: authn.Principal{Kind: authn.PrincipalUserSession},
			prepare: func(request *http.Request) {
				request.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: "session-secret"})
			},
			wantCSRF: true,
		},
		{
			name:      "team token",
			principal: authn.Principal{Kind: authn.PrincipalTeamToken},
			prepare: func(request *http.Request) {
				request.Header.Set(echo.HeaderAuthorization, "Bearer "+teamtoken.TokenPrefix+"secret")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resolver := &staticResolver{principal: test.principal}
			csrfCalled := false
			csrf := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
					csrfCalled = true
					return next(c)
				}
			}
			handlerCalled := false
			handler := Authenticate(AuthenticateConfig{Resolver: resolver, CSRF: csrf})(func(c *echo.Context) error {
				handlerCalled = true
				if _, ok := authn.PrincipalFromContext(c.Request().Context()); !ok {
					t.Fatal("principal was not attached to request context")
				}
				return nil
			})
			request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			test.prepare(request)
			ctx := echo.New().NewContext(request, httptest.NewRecorder())
			if err := handler(ctx); err != nil {
				t.Fatalf("Authenticate() error = %v", err)
			}
			if !handlerCalled || csrfCalled != test.wantCSRF {
				t.Fatalf("handler called = %v, CSRF called = %v, want CSRF %v", handlerCalled, csrfCalled, test.wantCSRF)
			}
		})
	}
}

func TestCredentialFromRequestKeepsBearerPrecedenceInputs(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set(echo.HeaderAuthorization, "Bearer "+teamtoken.TokenPrefix+"bearer")
	request.AddCookie(&http.Cookie{Name: authn.SessionCookieName, Value: "session"})
	credential, err := credentialFromRequest(request)
	if err != nil {
		t.Fatalf("credentialFromRequest() error = %v", err)
	}
	if credential.BearerToken == "" || credential.SessionToken == "" {
		t.Fatalf("credentialFromRequest() = %#v", credential)
	}
}
