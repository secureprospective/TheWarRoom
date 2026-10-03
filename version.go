package main

// Build-stamp variables injected at link time via `-ldflags -X` (see the
// Makefile `build`/`release` targets). The git tag is the single source of
// truth (D-V2): `git describe --tags --always --dirty` flows into `version`,
// the short SHA into `commit`, an RFC-3339 UTC timestamp into `buildDate`.
//
// The defaults are deliberately NOT empty — an un-stamped binary (plain
// `go build`, `wails dev`) must read as a visibly distinct DEV build, never a
// real release. There is no `version.json` and no duplicated TS constant: the
// binary is the authority and the frontend reads it through AppInfo() in one hop.
// These are the sanctioned single injection site for the build stamp.
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
