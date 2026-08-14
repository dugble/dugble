package users

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (handler *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := handler.service.List(c.Request().Context(), Filter{
		Query: strings.TrimSpace(c.QueryParam("q")), Limit: limit, Offset: offset,
	})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}

func (handler *Handler) Detail(c *echo.Context) error {
	detail, err := handler.service.Detail(c.Request().Context(), c.Param("user_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, detail)
}
