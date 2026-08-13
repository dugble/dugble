package middleware

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/coffeyvidzro/dugble/server/internal/authn"
	"github.com/coffeyvidzro/dugble/server/internal/authz"
	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
	"github.com/coffeyvidzro/dugble/server/pkg/httputil"
)

const defaultTenantParam = "team_id"
const defaultTenantHeader = "X-Team-ID"

type SelectTeamConfig struct {
	Memberships authz.MembershipRepository
	ParamName   string
	HeaderName  string
}

// SelectTeam establishes the team access boundary for an authenticated
// principal. It does not decide whether an operation is permitted.
func SelectTeam(config SelectTeamConfig) echo.MiddlewareFunc {
	paramName := config.ParamName
	if paramName == "" {
		paramName = defaultTenantParam
	}
	headerName := config.HeaderName
	if headerName == "" {
		headerName = defaultTenantHeader
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			principal, ok := authn.PrincipalFromContext(c.Request().Context())
			if !ok {
				return httputil.Error(c, apperrors.NewUnauthorized("Authentication is required"))
			}
			access, err := selectTeamAccess(c, config.Memberships, principal, paramName, headerName)
			if err != nil {
				return httputil.Error(c, err)
			}
			c.SetRequest(c.Request().WithContext(authz.ContextWithAccess(c.Request().Context(), access)))
			return next(c)
		}
	}
}

func selectTeamAccess(c *echo.Context, memberships authz.MembershipRepository, principal authn.Principal, paramName, headerName string) (authz.Access, error) {
	switch principal.Kind {
	case authn.PrincipalUserSession:
		if memberships == nil {
			return authz.Access{}, apperrors.NewInternal("Tenant membership store is not configured", nil)
		}
		teamID, present, err := teamIDFromRequest(c, paramName, headerName)
		if err != nil {
			return authz.Access{}, err
		}
		if !present {
			return authz.Access{}, apperrors.NewBadRequest("Team id is required")
		}
		membership, err := memberships.GetTenantMembership(c.Request().Context(), teamID, principal.UserID)
		if err != nil || !membership.Active() {
			return authz.Access{}, apperrors.NewForbidden("Active team membership is required")
		}
		return authz.Access{
			Actor: authz.Actor{Type: authz.ActorTypeUser, UserID: membership.UserID, SessionID: principal.SessionID},
			Scope: authz.TeamScope{TeamID: membership.TeamID, Role: membership.Role, Status: membership.Status, Scopes: principal.Scopes},
		}, nil
	case authn.PrincipalTeamToken:
		if principal.TeamID == nil || *principal.TeamID == uuid.Nil || principal.TokenID == nil || *principal.TokenID == uuid.Nil {
			return authz.Access{}, apperrors.NewUnauthorized("Team token is invalid")
		}
		requestedTeamID, present, err := teamIDFromRequest(c, paramName, headerName)
		if err != nil {
			return authz.Access{}, err
		}
		if present && requestedTeamID != *principal.TeamID {
			return authz.Access{}, apperrors.NewForbidden("Team token does not match requested team")
		}
		return authz.Access{
			Actor: authz.Actor{Type: authz.ActorTypeTeamToken, TokenID: *principal.TokenID},
			Scope: authz.TeamScope{TeamID: *principal.TeamID, Status: authz.StatusActive, Scopes: principal.Scopes, Permissions: authz.PermissionsImpliedByScopes(principal.Scopes).Permissions()},
		}, nil
	default:
		return authz.Access{}, apperrors.NewUnauthorized("Authenticated principal is not supported")
	}
}

// Authorize rejects requests whose selected access does not grant permission.
func Authorize(permission authz.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			access, decision := authz.ResolveAccess(c.Request().Context(), permission)
			if !decision.Allowed {
				return httputil.Error(c, apperrors.NewForbidden(decision.Reason))
			}
			c.SetRequest(c.Request().WithContext(authz.ContextWithAccess(c.Request().Context(), access)))
			return next(c)
		}
	}
}

// Chain constructs a middleware pipeline once, rather than composing
// middleware while handling each request.
func Chain(middlewares ...echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		for index := len(middlewares) - 1; index >= 0; index-- {
			if middlewares[index] != nil {
				next = middlewares[index](next)
			}
		}
		return next
	}
}

// Tenant is retained for session-only routes while delegating to the explicit
// selection and authorization stages.
type TenantConfig struct {
	Memberships authz.MembershipRepository
	ParamName   string
	HeaderName  string
	Required    authz.Permission
}

func Tenant(config TenantConfig) echo.MiddlewareFunc {
	return Chain(
		SelectTeam(SelectTeamConfig{Memberships: config.Memberships, ParamName: config.ParamName, HeaderName: config.HeaderName}),
		Authorize(config.Required),
	)
}

func teamIDFromRequest(c *echo.Context, paramName, headerName string) (uuid.UUID, bool, error) {
	pathValue := strings.TrimSpace(c.Param(paramName))
	headerValue := strings.TrimSpace(c.Request().Header.Get(headerName))
	if pathValue == "" && headerValue == "" {
		return uuid.Nil, false, nil
	}
	var pathID uuid.UUID
	if pathValue != "" {
		parsed, err := uuid.Parse(pathValue)
		if err != nil {
			return uuid.Nil, false, apperrors.NewBadRequest("Team id must be a valid UUID")
		}
		pathID = parsed
	}
	var headerID uuid.UUID
	if headerValue != "" {
		parsed, err := uuid.Parse(headerValue)
		if err != nil {
			return uuid.Nil, false, apperrors.NewBadRequest("Team id must be a valid UUID")
		}
		headerID = parsed
	}
	if pathValue != "" && headerValue != "" && pathID != headerID {
		return uuid.Nil, false, apperrors.NewBadRequest("Team id in path and header must match")
	}
	if pathValue != "" {
		return pathID, true, nil
	}
	return headerID, true, nil
}
