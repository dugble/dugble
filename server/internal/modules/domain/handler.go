package domain

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

const (
	emailInfrastructureRetryAfterSeconds   = 10
	emailInfrastructureProvisioningMessage = "Customer email infrastructure is being prepared"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *echo.Context) error {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	domains, err := h.service.List(c.Request().Context(), limit, offset)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, domains)
}

func (h *Handler) Get(c *echo.Context) error {
	domain, err := h.service.Get(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, domain)
}

func (h *Handler) Create(c *echo.Context) error {
	var req CreateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	result, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	if result.Provisioning {
		c.Response().Header().Set("Retry-After", strconv.Itoa(emailInfrastructureRetryAfterSeconds))
		return httputil.Accepted(c, ProvisioningResponse{
			Status:            "provisioning",
			Message:           emailInfrastructureProvisioningMessage,
			RetryAfterSeconds: emailInfrastructureRetryAfterSeconds,
		})
	}
	return httputil.Created(c, result.Domain)
}

func (h *Handler) Update(c *echo.Context) error {
	var req UpdateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	domain, err := h.service.Update(c.Request().Context(), c.Param("domain_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, domain)
}

func (h *Handler) Verify(c *echo.Context) error {
	domain, err := h.service.Verify(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, domain)
}

func (h *Handler) Delete(c *echo.Context) error {
	domain, err := h.service.Delete(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, domain)
}

func decodeJSON(c *echo.Context, dst any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	return nil
}
