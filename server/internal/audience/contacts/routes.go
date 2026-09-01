package contact

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	contacts := router.Group("/contacts")
	contacts.POST("", handler.Create, accessMiddleware(authz.PermissionContactsWrite))
	contacts.GET("", handler.List, accessMiddleware(authz.PermissionContactsRead))
	contacts.GET("/:contact_id/topics", handler.ListTopics, accessMiddleware(authz.PermissionContactsRead))
	contacts.PATCH("/:contact_id/topics", handler.UpdateTopics, accessMiddleware(authz.PermissionContactsWrite))
	contacts.GET("/:contact_id/segments", handler.ListSegments, accessMiddleware(authz.PermissionContactsRead))
	contacts.POST("/:contact_id/segments/:segment_id", handler.AddSegment, accessMiddleware(authz.PermissionContactsWrite))
	contacts.DELETE("/:contact_id/segments/:segment_id", handler.RemoveSegment, accessMiddleware(authz.PermissionContactsWrite))
	contacts.GET("/:contact_id", handler.Get, accessMiddleware(authz.PermissionContactsRead))
	contacts.PATCH("/:contact_id", handler.Update, accessMiddleware(authz.PermissionContactsWrite))
	contacts.DELETE("/:contact_id", handler.Delete, accessMiddleware(authz.PermissionContactsWrite))
}
