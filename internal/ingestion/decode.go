package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// mflAPIError is MFL's error envelope. MFL often returns HTTP 200 with {"error":{"$t":...}},
// so a fetcher where an empty result is valid (salaryadjustments) must check this first, or an
// error reads as "no data" and wipes real state.
type mflAPIError struct {
	Error *struct {
		Text string `json:"$t"`
	} `json:"error"`
}

// CheckAPIError returns an error when body is a well-formed MFL error payload. Malformed JSON
// is left to the caller's decode.
func CheckAPIError(body []byte) error {
	var e mflAPIError
	if json.Unmarshal(body, &e) == nil && e.Error != nil && strings.TrimSpace(e.Error.Text) != "" {
		return fmt.Errorf("ingestion: MFL API error: %s", strings.TrimSpace(e.Error.Text))
	}
	return nil
}

// MFLList decodes an MFL list field that is an array, or a bare object when MFL's XML→JSON
// converter collapses a single element. Use it for every MFL list. Null or empty gives nil;
// the caller decides whether that is an error.
type MFLList[T any] []T

// UnmarshalJSON decodes '[' as an array and anything else as one element.
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
