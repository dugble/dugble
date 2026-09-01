package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/dugble/dugble/server/internal/database/sqlc"
	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/dugble/dugble/server/pkg/pgconv"
)

var (
	ErrInvitationNotAccepted   = errors.New("invitation not accepted")
	ErrTeamMemberAlreadyExists = errors.New("team member already exists")
)

type Repository struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, queries: dbsqlc.New(db)}
}

func (r *Repository) IsBillingMarketEnabled(ctx context.Context, marketCode string) (bool, error) {
	var enabled bool
	err := r.db.QueryRow(ctx, `
		SELECT is_enabled
		FROM billing_markets
		WHERE code = $1`, marketCode).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check billing market: %w", err)
	}
	return enabled, nil
}

func (r *Repository) CreateWithOwner(ctx context.Context, name, marketCode, phone, address string, website *string, ownerID uuid.UUID) (Team, error) {
	row, err := r.queries.CreateTeamWithOwner(ctx, dbsqlc.CreateTeamWithOwnerParams{
		Name: name, MarketCode: marketCode, Phone: phone, Address: address, Website: website, OwnerID: &ownerID,
	})
	if err != nil {
		return Team{}, fmt.Errorf("create team with owner: %w", err)
	}
	return Team{
		ID: row.ID.String(), Name: row.Name, MarketCode: row.MarketCode,
		Phone: row.Phone, Address: row.Address, Website: row.Website,
		Status: row.Status, CreatedBy: stringPointer(row.CreatedBy),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Team, error) {
	row, err := r.queries.GetTeam(ctx, dbsqlc.GetTeamParams{ID: id})
	if err != nil {
		return Team{}, fmt.Errorf("get team: %w", err)
	}
	return teamFromSQLC(row), nil
}

func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]Team, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.market_code, t.phone, t.address, t.website, t.status, t.created_by, t.created_at, t.updated_at, tm.role
		FROM teams AS t
		JOIN team_members AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		  AND tm.status = 'active'
		  AND t.status = 'active'
		ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams for user: %w", err)
	}
	defer rows.Close()
	teams := make([]Team, 0)
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
			return nil, fmt.Errorf("scan team: %w", err)
		}
		team.ID = teamID.String()
		team.CreatedBy = stringPointer(createdBy)
		team.CreatedAt = pgconv.TimestamptzToTime(createdAt)
		team.UpdatedAt = pgconv.TimestamptzToTime(updatedAt)
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}
	return teams, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, name string) (Team, error) {
	row, err := r.queries.UpdateTeam(ctx, dbsqlc.UpdateTeamParams{ID: id, Name: name})
	if err != nil {
		return Team{}, fmt.Errorf("update team: %w", err)
	}
	return teamFromSQLC(row), nil
}

