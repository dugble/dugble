package dashboard

import "time"

type Stats struct {
	Users            int64
	Teams            int64
	SMSToday         int64
	FailedSMS24Hours int64
	PendingSenderIDs int64
	PendingDomains   int64
}

type Operations struct {
	Stats            Stats
	FailedSMS        []FailedSMS
	PendingSenderIDs []PendingSenderID
	PendingDomains   []PendingDomain
	RecentActivity   []Activity
}

type FailedSMS struct {
	ID, TeamName, ToNumber, Status, ErrorMessage string
	CreatedAt                                    time.Time
}

type PendingSenderID struct {
	ID, TeamName, Name, CountryCode string
	CreatedAt                       time.Time
}

type PendingDomain struct {
	ID, TeamName, Name string
	CreatedAt          time.Time
}

type Activity struct {
	Action, ResourceType, ResourceID, Outcome, ActorType string
	CreatedAt                                            time.Time
}
