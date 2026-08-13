package authn

import "testing"

func TestCredentialNormalizeAndEmpty(t *testing.T) {
	t.Parallel()

	credential := (Credential{BearerToken: "  bearer  ", SessionToken: "  session  "}).Normalize()
	if credential.BearerToken != "bearer" || credential.SessionToken != "session" {
		t.Fatalf("Normalize() = %#v", credential)
	}
	if credential.Empty() {
		t.Fatal("non-empty credential reported empty")
	}
	if !(Credential{BearerToken: " \t ", SessionToken: "\n"}).Empty() {
		t.Fatal("whitespace-only credential reported non-empty")
	}
}
