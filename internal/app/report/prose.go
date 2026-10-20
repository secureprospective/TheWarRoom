package report

import "strings"

// ProseName turns MFL's "Last, First" into the "First Last" a sentence reads.
func ProseName(name string) string {
	last, first, ok := strings.Cut(name, ", ")
	if !ok || first == "" || strings.Contains(first, ", ") {
		return name
	}
	return first + " " + last
}
