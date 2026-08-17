package senderid

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxNameCharacters = 11

var (
	nameCharactersPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	nameHasLetterPattern  = regexp.MustCompile(`[A-Za-z]`)
)

func NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

func NormalizeCountryCode(countryCode string) string {
	return strings.ToUpper(strings.TrimSpace(countryCode))
}

func ValidateName(name string) error {
	name = NormalizeName(name)
	switch {
	case name == "":
		return fmt.Errorf("sender ID name is required")
	case utf8.RuneCountInString(name) > MaxNameCharacters:
		return fmt.Errorf("sender ID name must be at most %d characters", MaxNameCharacters)
	case !nameCharactersPattern.MatchString(name):
		return fmt.Errorf("sender ID name may contain only letters and numbers")
	case !nameHasLetterPattern.MatchString(name):
		return fmt.Errorf("sender ID name must contain at least one letter")
	default:
		return nil
	}
}

func ValidateCountryCode(countryCode string) error {
	countryCode = NormalizeCountryCode(countryCode)
	if len(countryCode) != 2 {
		return fmt.Errorf("sender ID country code must be a 2-letter ISO country code")
	}
	for _, r := range countryCode {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("sender ID country code must contain only letters")
		}
	}
	return nil
}

func (request CreateRequest) Normalize() CreateRequest {
	request.Name = NormalizeName(request.Name)
	request.CountryCode = NormalizeCountryCode(request.CountryCode)
	request.Purpose = strings.TrimSpace(request.Purpose)
	return request
}

func (request CreateRequest) Validate() error {
	request = request.Normalize()
	if err := ValidateName(request.Name); err != nil {
		return err
	}
	if err := ValidateCountryCode(request.CountryCode); err != nil {
		return err
	}
	if request.Purpose == "" {
		return fmt.Errorf("sender ID purpose is required")
	}
	return nil
}
