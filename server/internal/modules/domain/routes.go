package domain

import (
	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	domains := router.Group("/domains")
	domains.GET("", handler.List, accessMiddleware(authz.PermissionSenderDomainsRead))
	domains.POST("", handler.Create, accessMiddleware(authz.PermissionSenderDomainsCreate))
	domains.GET("/:domain_id", handler.Get, accessMiddleware(authz.PermissionSenderDomainsRead))
	domains.POST("/:domain_id/verify", handler.Verify, accessMiddleware(authz.PermissionSenderDomainsCreate))
	domains.DELETE("/:domain_id", handler.Delete, accessMiddleware(authz.PermissionSenderDomainsDelete))
}
