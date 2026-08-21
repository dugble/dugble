package teamtoken

import (
	"context"
	"strings"
	"time"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"

	"github.com/dugble/dugble/server/internal/authn"
	"github.com/dugble/dugble/server/internal/authz"
	"github.com/dugble/dugble/server/internal/platform/audit"
	notifications "github.com/dugble/dugble/server/internal/platform/systemmail"
	"github.com/dugble/dugble/server/internal/security"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	maxTokenTTL   = 365 * 24 * time.Hour
	maxNameLength = 120
)

var allowedPermissions = map[authz.Permission]struct{}{
	authz.PermissionSenderIDsRead:          {},
	authz.PermissionSenderIDsCreate:        {},
	authz.PermissionSenderIDsDelete:        {},
	authz.PermissionSenderDomainsRead:      {},
	authz.PermissionSenderDomainsCreate:    {},
	authz.PermissionSenderDomainsDelete:    {},
	authz.PermissionSMSRead:                {},
	authz.PermissionSMSSend:                {},
	authz.PermissionEmailRead:              {},
	authz.PermissionEmailSend:              {},
	authz.PermissionVerifyRead:             {},
	authz.PermissionVerifySend:             {},
	authz.PermissionVerifyCheck:            {},
	authz.PermissionVerifyManage:           {},
	authz.PermissionContactsRead:           {},
	authz.PermissionContactsWrite:          {},
	authz.PermissionContactPropertiesRead:  {},
	authz.PermissionContactPropertiesWrite: {},
	authz.PermissionTopicsRead:             {},
	authz.PermissionTopicsWrite:            {},
	authz.PermissionSegmentsRead:           {},
	authz.PermissionSegmentsWrite:          {},
	authz.PermissionSuppressionsRead:       {},
	authz.PermissionSuppressionsWrite:      {},
	authz.PermissionBroadcastsRead:         {},
	authz.PermissionBroadcastsWrite:        {},
	authz.PermissionBroadcastsSend:         {},
	authz.PermissionTemplatesRead:          {},
	authz.PermissionTemplatesWrite:         {},
}

type Service struct {
	repository *Repository
	notifier   AdministrativeNotifier
}

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

type AdministrativeNotifier interface {
	SendTeamTokenCreated(context.Context, notifications.SendTeamTokenChangedInput) error
	SendTeamTokenRevoked(context.Context, notifications.SendTeamTokenChangedInput) error
}

func (s *Service) WithNotifier(notifier AdministrativeNotifier) *Service {
	s.notifier = notifier
	return s
}

func (s *Service) List(ctx context.Context) ([]Token, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionTeamTokensRead)
	if err != nil {
		return nil, err
	}
	tokens, err := s.repository.List(ctx, tenantContext.Scope.TeamID)
	if err != nil {
		return nil, apperrors.NewInternal("Unable to list team tokens", err)
	}
	return tokens, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreatedToken, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionTeamTokensCreate)
	if err != nil {
		return CreatedToken{}, err
	}
	if err := requireOwner(tenantContext); err != nil {
		return CreatedToken{}, err
	}
	name, permissions, expiresAt, err := validateMutation(req.Name, req.Permissions, req.ExpiresAt)
	if err != nil {
		return CreatedToken{}, err
	}
	secret, err := newTeamTokenSecret()
	if err != nil {
		return CreatedToken{}, apperrors.NewInternal("Unable to generate team token", err)
	}
	token, err := s.repository.Create(
		ctx,
		tenantContext.Scope.TeamID,
		name,
		security.HashSessionToken(secret),
		tokenDisplayPrefix(secret),
		permissions,
		tenantContext.Actor.UserID,
		expiresAt,
	)
	if err != nil {
		return CreatedToken{}, apperrors.NewInternal("Unable to create team token", err)
	}
	audit.Record(ctx, tenantContext, audit.Event{Action: "team_token.created", ResourceType: "team_token", ResourceID: token.ID, Metadata: map[string]any{"token_prefix": token.TokenPrefix}})
	s.notify(ctx, tenantContext, token, "created")
	return CreatedToken{Token: token, Secret: secret}, nil
}

