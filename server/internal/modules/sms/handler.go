package sms

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/platform/idempotency"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	messages, err := h.service.List(c.Request().Context(), ListRequest{Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, Responses(messages))
}

func (h *Handler) Get(c *echo.Context) error {
	message, err := h.service.Get(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, message.Response())
}

func (h *Handler) ListEvents(c *echo.Context) error {
	limit, _, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	response, err := h.service.ListEvents(c.Request().Context(), c.Param("message_id"), limit)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (h *Handler) Send(c *echo.Context) error {
	if _, err := idempotency.ValidateKey(c.Request().Header.Get(idempotency.Header)); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Idempotency-Key is required and must be at most 256 characters"))
	}
	var req SendRequest
	if err := httputil.DecodeJSON(c, &req, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	message, err := h.service.Send(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	c.Response().Header().Set("Location", "/sms/"+message.ID)
	return httputil.Accepted(c, message.SendResponse())
}

func (h *Handler) BatchSend(c *echo.Context) error {
	if _, err := idempotency.ValidateKey(c.Request().Header.Get(idempotency.Header)); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Idempotency-Key is required and must be at most 256 characters"))
	}
	var req BatchSendRequest
	if err := httputil.DecodeJSON(c, &req, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	response, err := h.service.BatchSend(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Accepted(c, SendResponses(response))
}

func (h *Handler) Cancel(c *echo.Context) error {
	response, err := h.service.Cancel(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (h *Handler) Update(c *echo.Context) error {
	var req UpdateRequest
	if err := httputil.DecodeJSON(c, &req, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	response, err := h.service.Update(c.Request().Context(), c.Param("message_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (h *Handler) SyncStatus(c *echo.Context) error {
	message, err := h.service.SyncStatus(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, message.Response())
}
