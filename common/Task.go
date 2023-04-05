package common

import "time"

type Provider struct {
	ID        string `bson:"id"`
	Country   string `bson:"country,omitempty"`
	Continent string `bson:"continent,omitempty"`
}

type ModuleName string

const (
	Stub ModuleName = "stub"
)

type Content struct {
	CID string `bson:"cid"`
}

type Strategy struct {
	Type string `bson:"type"`
}

type Task struct {
	Requester string            `bson:"requester"`
	Module    ModuleName        `bson:"module"`
	Metadata  map[string]string `bson:"metadata,omitempty"`
	Provider  Provider          `bson:"provider"`
	Content   Content           `bson:"content"`
	Strategy  Strategy          `bson:"strategy"`
	CreatedAt time.Time         `bson:"created_at"`
}
