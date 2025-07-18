package storm

import (
	"fmt"
	"runtime"
)

// Version information
const (
	Version      = "1.0.0-alpha"
	APIVersion   = "v1"
	MinGoVersion = "1.24"
)

// BuildInfo contains build information
var BuildInfo = struct {
	Version    string
	APIVersion string
	GitCommit  string
	BuildDate  string
	GoVersion  string
}{
	Version:    Version,
	APIVersion: APIVersion,
	GoVersion:  runtime.Version(),
}

// SetBuildInfo is called by the build process
func SetBuildInfo(commit, date, goVersion string) {
	BuildInfo.GitCommit = commit
	BuildInfo.BuildDate = date
	if goVersion != "" {
		BuildInfo.GoVersion = goVersion
	}
}

// VersionInfo returns formatted version information
func VersionInfo() string {
	return fmt.Sprintf("Storm %s (API %s)", BuildInfo.Version, BuildInfo.APIVersion)
}

// FullVersionInfo returns detailed version information
func FullVersionInfo() string {
	info := fmt.Sprintf("Storm %s\n", BuildInfo.Version)
	info += fmt.Sprintf("API Version: %s\n", BuildInfo.APIVersion)
	info += fmt.Sprintf("Go Version: %s\n", BuildInfo.GoVersion)

	if BuildInfo.GitCommit != "" {
		info += fmt.Sprintf("Git Commit: %s\n", BuildInfo.GitCommit)
	}

	if BuildInfo.BuildDate != "" {
		info += fmt.Sprintf("Build Date: %s\n", BuildInfo.BuildDate)
	}

	return info
}

// IsVersionCompatible checks if the current version is compatible with the minimum required version
func IsVersionCompatible(required string) bool {
	return Version >= required
}
