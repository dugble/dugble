package suppression

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	suppressions := router.Group("/suppressions")
	suppressions.POST("/batch/add", handler.BatchAdd, accessMiddleware(authz.PermissionSuppressionsWrite))
	suppressions.POST("/batch/remove", handler.BatchRemove, accessMiddleware(authz.PermissionSuppressionsWrite))
	suppressions.POST("", handler.Create, accessMiddleware(authz.PermissionSuppressionsWrite))
	suppressions.GET("", handler.List, accessMiddleware(authz.PermissionSuppressionsRead))
	suppressions.GET("/:suppression", handler.Get, accessMiddleware(authz.PermissionSuppressionsRead))
	suppressions.DELETE("/:suppression", handler.Delete, accessMiddleware(authz.PermissionSuppressionsWrite))
}
