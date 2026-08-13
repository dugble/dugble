package subscription

func StatusForOutcome(outcome Outcome) SubscriptionStatus {
	if outcome == OutcomeApplied {
		return StatusActive
	}
	return StatusPastDue
}
