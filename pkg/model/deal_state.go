package model

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

func EpochToTime(epoch int32) time.Time {
	//nolint:gomnd
	return time.Unix(int64(epoch*30+1598306400), 0)
}

func (s DealState) YeasTillExpiration() float64 {
	return time.Until(s.Expiration).Hours() / 24 / 365
}
