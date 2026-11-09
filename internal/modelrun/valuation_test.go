package modelrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// The exposed valuation must be the run's own evaluation, not a second one: for the same book and
// spec every player's score has to be bit-identical to what the run writes. A refit, a changed
// scale or a reordered blend would separate them.
func TestValuationIsTheSameEvaluationTheRunWrites(t *testing.T) {
	h, st, dir := league()
	spec := Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17}
	r, err := New(st, Held{dir, h}, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Run(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	book, err := Held{dir, h}.Book(context.Background(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewValuation(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	written := map[string]string{}
	for _, sc := range h.written[0].Scores {
		written[sc.MFLID] = fmt.Sprintf("%#v", sc)
	}
	if len(written) != 3 {
		t.Fatalf("the run wrote %d scores", len(written))
	}
	for mfl := range written {
		// An MFL link the record holds no player for resolves to the empty id, exactly as the run
		// resolves it, and is refused for the same reason.
		id, _ := v.Book().ID(mfl)
		got, reason := v.Value(id)
		if reason != "" {
			// The run records an excluded player as unavailable with the same reason.
			if want := fmt.Sprintf("%#v", excludedScore(mfl, reason)); want != written[mfl] {
				t.Errorf("%s refused as %q; the run wrote %s", mfl, reason, written[mfl])
			}
			continue
		}
		if fmt.Sprintf("%#v", got) != written[mfl] {
			t.Errorf("%s valuation = %#v\n   run wrote = %s", mfl, got, written[mfl])
		}
	}
}

func TestValuationWithoutABookRefusesRatherThanValuingNothing(t *testing.T) {
	v, err := NewValuation(playerBookZero(), Spec{Season: 2026, Params: params.DefaultSet()})
	if err != nil {
		t.Fatal(err)
	}
	if _, reason := v.Value("player.whoever"); reason == "" {
		t.Error("a valuation with no record read must say so, not return a value")
	}
}

func excludedScore(mfl, reason string) history.ModelScore {
	return history.ModelScore{MFLID: mfl, InputStatus: model.InputStatus{Unavailable: true, Reason: reason}}
}

func playerBookZero() playerrecord.Book { return playerrecord.Book{} }

func TestMirrorValuationBytesPreserved(t *testing.T) {
	h, st, dir := league()
	spec := Spec{Season: 2026, AsOf: time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC), Params: params.DefaultSet(), LastWeek: 17}
	st.roster[0].Salary, st.roster[0].ExpirationYear = 100, 2028
	r, err := New(st, Held{dir, h}, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(h.written[0].Scores)
	if err != nil {
		t.Fatal(err)
	}
	const baseline = "4dfa2da7e15ae4800bc66d9bbe1717382ac912f75b9eaa62fb8f1fa60f2f4ae7"
	if got := fmt.Sprintf("%x", sha256.Sum256(before)); got != baseline {
		t.Fatalf("mirror baseline changed: %s", got)
	}
	t.Logf("mirror baseline sha256=%s bytes=%d", baseline, len(before))
	book, err := (Held{dir, h}).Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewValuation(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	scores, err := v.Scores(st, spec)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(scores)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("mirror scores changed: before %s after %s", before, after)
	}
}
