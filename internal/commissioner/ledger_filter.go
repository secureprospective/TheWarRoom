package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type RawLedgerFilter struct {
	Person string `json:"person"`
	Kind   string `json:"kind"`
	From   string `json:"from"`
	Before string `json:"before"`
}

func (r RawLedgerFilter) Validate() error {
	switch r.Kind {
	case "", "setup", "rules", "people", "notices", "corrections":
	default:
		return fmt.Errorf("choose a kind of league change")
	}
	_, _, err := r.bounds()
	return err
}

func (r RawLedgerFilter) bounds() (time.Time, time.Time, error) {
	var from, before time.Time
	var err error
	if r.From != "" {
		from, err = time.Parse(time.RFC3339Nano, r.From)
		if err != nil {
			return from, before, fmt.Errorf("choose the ledger start date and time: %w", err)
		}
	}
	if r.Before != "" {
		before, err = time.Parse(time.RFC3339Nano, r.Before)
		if err != nil {
			return from, before, fmt.Errorf("choose the ledger end date and time: %w", err)
		}
	}
	if r.From != "" && r.Before != "" && !from.Before(before) {
		return from, before, fmt.Errorf("the ledger start must be earlier than its end")
	}
	return from, before, nil
}

func DecodeLedgerFilter(body string) (RawLedgerFilter, error) {
	var raw RawLedgerFilter
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read ledger filters: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("send exactly one set of ledger filters")
	}
	return raw, raw.Validate()
}

// FilteredReading filters only the authorized reading; its version remains the full league version.
func FilteredReading(league, principal string, records []Record, auth Authorizer,
	filter RawLedgerFilter) (Ledger, error) {
	if err := filter.Validate(); err != nil {
		return Ledger{}, err
	}
	from, before, err := filter.bounds()
	if err != nil {
		return Ledger{}, err
	}
	reading, err := Reading(league, principal, records, auth)
	if err != nil {
		return Ledger{}, err
	}
	kinds := make(map[string]string, len(records))
	for _, record := range records {
		kinds[record.Receipt.Entry] = ledgerKind(record.Command.Command)
	}
	entries := make([]Entry, 0, len(reading.Entries))
	for _, entry := range reading.Entries {
		if filter.matches(entry, kinds[entry.ID], from, before) {
			entries = append(entries, entry)
		}
	}
	reading.Entries = entries
	return reading, nil
}

func (r RawLedgerFilter) matches(entry Entry, kind string, from, before time.Time) bool {
	if !strings.Contains(strings.ToLower(entry.Who), strings.ToLower(strings.TrimSpace(r.Person))) ||
		(r.Kind != "" && r.Kind != kind) {
		return false
	}
	return (r.From == "" || !entry.When.Before(from)) && (r.Before == "" || entry.When.Before(before))
}

func ledgerKind(command string) string {
	switch command {
	case Create, Cutover:
		return "setup"
	case RulesSet:
		return "rules"
	case Revert:
		return "corrections"
	case NoticePublish, NotePost:
		return "notices"
	default:
		return "people"
	}
}
