package payments

import "github.com/labstack/echo/v5"

func RegisterRoutes(r *echo.Echo, h *Handler, m ...echo.MiddlewareFunc) {
	g := r.Group("/payments")
	g.Use(m...)
	g.GET("", h.List)
	g.GET("/:payment_id", h.Get)
	g.POST("/:payment_id/reconcile", h.Reconcile)
}
