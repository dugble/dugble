package subscription

import (
	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/labstack/echo/v5"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	subscription := router.Group("/subscription")
	subscription.GET("", handler.Get, accessMiddleware(authz.PermissionWalletRead))
	subscription.GET("/charges", handler.ListCharges, accessMiddleware(authz.PermissionWalletRead))
	subscription.POST("", handler.SelectPlan, accessMiddleware(authz.PermissionTeamUpdate))
	subscription.POST("/cancel-change", handler.CancelPendingPlanChange, accessMiddleware(authz.PermissionTeamUpdate))
	subscription.POST("/cancel", handler.Cancel, accessMiddleware(authz.PermissionTeamUpdate))
	subscription.POST("/reactivate", handler.Reactivate, accessMiddleware(authz.PermissionTeamUpdate))
}
