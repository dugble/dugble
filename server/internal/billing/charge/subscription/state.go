package subscription

type SubscriptionStatus string

const (
	StatusActive  SubscriptionStatus = "active"
	StatusPastDue SubscriptionStatus = "past_due"
)
