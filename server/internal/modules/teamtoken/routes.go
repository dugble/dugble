package teamtoken

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authz"
)

type TenantMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(
	router *echo.Echo,
	handler *Handler,
	authMiddleware echo.MiddlewareFunc,
	csrfMiddleware echo.MiddlewareFunc,
	tenantMiddleware TenantMiddleware,
) {
	tokens := router.Group("/team-tokens")
	tokens.Use(authMiddleware, csrfMiddleware)
	tokens.GET("", handler.List, tenantMiddleware(authz.PermissionTeamTokensRead))
	tokens.POST("", handler.Create, tenantMiddleware(authz.PermissionTeamTokensCreate))
	tokens.PATCH("/:token_id", handler.Update, tenantMiddleware(authz.PermissionTeamTokensUpdate))
	tokens.DELETE("/:token_id", handler.Revoke, tenantMiddleware(authz.PermissionTeamTokensRevoke))
}
