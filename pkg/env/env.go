package env

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"os"
	"strconv"
	"time"
)

func GetString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func GetInt(key string, defaultValue int) int {
	value := os.Getenv(key)
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

func GetRequiredInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		logging.Logger("env").Panicf("%s not set", key)
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		logging.Logger("env").Panicf("failed to parse %s as int", key)
	}

	return intValue
}

func GetRequiredString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		logging.Logger("env").Panicf("%s not set", key)
	}

	return value
}

func GetRequiredDuration(key string) time.Duration {
	value := GetRequiredString(key)
	duration, err := time.ParseDuration(value)
	if err != nil {
		logging.Logger("env").Panicf("%s not set", key)
	}

	return duration
}

func GetDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return GetRequiredDuration(key)
}

func MustSet(key, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		logging.Logger("env").Panicf("failed to set %s to %s", key, value)
	}
}

func MustSetAny(key string, value interface{}) {
	str := fmt.Sprintf("%v", value)
	err := os.Setenv(key, str)
	if err != nil {
		logging.Logger("env").Panicf("failed to set %s to %s", key, value)
	}
}
