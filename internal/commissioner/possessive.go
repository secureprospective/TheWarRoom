package commissioner

import "strings"

// Possessive is a franchise name's possessive: "Buffalo Bills'", "Miami Heat's".
func Possessive(name string) string {
	if strings.HasSuffix(name, "s") {
		return name + "'"
	}
	return name + "'s"
}
