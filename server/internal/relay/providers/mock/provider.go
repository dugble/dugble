package mock

import (
	"context"
	"sync"

	"github.com/dugble/dugble/server/internal/relay/sms"
)

type SendFunc func(context.Context, sms.Message) (sms.SendResult, error)

// Provider is a programmable SMS provider for tests and examples.
type Provider struct {
	name string
	send SendFunc

	mu       sync.Mutex
	messages []sms.Message
}

func New(name string, send SendFunc) *Provider { return &Provider{name: name, send: send} }
func (p *Provider) Name() string { return p.name }
func (p *Provider) Send(ctx context.Context, message sms.Message) (sms.SendResult, error) {
	p.mu.Lock(); p.messages = append(p.messages, message); p.mu.Unlock()
	if p.send == nil { return sms.SendResult{State: sms.SubmissionUnknown}, nil }
	return p.send(ctx, message)
}
func (p *Provider) Calls() int { p.mu.Lock(); defer p.mu.Unlock(); return len(p.messages) }
func (p *Provider) Messages() []sms.Message { p.mu.Lock(); defer p.mu.Unlock(); messages := make([]sms.Message, len(p.messages)); copy(messages, p.messages); return messages }
