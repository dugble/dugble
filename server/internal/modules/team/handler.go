package team

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authn"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
	"github.com/dugble/dugble/server/pkg/httputil"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *echo.Context) error {
	options, err := parseListOptions(
		c.QueryParam("page"),
		c.QueryParam("limit"),
		c.QueryParam("search"),
		c.QueryParam("status"),
	)
	if err != nil {
		return httputil.Error(c, err)
	}
	teams, meta, err := h.listTeamsPaginated(c.Request().Context(), options)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OKWithMeta(c, teams, meta)
}

func (h *Handler) listTeamsPaginated(ctx context.Context, options ListOptions) ([]Team, ListMeta, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return nil, ListMeta{}, apperrors.NewUnauthorized("Authentication is required")
	}

	searchPattern := "%" + options.Search + "%"
	var total int64
	if err := h.service.repository.db.QueryRow(ctx, `
		SELECT count(*)
		FROM teams AS t
		JOIN team_members AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.status = $2
		  AND ($3 = '' OR t.name ILIKE $4)`, principal.UserID, options.Status, options.Search, searchPattern).Scan(&total); err != nil {
		return nil, ListMeta{}, apperrors.NewInternal("Unable to list teams", fmt.Errorf("count teams for user: %w", err))
	}

	offset := (options.Page - 1) * options.Limit
	rows, err := h.service.repository.db.Query(ctx, `
		SELECT t.id, t.name, t.market_code, t.phone, t.address, t.website, t.status, t.created_by, t.created_at, t.updated_at, tm.role
		FROM teams AS t
		JOIN team_members AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.status = $2
		  AND ($3 = '' OR t.name ILIKE $4)
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $5 OFFSET $6`, principal.UserID, options.Status, options.Search, searchPattern, options.Limit, offset)
	if err != nil {
		return nil, ListMeta{}, apperrors.NewInternal("Unable to list teams", fmt.Errorf("list teams for user: %w", err))
	}
	defer rows.Close()

	teams := make([]Team, 0, options.Limit)
	for rows.Next() {
		var (
			team      Team
			teamID    uuid.UUID
			createdBy *uuid.UUID
			createdAt pgtype.Timestamptz
			updatedAt pgtype.Timestamptz
		)
		if err := rows.Scan(
			&teamID, &team.Name, &team.MarketCode, &team.Phone, &team.Address, &team.Website,
			&team.Status, &createdBy, &createdAt, &updatedAt, &team.UserRole,
		); err != nil {
			return nil, ListMeta{}, apperrors.NewInternal("Unable to list teams", fmt.Errorf("scan team: %w", err))
		}
		team.ID = teamID.String()
		team.CreatedBy = stringPointer(createdBy)
		team.CreatedAt = pgconv.TimestamptzToTime(createdAt)
		team.UpdatedAt = pgconv.TimestamptzToTime(updatedAt)
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, ListMeta{}, apperrors.NewInternal("Unable to list teams", fmt.Errorf("iterate teams: %w", err))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(options.Limit) - 1) / int64(options.Limit))
	}
	return teams, ListMeta{Pagination: PaginationMeta{
		Page:       options.Page,
		Limit:      options.Limit,
		Total:      total,
		TotalPages: totalPages,
	}}, nil
}

func (h *Handler) Create(c *echo.Context) error {
	var req CreateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	team, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, team)
}

func (h *Handler) Get(c *echo.Context) error {
	team, err := h.service.Get(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, team)
}

func (h *Handler) Update(c *echo.Context) error {
	var req UpdateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	team, err := h.service.Update(c.Request().Context(), c.Param("team_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, team)
}

func (h *Handler) Delete(c *echo.Context) error {
	team, err := h.service.Delete(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, team)
}

func (h *Handler) ListMembers(c *echo.Context) error {
	members, err := h.service.ListMembers(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, members)
}

func (h *Handler) Leave(c *echo.Context) error {
	if err := h.service.Leave(c.Request().Context(), c.Param("team_id")); err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, map[string]bool{"left": true})
}

func (h *Handler) RemoveMember(c *echo.Context) error {
	if err := h.service.RemoveMember(
		c.Request().Context(),
		c.Param("team_id"),
		c.Param("user_id"),
	); err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, map[string]bool{"removed": true})
}

func (h *Handler) UpdateMemberRole(c *echo.Context) error {
	var req UpdateMemberRoleRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	member, err := h.service.UpdateMemberRole(
		c.Request().Context(),
		c.Param("team_id"),
		c.Param("user_id"),
		req,
	)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, member)
}

func (h *Handler) ListPendingInvitations(c *echo.Context) error {
	invitations, err := h.service.ListPendingInvitations(c.Request().Context())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitations)
}

func (h *Handler) AcceptPendingInvitation(c *echo.Context) error {
	invitationID, err := validateInvitationID(c.Param("invitation_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	invitation, err := h.service.AcceptInvitation(c.Request().Context(), invitationID.String())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func (h *Handler) DeclinePendingInvitation(c *echo.Context) error {
	invitationID, err := validateInvitationID(c.Param("invitation_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	invitation, err := h.service.DeclineInvitation(c.Request().Context(), invitationID.String())
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func (h *Handler) ListTeamInvitations(c *echo.Context) error {
	invitations, err := h.service.ListTeamInvitations(c.Request().Context(), c.Param("team_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitations)
}

func (h *Handler) RevokeInvitation(c *echo.Context) error {
	invitation, err := h.service.RevokeInvitation(c.Request().Context(), c.Param("team_id"), c.Param("invitation_id"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func (h *Handler) InviteMember(c *echo.Context) error {
	var req InviteMemberRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}
	invitation, err := h.service.InviteMember(c.Request().Context(), c.Param("team_id"), req)
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Created(c, invitation)
}

func (h *Handler) GetInvitation(c *echo.Context) error {
	invitation, err := h.service.GetInvitation(c.Request().Context(), c.Param("token"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func (h *Handler) AcceptInvitation(c *echo.Context) error {
	invitation, err := h.service.AcceptInvitation(c.Request().Context(), c.Param("token"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func (h *Handler) DeclineInvitation(c *echo.Context) error {
	invitation, err := h.service.DeclineInvitation(c.Request().Context(), c.Param("token"))
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.OK(c, invitation)
}

func decodeJSON(c *echo.Context, dst any) error {
	if err := json.NewDecoder(c.Request().Body).Decode(dst); err != nil {
		return httputil.Error(c, apperrors.NewBadRequest("Invalid JSON request body"))
	}
	return nil
}
