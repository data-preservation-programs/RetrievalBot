package common

import "time"

type Provider struct {
	ID        string `bson:"id"`
	Country   string `bson:"country,omitempty"`
	Continent string `bson:"continent,omitempty"`
}

type ModuleName string

const (
	Stub      ModuleName = "stub"
	GraphSync ModuleName = "graphsync"
	HTTP      ModuleName = "http"
	Bitswap   ModuleName = "bitswap"
)

type Content struct {
	CID string `bson:"cid"`
}

type Task struct {
	Requester string                 `bson:"requester"`
	Module    ModuleName             `bson:"module"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty"`
	Provider  Provider               `bson:"provider"`
	Content   Content                `bson:"content"`
	Options   map[string]interface{} `bson:"options,omitempty"`
	CreatedAt time.Time              `bson:"created_at"`
}
