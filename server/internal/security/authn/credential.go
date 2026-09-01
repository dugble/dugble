package authn

import "strings"

// Credential contains the request credentials supported by the application.
// A resolver defines precedence when more than one credential is present.
type Credential struct {
	BearerToken  string
	SessionToken string
}

// Normalize trims transport whitespace without otherwise changing credentials.
func (credential Credential) Normalize() Credential {
	credential.BearerToken = strings.TrimSpace(credential.BearerToken)
	credential.SessionToken = strings.TrimSpace(credential.SessionToken)
	return credential
}

// Empty reports whether no supported credential was supplied.
func (credential Credential) Empty() bool {
	credential = credential.Normalize()
	return credential.BearerToken == "" && credential.SessionToken == ""
}

// SessionCookieName is the cookie transport used for first-party sessions.
const SessionCookieName = "dugble_session"
