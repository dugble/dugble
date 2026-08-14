package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	smscampaign "github.com/dugble/dugble/server/internal/modules/campaign"
	smsmodule "github.com/dugble/dugble/server/internal/modules/sms"
)

type repository interface {
	QueueNextDue(context.Context) (smscampaign.Campaign, bool, error)
	MaterializeNext(context.Context) (smscampaign.Campaign, bool, error)
	BeginFanoutTx(context.Context) (pgx.Tx, error)
	ClaimNextRecipientTx(context.Context, pgx.Tx) (smscampaign.FanoutRecipient, bool, error)
	RecheckRecipientTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient) (string, bool, error)
	SetRecipientQueuedTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient, uuid.UUID, string, int32, int64) error
	ClaimNextEstimateTx(context.Context, pgx.Tx) (smscampaign.FanoutRecipient, bool, error)
	SetRecipientEstimateTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient, string, string, int32) error
	FailRecipientEstimateTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient, string, string) error
	FinalizeCostPreflightTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient) (smscampaign.Campaign, bool, error)
	FailRecipientTx(context.Context, pgx.Tx, smscampaign.FanoutRecipient, string, string) error
	FinalizeFanoutTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) (smscampaign.Campaign, bool, error)
}

type enqueuer interface {
	EnqueueCampaignTx(context.Context, pgx.Tx, uuid.UUID, smsmodule.CampaignEnqueueRequest) (smsmodule.CampaignQueuedMessage, error)
	ObserveCampaignCommitted(context.Context, smsmodule.CampaignQueuedMessage)
}

type Processor struct {
	repository repository
	sms        enqueuer
}

func NewProcessor(repository repository, sms enqueuer) *Processor {
	return &Processor{repository: repository, sms: sms}
}

func (p *Processor) ProcessBatch(ctx context.Context, size int) error {
	if p == nil || p.repository == nil || p.sms == nil {
		return errors.New("SMS campaign processor is not configured")
	}
	if size <= 0 {
		size = 100
	}
	for i := 0; i < size; i++ {
		if _, found, err := p.repository.QueueNextDue(ctx); err != nil {
			return err
		} else if !found {
			break
		}
	}
	for i := 0; i < size; i++ {
		if _, found, err := p.repository.MaterializeNext(ctx); err != nil {
			return err
		} else if !found {
			break
		}
	}
	for i := 0; i < size; i++ {
		found, err := p.estimateRecipient(ctx)
		if err != nil {
			return err
		}
		if !found {
			break
		}
	}
	for i := 0; i < size; i++ {
		found, err := p.processRecipient(ctx)
		if err != nil {
			return err
		}
		if !found {
			break
		}
	}
	return nil
}

func (p *Processor) processRecipient(ctx context.Context) (bool, error) {
	tx, err := p.repository.BeginFanoutTx(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	recipient, found, err := p.repository.ClaimNextRecipientTx(ctx, tx)
	if err != nil || !found {
		return found, err
	}
	_, excluded, err := p.repository.RecheckRecipientTx(ctx, tx, recipient)
	if err != nil {
		return false, err
	}
	if excluded {
		if _, _, err = p.repository.FinalizeFanoutTx(ctx, tx, recipient.TeamID, recipient.CampaignID); err != nil {
			return false, err
		}
		if err = tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}
	body := recipient.RenderedBody
	if body == "" {
		return p.fail(ctx, tx, recipient, "estimate_missing", "recipient cost estimate is missing")
	}
	metadata, _ := json.Marshal(map[string]string{"campaign_id": recipient.CampaignID.String(), "campaign_recipient_id": recipient.ID.String()})
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin SMS campaign enqueue savepoint: %w", err)
	}
	queued, err := p.sms.EnqueueCampaignTx(ctx, savepoint, recipient.TeamID, smsmodule.CampaignEnqueueRequest{SenderID: recipient.SenderID, To: recipient.Phone, From: recipient.SenderName, Body: body, Metadata: metadata})
	if err != nil {
		_ = savepoint.Rollback(ctx)
		return p.fail(ctx, tx, recipient, "enqueue_failed", err.Error())
	}
	if err = savepoint.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit SMS campaign enqueue savepoint: %w", err)
	}
	messageID, err := uuid.Parse(queued.Message.ID)
	if err != nil {
		return false, err
	}
	if err = p.repository.SetRecipientQueuedTx(ctx, tx, recipient, messageID, body, queued.Message.Segments, queued.Charge.AmountUnits); err != nil {
		return false, err
	}
	if _, _, err = p.repository.FinalizeFanoutTx(ctx, tx, recipient.TeamID, recipient.CampaignID); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit SMS campaign fanout: %w", err)
	}
	p.sms.ObserveCampaignCommitted(ctx, queued)
	return true, nil
}

func (p *Processor) estimateRecipient(ctx context.Context) (bool, error) {
	tx, err := p.repository.BeginFanoutTx(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	recipient, found, err := p.repository.ClaimNextEstimateTx(ctx, tx)
	if err != nil || !found {
		return found, err
	}
	body, err := renderBody(recipient.CampaignBody, recipient.ContactSnapshot)
	if err != nil {
		if failErr := p.repository.FailRecipientEstimateTx(ctx, tx, recipient, "render_failed", err.Error()); failErr != nil {
			return false, failErr
		}
	} else {
		encoding, _, segments := smsmodule.AnalyzeBody(body)
		if err = p.repository.SetRecipientEstimateTx(ctx, tx, recipient, body, encoding, segments); err != nil {
			if failErr := p.repository.FailRecipientEstimateTx(ctx, tx, recipient, "pricing_failed", err.Error()); failErr != nil {
				return false, failErr
			}
		}
	}
	if _, _, err = p.repository.FinalizeCostPreflightTx(ctx, tx, recipient); err != nil {
		return false, err
	}
	if err = smscampaign.ApplyCommunicationCreditPreflight(ctx, tx, recipient.TeamID, recipient.CampaignID); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit SMS campaign cost estimate: %w", err)
	}
	return true, nil
}

func (p *Processor) fail(ctx context.Context, tx pgx.Tx, recipient smscampaign.FanoutRecipient, code, message string) (bool, error) {
	if err := p.repository.FailRecipientTx(ctx, tx, recipient, code, message); err != nil {
		return false, err
	}
	if _, _, err := p.repository.FinalizeFanoutTx(ctx, tx, recipient.TeamID, recipient.CampaignID); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

var mustacheVariable = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

func renderBody(body string, snapshot map[string]any) (string, error) {
	variables := make(map[string]any, len(snapshot)+8)
	for key, value := range snapshot {
		if value == nil {
			value = ""
		}
		variables[key] = value
	}
	if properties, ok := snapshot["properties"].(map[string]any); ok {
		for key, value := range properties {
			if value == nil {
				value = ""
			}
			if _, exists := variables[key]; !exists {
				variables[key] = value
			}
		}
	}
	normalized := mustacheVariable.ReplaceAllString(body, "{{ .$1 }}")
	tmpl, err := template.New("sms_campaign").Option("missingkey=error").Parse(normalized)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	if err = tmpl.Execute(&rendered, variables); err != nil {
		return "", err
	}
	result := rendered.String()
	if strings.TrimSpace(result) == "" {
		return "", errors.New("rendered SMS body is empty")
	}
	if _, characters, _ := smsmodule.AnalyzeBody(result); characters > 1600 {
		return "", errors.New("rendered SMS body exceeds 1600 characters")
	}
	return result, nil
}
