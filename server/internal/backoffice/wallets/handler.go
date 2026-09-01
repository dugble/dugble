package wallets

import (
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authn"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *echo.Context) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := h.service.List(c.Request().Context(), ListInput{Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}
func (h *Handler) Get(c *echo.Context) error {
	item, err := h.service.Get(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, item)
}
func (h *Handler) ListTransactions(c *echo.Context) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	page, err := h.service.ListTransactions(c.Request().Context(), TransactionListInput{TeamID: c.Param("team_id"), Limit: limit, Offset: offset})
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, page)
}
func (h *Handler) GetTransaction(c *echo.Context) error {
	item, err := h.service.GetTransaction(c.Request().Context(), c.Param("team_id"), c.Param("transaction_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, item)
}
func (h *Handler) Adjust(c *echo.Context) error {
	var input AdjustmentInput
	if err := httputil.DecodeJSON(c, &input, httputil.DefaultMaxRequestBodyBytes); err != nil {
		return httputil.Error(c, err)
	}
	principal := authn.MustPrincipalFromContext(c.Request().Context())
	input.ActorUserID, input.SessionID = principal.UserID.String(), principal.SessionID
	item, err := h.service.Adjust(c.Request().Context(), c.Param("team_id"), input)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, item)
}
func pagination(c *echo.Context) (int32, int32, error) {
	parse := func(value, name string) (int32, error) {
		if value == "" {
			return 0, nil
		}
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, apperrors.NewBadRequest("Invalid " + name)
		}
		return int32(n), nil
	}
	limit, err := parse(c.QueryParam("limit"), "limit")
	if err != nil {
		return 0, 0, err
	}
	offset, err := parse(c.QueryParam("offset"), "offset")
	return limit, offset, err
}
