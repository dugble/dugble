package smsrates

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := handler.service.List(c.Request().Context(), ListInput{Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}

func (handler *Handler) Get(c *echo.Context) error {
	item, err := handler.service.Get(c.Request().Context(), c.Param("rate_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, item)
}

func (handler *Handler) Create(c *echo.Context) error {
	var input CreateInput
	if err := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	item, err := handler.service.Create(c.Request().Context(), input)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, item)
}

func (handler *Handler) Close(c *echo.Context) error {
	var input CloseInput
	if err := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	item, err := handler.service.Close(c.Request().Context(), c.Param("rate_id"), input)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, item)
}

func (handler *Handler) Replace(c *echo.Context) error {
	var input ReplaceInput
	if err := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	item, err := handler.service.Replace(c.Request().Context(), c.Param("rate_id"), input)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, item)
}
