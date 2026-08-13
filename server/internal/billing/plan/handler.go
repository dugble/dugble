package plan

import (
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(c *echo.Context) error {
	plans, err := h.service.List(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, plans)
}
