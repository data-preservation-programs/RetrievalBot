package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DealState struct {
	DealID      int32  `bson:"deal_id"`
	PieceCID    string `bson:"piece_cid"`
	PieceSize   uint64 `bson:"piece_size"`
	Label       string `bson:"label"`
	Verified    bool   `bson:"verified"`
	Client      string `bson:"client"`
	Provider    string `bson:"provider"`
	Start       int32  `bson:"start"`
	End         int32  `bson:"end"`
	SectorStart int32  `bson:"sector_start"`
	Slashed     int32  `bson:"slashed"`
	LastUpdated int32  `bson:"last_updated"`
}

type DealIDLastUpdated struct {
	ID          primitive.ObjectID `bson:"_id"`
	DealID      int32              `bson:"deal_id"`
	LastUpdated int32              `bson:"last_updated"`
}

func EpochToTime(epoch int32) time.Time {
	if epoch < 0 {
		return time.Time{}
	}
	//nolint:gomnd
	return time.Unix(int64(epoch*30+1598306400), 0).UTC()
}

func TimeToEpoch(t time.Time) int32 {
	if t.IsZero() {
		return -1
	}
	//nolint:gomnd
	return int32(t.Unix()-1598306400) / 30
}

func (s DealState) AgeInYears() float64 {
	return time.Since(EpochToTime(s.SectorStart)).Hours() / 24 / 365
}
