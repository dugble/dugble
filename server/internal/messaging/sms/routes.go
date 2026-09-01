package sms

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	messages := router.Group("/sms")
	messages.GET("", handler.List, accessMiddleware(authz.PermissionSMSRead))
	messages.GET("/analytics", handler.GetAnalytics, accessMiddleware(authz.PermissionSMSRead))
	messages.POST("", handler.Send, accessMiddleware(authz.PermissionSMSSend))
	messages.POST("/batch", handler.BatchSend, accessMiddleware(authz.PermissionSMSSend))
	messages.POST("/:message_id/cancel", handler.Cancel, accessMiddleware(authz.PermissionSMSSend))
	messages.PATCH("/:message_id", handler.Update, accessMiddleware(authz.PermissionSMSSend))
	messages.GET("/:message_id", handler.Get, accessMiddleware(authz.PermissionSMSRead))
	messages.GET("/:message_id/events", handler.ListEvents, accessMiddleware(authz.PermissionSMSRead))
	messages.POST("/:message_id/sync-status", handler.SyncStatus, accessMiddleware(authz.PermissionSMSSend))
}
