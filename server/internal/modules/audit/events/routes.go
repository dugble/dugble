package auditevent

import (
	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/labstack/echo/v5"
)

type TenantMiddleware func(authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, authMiddleware, csrfMiddleware echo.MiddlewareFunc, tenantMiddleware TenantMiddleware) {
	group := router.Group("/teams/:team_id/audit-events")
	group.Use(authMiddleware, csrfMiddleware)
	group.GET("", handler.List, tenantMiddleware(authz.PermissionAuditEventsRead))
}
