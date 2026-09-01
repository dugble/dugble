package email

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type TenantMiddleware func(authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, tenantAccess TenantMiddleware) {
	emails := router.Group("/emails")
	emails.GET("", handler.List, tenantAccess(authz.PermissionEmailRead))
	emails.GET("/analytics", handler.GetAnalytics, tenantAccess(authz.PermissionEmailRead))
	emails.POST("", handler.Send, tenantAccess(authz.PermissionEmailSend))
	emails.POST("/batch", handler.BatchSend, tenantAccess(authz.PermissionEmailSend))
	emails.POST("/:message_id/cancel", handler.Cancel, tenantAccess(authz.PermissionEmailSend))
	emails.PATCH("/:message_id", handler.Update, tenantAccess(authz.PermissionEmailSend))
	emails.GET("/:message_id", handler.Get, tenantAccess(authz.PermissionEmailRead))
	emails.GET("/:message_id/events", handler.ListEvents, tenantAccess(authz.PermissionEmailRead))
}
