package auditlog

import "github.com/labstack/echo/v5"

func RegisterRoutes(router *echo.Echo, handler *Handler, middleware ...echo.MiddlewareFunc) {
	group := router.Group("/audit-log")
	group.Use(middleware...)
	group.GET("", handler.List)
}
