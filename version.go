package main

// Build stamp, set at link time by -ldflags -X from git describe (see the Makefile). The
// defaults make an un-stamped binary read as a DEV build, never a release.
//
//nolint:gochecknoglobals // ldflags -X requires package-level string vars.
var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

// AppInfo is what the shell reads once on load: the build stamp, and the startup failure if
// there was one (empty when startup succeeded).
type AppInfo struct {
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	BuildDate    string `json:"buildDate"`
	StartupError string `json:"startupError,omitempty"`
}

// AppInfo returns the build stamp and startup status.
func (a *App) AppInfo() AppInfo {
	info := AppInfo{Version: version, Commit: commit, BuildDate: buildDate}
	if a.startupErr != nil {
		info.StartupError = a.startupErr.Error()
	}
	return info
}
