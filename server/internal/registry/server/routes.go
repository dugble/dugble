package server

import (
	"github.com/labstack/echo/v5"

	healthmodule "github.com/dugble/dugble/server/internal/modules/health"
	httpmiddleware "github.com/dugble/dugble/server/internal/transport/middleware"
	providersns "github.com/dugble/dugble/server/internal/transport/provider/aws/sns"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

func (registry *Registry) registerRoutes(router *echo.Echo) error {
	healthmodule.RegisterRoutes(router, healthmodule.NewHandler(registry.postgres, registry.redis))
	if registry.providerSNS != nil {
		providersns.RegisterRoutes(router, registry.providerSNS)
	}

	middleware := registry.moduleMiddleware()
	registerCSRFRoute(router, middleware.csrf)
	return registry.registerModulesWithMiddleware(router, middleware)
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
