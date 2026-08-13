package email

import (
	"context"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type service interface {
	List(context.Context, Filter) ([]Row, error)
	Detail(context.Context, string) (Detail, error)
}
type Handler struct{ service service }

func NewHandler(s service) *Handler { return &Handler{service: s} }
func (h *Handler) List(c *echo.Context) error {
	f := Filter{Query: strings.TrimSpace(c.QueryParam("q")), Status: strings.TrimSpace(c.QueryParam("status"))}
	rows, err := h.service.List(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return httputil.OK(c, rows)
}
func (h *Handler) Detail(c *echo.Context) error {
	d, err := h.service.Detail(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return err
	}
	return httputil.OK(c, d)
}
