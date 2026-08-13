package email

import (
	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/platform/idempotency"
	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Send(c *echo.Context) error {
	if handler == nil || handler.service == nil {
		return httputil.Error(c, apperrors.NewServiceUnavailable("Email service is not configured", nil))
	}
	if _, err := idempotency.ValidateKey(c.Request().Header.Get(idempotency.Header)); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Idempotency-Key is required and must be at most 256 characters"))
	}
	var request SendRequest
	if err := httputil.DecodeJSON(c, &request, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	message, err := handler.service.Send(c.Request().Context(), request)
	if err != nil {
		return httputil.Error(c, err)
	}
	c.Response().Header().Set("Location", "/emails/"+message.ID)
	return httputil.Accepted(c, message.SendResponse())
}

func (handler *Handler) Get(c *echo.Context) error {
	message, err := handler.service.Get(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, message.RetrieveResponse())
}

func (handler *Handler) ListEvents(c *echo.Context) error {
	limit, _, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	response, err := handler.service.ListEvents(c.Request().Context(), c.Param("message_id"), limit)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (handler *Handler) Cancel(c *echo.Context) error {
	response, err := handler.service.Cancel(c.Request().Context(), c.Param("message_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (handler *Handler) Update(c *echo.Context) error {
	var request UpdateRequest
	if err := httputil.DecodeJSON(c, &request, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	response, err := handler.service.Update(c.Request().Context(), c.Param("message_id"), request)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (handler *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	messages, err := handler.service.List(c.Request().Context(), ListRequest{
		Limit: limit, Offset: offset,
	})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, messages)
}

func (handler *Handler) BatchSend(c *echo.Context) error {
	if _, err := idempotency.ValidateKey(c.Request().Header.Get(idempotency.Header)); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Idempotency-Key is required and must be at most 256 characters"))
	}
	var request BatchSendRequest
	if err := httputil.DecodeJSON(c, &request, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	messages, err := handler.service.BatchSend(c.Request().Context(), request)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Accepted(c, SendResponses(messages))
}
