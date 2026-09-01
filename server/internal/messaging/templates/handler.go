package messagetemplate

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/labstack/echo/v5"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *echo.Context) error {
	var req APICreateRequest
	if err := decodeJSON(c, &req, false); err != nil {
		return err
	}
	value, err := h.service.CreateAPI(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) List(c *echo.Context) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	value, err := h.service.List(c.Request().Context(), ListRequest{Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) Get(c *echo.Context) error {
	value, err := h.service.GetAPI(c.Request().Context(), c.Param("template"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Update(c *echo.Context) error {
	var req APIUpdateRequest
	if err := decodeJSON(c, &req, false); err != nil {
		return err
	}
	value, err := h.service.UpdateAPI(c.Request().Context(), c.Param("template"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Delete(c *echo.Context) error {
	value, err := h.service.DeleteAPI(c.Request().Context(), c.Param("template"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Publish(c *echo.Context) error {
	value, err := h.service.PublishAPI(c.Request().Context(), c.Param("template"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Duplicate(c *echo.Context) error {
	value, err := h.service.DuplicateAPI(c.Request().Context(), c.Param("template"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) ListVersions(c *echo.Context) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	values, err := h.service.ListVersions(c.Request().Context(), c.Param("template"), ListRequest{Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, values)
}
func (h *Handler) GetVersion(c *echo.Context) error {
	value, err := h.service.GetVersion(c.Request().Context(), c.Param("template"), c.Param("version_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Revert(c *echo.Context) error {
	value, err := h.service.Revert(c.Request().Context(), c.Param("template"), c.Param("version_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) Preview(c *echo.Context) error {
	var req PreviewRequest
	if err := decodeJSON(c, &req, true); err != nil {
		return err
	}
	value, err := h.service.Preview(c.Request().Context(), c.Param("template"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}
func (h *Handler) TestSend(c *echo.Context) error {
	var req TestSendRequest
	if err := decodeJSON(c, &req, false); err != nil {
		return err
	}
	value, err := h.service.TestSend(c.Request().Context(), c.Param("template"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Accepted(c, value)
}

func decodeJSON(c *echo.Context, dst any, optional bool) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if optional && errors.Is(err, io.EOF) {
			return nil
		}
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	return nil
}

func pagination(c *echo.Context) (int32, int32, error) {
	limit, offset, err := httputil.Pagination(c)
	if err != nil {
		return 0, 0, err
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset, nil
}
