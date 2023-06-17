package env

import (
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

var (
	_startTime  string
	_goVersion  string
	_appVersion string
)

// build info
var (
	_buildTime string // build time
	_gitTag    string // git tag
	_gitCommit string // sha1 from git, output of $(git rev-parse HEAD)
)

func init() {
	_startTime = time.Now().Format("2006-01-02 15:04:05")
	_buildTime = strings.Replace(_buildTime, "--", " ", 1)
	_goVersion = runtime.Version()

	// milli version
	_appVersion = "unknown version"
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, value := range info.Deps {
			if value.Path == "github.com/nextmicro/next" {
				_appVersion = value.Version
			}
		}
	}
}

// StartTime get start time
func StartTime() string {
	return _startTime
}

// BuildTime get buildTime
func BuildTime() string {
	return _buildTime
}

// AppVersion get app version
func AppVersion() string {
	return _appVersion
}

// GoVersion get go version
func GoVersion() string {
	return _goVersion
}

// GitTag git tag
func GitTag() string {
	return _gitTag
}

// GitCommit git commit
func GitCommit() string {
	return _gitCommit
}
