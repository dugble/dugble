package subscriptions

import "github.com/labstack/echo/v5"

func RegisterRoutes(r *echo.Echo, h *Handler, m ...echo.MiddlewareFunc) {
	g := r.Group("/subscriptions")
	g.Use(m...)
	g.GET("", h.List)
	g.GET("/:subscription_id", h.Get)
	g.GET("/:subscription_id/charges", h.Charges)
	g.POST("/:subscription_id/change-plan", h.ChangePlan)
	g.POST("/:subscription_id/cancel-pending-change", h.CancelPlanChange)
	g.POST("/:subscription_id/cancel", h.Cancel)
	g.POST("/:subscription_id/reactivate", h.Reactivate)
}
