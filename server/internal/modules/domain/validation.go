package domain

import (
	"regexp"
	"strings"

	"github.com/google/uuid"

	platformemail "github.com/dugble/dugble/server/internal/providers/aws/ses"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

const maxDomainLength = 253

var domainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

func validateCreate(req CreateRequest) (string, string, DomainConfiguration, error) {
	return validateDomainConfiguration(req.Name, req.Region, req.TLS)
}

func validateDomainConfiguration(name, region, tls string) (string, string, DomainConfiguration, error) {
	domainName := normalizeDomain(name)
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		return "", "", DomainConfiguration{}, apperrors.NewBadRequest("Sender domain region is required")
	}
	if domainName == "" {
		return "", "", DomainConfiguration{}, apperrors.NewBadRequest("Sender domain is required")
	}
	if len(domainName) > maxDomainLength || !domainPattern.MatchString(domainName) {
		return "", "", DomainConfiguration{}, apperrors.NewBadRequest("Sender domain must be a valid domain name")
	}
	if err := validateRegion(region); err != nil {
		return "", "", DomainConfiguration{}, err
	}

	configuration := DomainConfiguration{
		TLS:              DefaultTLSMode,
		CustomReturnPath: DefaultCustomReturnPath,
	}
	if strings.TrimSpace(tls) != "" {
		configuration.TLS = strings.ToLower(strings.TrimSpace(tls))
	}
	if configuration.TLS != "opportunistic" && configuration.TLS != "enforced" {
		return "", "", DomainConfiguration{}, apperrors.NewBadRequest("TLS must be opportunistic or enforced")
	}
	return domainName, region, configuration, nil
}

func validateUpdate(current SenderDomain, req UpdateRequest) (DomainConfiguration, error) {
	configuration := DomainConfiguration{
		TLS:              current.TLS,
		CustomReturnPath: current.CustomReturnPath,
	}
	if req.TLS != nil {
		configuration.TLS = strings.ToLower(strings.TrimSpace(*req.TLS))
		if configuration.TLS != "opportunistic" && configuration.TLS != "enforced" {
			return DomainConfiguration{}, apperrors.NewBadRequest("TLS must be opportunistic or enforced")
		}
	}
	return configuration, nil
}

func validateRegion(region string) error {
	if _, ok := platformemail.NormalizeSESRegion(region); !ok {
		return apperrors.NewBadRequest("Sender domain region is not supported")
	}
	return nil
}

func normalizeDomain(value string) string {
	domainName := strings.TrimSpace(strings.ToLower(value))
	domainName = strings.TrimPrefix(domainName, "http://")
	domainName = strings.TrimPrefix(domainName, "https://")
	domainName = strings.TrimSuffix(domainName, ".")
	if before, _, ok := strings.Cut(domainName, "/"); ok {
		domainName = before
	}
	return domainName
}

func parseDomainID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Sender domain id must be a valid UUID")
	}
	return id, nil
}
