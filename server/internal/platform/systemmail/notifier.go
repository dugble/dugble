package systemmail

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	sentrymonitoring "github.com/dugble/dugble/server/internal/adapters/monitoring/sentry"
	"github.com/jackc/pgx/v5"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

const (
	verifyEmailTemplate           = "verify_email.html"
	forgotPasswordTemplate        = "forgot_password.html"
	passwordChangedTemplate       = "password_changed.html"
	emailChangedTemplate          = "email_changed.html"
	mfaEnabledTemplate            = "mfa_enabled.html"
	mfaDisabledTemplate           = "mfa_disabled.html"
	recoveryCodeUsedTemplate      = "recovery_code_used.html"
	mfaLoginFailedTemplate        = "mfa_login_failed.html"
	newLoginTemplate              = "new_login.html"
	accountDeletedTemplate        = "account_deleted.html"
	teamMemberRemovedTemplate     = "team_member_removed.html"
	teamMemberRoleChangedTemplate = "team_member_role_changed.html"
	teamTokenCreatedTemplate      = "team_token_created.html"
	teamTokenRevokedTemplate      = "team_token_revoked.html"
	teamInvitationTemplate        = "team_invitation.html"
	subscriptionPastDueTemplate   = "subscription_past_due.html"
	walletBalanceAlertTemplate    = "wallet_balance_alert.html"
	walletTopUpSucceededTemplate  = "wallet_top_up_succeeded.html"
	walletTopUpFailedTemplate     = "wallet_top_up_failed.html"
)

type EmailService struct {
	sender      EmailSender
	renderer    *Renderer
	frontendURL string
	fromEmail   string
}

func NewEmailService(sender EmailSender, renderer *Renderer, frontendURL, fromEmail string) *EmailService {
	return &EmailService{
		sender:      sender,
		renderer:    renderer,
		frontendURL: strings.TrimRight(strings.TrimSpace(frontendURL), "/"),
		fromEmail:   strings.TrimSpace(fromEmail),
	}
}

type SendTemplateEmailInput struct {
	To           string
	Subject      string
	TemplateName string
	Data         any
}

func (s *EmailService) SendTemplateEmail(ctx context.Context, input SendTemplateEmailInput) error {
	return s.sendTemplateEmail(ctx, nil, input)
}

func (s *EmailService) sendTemplateEmail(ctx context.Context, tx pgx.Tx, input SendTemplateEmailInput) error {
	if s == nil {
		return errors.New("email service is not configured")
	}
	if s.renderer == nil {
		return errors.New("email renderer is not configured")
	}
	if s.sender == nil {
		return errors.New("email sender is not configured")
	}
	body, err := s.renderer.Render(input.TemplateName, input.Data)
	if err != nil {
		return err
	}
	message := platformemail.Message{
		From:    platformemail.Address{Email: s.fromEmail, Name: "Dugble"},
		To:      []platformemail.Address{{Email: input.To}},
		Subject: input.Subject,
		HTML:    body,
	}
	if tx == nil {
		_, err = s.sender.Send(ctx, message)
	} else if sender, ok := s.sender.(TransactionalEmailSender); ok {
		_, err = sender.SendTx(ctx, tx, message)
	} else {
		return errors.New("transactional email sender is not configured")
	}
	if err != nil {
		sentrymonitoring.Warn("failed to send email", "error", err, "to", input.To, "template", input.TemplateName)
		return fmt.Errorf("send email %s to %s: %w", input.TemplateName, input.To, err)
	}
	return nil
}

