package topic

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	topics := router.Group("/topics")
	topics.POST("", handler.Create, accessMiddleware(authz.PermissionTopicsWrite))
	topics.GET("", handler.List, accessMiddleware(authz.PermissionTopicsRead))
	topics.GET("/:topic_id", handler.Get, accessMiddleware(authz.PermissionTopicsRead))
	topics.PATCH("/:topic_id", handler.Update, accessMiddleware(authz.PermissionTopicsWrite))
	topics.DELETE("/:topic_id", handler.Delete, accessMiddleware(authz.PermissionTopicsWrite))
}
