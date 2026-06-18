package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// mflAPIError is MFL's standard error envelope. MFL returns HTTP 200 with
// {"error":{"$t":"..."}} for many failures (invalid league id, maintenance,
// rate-limit) rather than a non-2xx status. A fetcher that treats an empty decode as
// a VALID empty result — e.g. salaryadjustments, where an empty ledger is legitimate
// and there is no zero-length sentinel — MUST check this BEFORE flattening, or an
// error payload silently reads as "no data" and wipes real state downstream.
type mflAPIError struct {
	Error *struct {
		Text string `json:"$t"`
	} `json:"error"`
}

// CheckAPIError returns a non-nil error when body is an MFL API-error payload. It is
// decode-tolerant: malformed JSON is not its concern (the caller's real decode will
// surface that), so it fires only on a well-formed MFL {"error":{"$t":...}} shape.
func CheckAPIError(body []byte) error {
	var e mflAPIError
	// Decode-tolerant: only act on a cleanly-decoded MFL error shape. Malformed
	// JSON is left for the caller's real decode to surface (hence we proceed only
	// when Unmarshal succeeds, rather than returning nil on its error).
	if json.Unmarshal(body, &e) == nil && e.Error != nil && strings.TrimSpace(e.Error.Text) != "" {
		return fmt.Errorf("ingestion: MFL API error: %s", strings.TrimSpace(e.Error.Text))
	}
	return nil
}

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
