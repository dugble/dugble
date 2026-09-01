package domainclaim

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	domains := router.Group("/domains")
	domains.POST("/claim", handler.Start, accessMiddleware(authz.PermissionSenderDomainsCreate))
	domains.GET("/:domain_id/claim", handler.Get, accessMiddleware(authz.PermissionSenderDomainsRead))
	domains.POST("/:domain_id/claim/verify", handler.Verify, accessMiddleware(authz.PermissionSenderDomainsCreate))
	domains.DELETE("/:domain_id/claim", handler.Cancel, accessMiddleware(authz.PermissionSenderDomainsDelete))
}