func (s *EmailService) SendSubscriptionPastDue(ctx context.Context, tx pgx.Tx, input SendSubscriptionPastDueInput) error {
	data := map[string]string{
		"Name": displayName(input.Name), "PreviewText": "Your Dugble subscription payment failed.",
		"Team": displayName(input.TeamName), "Plan": displayValue(input.PlanCode),
		"Amount": formatMoney(input.Currency, input.AmountUnits), "Balance": formatMoney(input.Currency, input.BalanceUnits),
		"BillingURL": s.frontendURL + "/settings/billing",
	}
	return s.sendTemplateEmail(ctx, tx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Action required: your Dugble subscription is past due", TemplateName: subscriptionPastDueTemplate, Data: data})
}

func formatMoney(currency string, units int64) string {
	negative := units < 0
	if negative {
		units = -units
	}
	value := strconv.FormatInt(units/100, 10) + "." + fmt.Sprintf("%02d", units%100)
	if negative {
		value = "-" + value
	}
	return strings.TrimSpace(currency) + " " + value
}

func (s *EmailService) SendWalletBalanceAlert(ctx context.Context, input SendWalletBalanceAlertInput) error {
	exhausted := strings.EqualFold(strings.TrimSpace(input.Level), "exhausted")
	subject := "Your Dugble wallet balance is low"
	preview := "Add funds to your Dugble wallet to avoid interrupted service."
	message := "Your wallet is running low. Add funds soon to keep your messages sending."
	if exhausted {
		subject = "Action required: your Dugble wallet is empty"
		preview = "Your Dugble wallet is empty."
		message = "Your wallet is empty. Add funds now to restore paid message sending."
	}
	data := map[string]string{
		"Name": displayName(input.Name), "PreviewText": preview,
		"Team": displayName(input.TeamName), "Balance": formatMoney(input.Currency, input.BalanceUnits),
		"Message": message, "BillingURL": s.frontendURL + "/settings/billing",
	}
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: subject, TemplateName: walletBalanceAlertTemplate, Data: data})
}

func (s *EmailService) SendWalletTopUpResult(ctx context.Context, input SendWalletTopUpResultInput) error {
	succeeded := strings.EqualFold(strings.TrimSpace(input.Status), "paid")
	subject := "Your Dugble wallet top-up failed"
	templateName := walletTopUpFailedTemplate
	preview := "Your Dugble wallet top-up was not completed."
	if succeeded {
		subject = "Your Dugble wallet top-up succeeded"
		templateName = walletTopUpSucceededTemplate
		preview = "Funds were added to your Dugble wallet."
	}
	data := map[string]string{
		"Name": displayName(input.Name), "PreviewText": preview,
		"Team": displayName(input.TeamName), "Amount": formatMoney(input.Currency, input.AmountUnits),
		"Reference": displayValue(input.ClientReference), "BillingURL": s.frontendURL + "/dashboard/billing/transactions",
	}
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: subject, TemplateName: templateName, Data: data})
}

func (s *EmailService) SendEmailVerification(ctx context.Context, input SendEmailVerificationInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Verify your Dugble email address", TemplateName: verifyEmailTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Verify your dugble email address.", "VerificationURL": s.verificationURL(input.ToEmail, input.Token)}})
}

func (s *EmailService) SendPasswordReset(ctx context.Context, input SendPasswordResetInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Reset your Dugble password", TemplateName: forgotPasswordTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Reset your dugble password.", "ResetURL": s.passwordResetURL(input.ToEmail, input.Token)}})
}

func (s *EmailService) SendPasswordChanged(ctx context.Context, input SendPasswordChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Your Dugble password was changed", TemplateName: passwordChangedTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Your dugble password was changed."}})
}

func (s *EmailService) SendEmailChanged(ctx context.Context, input SendEmailChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Your Dugble email address was changed", TemplateName: emailChangedTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Your Dugble email address was changed.", "Email": input.Email}})
}

func (s *EmailService) SendMFAEnabled(ctx context.Context, input SendSecurityEventInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Authenticator MFA was enabled", TemplateName: mfaEnabledTemplate, Data: securityEventData(input, "Authenticator MFA was enabled on your Dugble account.")})
}

func (s *EmailService) SendMFADisabled(ctx context.Context, input SendSecurityEventInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Authenticator MFA was disabled", TemplateName: mfaDisabledTemplate, Data: securityEventData(input, "Authenticator MFA was disabled on your Dugble account.")})
}

