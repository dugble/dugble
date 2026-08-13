package hubtel

type InitiateCheckoutRequest struct {
	TotalAmount           float64 `json:"totalAmount"`
	Description           string  `json:"description"`
	CallbackURL           string  `json:"callbackUrl"`
	ReturnURL             string  `json:"returnUrl"`
	MerchantAccountNumber string  `json:"merchantAccountNumber"`
	CancellationURL       string  `json:"cancellationUrl"`
	ClientReference       string  `json:"clientReference"`
}

type InitiateCheckoutResponse struct {
	ResponseCode string       `json:"responseCode"`
	Status       string       `json:"status"`
	Data         CheckoutData `json:"data"`
}

type CheckoutData struct {
	CheckoutURL       string `json:"checkoutUrl"`
	CheckoutID        string `json:"checkoutId"`
	ClientReference   string `json:"clientReference"`
	Message           string `json:"message"`
	CheckoutDirectURL string `json:"checkoutDirectUrl"`
}

type CallbackPayload struct {
	ResponseCode string       `json:"ResponseCode"`
	Status       string       `json:"Status"`
	Data         CallbackData `json:"Data"`
}

type CallbackData struct {
	CheckoutID          string         `json:"CheckoutId"`
	SalesInvoiceID      string         `json:"SalesInvoiceId"`
	ClientReference     string         `json:"ClientReference"`
	Status              string         `json:"Status"`
	Amount              float64        `json:"Amount"`
	CustomerPhoneNumber string         `json:"CustomerPhoneNumber"`
	PaymentDetails      PaymentDetails `json:"PaymentDetails"`
	Description         string         `json:"Description"`
}

type PaymentDetails struct {
	MobileMoneyNumber string `json:"MobileMoneyNumber"`
	PaymentType       string `json:"PaymentType"`
	Channel           string `json:"Channel"`
}

type TransactionStatusResponse struct {
	Message      string                `json:"message"`
	ResponseCode string                `json:"responseCode"`
	Data         TransactionStatusData `json:"data"`
}

type TransactionStatusData struct {
	Date                  string  `json:"date"`
	Status                string  `json:"status"`
	TransactionID         string  `json:"transactionId"`
	ExternalTransactionID string  `json:"externalTransactionId"`
	PaymentMethod         string  `json:"paymentMethod"`
	ClientReference       string  `json:"clientReference"`
	CurrencyCode          *string `json:"currencyCode"`
	Amount                float64 `json:"amount"`
	Charges               float64 `json:"charges"`
	AmountAfterCharges    float64 `json:"amountAfterCharges"`
	IsFulfilled           *bool   `json:"isFulfilled"`
}

type PaymentStatus struct {
	ClientReference string
	Status          string
	Provider        string
	Raw             []byte
}
