package messagetemplate

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	templates := router.Group("/templates")
	templates.POST("", handler.Create, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.GET("", handler.List, accessMiddleware(authz.PermissionTemplatesRead))
	templates.GET("/:template", handler.Get, accessMiddleware(authz.PermissionTemplatesRead))
	templates.PATCH("/:template", handler.Update, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.DELETE("/:template", handler.Delete, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.POST("/:template/publish", handler.Publish, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.POST("/:template/duplicate", handler.Duplicate, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.GET("/:template/versions", handler.ListVersions, accessMiddleware(authz.PermissionTemplatesRead))
	templates.GET("/:template/versions/:version_id", handler.GetVersion, accessMiddleware(authz.PermissionTemplatesRead))
	templates.POST("/:template/versions/:version_id/revert", handler.Revert, accessMiddleware(authz.PermissionTemplatesWrite))
	templates.POST("/:template/preview", handler.Preview, accessMiddleware(authz.PermissionTemplatesRead))
	templates.POST("/:template/test-send", handler.TestSend, accessMiddleware(authz.PermissionTemplatesWrite))
}
