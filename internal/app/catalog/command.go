// Package catalog describes feature commands independently of their transport.
package catalog

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const Unset = "unset"

type Mode string

const (
	Read    Mode = "read"
	Preview Mode = "preview"
	Execute Mode = "execute"
)

// Command describes a logical operation. Method is its existing desktop adapter, not a route.
// Role is legacy desktop display metadata; Permission is the argument-aware domain policy.
// Version versions this operation, independently of the transport protocol and durable records.
type Command struct {
	ID, Method, Version, Role, Gravity, Availability string
	Arguments                                        Schema
	Permission                                       string
	Modes                                            []Mode
	Undo, Facts, Safeguards                          string
}

// ValidatePublication fails closed before a publisher registers any routes. Metadata is not
// authorization: dispatch must still validate Raw input and apply current credential/domain policy.
func (c Command) ValidatePublication() error {
	for name, value := range map[string]string{
		"id": c.ID, "method": c.Method, "role": c.Role, "permission": c.Permission,
		"undo": c.Undo, "facts": c.Facts, "safeguards": c.Safeguards,
	} {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value || value == Unset {
			return fmt.Errorf("catalog %s: missing %s", c.Method, name)
		}
	}
	version, err := strconv.Atoi(c.Version)
	if err != nil || version < 1 || strconv.Itoa(version) != c.Version {
		return fmt.Errorf("catalog %s: version must be a positive integer", c.Method)
	}
	if !slices.Contains([]string{"G0", "G1", "G2", "G3", "argument-dependent"}, c.Gravity) ||
		!slices.Contains([]string{"both", "desktop", "server"}, c.Availability) {
		return fmt.Errorf("catalog %s: invalid gravity or availability", c.Method)
	}
	if err := validateModes(c.Method, c.Modes); err != nil {
		return err
	}
	if err := c.Arguments.Validate(); err != nil {
		return fmt.Errorf("catalog %s arguments: %w", c.Method, err)
	}
	return nil
}

// Publication validates the entire set before returning anything a transport can register.
// Several logical commands may share an adapter; their IDs must still be unique.
func Publication(commands []Command) ([]Command, error) {
	seen := map[string]bool{}
	for _, command := range commands {
		if err := command.ValidatePublication(); err != nil {
			return nil, err
		}
		if seen[command.ID] {
			return nil, fmt.Errorf("catalog: duplicate command %s", command.ID)
		}
		seen[command.ID] = true
	}
	return slices.Clone(commands), nil
}

func validateModes(method string, modes []Mode) error {
	if len(modes) == 0 {
		return fmt.Errorf("catalog %s: missing modes", method)
	}
	seen := map[Mode]bool{}
	for _, mode := range modes {
		if (mode != Read && mode != Preview && mode != Execute) || seen[mode] {
			return fmt.Errorf("catalog %s: invalid or duplicate mode", method)
		}
		seen[mode] = true
	}
	if seen[Read] && (seen[Preview] || seen[Execute]) {
		return fmt.Errorf("catalog %s: reads cannot execute", method)
	}
	return nil
}
