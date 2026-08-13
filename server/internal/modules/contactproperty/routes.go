package contactproperty

import (
	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	properties := router.Group("/contact-properties")
	properties.POST("", handler.Create, accessMiddleware(authz.PermissionContactPropertiesWrite))
	properties.GET("", handler.List, accessMiddleware(authz.PermissionContactPropertiesRead))
	properties.GET("/:property_id", handler.Get, accessMiddleware(authz.PermissionContactPropertiesRead))
	properties.PATCH("/:property_id", handler.Update, accessMiddleware(authz.PermissionContactPropertiesWrite))
	properties.DELETE("/:property_id", handler.Delete, accessMiddleware(authz.PermissionContactPropertiesWrite))
}
