package email

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/dugble/dugble/server/internal/authn"
	platformemail "github.com/dugble/dugble/server/internal/providers/aws/ses"
	apperrors "github.com/dugble/dugble/server/pkg/errors"
)

type sandboxRecipientStub struct {
	email    string
	verified bool
	err      error
}

func (stub sandboxRecipientStub) ResolveSandboxRecipientForToken(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
	return stub.email, stub.verified, stub.err
}

func TestAuthorizeSandboxSenderForTeamToken(t *testing.T) {
	t.Parallel()

	teamID, tokenID := uuid.New(), uuid.New()
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		Kind: authn.PrincipalTeamToken, TeamID: &teamID, TokenID: &tokenID,
	})
	service := &Service{
		config:            ServiceConfig{DefaultProvider: "aws_ses", DefaultRegion: "eu-north-1"},
		sandboxRecipients: sandboxRecipientStub{email: "Owner@Example.com", verified: true},
	}
	message := validatedSend{
		MessageType: MessageTypeTransactional,
		FromEmail:   platformemail.SandboxFromEmail,
		To:          []EmailAddress{{Email: "owner@example.com"}},
		CC:          []EmailAddress{{Email: "OWNER@example.com"}},
	}

	if err := service.authorizeSender(ctx, teamID, &message); err != nil {
		t.Fatalf("authorizeSender() error = %v", err)
	}
	if message.SenderDomainID != nil {
		t.Fatalf("SenderDomainID = %v, want nil", message.SenderDomainID)
	}
	if message.Provider != "aws_ses" || message.ProviderRegion != "eu-north-1" {
		t.Fatalf("provider route = %s/%s", message.Provider, message.ProviderRegion)
	}
	wantRoute := platformemail.SandboxDeliveryRoute()
	if message.DeliveryRoute != wantRoute {
		t.Fatalf("DeliveryRoute = %#v, want %#v", message.DeliveryRoute, wantRoute)
	}
}

func TestAuthorizeSandboxSenderRejectsAnyOtherRecipient(t *testing.T) {
	t.Parallel()

	teamID, tokenID := uuid.New(), uuid.New()
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		Kind: authn.PrincipalTeamToken, TeamID: &teamID, TokenID: &tokenID,
	})
	service := &Service{
		config:            ServiceConfig{DefaultProvider: "aws_ses", DefaultRegion: "us-east-1"},
		sandboxRecipients: sandboxRecipientStub{email: "owner@example.com", verified: true},
	}
	message := validatedSend{
		MessageType: MessageTypeTransactional,
		FromEmail:   platformemail.SandboxFromEmail,
		To:          []EmailAddress{{Email: "owner@example.com"}},
		BCC:         []EmailAddress{{Email: "other@example.com"}},
	}

	err := service.authorizeSender(ctx, teamID, &message)
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("authorizeSender() error = %v, want AppError", err)
	}
	if appErr.Status != http.StatusForbidden || appErr.Code != "SANDBOX_RECIPIENT_RESTRICTED" {
		t.Fatalf("sandbox error = %#v", appErr)
	}
}

func TestAuthorizeSandboxSenderUsesVerifiedSessionEmail(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		Kind: authn.PrincipalUserSession, UserID: uuid.New(),
		Email: "Developer@Example.com", EmailVerified: true,
	})
	service := &Service{config: ServiceConfig{DefaultProvider: "aws_ses", DefaultRegion: "eu-north-1"}}
	message := validatedSend{
		MessageType: MessageTypeTransactional,
		FromEmail:   platformemail.SandboxFromEmail,
		To:          []EmailAddress{{Email: "developer@example.com"}},
	}

	if err := service.authorizeSender(ctx, teamID, &message); err != nil {
		t.Fatalf("authorizeSender() error = %v", err)
	}
}

func TestAuthorizeSandboxSenderRejectsOtherDugbleMailbox(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		Kind: authn.PrincipalUserSession, UserID: uuid.New(),
		Email: "developer@example.com", EmailVerified: true,
	})
	service := &Service{config: ServiceConfig{DefaultProvider: "aws_ses", DefaultRegion: "eu-north-1"}}
	message := validatedSend{
		MessageType: MessageTypeTransactional,
		FromEmail:   "billing@dugble.me",
		To:          []EmailAddress{{Email: "developer@example.com"}},
	}

	err := service.authorizeSender(ctx, teamID, &message)
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Status != http.StatusForbidden {
		t.Fatalf("authorizeSender() error = %v, want forbidden", err)
	}
}

func TestAuthorizeSandboxSenderRejectsMarketing(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		Kind: authn.PrincipalUserSession, UserID: uuid.New(),
		Email: "developer@example.com", EmailVerified: true,
	})
	service := &Service{config: ServiceConfig{DefaultProvider: "aws_ses", DefaultRegion: "eu-north-1"}}
	message := validatedSend{
		MessageType: MessageTypeMarketing,
		FromEmail:   platformemail.SandboxFromEmail,
		To:          []EmailAddress{{Email: "developer@example.com"}},
	}

	err := service.authorizeSender(ctx, teamID, &message)
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Status != http.StatusForbidden {
		t.Fatalf("authorizeSender() error = %v, want forbidden", err)
	}
}
