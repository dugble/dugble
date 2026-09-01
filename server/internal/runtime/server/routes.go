package server

import (
	"github.com/labstack/echo/v5"

	httpmiddleware "github.com/dugble/dugble/server/internal/runtime/middleware"
	providersns "github.com/dugble/dugble/server/internal/runtime/provider/aws/sns"
	healthmodule "github.com/dugble/dugble/server/internal/runtime/server/health"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

func (registry *Registry) registerRoutes(router *echo.Echo) error {
	healthmodule.RegisterRoutes(router, healthmodule.NewHandler(registry.postgres, registry.redis))
	if registry.providerSNS != nil {
		providersns.RegisterRoutes(router, registry.providerSNS)
	}
	csrfMiddleware := httpmiddleware.CSRF(httpmiddleware.CSRFConfig{
		Development:    registry.config.IsDevelopment(),
		TrustedOrigins: registry.config.CORSOrigins,
	})
	registerCSRFRoute(router, csrfMiddleware)
	return registry.registerModules(router)
}

func registerCSRFRoute(router *echo.Echo, csrfMiddleware echo.MiddlewareFunc) {
	router.GET("/csrf", func(c *echo.Context) error {
		token, ok := c.Get(httpmiddleware.CSRFContextKey).(string)
		if !ok || token == "" {
			return httputil.Error(c, apperrors.NewInternal("CSRF token is not available", nil))
		}
		return httputil.OK(c, map[string]string{"csrf_token": token})
	}, csrfMiddleware)
}
