package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authn"
	"github.com/coffeyvidzro/dugble/server/internal/authz"
	"github.com/coffeyvidzro/dugble/server/internal/modules/teamtoken"
	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type StepUpConfig struct {
	Assurance authn.AssuranceLevel
	MaxAge    time.Duration
}

func RequireRecentAuthentication(config StepUpConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			principal, ok := authn.PrincipalFromContext(c.Request().Context())
			if !ok {
				return httputil.Error(c, apperrors.NewUnauthorized("Authentication is required"))
			}
			if config.MaxAge <= 0 || !principal.RecentlyAuthenticated(config.Assurance, config.MaxAge, time.Now().UTC()) {
				return httputil.Error(c, apperrors.NewStepUpRequired("Recent stronger authentication is required"))
			}
			return next(c)
		}
	}
}

type AuthenticateConfig struct {
	Resolver authn.Resolver
	CSRF     echo.MiddlewareFunc
}

// Authenticate extracts request credentials once, resolves one principal, and
// applies CSRF only when the selected credential is a cookie-backed session.
func Authenticate(config AuthenticateConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		csrfNext := next
		if config.CSRF != nil {
			csrfNext = config.CSRF(next)
		}
		return func(c *echo.Context) error {
			credential, err := credentialFromRequest(c.Request())
			if err != nil {
				return httputil.Error(c, apperrors.NewUnauthorized("Authorization header is invalid"))
			}
			return authenticate(c, config.Resolver, credential, next, csrfNext)
		}
	}
}

// SessionAuth remains the session-only middleware used by account and
// backoffice routes. Tenant APIs use Authenticate instead.
type SessionAuthConfig struct {
	Sessions SessionStore
	Users    PrincipalRepository
}

func SessionAuth(config SessionAuthConfig) echo.MiddlewareFunc {
	resolver := CredentialResolver{Sessions: config.Sessions, Users: config.Users}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			cookie, err := c.Request().Cookie(authn.SessionCookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				return httputil.Error(c, apperrors.NewUnauthorized("Authentication is required"))
			}
			return authenticate(c, resolver, authn.Credential{SessionToken: cookie.Value}, next, next)
		}
	}
}

func authenticate(c *echo.Context, resolver authn.Resolver, credential authn.Credential, next, csrfNext echo.HandlerFunc) error {
	if resolver == nil {
		return httputil.Error(c, apperrors.NewInternal("Authentication is not configured", nil))
	}
	principal, err := resolver.Resolve(c.Request().Context(), credential)
	if err != nil {
		var internal resolutionError
		if errors.As(err, &internal) {
			return httputil.Error(c, apperrors.NewInternal("Unable to authenticate request", internal.err))
		}
		return httputil.Error(c, apperrors.NewUnauthorized("Authentication is invalid or expired"))
	}
	c.SetRequest(c.Request().WithContext(authn.ContextWithPrincipal(c.Request().Context(), principal)))
	ObservePrincipal(c, principal)
	if principal.Kind == authn.PrincipalUserSession {
		return csrfNext(c)
	}
	return next(c)
}

func credentialFromRequest(request *http.Request) (authn.Credential, error) {
	credential := authn.Credential{}
	authorization := strings.TrimSpace(request.Header.Get(echo.HeaderAuthorization))
	if authorization != "" {
		secret, ok := parseBearerToken(authorization)
		if !ok {
			return authn.Credential{}, authn.ErrUnauthenticated
		}
		credential.BearerToken = secret
	}
	if cookie, err := request.Cookie(authn.SessionCookieName); err == nil {
		credential.SessionToken = cookie.Value
	}
	return credential.Normalize(), nil
}

func parseBearerToken(value string) (string, bool) {
	prefix, token, ok := strings.Cut(strings.TrimSpace(value), " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}

func tokenPermissions(values []string) ([]authz.Permission, bool) {
	permissions := make([]authz.Permission, 0, len(values))
	for _, value := range values {
		permission := authz.Permission(strings.TrimSpace(value))
		if permission == "" || !teamtoken.IsAllowedPermission(permission) {
			return nil, false
		}
		permissions = append(permissions, permission)
	}
	return permissions, len(permissions) > 0
}
