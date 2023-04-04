package common

import "time"

type Provider struct {
	ID        string `bson:"id"`
	Country   string `bson:"country,omitempty"`
	Continent string `bson:"continent,omitempty"`
}

type ProtocolName string

const (
	Stub      ProtocolName = "stub"
	GraphSync ProtocolName = "graphsync"
	Bitswap   ProtocolName = "bitswap"
	HTTP      ProtocolName = "http"
)

type Protocol struct {
	Name ProtocolName `bson:"name"`
}

type Content struct {
	CID string `bson:"cid"`
}

type Strategy struct {
	Type string `bson:"type"`
}

type Task struct {
	Requester string            `bson:"requester"`
	Metadata  map[string]string `bson:"metadata,omitempty"`
	Provider  Provider          `bson:"provider"`
	Protocol  Protocol          `bson:"protocol"`
	Content   Content           `bson:"content"`
	Strategy  Strategy          `bson:"strategy"`
	CreatedAt time.Time         `bson:"created_at"`
}
