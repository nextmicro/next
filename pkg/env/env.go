package env

import (
	"os"
	"strconv"
	"sync"

	"github.com/nextmicro/logger"
)

var (
	envs    = make(map[string]string)
	envLock sync.RWMutex
)

// Env returns the value of the given environment variable.
func Env(name string) string {
	envLock.RLock()
	val, ok := envs[name]
	envLock.RUnlock()

	if ok {
		return val
	}

	val = os.Getenv(name)
	envLock.Lock()
	envs[name] = val
	envLock.Unlock()

	return val
}

// EnvInt returns an int value of the given environment variable.
func EnvInt(name string) (int, bool) {
	val := Env(name)
	if len(val) == 0 {
		return 0, false
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}

	return n, true
}

// IntEnvOr returns the int value of the environment variable with name key if
// it exists and the value is an int. Otherwise, defaultValue is returned.
func IntEnvOr(key string, defaultValue int) int {
	val := Env(key)
	if len(val) == 0 {
		return defaultValue
	}

	intValue, err := strconv.Atoi(val)
	if err != nil {
		logger.Warn("got invalid value, number value expected.", key, val)
		return defaultValue
	}

	return intValue
}

// StringEnvOr returns the string value of the environment variable with name key if
// it exists and the value is an string. Otherwise, defaultValue is returned.
func StringEnvOr(key string, defaultValue string) string {
	value := Env(key)
	if len(value) == 0 {
		return defaultValue
	}

	return value
}
