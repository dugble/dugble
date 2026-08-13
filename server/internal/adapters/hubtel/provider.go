package hubtel

import "context"

type Provider struct{ client *Client }

func NewProvider(client *Client) *Provider { return &Provider{client: client} }

func (p *Provider) InitiateCheckout(ctx context.Context, req InitiateCheckoutRequest) (InitiateCheckoutResponse, error) {
	return p.client.InitiateCheckout(ctx, req)
}

func (p *Provider) VerifyTransaction(ctx context.Context, clientReference string) (PaymentStatus, error) {
	response, err := p.client.CheckTransactionStatus(ctx, clientReference)
	if err != nil {
		return PaymentStatus{}, err
	}
	return PaymentStatusFromTransactionStatus(response)
}

func (p *Provider) MapCallback(payload CallbackPayload) (PaymentStatus, error) {
	return PaymentStatusFromCallback(payload)
}
