package veteranfilm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// FTN columns (the charting), located by NAME in the header.
const (
	colFTNGame  = "nflverse_game_id"
	colFTNPlay  = "nflverse_play_id"
	colContest  = "is_contested_ball"
	colCreated  = "is_created_reception"
	colDrop     = "is_drop"
	colIntWorth = "is_interception_worthy"
	colThrowAwy = "is_throw_away"
)

// pbp columns (the gsis ids + join keys). Only these are bound — no fantasy/EPA/WP
// column is read, so the zero-leak constraint holds structurally.
const (
	colPBPPlay = "play_id"
	colPBPGame = "game_id"
	colPasser  = "passer_player_id"
	colReceivr = "receiver_player_id"
)

// playKey identifies a charted play across the two sources.
type playKey struct {
	game string
	play string
}

// ftnFlags is one charted play's booleans (0/1), split by the role each is attributed
// to: contested/created/drop to the play's receiver, intWorth/throwAway to the passer.
type ftnFlags struct {
	contested int
	created   int
	drop      int
	intWorth  int
	throwAway int
}

// loadFTN reads one season's FTN charting (small; buffered FetchCSV is fine) into a
// (game,play)->flags map. FTN is one row per play; a duplicate (game,play) is a
// source-integrity error and fails loud (trusted-shape policy).
func loadFTN(ctx context.Context, client *http.Client, url string) (map[playKey]ftnFlags, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("veteranfilm: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("veteranfilm: FTN %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0],
		colFTNGame, colFTNPlay, colContest, colCreated, colDrop, colIntWorth, colThrowAwy)
	if err != nil {
		return nil, fmt.Errorf("veteranfilm: %w", err)
	}

	out := make(map[playKey]ftnFlags, len(records))
	for _, rec := range records[1:] {
		key := playKey{game: rec[cols[colFTNGame]], play: rec[cols[colFTNPlay]]}
		if ingestion.IsMissing(key.game) || ingestion.IsMissing(key.play) {
			continue // an unidentifiable charted row cannot be joined
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("veteranfilm: FTN %q has duplicate play %s/%s", url, key.game, key.play)
		}
		f, err := parseFlags(cols, rec)
		if err != nil {
			return nil, err
		}
		out[key] = f
	}
	return out, nil
}

// parseFlags reads the five charted booleans for one FTN row (fail loud on a non-bool).
func parseFlags(cols map[string]int, rec []string) (ftnFlags, error) {
	var f ftnFlags
	var err error
	if f.contested, err = boolFlag(rec, cols[colContest], colContest); err != nil {
		return ftnFlags{}, err
	}
	if f.created, err = boolFlag(rec, cols[colCreated], colCreated); err != nil {
		return ftnFlags{}, err
	}
	if f.drop, err = boolFlag(rec, cols[colDrop], colDrop); err != nil {
		return ftnFlags{}, err
	}
	if f.intWorth, err = boolFlag(rec, cols[colIntWorth], colIntWorth); err != nil {
		return ftnFlags{}, err
	}
	if f.throwAway, err = boolFlag(rec, cols[colThrowAwy], colThrowAwy); err != nil {
		return ftnFlags{}, err
	}
	return f, nil
}

// recvAcc / passAcc are the per-player running counters (numerators + denominator).
type recvAcc struct {
	targets   int
	contested int
	created   int
	drops     int
}

type passAcc struct {
	attempts  int
	intWorth  int
	throwAway int
}

// aggregator holds the cross-season counters keyed by gsis.
type aggregator struct {
	recv map[string]*recvAcc
	pass map[string]*passAcc
}

func newAggregator() *aggregator {
	return &aggregator{recv: map[string]*recvAcc{}, pass: map[string]*passAcc{}}
}

// streamPBP streams one season's pbp and attributes each charted play's flags to the
// play's receiver (receiver traits) and passer (passer traits). A play absent from
// flags is uncharted and skipped; an empty role id contributes nothing to that role.
func (a *aggregator) streamPBP(ctx context.Context, client *http.Client, url string, flags map[playKey]ftnFlags) error {
	err := ingestion.StreamCSV(ctx, client, url, PBPMaxBytes,
		[]string{colPBPPlay, colPBPGame, colPasser, colReceivr},
		func(cols map[string]int, rec []string) error {
			key := playKey{game: rec[cols[colPBPGame]], play: rec[cols[colPBPPlay]]}
			f, charted := flags[key]
			if !charted {
				return nil // play was not charted by FTN
			}

			if recv := rec[cols[colReceivr]]; !ingestion.IsMissing(recv) {
				acc := a.recv[recv]
				if acc == nil {
					acc = &recvAcc{}
					a.recv[recv] = acc
				}
				acc.targets++
				acc.contested += f.contested
				acc.created += f.created
				acc.drops += f.drop
			}
			if pass := rec[cols[colPasser]]; !ingestion.IsMissing(pass) {
				acc := a.pass[pass]
				if acc == nil {
					acc = &passAcc{}
					a.pass[pass] = acc
				}
				acc.attempts++
				acc.intWorth += f.intWorth
				acc.throwAway += f.throwAway
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("veteranfilm: %w", err)
	}
	return nil
}

// finalize converts the pooled counters to rates, emitting a role group only when its
// charted-play count meets the floor. A player with neither group above its floor is
// omitted entirely.
func (a *aggregator) finalize(recvFloor, passFloor int) map[string]RawVeteranFilm {
	out := make(map[string]RawVeteranFilm)

	for gsis, r := range a.recv {
		if r.targets < recvFloor {
			continue
		}
		rec := out[gsis]
		rec.GSISID = gsis
		rec.Receiver = &ReceiverFilm{
			Targets:              r.targets,
			ContestedCatchRate:   rate(r.contested, r.targets),
			CreatedReceptionRate: rate(r.created, r.targets),
			DropRate:             rate(r.drops, r.targets),
		}
		out[gsis] = rec
	}

	for gsis, p := range a.pass {
		if p.attempts < passFloor {
			continue
		}
		rec := out[gsis]
		rec.GSISID = gsis
		rec.Passer = &PasserFilm{
			Attempts:               p.attempts,
			InterceptionWorthyRate: rate(p.intWorth, p.attempts),
			ThrowawayRate:          rate(p.throwAway, p.attempts),
		}
		out[gsis] = rec
	}
	return out
}

// rate is num/denom as a float64; denom is guaranteed > 0 by the floor (>= 1) at the
// only call sites.
func rate(num, denom int) float64 {
	return float64(num) / float64(denom)
}
