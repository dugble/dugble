package senderids

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/pkg/httputil"
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

	senderIDs, err := handler.service.List(c.Request().Context(), filter)
	if err != nil {
		return err
	}

	return httputil.OK(c, senderIDs)
}

func (handler *Handler) Detail(c *echo.Context) error {
	detail, err := handler.service.Detail(c.Request().Context(), c.Param("sender_id"))
	if err != nil {
		return err
	}

	return httputil.OK(c, detail)
}

func (handler *Handler) UpdateStatus(c *echo.Context) error {
	senderID := c.Param("sender_id")
	var request StatusRequest
	if err := httputil.DecodeJSON(c, &request, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return err
	}
	if err := handler.service.UpdateStatus(c.Request().Context(), senderID, request); err != nil {
		return err
	}

	return httputil.OK(c, map[string]string{"sender_id": senderID})
}
