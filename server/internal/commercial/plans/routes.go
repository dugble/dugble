package plan

import (
	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/labstack/echo/v5"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	router.GET("/plans", handler.List, accessMiddleware(authz.PermissionWalletRead))
}
