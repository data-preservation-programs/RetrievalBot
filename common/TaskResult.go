package common

import "time"

type Retriever struct {
	PublicIP  string  `bson:"ip"`
	City      string  `bson:"city"`
	Region    string  `bson:"region"`
	Country   string  `bson:"country"`
	Continent string  `bson:"continent"`
	ASN       string  `bson:"asn"`
	Org       string  `bson:"org"`
	Latitude  float32 `bson:"lat"`
	Longitude float32 `bson:"long"`
}

type RetrievalResult struct {
	Success    bool    `bson:"success"`
	TTFB       int32   `bson:"ttfb,omitempty"`
	Speed      float32 `bson:"speed,omitempty"`
	Duration   int32   `bson:"duration,omitempty"`
	Downloaded int64   `bson:"downloaded,omitempty"`
}

type TaskResult struct {
	Task
	Retriever Retriever       `bson:"retriever"`
	Result    RetrievalResult `bson:"result"`
	CreatedAt time.Time       `bson:"created_at"`
}
