package domainclaim

import (
	"encoding/json"
	"io"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Start(c *echo.Context) error {
	var req Request
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	claim, err := h.service.Start(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, claim)
}

func (h *Handler) Get(c *echo.Context) error {
	claim, err := h.service.Get(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, claim)
}

func (h *Handler) Verify(c *echo.Context) error {
	claim, err := h.service.Verify(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, claim)
}

func (h *Handler) Cancel(c *echo.Context) error {
	claim, err := h.service.Cancel(c.Request().Context(), c.Param("domain_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, claim)
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
