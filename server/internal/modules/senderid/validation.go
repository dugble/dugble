package senderid

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxNameCharacters = 11

var (
	nameCharactersPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	nameHasLetterPattern  = regexp.MustCompile(`[A-Za-z]`)
)

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func validateName(name string) error {
	name = normalizeName(name)
	switch {
	case name == "":
		return fmt.Errorf("sender ID name is required")
	case utf8.RuneCountInString(name) > maxNameCharacters:
		return fmt.Errorf("sender ID name must be at most %d characters", maxNameCharacters)
	case !nameCharactersPattern.MatchString(name):
		return fmt.Errorf("sender ID name may contain only letters and numbers")
	case !nameHasLetterPattern.MatchString(name):
		return fmt.Errorf("sender ID name must contain at least one letter")
	default:
		return nil
	}
}
