package segment

import (
	"encoding/json"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *echo.Context) error {
	var req CreateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	value, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, value)
}

func (h *Handler) List(c *echo.Context) error {
	req, err := listRequest(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	values, err := h.service.List(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, values)
}

func (h *Handler) Get(c *echo.Context) error {
	value, err := h.service.Get(c.Request().Context(), c.Param("segment_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) GetAudienceSize(c *echo.Context) error {
	value, err := h.service.GetAudienceSize(c.Request().Context(), c.Param("segment_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) ListContacts(c *echo.Context) error {
	req, err := listRequest(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	values, err := h.service.ListContacts(c.Request().Context(), c.Param("segment_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, values)
}

func (h *Handler) Delete(c *echo.Context) error {
	value, err := h.service.Delete(c.Request().Context(), c.Param("segment_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func decodeJSON(c *echo.Context, dst any) error {
	if err := json.NewDecoder(c.Request().Body).Decode(dst); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	return nil
}

func listRequest(c *echo.Context) (ListRequest, error) {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return ListRequest{}, err
	}
	return ListRequest{Limit: limit, Offset: offset}, nil
}
