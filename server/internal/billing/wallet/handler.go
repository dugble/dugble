package wallet

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	hubteladapter "github.com/dugble/dugble/server/internal/adapters/hubtel"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(c *echo.Context) error {
	wallet, err := h.service.Get(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, wallet)
}

func (h *Handler) ListLedger(c *echo.Context) error {
	limit, err := parseInt32Query(c, "limit")
	if err != nil {
		return httputil.Error(c, err)
	}
	offset, err := parseInt32Query(c, "offset")
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := h.service.ListLedger(c.Request().Context(), limit, offset)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}

func (h *Handler) TopUp(c *echo.Context) error {
	var req TopUpRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	transaction, err := h.service.TopUp(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, transaction)
}

func (h *Handler) HubtelWebhook(c *echo.Context) error {
	var payload hubteladapter.CallbackPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	transaction, err := h.service.HandleHubtelCallback(c.Request().Context(), payload)
	if err != nil {
		return httputil.Error(c, err)
	}
	if transaction == nil {
		return httputil.NoContent(c)
	}
	return httputil.OK(c, transaction)
}

func parseInt32Query(c *echo.Context, name string) (int32, error) {
	value := strings.TrimSpace(c.QueryParam(name))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, apperrors.NewBadRequest("Wallet " + name + " must be an integer")
	}
	return int32(parsed), nil
}
