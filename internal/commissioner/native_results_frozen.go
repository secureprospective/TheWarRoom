package commissioner

import (
	"encoding/json"
	"fmt"
)

// frozenResults is the encoded, viewer-independent body of a week's results. A kept reading is served to every
// phone polling the week; encoding its starters' stat lines (maps, re-sorted on every encode) cost more than the
// reading itself, so it is encoded once when kept.
type frozenResults struct {
	games, scores, missed json.RawMessage
}

// Freeze encodes the games, scores and missed lineups once. A frozen reading is shared read-only: its slices
// must not change afterwards. Evidence and Revision, set per read, are still encoded on every read.
func (r *NativeResults) Freeze() error {
	games, err := json.Marshal(r.Games)
	if err != nil {
		return fmt.Errorf("encode results games: %w", err)
	}
	scores, err := json.Marshal(r.Scores)
	if err != nil {
		return fmt.Errorf("encode results scores: %w", err)
	}
	missed, err := json.Marshal(r.Missed)
	if err != nil {
		return fmt.Errorf("encode results missed: %w", err)
	}
	r.frozen = &frozenResults{games: games, scores: scores, missed: missed}
	return nil
}

// Frozen reports whether the reading carries its encoded body (Freeze).
func (r NativeResults) Frozen() bool { return r.frozen != nil }

// resultsHead and resultsBye are the per-read fields of NativeResults' wire shape, in its field order and tags;
// the frozen body is spliced between them.
type resultsHead struct {
	Evidence  *ScoreEvidence `json:"evidence,omitempty"`
	League    string         `json:"league"`
	Season    int            `json:"season"`
	Week      int            `json:"week"`
	Revision  int64          `json:"revision"`
	Published bool           `json:"published"`
	Words     string         `json:"words"`
}

type resultsBye struct {
	Bye     string `json:"bye,omitempty"`
	ByeName string `json:"byeName,omitempty"`
}

// MarshalJSON encodes the reading; a frozen one reuses its encoded body, byte for byte what a fresh encode gives.
func (r NativeResults) MarshalJSON() ([]byte, error) {
	if r.frozen != nil {
		return r.EncodedJSON()
	}
	type plain NativeResults
	body, err := json.Marshal(plain(r))
	if err != nil {
		return nil, fmt.Errorf("encode results: %w", err)
	}
	return body, nil
}

// EncodedJSON is the reading's compact JSON. A frozen reading only encodes its per-read fields; callers that
// write the answer themselves (workflowJSON) use it to skip the encoder's re-compaction of MarshalJSON output.
func (r NativeResults) EncodedJSON() ([]byte, error) {
	if r.frozen == nil {
		return r.MarshalJSON()
	}
	head, err := json.Marshal(resultsHead{Evidence: r.Evidence, League: r.League, Season: r.Season, Week: r.Week,
		Revision: r.Revision, Published: r.Published, Words: r.Words})
	if err != nil {
		return nil, fmt.Errorf("encode results head: %w", err)
	}
	bye, err := json.Marshal(resultsBye{Bye: r.Bye, ByeName: r.ByeName})
	if err != nil {
		return nil, fmt.Errorf("encode results bye: %w", err)
	}
	f := r.frozen
	out := make([]byte, 0, len(head)+len(bye)+len(f.games)+len(f.scores)+len(f.missed)+32)
	out = append(out, head[:len(head)-1]...)
	out = append(out, `,"games":`...)
	out = append(out, f.games...)
	if len(bye) > 2 {
		out = append(out, ',')
		out = append(out, bye[1:len(bye)-1]...)
	}
	out = append(out, `,"scores":`...)
	out = append(out, f.scores...)
	out = append(out, `,"missed":`...)
	out = append(out, f.missed...)
	return append(out, '}'), nil
}
