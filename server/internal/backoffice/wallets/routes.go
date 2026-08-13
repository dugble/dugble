package wallets

import "github.com/labstack/echo/v5"

func RegisterRoutes(router *echo.Echo, handler *Handler, middleware ...echo.MiddlewareFunc) {
	wallets := router.Group("/wallets")
	wallets.Use(middleware...)
	wallets.GET("", handler.List)
	wallets.GET("/:team_id", handler.Get)
	wallets.GET("/:team_id/transactions", handler.ListTransactions)
	wallets.GET("/:team_id/transactions/:transaction_id", handler.GetTransaction)
	wallets.POST("/:team_id/adjustments", handler.Adjust)
	transactions := router.Group("/wallet-transactions")
	transactions.Use(middleware...)
	transactions.GET("", handler.ListTransactions)
}
