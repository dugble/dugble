package email

import "github.com/labstack/echo/v5"

func RegisterRoutes(r *echo.Echo, h *Handler, m ...echo.MiddlewareFunc) {
	g := r.Group("/email-messages")
	g.Use(m...)
	g.GET("", h.List)
	g.GET("/:message_id", h.Detail)
}
