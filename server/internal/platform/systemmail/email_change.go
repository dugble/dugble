package systemmail

import (
	"context"
	"net/url"
)

func (s *EmailService) SendEmailChangeVerification(ctx context.Context, input SendEmailChangeVerificationInput) error {
	query := url.Values{}
	query.Set("token", input.Token)
	verificationURL := s.frontendURL + "/verify-email-change?" + query.Encode()

	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{
		To:           input.ToEmail,
		Subject:      "Verify your new Dugble email address",
		TemplateName: verifyEmailTemplate,
		Data: map[string]string{
			"Name":            displayName(input.Name),
			"PreviewText":     "Verify your new Dugble email address.",
			"VerificationURL": verificationURL,
		},
	})
}
