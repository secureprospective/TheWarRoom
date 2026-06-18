package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MFLList decodes an MFL JSON field that is either a JSON array or — when MFL's
// legacy XML→JSON converter emits exactly one element — a bare object with the
// brackets stripped. It always yields a slice, so callers never special-case the
// singleton. Every MFL list field (franchise[], player[], …) should use this type;
// it is the single place the array/object-collapse quirk is handled for the whole
// ingestion layer. A null or empty field yields a nil slice — the caller's own
// emptiness check decides whether that is an error (a 32-team league returning zero
// franchises is a glitch, not a valid state).
type MFLList[T any] []T

// UnmarshalJSON implements the array-or-single decode by sniffing the first
// non-space byte: '[' decodes as an array, anything else as a single element
// wrapped into a one-item slice.
func (l *MFLList[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*l = nil
		return nil
	}

	if trimmed[0] == '[' {
		var s []T
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return fmt.Errorf("ingestion: decode MFL list: %w", err)
		}
		*l = s
		return nil
	}

	var one T
	if err := json.Unmarshal(trimmed, &one); err != nil {
		return fmt.Errorf("ingestion: decode MFL singleton: %w", err)
	}
	*l = []T{one}
	return nil
}
