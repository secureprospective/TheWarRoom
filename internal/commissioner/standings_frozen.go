package commissioner

import (
	"encoding/json"
	"fmt"
)

// frozenStandings is the encoded, viewer-independent body of a season's standings, kept like frozenResults: a
// kept reading is served to every phone polling the season, so its rows are encoded once when kept.
type frozenStandings struct {
	lead, rest json.RawMessage
}

// standingsLead and standingsRest are NativeStandings' wire fields in their order and tags, split around the
// per-read Revision (Evidence leads, and is also per-read).
type standingsLead struct {
	League string `json:"league"`
	Name   string `json:"name"`
	Season int    `json:"season"`
}

type standingsRest struct {
	Status   string                 `json:"status"`
	Words    string                 `json:"words"`
	Through  int                    `json:"through"`
	Decided  int                    `json:"decided"`
	Pending  int                    `json:"pending"`
	Criteria []string               `json:"criteria"`
	NoData   []string               `json:"noData"`
	Computed []StandingRow          `json:"computed"`
	Override *StandingsOverrideView `json:"override,omitempty"`
	Changes  []ScoreChange          `json:"changes"`
}

// Freeze encodes the reading's body once. A frozen reading is shared read-only: its slices must not change
// afterwards. Evidence and Revision, set per read, are still encoded on every read.
func (r *NativeStandings) Freeze() error {
	lead, err := json.Marshal(standingsLead{League: r.League, Name: r.Name, Season: r.Season})
	if err != nil {
		return fmt.Errorf("encode standings head: %w", err)
	}
	rest, err := json.Marshal(standingsRest{Status: r.Status, Words: r.Words, Through: r.Through, Decided: r.Decided,
		Pending: r.Pending, Criteria: r.Criteria, NoData: r.NoData, Computed: r.Computed, Override: r.Override,
		Changes: r.Changes})
	if err != nil {
		return fmt.Errorf("encode standings body: %w", err)
	}
	r.frozen = &frozenStandings{lead: lead[1 : len(lead)-1], rest: rest[1 : len(rest)-1]}
	return nil
}

// Frozen reports whether the reading carries its encoded body (Freeze).
func (r NativeStandings) Frozen() bool { return r.frozen != nil }

// MarshalJSON encodes the reading; a frozen one reuses its encoded body, byte for byte what a fresh encode gives.
func (r NativeStandings) MarshalJSON() ([]byte, error) {
	if r.frozen != nil {
		return r.EncodedJSON()
	}
	type plain NativeStandings
	body, err := json.Marshal(plain(r))
	if err != nil {
		return nil, fmt.Errorf("encode standings: %w", err)
	}
	return body, nil
}

// EncodedJSON is the reading's compact JSON; a frozen reading only encodes its per-read fields.
func (r NativeStandings) EncodedJSON() ([]byte, error) {
	f := r.frozen
	if f == nil {
		return r.MarshalJSON()
	}
	out := make([]byte, 0, len(f.lead)+len(f.rest)+128)
	out = append(out, '{')
	if r.Evidence != nil {
		evidence, err := json.Marshal(r.Evidence)
		if err != nil {
			return nil, fmt.Errorf("encode standings evidence: %w", err)
		}
		out = append(out, `"evidence":`...)
		out = append(out, evidence...)
		out = append(out, ',')
	}
	out = append(out, f.lead...)
	out = append(out, `,"revision":`...)
	out = fmt.Appendf(out, "%d,", r.Revision)
	out = append(out, f.rest...)
	return append(out, '}'), nil
}
