package subscriptions

import (
	"context"

	"github.com/dugble/dugble/server/internal/authn"
	"github.com/dugble/dugble/server/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) List(c *echo.Context) error {
	l, o, e := httputil.Pagination(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	p, e := h.service.List(c.Request().Context(), Filter{TeamID: c.QueryParam("team_id"), Status: c.QueryParam("status"), Limit: l, Offset: o})
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, p)
}
func (h *Handler) ChangePlan(c *echo.Context) error {
	var input ChangePlanInput
	if e := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); e != nil {
		return httputil.Error(c, e)
	}
	p := authn.MustPrincipalFromContext(c.Request().Context())
	input.ActorUserID, input.SessionID = p.UserID.String(), p.SessionID
	x, e := h.service.ChangePlan(c.Request().Context(), c.Param("subscription_id"), input)
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, x)
}
func (h *Handler) CancelPlanChange(c *echo.Context) error {
	return h.action(c, h.service.CancelPlanChange)
}
func (h *Handler) Cancel(c *echo.Context) error     { return h.action(c, h.service.Cancel) }
func (h *Handler) Reactivate(c *echo.Context) error { return h.action(c, h.service.Reactivate) }
func (h *Handler) action(c *echo.Context, operation func(context.Context, string, ActionInput) (Subscription, error)) error {
	var input ActionInput
	if e := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); e != nil {
		return httputil.Error(c, e)
	}
	p := authn.MustPrincipalFromContext(c.Request().Context())
	input.ActorUserID, input.SessionID = p.UserID.String(), p.SessionID
	x, e := operation(c.Request().Context(), c.Param("subscription_id"), input)
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, x)
}
func (h *Handler) Get(c *echo.Context) error {
	x, e := h.service.Get(c.Request().Context(), c.Param("subscription_id"))
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, x)
}
func (h *Handler) Charges(c *echo.Context) error {
	l, o, e := httputil.Pagination(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	x, e := h.service.Charges(c.Request().Context(), c.Param("subscription_id"), l, o)
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, x)
}
