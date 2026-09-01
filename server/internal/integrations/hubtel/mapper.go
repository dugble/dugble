package hubtel

import "encoding/json"

func PaymentStatusFromCallback(payload CallbackPayload) (PaymentStatus, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return PaymentStatus{}, err
	}
	return PaymentStatus{
		ClientReference: payload.Data.ClientReference,
		Status:          payload.Data.Status,
		Provider:        "hubtel",
		Raw:             raw,
	}, nil
}

func PaymentStatusFromTransactionStatus(response TransactionStatusResponse) (PaymentStatus, error) {
	raw, err := json.Marshal(response)
	if err != nil {
		return PaymentStatus{}, err
	}
	return PaymentStatus{
		ClientReference: response.Data.ClientReference,
		Status:          response.Data.Status,
		Provider:        "hubtel",
		Raw:             raw,
	}, nil
}

func IsPaidStatus(status string) bool { return status == "Paid" || status == "Success" }
