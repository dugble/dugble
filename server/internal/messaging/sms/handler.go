package sms

import (
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/platform/idempotency"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetAnalytics(c *echo.Context) error {
	response, err := h.service.GetAnalytics(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, response)
}

func (h *Handler) List(c *echo.Context) error {
	req, err := listRequest(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	messages, err := h.service.List(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, Responses(messages))
}

func listRequest(c *echo.Context) (ListRequest, error) {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return ListRequest{}, err
	}
	status := strings.ToLower(strings.TrimSpace(c.QueryParam("status")))
	if status != "" && !isValidStatus(status) {
		return ListRequest{}, apperrors.NewBadRequest("status must be a valid SMS status")
	}
	startDate, err := queryDate(c, "start_date")
	if err != nil {
		return ListRequest{}, err
	}
	endDate, err := queryDate(c, "end_date")
	if err != nil {
		return ListRequest{}, err
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		return ListRequest{}, apperrors.NewBadRequest("start_date must be before or equal to end_date")
	}
	return ListRequest{
		Limit: limit, Offset: offset, Status: status,
		Sender:    strings.TrimSpace(c.QueryParam("sender")),
		StartDate: startDate, EndDate: endDate,
		Search: strings.TrimSpace(c.QueryParam("search")),
	}, nil
}

func queryDate(c *echo.Context, name string) (*time.Time, error) {
	value := strings.TrimSpace(c.QueryParam(name))
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, apperrors.NewBadRequest(name + " must be a valid RFC 3339 timestamp")
	}
	return &parsed, nil
}

func isValidStatus(status string) bool {
	switch status {
	case StatusQueued, StatusProcessing, StatusSubmitted, StatusSent, StatusDelivered,
		StatusUndelivered, StatusRejected, StatusFailed, StatusExpired, StatusUnknown, StatusCanceled:
		return true
	default:
		return false
	}
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
