package auditlog

import "time"

type Filter struct {
	Query, Outcome, ActorType string
	Limit, Offset             int32
}

type Event struct {
	ID, Action, ResourceType, ResourceID, Outcome string
	ActorType, ActorEmail, TeamName               string
	RequestID, IPAddress, Metadata                string
	CreatedAt                                     time.Time
}

type Page struct {
	Events                     []Event
	Filter                     Filter
	PreviousOffset, NextOffset int32
	HasPrevious, HasNext       bool
}
