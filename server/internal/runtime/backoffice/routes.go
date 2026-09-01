package backoffice

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	backofficehealth "github.com/dugble/dugble/server/internal/backoffice/health"
	authmodule "github.com/dugble/dugble/server/internal/identity/auth"
	sessionmodule "github.com/dugble/dugble/server/internal/identity/sessions"
	httpmiddleware "github.com/dugble/dugble/server/internal/runtime/middleware"
	"github.com/dugble/dugble/server/internal/security/authn"
)

func (registry *Registry) registerRoutes(router *echo.Echo) error {
	backofficehealth.RegisterRoutes(router, backofficehealth.NewHandler(registry.postgres))

	sessionRepository := sessionmodule.NewRepository(registry.postgres)
	authRepository := authmodule.NewRepository(registry.postgres)
	authMiddleware := httpmiddleware.SessionAuth(httpmiddleware.SessionAuthConfig{
		Sessions: sessionRepository,
		Users:    authRepository,
	})
	adminMiddleware := requireAdmin(registry.config.Backoffice.AdminEmails)
	csrfMiddleware := httpmiddleware.CSRF(httpmiddleware.CSRFConfig{
		Development:    registry.config.IsDevelopment(),
		TrustedOrigins: registry.config.CORSOrigins,
		TokenLookup:    "header:" + echo.HeaderXCSRFToken,
		CookieName:     "dugble_backoffice_csrf",
	})

	registry.registerModules(router, authMiddleware, adminMiddleware, csrfMiddleware)
	return nil
}

func requireAdmin(adminEmails []string) echo.MiddlewareFunc {
	allowed := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" {
			allowed[email] = struct{}{}
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if len(allowed) == 0 {
				return echo.NewHTTPError(http.StatusForbidden, "backoffice admin allowlist is not configured")
			}

			principal, ok := authn.PrincipalFromContext(c.Request().Context())
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication is required")
			}
			if _, ok := allowed[strings.ToLower(principal.Email)]; !ok {
				return echo.NewHTTPError(http.StatusForbidden, "backoffice access is forbidden")
			}
			return next(c)
		}
	}
}
