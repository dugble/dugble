package allowances

import "github.com/labstack/echo/v5"

func RegisterRoutes(router *echo.Echo, handler *Handler, middleware ...echo.MiddlewareFunc) {
	group := router.Group("/allowances")
	group.Use(middleware...)
	group.GET("", handler.List)
	group.GET("/:allowance_id", handler.Get)
	group.POST("", handler.Create)
	group.POST("/:allowance_id/close", handler.Close)
	group.POST("/:allowance_id/replace", handler.Replace)
}
