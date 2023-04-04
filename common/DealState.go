package common

import "time"

type DealState struct {
	DealID     int32     `bson:"deal_id"`
	PieceCID   string    `bson:"piece_cid"`
	Label      string    `bson:"label"`
	Verified   bool      `bson:"verified"`
	Client     string    `bson:"client"`
	Provider   string    `bson:"provider"`
	Expiration time.Time `bson:"expiration"`
}

type DealID struct {
	DealID int32 `bson:"deal_id"`
}
