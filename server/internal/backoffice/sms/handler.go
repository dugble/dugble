package sms

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) List(c *echo.Context) error {
	filter := Filter{
		Query:  strings.TrimSpace(c.QueryParam("q")),
		Status: strings.TrimSpace(c.QueryParam("status")),
	}

	messages, err := handler.service.List(c.Request().Context(), filter)
	if err != nil {
		return err
	}

	return httputil.OK(c, messages)
}

func (handler *Handler) Detail(c *echo.Context) error {
	detail, err := handler.service.Detail(c.Request().Context(), c.Param("sms_id"))
	if err != nil {
		return err
	}

	return httputil.OK(c, detail)
}
