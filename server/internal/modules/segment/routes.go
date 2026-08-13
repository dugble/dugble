package segment

import (
	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	segments := router.Group("/segments")
	segments.POST("", handler.Create, accessMiddleware(authz.PermissionSegmentsWrite))
	segments.GET("", handler.List, accessMiddleware(authz.PermissionSegmentsRead))
	segments.GET("/:segment_id", handler.Get, accessMiddleware(authz.PermissionSegmentsRead))
	segments.GET("/:segment_id/contacts", handler.ListContacts, accessMiddleware(authz.PermissionSegmentsRead))
	segments.DELETE("/:segment_id", handler.Delete, accessMiddleware(authz.PermissionSegmentsWrite))
}
