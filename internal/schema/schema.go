// Package schema is the boundary-validation pattern for external input:
// MFL API responses, CSV ingestion, IPC payloads from the frontend.
//
// Mirrors templates/typescript/schemas/example.ts, but hand-written rather
// than reflection/struct-tag driven (agy web-research, companion plan
// Section 8B: "hand-written Validate() error methods over reflection-based
// go-playground/validator — compile-time safe, no struct tags an LLM can
// hallucinate").
//
// RULE: never pass a raw external value (a string from JSON, a CSV cell)
// into business logic. Decode it into a Raw* struct, call Validate(), and
// only then convert into a domain type.
package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// RawPlayerRecord is the shape of one player record as it arrives from the
// MFL `players` endpoint. Every field is a string — MFL's JSON encodes
// numbers as strings (companion plan Section 5: "salary and cap fields
// arrive as strings — parse them, don't assume float").
type RawPlayerRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Salary   string `json:"salary"`
}

// DecodePlayerRecord reads exactly one JSON object and rejects unknown
// fields. DisallowUnknownFields is the Go equivalent of Zod's .strict() —
// extra fields from an API change fail loudly instead of passing silently.
func DecodePlayerRecord(r io.Reader) (RawPlayerRecord, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rec RawPlayerRecord
	if err := dec.Decode(&rec); err != nil {
		return RawPlayerRecord{}, fmt.Errorf("schema: decode player record: %w", err)
	}
	return rec, nil
}

// Validate checks the raw record's shape before any normalization runs.
// It does NOT convert types (that's normalize's job, companion plan WF 1C)
// — it only rejects records that cannot possibly be valid.
func (r RawPlayerRecord) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("schema: player record missing id")
	}
	if _, err := strconv.Atoi(strings.TrimSpace(r.ID)); err != nil {
		return fmt.Errorf("schema: player id %q is not numeric: %w", r.ID, err)
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("schema: player record %q missing name", r.ID)
	}
	if strings.TrimSpace(r.Position) == "" {
		return fmt.Errorf("schema: player record %q missing position", r.ID)
	}
	// Salary is validated as a parseable number here, but kept as a string —
	// normalize.Roster (WF 1C) does the float conversion. Validation and
	// transformation are different jobs (Section 2, item 1: fetchers
	// transform nothing).
	if strings.TrimSpace(r.Salary) != "" {
		if _, err := strconv.ParseFloat(strings.TrimSpace(r.Salary), 64); err != nil {
			return fmt.Errorf("schema: player record %q has non-numeric salary %q: %w", r.ID, r.Salary, err)
		}
	}
	return nil
}
