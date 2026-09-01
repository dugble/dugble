package senderid

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	senderIDs := router.Group("/sender-ids")
	senderIDs.GET("", handler.List, accessMiddleware(authz.PermissionSenderIDsRead))
	senderIDs.POST("", handler.Create, accessMiddleware(authz.PermissionSenderIDsCreate))
	senderIDs.GET("/:sender_id", handler.Get, accessMiddleware(authz.PermissionSenderIDsRead))
	senderIDs.DELETE("/:sender_id", handler.Delete, accessMiddleware(authz.PermissionSenderIDsDelete))
}