func (s *Service) Update(ctx context.Context, tokenID string, req UpdateRequest) (Token, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionTeamTokensUpdate)
	if err != nil {
		return Token{}, err
	}
	if err := requireOwner(tenantContext); err != nil {
		return Token{}, err
	}
	parsedTokenID, err := validateTokenID(tokenID)
	if err != nil {
		return Token{}, err
	}
	name, permissions, expiresAt, err := validateMutation(req.Name, req.Permissions, req.ExpiresAt)
	if err != nil {
		return Token{}, err
	}
	token, err := s.repository.Update(
		ctx,
		parsedTokenID,
		tenantContext.Scope.TeamID,
		name,
		permissions,
		expiresAt,
	)
	if err != nil {
		return Token{}, apperrors.NewNotFound("Team token not found")
	}
	audit.Record(ctx, tenantContext, audit.Event{Action: "team_token.updated", ResourceType: "team_token", ResourceID: token.ID, Metadata: map[string]any{"token_prefix": token.TokenPrefix}})
	return token, nil
}

func (s *Service) Revoke(ctx context.Context, tokenID string) (Token, error) {
	tenantContext, err := requireTenantPermission(ctx, authz.PermissionTeamTokensRevoke)
	if err != nil {
		return Token{}, err
	}
	if err := requireOwner(tenantContext); err != nil {
		return Token{}, err
	}
	parsedTokenID, err := validateTokenID(tokenID)
	if err != nil {
		return Token{}, err
	}
	token, err := s.repository.Revoke(ctx, parsedTokenID, tenantContext.Scope.TeamID)
	if err != nil {
		return Token{}, apperrors.NewNotFound("Team token not found")
	}
	audit.Record(ctx, tenantContext, audit.Event{Action: "team_token.revoked", ResourceType: "team_token", ResourceID: token.ID, Metadata: map[string]any{"token_prefix": token.TokenPrefix}})
	s.notify(ctx, tenantContext, token, "revoked")
	return token, nil
}

func (s *Service) notify(ctx context.Context, access authz.Access, token Token, event string) {
	if s.notifier == nil {
		return
	}
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok || strings.TrimSpace(principal.Email) == "" {
		return
	}
	input := notifications.SendTeamTokenChangedInput{ToEmail: principal.Email, Name: principal.Name, TeamID: access.Scope.TeamID.String(), TokenName: token.Name, TokenPrefix: token.TokenPrefix}
	var err error
	if event == "created" {
		err = s.notifier.SendTeamTokenCreated(ctx, input)
	} else {
		err = s.notifier.SendTeamTokenRevoked(ctx, input)
	}
	if err != nil {
		sentrymonitoring.Warn("failed to send team token notification", "error", err, "event", event, "team_id", access.Scope.TeamID, "token_id", token.ID)
	}
}

func requireTenantPermission(
	ctx context.Context,
	permission authz.Permission,
) (authz.Access, error) {
	tenantContext, decision := authz.ResolveAccess(ctx, permission)
	if !decision.Allowed {
		return authz.Access{}, apperrors.NewForbidden(decision.Reason)
	}
	return tenantContext, nil
}

func requireOwner(tenantContext authz.Access) error {
	if tenantContext.Scope.Role != authz.RoleOwner {
		return apperrors.NewForbidden("Team owner role is required")
	}
	return nil
}

func newTeamTokenSecret() (string, error) {
	token, err := security.NewSessionToken()
	if err != nil {
		return "", err
	}
	return TokenPrefix + token, nil
}

func tokenDisplayPrefix(secret string) string {
	if len(secret) <= 18 {
		return secret
	}
	return secret[:18]
}
