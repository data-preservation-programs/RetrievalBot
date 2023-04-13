package env

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"os"
	"strconv"
	"time"
)

type Key string

//nolint:gosec
const (
	ProcessModules                Key = "PROCESS_MODULES"
	ProcessErrorInterval          Key = "PROCESS_ERROR_INTERVAL"
	TaskWorkerPollInterval        Key = "TASK_WORKER_POLL_INTERVAL"
	TaskWorkerTimeoutBuffer       Key = "TASK_WORKER_TIMEOUT_BUFFER"
	LotusAPIUrl                   Key = "LOTUS_API_URL"
	LotusAPIToken                 Key = "LOTUS_API_TOKEN"
	QueueMongoURI                 Key = "QUEUE_MONGO_URI"
	QueueMongoDatabase            Key = "QUEUE_MONGO_DATABASE"
	ResultMongoURI                Key = "RESULT_MONGO_URI"
	ResultMongoDatabase           Key = "RESULT_MONGO_DATABASE"
	FilplusIntegrationBatchSize   Key = "FILPLUS_INTEGRATION_BATCH_SIZE"
	FilplusIntegrationTaskTimeout Key = "FILPLUS_INTEGRATION_TASK_TIMEOUT"
	FilplusIntegrationRandConst   Key = "FILPLUS_INTEGRATION_RANDOM_CONSTANT"
	StatemarketdealsMongoURI      Key = "STATEMARKETDEALS_MONGO_URI"
	StatemarketdealsMongoDatabase Key = "STATEMARKETDEALS_MONGO_DATABASE"
	StatemarketdealsBatchSize     Key = "STATEMARKETDEALS_BATCH_SIZE"
	StatemarketdealsInterval      Key = "STATEMARKETDEALS_INTERVAL"
	PublicIP                      Key = "_PUBLIC_IP"
	City                          Key = "_CITY"
	Region                        Key = "_REGION"
	Country                       Key = "_COUNTRY"
	Continent                     Key = "_CONTINENT"
	ASN                           Key = "_ASN"
	ISP                           Key = "_ISP"
	Latitude                      Key = "_LATITUDE"
	Longitude                     Key = "_LONGITUDE"
	ProviderCacheTTL              Key = "PROVIDER_CACHE_TTL"
	LocationCacheTTL              Key = "LOCATION_CACHE_TTL"
	AcceptedContinents            Key = "ACCEPTED_CONTINENTS"
	AcceptedCountries             Key = "ACCEPTED_COUNTRIES"
	IPInfoToken                   Key = "IPINFO_TOKEN"
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
