package wallet

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, accessMiddleware AccessMiddleware) {
	wallet := router.Group("/wallet")
	wallet.GET("", handler.Get, accessMiddleware(authz.PermissionWalletRead))
	wallet.GET("/ledger", handler.ListLedger, accessMiddleware(authz.PermissionWalletLedgerRead))
	wallet.POST("/topup", handler.TopUp, accessMiddleware(authz.PermissionTeamUpdate))
	router.POST("/wallet/webhook/hubtel", handler.HubtelWebhook)
}
