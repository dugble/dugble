package subscription

import (
	"strconv"
	"strings"

	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) Get(c *echo.Context) error {
	result, err := h.service.Get(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}
func (h *Handler) ListCharges(c *echo.Context) error {
	limit, err := parsePageValue(c.QueryParam("limit"), "limit")
	if err != nil {
		return httputil.Error(c, err)
	}
	offset, err := parsePageValue(c.QueryParam("offset"), "offset")
	if err != nil {
		return httputil.Error(c, err)
	}
	result, err := h.service.ListCharges(c.Request().Context(), limit, offset)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}

func parsePageValue(value, name string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, apperrors.NewBadRequest("Subscription charge " + name + " must be an integer")
	}
	return int32(parsed), nil
}
func (h *Handler) SelectPlan(c *echo.Context) error {
	var input SelectPlanInput
	if err := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	result, err := h.service.SelectPlan(c.Request().Context(), input)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}
func (h *Handler) CancelPendingPlanChange(c *echo.Context) error {
	result, err := h.service.CancelPendingPlanChange(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}

func (h *Handler) Cancel(c *echo.Context) error {
	result, err := h.service.Cancel(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}

func (h *Handler) Reactivate(c *echo.Context) error {
	result, err := h.service.Reactivate(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, result)
}
