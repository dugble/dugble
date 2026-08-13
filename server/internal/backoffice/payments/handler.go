package payments

import (
	"strings"

	"github.com/coffeyvidzro/dugble/server/internal/authn"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) List(c *echo.Context) error {
	limit, offset, e := httputil.Pagination(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	page, e := h.service.List(c.Request().Context(), Filter{TeamID: c.QueryParam("team_id"), Status: strings.TrimSpace(c.QueryParam("status")), Provider: strings.TrimSpace(c.QueryParam("provider")), Limit: limit, Offset: offset})
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, page)
}
func (h *Handler) Reconcile(c *echo.Context) error {
	var input ReconcileInput
	if e := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); e != nil {
		return httputil.Error(c, e)
	}
	principal := authn.MustPrincipalFromContext(c.Request().Context())
	input.ActorUserID, input.SessionID = principal.UserID.String(), principal.SessionID
	item, e := h.service.Reconcile(c.Request().Context(), c.Param("payment_id"), input)
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, item)
}
func (h *Handler) Get(c *echo.Context) error {
	item, e := h.service.Get(c.Request().Context(), c.Param("payment_id"))
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.OK(c, item)
}
