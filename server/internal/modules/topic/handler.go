package topic

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
	var req CreateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	value, err := h.service.CreateAPI(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

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
	values, err := h.service.List(c.Request().Context(), ListRequest{Limit: limit + 1, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	hasMore := len(values) > int(limit)
	if hasMore {
		values = values[:limit]
	}
	data := make([]Resource, 0, len(values))
	for _, value := range values {
		data = append(data, resourceFromTopic(value))
	}
	return httputil.OK(c, ListResponse{Object: ObjectList, HasMore: hasMore, Data: data})
}

func (h *Handler) Get(c *echo.Context) error {
	value, err := h.service.GetAPI(c.Request().Context(), c.Param("topic_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) Update(c *echo.Context) error {
	var req UpdateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	value, err := h.service.UpdateAPI(c.Request().Context(), c.Param("topic_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func (h *Handler) Delete(c *echo.Context) error {
	value, err := h.service.DeleteAPI(c.Request().Context(), c.Param("topic_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, value)
}

func decodeJSON(c *echo.Context, dst any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	return nil
}
