package dashboard

import (
	"context"

	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

type service interface {
	Operations(context.Context) (Operations, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Index(c *echo.Context) error {
	operations, err := handler.service.Operations(c.Request().Context())
	if err != nil {
		return err
	}

	return httputil.OK(c, operations)
}
