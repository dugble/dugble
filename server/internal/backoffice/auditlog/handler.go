package auditlog

import (
	"context"
	"strconv"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/pkg/httputil"
)

type service interface {
	List(context.Context, Filter) (Page, error)
}
type Handler struct{ service service }

func NewHandler(service service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *echo.Context) error {
	limit, err := number(c.QueryParam("limit"))
	if err != nil {
		return err
	}
	offset, err := number(c.QueryParam("offset"))
	if err != nil {
		return err
	}
	filter := Filter{Query: c.QueryParam("q"), Outcome: c.QueryParam("outcome"), ActorType: c.QueryParam("actor_type"), Limit: limit, Offset: offset}
	page, err := h.service.List(c.Request().Context(), filter)
	if err != nil {
		return err
	}
	return httputil.OK(c, page)
}

func number(value string) (int32, error) {
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, apperrors.NewBadRequest("Invalid pagination")
	}
	return int32(n), nil
}
