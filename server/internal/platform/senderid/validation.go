package senderid

import (
	"strings"

	relaysenderid "github.com/dugble/dugble/server/internal/relay/senderid"
)

const MaxNameCharacters = relaysenderid.MaxNameCharacters

func NormalizeName(name string) string {
	return relaysenderid.NormalizeName(name)
}

func ValidateName(name string) error {
	return relaysenderid.ValidateName(name)
}

func (request CreateRequest) Normalize() CreateRequest {
	request.SenderID = NormalizeName(request.SenderID)
	request.Purpose = strings.TrimSpace(request.Purpose)
	return request
}

func (request CreateRequest) Validate() error {
	return ValidateName(request.SenderID)
}
