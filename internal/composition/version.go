package composition

// Build stamp, set at link time by -ldflags -X from git describe (see the Makefile). The
// defaults make an un-stamped binary read as a DEV build, never a release.
//
//nolint:gochecknoglobals // ldflags -X requires package-level string vars.
var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

// buildLabel is the version, with its commit when the build was stamped with one.
func buildLabel() string {
	if commit == "" {
		return version
	}
	return version + " (" + commit + ")"
}
