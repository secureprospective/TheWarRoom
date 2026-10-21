// Package catalog describes feature commands independently of their transport.
package catalog

// Unset is explicit missing metadata, not permission to publish. A future server publisher
// must reject it before exposing a command; this catalog changes no current enforcement.
const Unset = "unset"

// Role is a comma-separated set when the current desktop command permits several roles.
type Command struct {
	ID, Method, Version, Role, Gravity, Availability string
}