func (r *Repository) Disable(ctx context.Context, id uuid.UUID) (Team, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Team{}, fmt.Errorf("begin disable team transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := r.queries.WithTx(tx).DisableTeam(ctx, dbsqlc.DisableTeamParams{ID: id})
	if err != nil {
		return Team{}, fmt.Errorf("disable team: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE webhook_deliveries AS delivery
SET status = 'canceled', last_error = 'Team disabled before webhook delivery',
    locked_at = NULL, locked_by = NULL, updated_at = now()
FROM webhook_events AS event
WHERE event.id = delivery.event_id AND event.team_id = $1
  AND delivery.status IN ('pending', 'retrying')`, id); err != nil {
		return Team{}, fmt.Errorf("cancel team webhook deliveries: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Team{}, fmt.Errorf("commit disable team transaction: %w", err)
	}
	return teamFromSQLC(row), nil
}

func (r *Repository) GetMember(ctx context.Context, teamID, userID uuid.UUID) (Member, error) {
	row, err := r.queries.GetTeamMember(ctx, dbsqlc.GetTeamMemberParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return Member{}, fmt.Errorf("get team member: %w", err)
	}
	return memberFromSQLC(row), nil
}

func (r *Repository) GetTenantMembership(ctx context.Context, teamID, userID uuid.UUID) (authz.Membership, error) {
	member, err := r.GetMember(ctx, teamID, userID)
	if err != nil {
		return authz.Membership{}, err
	}
	team, err := r.Get(ctx, teamID)
	if err != nil {
		return authz.Membership{}, err
	}
	return authz.Membership{TeamID: teamID, UserID: userID, Role: member.Role, Status: member.Status, TeamStatus: team.Status}, nil
}

func (r *Repository) ListMembers(ctx context.Context, teamID uuid.UUID) ([]Member, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tm.team_id, tm.user_id, tm.role, tm.status, tm.created_at, tm.updated_at, u.name, u.email
		FROM team_members AS tm
		JOIN users AS u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY tm.created_at ASC`, teamID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()
	members := make([]Member, 0)
	for rows.Next() {
		var (
			member    Member
			profile   MemberProfile
			teamID    uuid.UUID
			userID    uuid.UUID
			createdAt pgtype.Timestamptz
			updatedAt pgtype.Timestamptz
		)
		if err := rows.Scan(
			&teamID, &userID, &member.Role, &member.Status, &createdAt, &updatedAt,
			&profile.Name, &profile.Email,
		); err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		member.TeamID = teamID.String()
		member.UserID = userID.String()
		member.CreatedAt = pgconv.TimestamptzToTime(createdAt)
		member.UpdatedAt = pgconv.TimestamptzToTime(updatedAt)
		profile.ID = member.UserID
		member.User = &profile
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team members: %w", err)
	}
	return members, nil
}

func (r *Repository) ListPendingInvitationsForEmail(ctx context.Context, email string) ([]Invitation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT invitation.id, invitation.team_id, team.name, invitation.email, invitation.role, invitation.status,
		       invitation.invited_by, invitation.expires_at, invitation.accepted_at, invitation.declined_at,
		       invitation.created_at, invitation.updated_at
		FROM team_invitations AS invitation
		JOIN teams AS team ON team.id = invitation.team_id
		WHERE lower(invitation.email) = lower($1)
		  AND invitation.status = 'pending'
		  AND invitation.expires_at > now()
		  AND team.status = 'active'
		ORDER BY invitation.created_at DESC`, email)
	if err != nil {
		return nil, fmt.Errorf("list pending invitations for email: %w", err)
	}
	defer rows.Close()
	invitations := make([]Invitation, 0)
	for rows.Next() {
		invitation, err := scanInvitationWithTeamName(rows)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending invitations: %w", err)
	}
	return invitations, nil
}

func (r *Repository) ListPendingInvitationsForTeam(ctx context.Context, teamID uuid.UUID) ([]Invitation, error) {
	rows, err := r.queries.ListPendingTeamInvitations(ctx, dbsqlc.ListPendingTeamInvitationsParams{TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("list pending team invitations: %w", err)
	}
	invitations := make([]Invitation, 0, len(rows))
	for _, row := range rows {
		invitations = append(invitations, invitationFromSQLC(row))
	}
	return invitations, nil
}

func (r *Repository) RevokeInvitation(ctx context.Context, teamID, invitationID uuid.UUID) (Invitation, error) {
	row, err := r.db.Query(ctx, `
		UPDATE team_invitations AS invitation
		SET status = 'revoked', updated_at = now()
		FROM teams AS team
		WHERE invitation.id = $1
		  AND invitation.team_id = $2
		  AND invitation.status = 'pending'
		  AND team.id = invitation.team_id
		  AND team.status = 'active'
		RETURNING invitation.id, invitation.team_id, team.name, invitation.email, invitation.role, invitation.status,
		          invitation.invited_by, invitation.expires_at, invitation.accepted_at, invitation.declined_at,
		          invitation.created_at, invitation.updated_at`, invitationID, teamID)
	if err != nil {
		return Invitation{}, fmt.Errorf("revoke team invitation: %w", err)
	}
	defer row.Close()
	if !row.Next() {
		if err := row.Err(); err != nil {
			return Invitation{}, fmt.Errorf("revoke team invitation: %w", err)
		}
		return Invitation{}, pgx.ErrNoRows
	}
	invitation, err := scanInvitationWithTeamName(row)
	if err != nil {
		return Invitation{}, err
	}
	if err := row.Err(); err != nil {
		return Invitation{}, fmt.Errorf("revoke team invitation: %w", err)
	}
	return invitation, nil
}

func (r *Repository) GetPendingInvitationByIDForEmail(ctx context.Context, invitationID uuid.UUID, email string) (Invitation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT invitation.id, invitation.team_id, team.name, invitation.email, invitation.role, invitation.status,
		       invitation.invited_by, invitation.expires_at, invitation.accepted_at, invitation.declined_at,
		       invitation.created_at, invitation.updated_at
		FROM team_invitations AS invitation
		JOIN teams AS team ON team.id = invitation.team_id
		WHERE invitation.id = $1
		  AND lower(invitation.email) = lower($2)
		  AND invitation.status = 'pending'
		  AND invitation.expires_at > now()
		  AND team.status = 'active'`, invitationID, email)
	return scanInvitationWithTeamName(row)
}

func (r *Repository) AcceptInvitationByIDAndCreateMember(ctx context.Context, invitationID, teamID, userID uuid.UUID, role, status string) (Invitation, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Invitation{}, fmt.Errorf("begin invitation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row := tx.QueryRow(ctx, `
		UPDATE team_invitations AS invitation
		SET status = 'accepted', accepted_at = now(), updated_at = now()
		FROM teams AS team
		WHERE invitation.id = $1
		  AND invitation.status = 'pending'
		  AND invitation.expires_at > now()
		  AND team.id = invitation.team_id
		  AND team.status = 'active'
		RETURNING invitation.id, invitation.team_id, team.name, invitation.email, invitation.role, invitation.status,
		          invitation.invited_by, invitation.expires_at, invitation.accepted_at, invitation.declined_at,
		          invitation.created_at, invitation.updated_at`, invitationID)
	invitation, err := scanInvitationWithTeamName(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invitation{}, ErrInvitationNotAccepted
		}
		return Invitation{}, fmt.Errorf("accept team invitation by id: %w", err)
	}
	if _, err := r.queries.WithTx(tx).CreateTeamMember(ctx, dbsqlc.CreateTeamMemberParams{TeamID: teamID, UserID: userID, Role: role, Status: status}); err != nil {
		if isUniqueViolation(err) {
			return Invitation{}, ErrTeamMemberAlreadyExists
		}
		return Invitation{}, fmt.Errorf("create team member: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Invitation{}, fmt.Errorf("commit invitation acceptance: %w", err)
	}
	return invitation, nil
}

func (r *Repository) DeclineInvitationByIDForEmail(ctx context.Context, invitationID uuid.UUID, email string) (Invitation, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE team_invitations AS invitation
		SET status = 'declined', declined_at = now(), updated_at = now()
		FROM teams AS team
		WHERE invitation.id = $1
		  AND lower(invitation.email) = lower($2)
		  AND invitation.status = 'pending'
		  AND invitation.expires_at > now()
		  AND team.id = invitation.team_id
		  AND team.status = 'active'
		RETURNING invitation.id, invitation.team_id, team.name, invitation.email, invitation.role, invitation.status,
		          invitation.invited_by, invitation.expires_at, invitation.accepted_at, invitation.declined_at,
		          invitation.created_at, invitation.updated_at`, invitationID, email)
	return scanInvitationWithTeamName(row)
}

func (r *Repository) CreateInvitation(ctx context.Context, teamID uuid.UUID, email, role, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (Invitation, error) {
	row, err := r.queries.CreateTeamInvitation(ctx, dbsqlc.CreateTeamInvitationParams{
		TeamID: teamID, Email: email, Role: role, TokenHash: tokenHash,
		InvitedBy: &invitedBy, ExpiresAt: pgconv.TimestamptzFromTime(expiresAt),
	})
	if err != nil {
		return Invitation{}, fmt.Errorf("create team invitation: %w", err)
	}
	return invitationFromSQLC(row), nil
}

func (r *Repository) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (Invitation, error) {
	row, err := r.queries.GetTeamInvitationByTokenHash(ctx, dbsqlc.GetTeamInvitationByTokenHashParams{TokenHash: tokenHash})
	if err != nil {
		return Invitation{}, fmt.Errorf("get team invitation by token hash: %w", err)
	}
	return invitationFromSQLC(row), nil
}

func (r *Repository) AcceptInvitation(ctx context.Context, tokenHash string) (Invitation, error) {
	row, err := r.queries.AcceptTeamInvitation(ctx, dbsqlc.AcceptTeamInvitationParams{TokenHash: tokenHash})
	if err != nil {
		return Invitation{}, fmt.Errorf("accept team invitation: %w", err)
	}
	return invitationFromSQLC(row), nil
}

func (r *Repository) AcceptInvitationAndCreateMember(ctx context.Context, tokenHash string, teamID, userID uuid.UUID, role, status string) (Invitation, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Invitation{}, fmt.Errorf("begin invitation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.AcceptTeamInvitation(ctx, dbsqlc.AcceptTeamInvitationParams{TokenHash: tokenHash})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invitation{}, ErrInvitationNotAccepted
		}
		return Invitation{}, fmt.Errorf("accept team invitation: %w", err)
	}
	if _, err := q.CreateTeamMember(ctx, dbsqlc.CreateTeamMemberParams{TeamID: teamID, UserID: userID, Role: role, Status: status}); err != nil {
		if isUniqueViolation(err) {
			return Invitation{}, ErrTeamMemberAlreadyExists
		}
		return Invitation{}, fmt.Errorf("create team member: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Invitation{}, fmt.Errorf("commit invitation acceptance: %w", err)
	}
	return invitationFromSQLC(row), nil
}

func (r *Repository) DeclineInvitation(ctx context.Context, tokenHash string) (Invitation, error) {
	row, err := r.queries.DeclineTeamInvitation(ctx, dbsqlc.DeclineTeamInvitationParams{TokenHash: tokenHash})
	if err != nil {
		return Invitation{}, fmt.Errorf("decline team invitation: %w", err)
	}
	return invitationFromSQLC(row), nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, teamID, userID uuid.UUID, role string) (Member, error) {
	row, err := r.queries.UpdateTeamMemberRole(ctx, dbsqlc.UpdateTeamMemberRoleParams{TeamID: teamID, UserID: userID, Role: role})
	if err != nil {
		return Member{}, fmt.Errorf("update team member role: %w", err)
	}
	return memberFromSQLC(row), nil
}

func (r *Repository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := r.queries.RemoveTeamMember(ctx, dbsqlc.RemoveTeamMemberParams{TeamID: teamID, UserID: userID}); err != nil {
		return fmt.Errorf("remove team member: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func teamFromSQLC(row dbsqlc.Team) Team {
	return Team{
		ID: row.ID.String(), Name: row.Name, MarketCode: row.MarketCode,
		Phone: row.Phone, Address: row.Address, Website: row.Website,
		Status: row.Status, CreatedBy: stringPointer(row.CreatedBy),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func stringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	text := value.String()
	return &text
}

func invitationFromSQLC(row dbsqlc.TeamInvitation) Invitation {
	var invitedBy *string
	if row.InvitedBy != nil {
		value := row.InvitedBy.String()
		invitedBy = &value
	}
	var acceptedAt, declinedAt *time.Time
	if row.AcceptedAt.Valid {
		acceptedAt = &row.AcceptedAt.Time
	}
	if row.DeclinedAt.Valid {
		declinedAt = &row.DeclinedAt.Time
	}
	return Invitation{
		ID: row.ID.String(), TeamID: row.TeamID.String(), Email: row.Email,
		Role: row.Role, Status: row.Status, InvitedBy: invitedBy, ExpiresAt: row.ExpiresAt.Time,
		AcceptedAt: acceptedAt, DeclinedAt: declinedAt, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

type invitationScanner interface {
	Scan(dest ...any) error
}

func scanInvitationWithTeamName(row invitationScanner) (Invitation, error) {
	var (
		id         uuid.UUID
		teamID     uuid.UUID
		invitedBy  *uuid.UUID
		expiresAt  pgtype.Timestamptz
		acceptedAt pgtype.Timestamptz
		declinedAt pgtype.Timestamptz
		createdAt  pgtype.Timestamptz
		updatedAt  pgtype.Timestamptz
		invitation Invitation
	)
	if err := row.Scan(
		&id, &teamID, &invitation.TeamName, &invitation.Email, &invitation.Role, &invitation.Status,
		&invitedBy, &expiresAt, &acceptedAt, &declinedAt, &createdAt, &updatedAt,
	); err != nil {
		return Invitation{}, err
	}
	invitation.ID = id.String()
	invitation.TeamID = teamID.String()
	invitation.InvitedBy = stringPointer(invitedBy)
	invitation.ExpiresAt = pgconv.TimestamptzToTime(expiresAt)
	invitation.AcceptedAt = pgconv.TimestamptzToTimePtr(acceptedAt)
	invitation.DeclinedAt = pgconv.TimestamptzToTimePtr(declinedAt)
	invitation.CreatedAt = pgconv.TimestamptzToTime(createdAt)
	invitation.UpdatedAt = pgconv.TimestamptzToTime(updatedAt)
	return invitation, nil
}

func memberFromSQLC(row dbsqlc.TeamMember) Member {
	return Member{TeamID: row.TeamID.String(), UserID: row.UserID.String(), Role: row.Role, Status: row.Status, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
