package db

import "strings"

// fileURI makes a SQLite file: URI for path. SQLite reads ? and # as URI delimiters and %
// as an escape, so a folder named "Bob#2" would otherwise open a different file.
func fileURI(path string) string {
	return "file:" + strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(path)
}
