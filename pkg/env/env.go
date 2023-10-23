package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

type Key string

//nolint:gosec
const (
	AcceptedContinents            Key = "ACCEPTED_CONTINENTS"
	AcceptedCountries             Key = "ACCEPTED_COUNTRIES"
	ASN                           Key = "_ASN"
	City                          Key = "_CITY"
	Continent                     Key = "_CONTINENT"
	Country                       Key = "_COUNTRY"
	FilplusIntegrationBatchSize   Key = "FILPLUS_INTEGRATION_BATCH_SIZE"
	FilplusIntegrationRandConst   Key = "FILPLUS_INTEGRATION_RANDOM_CONSTANT"
	FilplusIntegrationTaskTimeout Key = "FILPLUS_INTEGRATION_TASK_TIMEOUT"
	IPInfoToken                   Key = "IPINFO_TOKEN"
	ISP                           Key = "_ISP"
	Latitude                      Key = "_LATITUDE"
	LocationCacheTTL              Key = "LOCATION_CACHE_TTL"
	Longitude                     Key = "_LONGITUDE"
	LotusAPIToken                 Key = "LOTUS_API_TOKEN"
	LotusAPIUrl                   Key = "LOTUS_API_URL"
	ProcessErrorInterval          Key = "PROCESS_ERROR_INTERVAL"
	ProcessModules                Key = "PROCESS_MODULES"
	ProviderCacheTTL              Key = "PROVIDER_CACHE_TTL"
	PublicIP                      Key = "_PUBLIC_IP"
	QueueMongoDatabase            Key = "QUEUE_MONGO_DATABASE"
	QueueMongoURI                 Key = "QUEUE_MONGO_URI"
	Region                        Key = "_REGION"
	ResultMongoDatabase           Key = "RESULT_MONGO_DATABASE"
	ResultMongoURI                Key = "RESULT_MONGO_URI"
	SpadeIntegrationTaskTimeout   Key = "SPADE_INTEGRATION_TASK_TIMEOUT"
	StatemarketdealsBatchSize     Key = "STATEMARKETDEALS_BATCH_SIZE"
	StatemarketdealsInterval      Key = "STATEMARKETDEALS_INTERVAL"
	StatemarketdealsMongoDatabase Key = "STATEMARKETDEALS_MONGO_DATABASE"
	StatemarketdealsMongoURI      Key = "STATEMARKETDEALS_MONGO_URI"
	TaskWorkerPollInterval        Key = "TASK_WORKER_POLL_INTERVAL"
	TaskWorkerTimeoutBuffer       Key = "TASK_WORKER_TIMEOUT_BUFFER"
)

func GetString(key Key, defaultValue string) string {
	value := os.Getenv(string(key))
	if value == "" {
		return defaultValue
	}

	return value
}

func GetInt(key Key, defaultValue int) int {
	value := os.Getenv(string(key))
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		logging.Logger("env").Debugf("failed to parse %s as int", key)
		return defaultValue
	}

	return intValue
}

func GetRequiredInt(key Key) int {
	value := os.Getenv(string(key))
	if value == "" {
		logging.Logger("env").Panicf("%s not set", key)
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		logging.Logger("env").Panicf("failed to parse %s as int", key)
	}

	return intValue
}

func GetRequiredString(key Key) string {
	value := os.Getenv(string(key))
	if value == "" {
		logging.Logger("env").Panicf("%s not set", key)
	}

	return value
}

func GetRequiredFloat32(key Key) float32 {
	value := GetRequiredString(key)
	floatValue, err := strconv.ParseFloat(value, 32)
	if err != nil {
		logging.Logger("env").Panicf("failed to parse %s as float32", key)
	}

	return float32(floatValue)
}

func GetFloat64(key Key, defaultValue float64) float64 {
	value := os.Getenv(string(key))
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		logging.Logger("env").Debugf("failed to parse %s as float", key)
		return defaultValue
	}

	return floatValue
}

func GetRequiredDuration(key Key) time.Duration {
	value := GetRequiredString(key)
	duration, err := time.ParseDuration(value)
	if err != nil {
		logging.Logger("env").Panicf("%s not set", key)
	}

	return duration
}

func GetDuration(key Key, defaultValue time.Duration) time.Duration {
	value := os.Getenv(string(key))
	if value == "" {
		return defaultValue
	}

	return GetRequiredDuration(key)
}

func MustSet(key Key, value string) {
	err := os.Setenv(string(key), value)
	if err != nil {
		logging.Logger("env").Panicf("failed to set %s to %s", key, value)
	}
}

func MustSetAny(key Key, value interface{}) {
	str := fmt.Sprintf("%v", value)
	err := os.Setenv(string(key), str)
	if err != nil {
		logging.Logger("env").Panicf("failed to set %s to %s", key, value)
	}
}
