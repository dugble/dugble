package teams

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (handler *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := handler.service.List(c.Request().Context(), Filter{
		Query: strings.TrimSpace(c.QueryParam("q")), Status: strings.TrimSpace(c.QueryParam("status")), Limit: limit, Offset: offset,
	})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}

func (handler *Handler) Detail(c *echo.Context) error {
	detail, err := handler.service.Detail(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, detail)
}

func (handler *Handler) UpdateStatus(c *echo.Context) error {
	var request StatusRequest
	if err := httputil.DecodeJSON(c, &request, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	detail, err := handler.service.UpdateStatus(c.Request().Context(), c.Param("team_id"), request)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, detail)
}
