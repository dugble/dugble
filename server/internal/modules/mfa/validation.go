package mfa

import (
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/coffeyvidzro/dugble/server/internal/security"
	apperrors "github.com/coffeyvidzro/dugble/server/pkg/errors"
)

func normalizeAuthenticationCode(value string) string { return strings.TrimSpace(value) }

func validateRecoveryCode(value string) (string, error) {
	code := normalizeAuthenticationCode(value)
	if code == "" {
		return "", apperrors.NewBadRequest("Recovery code is required")
	}
	return code, nil
}

func validateLoginChallengeToken(value string) (string, error) {
	token := strings.TrimSpace(value)
	if !strings.HasPrefix(token, loginChallengePrefix) {
		return "", pgx.ErrNoRows
	}
	return security.HashSessionToken(token), nil
}
