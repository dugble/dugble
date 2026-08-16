package team

import (
	"net/mail"
	"strconv"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const (
	defaultTeamListPage  = 1
	defaultTeamListLimit = 20
	maxTeamListLimit     = 100
)

func validateTeamName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", apperrors.NewBadRequest("Team name is required")
	}
	return name, nil
}

func validateMarketCode(value string) (string, error) {
	marketCode := strings.ToUpper(strings.TrimSpace(value))
	if marketCode == "" {
		return "", apperrors.NewBadRequest("Billing market is required")
	}
	return marketCode, nil
}

func validateRequiredTeamField(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewBadRequest(field + " is required")
	}
	return value, nil
}

func normalizeOptionalTeamField(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func parseListOptions(pageValue, limitValue, searchValue, statusValue string) (ListOptions, error) {
	options := ListOptions{
		Page:   defaultTeamListPage,
		Limit:  defaultTeamListLimit,
		Search: strings.TrimSpace(searchValue),
		Status: TeamStatusActive,
	}

	if value := strings.TrimSpace(pageValue); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			return ListOptions{}, apperrors.NewBadRequest("Page must be a positive integer")
		}
		options.Page = page
	}

	if value := strings.TrimSpace(limitValue); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > maxTeamListLimit {
			return ListOptions{}, apperrors.NewBadRequest("Limit must be between 1 and 100")
		}
		options.Limit = limit
	}

	if value := strings.ToLower(strings.TrimSpace(statusValue)); value != "" {
		if value != TeamStatusActive && value != TeamStatusDisabled {
			return ListOptions{}, apperrors.NewBadRequest("Status must be active or disabled")
		}
		options.Status = value
	}

	return options, nil
}

func validateTeamID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Team id must be a valid UUID")
	}
	return id, nil
}

func validateMemberID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("User id must be a valid UUID")
	}
	return id, nil
}

func validateInvitationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invitation id must be a valid UUID")
	}
	return id, nil
}

func validateMemberRole(value string) (string, error) {
	role := strings.TrimSpace(value)
	if role != RoleAdmin && role != RoleMember {
		return "", apperrors.NewBadRequest("Role must be admin or member")
	}
	return role, nil
}

func normalizeInvitationEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", apperrors.NewBadRequest("A valid invitee email is required")
	}
	return email, nil
}

func validateInvitationRole(value string) (string, error) {
	role := strings.TrimSpace(value)
	if role == "" {
		return RoleMember, nil
	}
	return validateMemberRole(role)
}

func normalizeInvitationToken(value string) string { return strings.TrimSpace(value) }

func validateInvitationToken(value string) (string, error) {
	token := normalizeInvitationToken(value)
	if token == "" {
		return "", apperrors.NewBadRequest("Invitation token is required")
	}
	return token, nil
}
