package systemmail

import (
	"context"

	"github.com/jackc/pgx/v5"

	platformemail "github.com/dugble/dugble/server/internal/platform/awsses"
)

type Recipient struct {
	Name  string
	Email string
}

type SendEmailVerificationInput struct {
	ToEmail string
	Name    string
	Token   string
}

type SendPasswordResetInput struct {
	ToEmail string
	Name    string
	Token   string
}

type SendPasswordChangedInput struct {
	ToEmail string
	Name    string
}

type SendEmailChangedInput struct {
	ToEmail string
	Name    string
	Email   string
}

type SendSecurityEventInput struct {
	ToEmail string
	Name    string
}

type SendNewLoginInput struct {
	ToEmail   string
	Name      string
	IPAddress string
	UserAgent string
	Method    string
}

type SendTeamMemberChangedInput struct {
	ToEmail string
	Name    string
	Team    string
	Role    string
}

type SendTeamTokenChangedInput struct {
	ToEmail     string
	Name        string
	TeamID      string
	TokenName   string
	TokenPrefix string
}

type SendTeamInvitationInput struct {
	ToEmail     string
	Name        string
	TeamName    string
	InviterName string
	Role        string
	Token       string
}

type SendSubscriptionPastDueInput struct {
	ToEmail      string
	Name         string
	TeamName     string
	PlanCode     string
	Currency     string
	AmountUnits  int64
	BalanceUnits int64
}

type SendWalletBalanceAlertInput struct {
	ToEmail      string
	Name         string
	TeamName     string
	Currency     string
	BalanceUnits int64
	Level        string
}

type SendWalletTopUpResultInput struct {
	ToEmail         string
	Name            string
	TeamName        string
	Currency        string
	AmountUnits     int64
	ClientReference string
	Status          string
}

type SendSenderDomainStatusInput struct {
	ToEmail string
	Name    string
	Domain  string
	Status  string
	Reason  string
}

type SendSenderIDStatusInput struct {
	ToEmail  string
	Name     string
	SenderID string
	Status   string
	Reason   string
}

type EmailSender = platformemail.Sender

type TransactionalEmailSender interface {
	SendTx(context.Context, pgx.Tx, platformemail.Message) (platformemail.Result, error)
}