func (s *EmailService) SendRecoveryCodeUsed(ctx context.Context, input SendSecurityEventInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "A recovery code was used", TemplateName: recoveryCodeUsedTemplate, Data: securityEventData(input, "A recovery code was used on your Dugble account.")})
}

func (s *EmailService) SendMFALoginFailed(ctx context.Context, input SendSecurityEventInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "A failed MFA sign-in attempt was detected", TemplateName: mfaLoginFailedTemplate, Data: securityEventData(input, "A failed MFA sign-in attempt was detected on your Dugble account.")})
}

func (s *EmailService) SendAccountDeleted(ctx context.Context, input SendSecurityEventInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Your Dugble account was deleted", TemplateName: accountDeletedTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Your Dugble account was deleted."}})
}

func (s *EmailService) SendNewLogin(ctx context.Context, input SendNewLoginInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "New sign-in to your Dugble account", TemplateName: newLoginTemplate, Data: map[string]string{
		"Name": displayName(input.Name), "PreviewText": "A new sign-in was detected on your Dugble account.",
		"Method": displayValue(input.Method), "IPAddress": displayValue(input.IPAddress), "UserAgent": displayValue(input.UserAgent),
	}})
}

func securityEventData(input SendSecurityEventInput, preview string) map[string]string {
	return map[string]string{"Name": displayName(input.Name), "PreviewText": preview}
}

func (s *EmailService) SendTeamMemberRemoved(ctx context.Context, input SendTeamMemberChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "You were removed from a Dugble team", TemplateName: teamMemberRemovedTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Your Dugble team membership changed.", "Team": input.Team}})
}

func (s *EmailService) SendTeamMemberRoleChanged(ctx context.Context, input SendTeamMemberChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "Your Dugble team role changed", TemplateName: teamMemberRoleChangedTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "Your Dugble team role changed.", "Team": input.Team, "Role": displayRole(input.Role)}})
}

func (s *EmailService) SendTeamTokenCreated(ctx context.Context, input SendTeamTokenChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "A Dugble team token was created", TemplateName: teamTokenCreatedTemplate, Data: teamTokenEventData(input, "A team API token was created.")})
}

func (s *EmailService) SendTeamTokenRevoked(ctx context.Context, input SendTeamTokenChangedInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "A Dugble team token was revoked", TemplateName: teamTokenRevokedTemplate, Data: teamTokenEventData(input, "A team API token was revoked.")})
}

func teamTokenEventData(input SendTeamTokenChangedInput, preview string) map[string]string {
	return map[string]string{"Name": displayName(input.Name), "PreviewText": preview, "TeamID": input.TeamID, "TokenName": input.TokenName, "TokenPrefix": input.TokenPrefix}
}

func (s *EmailService) SendTeamInvitation(ctx context.Context, input SendTeamInvitationInput) error {
	return s.SendTemplateEmail(ctx, SendTemplateEmailInput{To: input.ToEmail, Subject: "You were invited to join a dugble team", TemplateName: teamInvitationTemplate, Data: map[string]string{"Name": displayName(input.Name), "PreviewText": "You were invited to join a dugble team.", "TeamName": displayName(input.TeamName), "InviterName": displayName(input.InviterName), "Role": displayRole(input.Role), "InvitationURL": s.teamInvitationURL(input.Token)}})
}

func (s *EmailService) verificationURL(email, token string) string {
	query := url.Values{}
	query.Set("email", email)
	query.Set("token", token)
	return s.frontendURL + "/verify-email?" + query.Encode()
}

func (s *EmailService) passwordResetURL(email, token string) string {
	query := url.Values{}
	query.Set("email", email)
	query.Set("token", token)
	return s.frontendURL + "/reset-password?" + query.Encode()
}

func (s *EmailService) teamInvitationURL(token string) string {
	query := url.Values{}
	query.Set("token", token)
	return s.frontendURL + "/team-invitations?" + query.Encode()
}

func displayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "there"
	}
	return name
}

func displayRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "member"
	}
	return role
}

func displayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
