package env

import (
	"os"
	"path/filepath"
)

const (
	NextEnvPrefix        = "NEXT_"
	HostIDKey            = "HOST_ID"
	HostnameKey          = "HOST_NAME"
	WorkPathKey          = "WORK_PATH"
	DeployColorKey       = "DEPLOY_COLOR"
	DeployEnvironmentKey = "DEPLOY_ENV"
)

const (
	Dev  = "dev"        // 开发环境
	Test = "testing"    // 环境环境
	Pre  = "pre"        // 灰度环境
	Prod = "production" // 线上环境
)

var (
	_hostId            int
	_hostname          string
	_workPath          string
	_deployColor       string
	_deployEnvironment = Dev
)

func init() {
	_deployColor = StringEnvOr(GetEnvKey(DeployColorKey), "")
	_deployEnvironment = StringEnvOr(GetEnvKey(DeployEnvironmentKey), Dev)
	_hostId = IntEnvOr(GetEnvKey(HostIDKey), 0)
	_hostname = StringEnvOr(GetEnvKey(HostnameKey), "")
	if _hostname == "" {
		_hostname, _ = os.Hostname()
	}

	_workPath = StringEnvOr(GetEnvKey(WorkPathKey), "")
	if _workPath == "" && _deployEnvironment == Dev {
		_workPath, _ = os.Getwd()
	} else if _workPath == "" {
		_workPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	}
}

// GetEnvKey get env key: NEXT_{env}
func GetEnvKey(env string) string {
	return NextEnvPrefix + env
}

// WorkDir work path.
func WorkDir() string {
	return _workPath
}

// HostID get host id
func HostID() int {
	return _hostId
}

// Hostname get hostname
func Hostname() string {
	return _hostname
}

// DeployColor is the identification of different experimental group in one caster cluster.
func DeployColor() string {
	return _deployColor
}

// DeployEnvironment get deploy environment
func DeployEnvironment() string {
	return _deployEnvironment
}

// IsDev is dev environment
func IsDev() bool {
	return DeployEnvironment() == Dev
}

// IsTest is test environment
func IsTest() bool {
	return DeployEnvironment() == Test
}

// IsPre is pre environment
func IsPre() bool {
	return DeployEnvironment() == Pre
}

// IsProduction is production environment
func IsProduction() bool {
	return DeployEnvironment() == Prod
}
