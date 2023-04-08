package task

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

type ErrorCode string

const (
	ErrorCodeNone        ErrorCode = ""
	CannotConnect        ErrorCode = "cannot_connect"
	NotFound             ErrorCode = "not_found"
	RetrievalFailure     ErrorCode = "retrieval_failure"
	ProtocolNotSupported ErrorCode = "protocol_not_supported"
	Timeout              ErrorCode = "timeout"
)

func NewErrorRetrievalResult(code ErrorCode, err error) *RetrievalResult {
	return &RetrievalResult{
		Success:      false,
		ErrorCode:    code,
		ErrorMessage: err.Error(),
		TTFB:         0,
		Speed:        0,
		Duration:     0,
		Downloaded:   0,
	}
}

func NewSuccessfulRetrievalResult(ttfb time.Duration, downloaded int64, duration time.Duration) *RetrievalResult {
	return &RetrievalResult{
		Success:      true,
		ErrorCode:    ErrorCodeNone,
		ErrorMessage: "",
		TTFB:         ttfb,
		Speed:        float64(downloaded) / duration.Seconds(),
		Duration:     duration,
		Downloaded:   downloaded,
	}
}

type RetrievalResult struct {
	Success      bool          `bson:"success"`
	ErrorCode    ErrorCode     `bson:"error_code,omitempty"`
	ErrorMessage string        `bson:"error_message,omitempty"`
	TTFB         time.Duration `bson:"ttfb,omitempty"`
	Speed        float64       `bson:"speed,omitempty"`
	Duration     time.Duration `bson:"duration,omitempty"`
	Downloaded   int64         `bson:"downloaded,omitempty"`
}

type Result struct {
	Task
	Retriever Retriever       `bson:"retriever"`
	Result    RetrievalResult `bson:"result"`
	CreatedAt time.Time       `bson:"created_at"`
}
