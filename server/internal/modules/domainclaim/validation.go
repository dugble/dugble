package domainclaim

import (
	"regexp"
	"strings"

	"github.com/google/uuid"

	platformemail "github.com/dugble/dugble/server/internal/providers/aws/ses"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const maxDomainLength = 253

var domainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

func validateRequest(req Request) (string, string, configuration, error) {
	name := normalizeDomain(req.Name)
	region := strings.ToLower(strings.TrimSpace(req.Region))
	if region == "" {
		return "", "", configuration{}, apperrors.NewBadRequest("Sender domain region is required")
	}
	if name == "" {
		return "", "", configuration{}, apperrors.NewBadRequest("Sender domain is required")
	}
	if len(name) > maxDomainLength || !domainPattern.MatchString(name) {
		return "", "", configuration{}, apperrors.NewBadRequest("Sender domain must be a valid domain name")
	}
	if _, ok := platformemail.NormalizeSESRegion(region); !ok {
		return "", "", configuration{}, apperrors.NewBadRequest("Sender domain region is not supported")
	}

	cfg := configuration{TLS: "opportunistic", CustomReturnPath: "send"}
	if strings.TrimSpace(req.TLS) != "" {
		cfg.TLS = strings.ToLower(strings.TrimSpace(req.TLS))
	}
	if cfg.TLS != "opportunistic" && cfg.TLS != "enforced" {
		return "", "", configuration{}, apperrors.NewBadRequest("TLS must be opportunistic or enforced")
	}
	return name, region, cfg, nil
}

func normalizeDomain(value string) string {
	name := strings.TrimSpace(strings.ToLower(value))
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimSuffix(name, ".")
	if before, _, ok := strings.Cut(name, "/"); ok {
		name = before
	}
	return name
}

func parseDomainID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Sender domain id must be a valid UUID")
	}
	return id, nil
}
