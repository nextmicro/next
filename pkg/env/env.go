package env

import (
	"os"
	"strconv"

	"github.com/nextmicro/logger"
)

// IntEnvOr returns the int value of the environment variable with name key if
// it exists and the value is an int. Otherwise, defaultValue is returned.
func IntEnvOr(key string, defaultValue int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		logger.Warn("got invalid value, number value expected.", key, value)
		return defaultValue
	}

	return intValue
}

// StringEnvOr returns the string value of the environment variable with name key if
// it exists and the value is an string. Otherwise, defaultValue is returned.
func StringEnvOr(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	return value
}
