package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DealState struct {
	DealID      uint64    `bson:"deal_id"`
	PieceCID    string    `bson:"piece_cid"`
	PieceSize   int64     `bson:"piece_size"`
	Label       string    `bson:"label"`
	Verified    bool      `bson:"verified"`
	Client      string    `bson:"client"`
	Provider    string    `bson:"provider"`
	Expiration  time.Time `bson:"expiration"`
	Start       time.Time `bson:"start"`
	Slashed     time.Time `bson:"slashed"`
	LastUpdated time.Time `bson:"last_updated"`
}

type DealIDLastUpdated struct {
	ID          primitive.ObjectID `bson:"_id"`
	DealID      int32              `bson:"deal_id"`
	LastUpdated time.Time          `bson:"last_updated"`
}

func EpochToTime(epoch int32) time.Time {
	//nolint:gomnd
	return time.Unix(int64(epoch*30+1598306400), 0)
}

func (s DealState) AgeInYears() float64 {
	return time.Since(s.Start).Hours() / 24 / 365
}
