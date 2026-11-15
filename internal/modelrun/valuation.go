package modelrun

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Valuation is one pass's evaluation on its own: the same params, scales and per-player rules a
// run uses, without a league-state read and without writing a run. It exists so a caller that
// has already assembled a book — a league valued from its own settings, a replay — can read the
// values the model gives without a second evaluation existing anywhere. It refits nothing.
type Valuation struct {
	pass pass
}

// NewValuation prepares the pass over book with spec's params. It is the same preparation Run
// makes, so a value read through it is the value the run writes.
func NewValuation(book playerrecord.Book, spec Spec) (Valuation, error) {
	p, err := newPass(book, spec)
	if err != nil {
		return Valuation{}, err
	}
	return Valuation{pass: p}, nil
}

// Value scores one player by our permanent id. reason is why he could not be valued, empty when
// he was; the score is then meaningless.
func (v Valuation) Value(id string) (score history.ModelScore, reason string) {
	if v.pass.book.Records == nil {
		return history.ModelScore{}, "no player record was read for this valuation"
	}
	sc, _, why := v.pass.value(id)
	return sc, why
}

// Book is the record the valuation reads, so a caller can resolve an outside link to our id
// without holding a second copy.
func (v Valuation) Book() playerrecord.Book { return v.pass.book }

// Season is the season the valuation values.
func (v Valuation) Season() int { return v.pass.season }

// Scores reads the roster through the run's own universe, evaluation and contract tilt.
// It writes nothing; unavailable players retain the run's refusal.
func (v Valuation) Scores(st state.Reader, spec Spec) ([]history.ModelScore, error) {
	r := Runner{state: st}
	universe, err := r.universe(v.pass.book, spec.FreeAgents)
	if err != nil {
		return nil, fmt.Errorf("valuation roster: %w", err)
	}
	scores := []history.ModelScore{}
	inputs := []playerInput{}
	excluded := []history.ModelScore{}
	for _, p := range universe {
		sc, in, reason := v.pass.value(p.id)
		if reason != "" {
			excluded = append(excluded, history.ModelScore{MFLID: p.mfl, InputStatus: model.InputStatus{Unavailable: true, Reason: reason}})
			continue
		}
		scores, inputs = append(scores, sc), append(inputs, in)
	}
	if err := r.tilt(v.pass.book, scores, inputs, spec, v.pass); err != nil {
		return nil, fmt.Errorf("valuation contracts: %w", err)
	}
	return append(scores, excluded...), nil
}
