package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// ModelScore is one player's measurables in a model run, and the inputs that fed them.
type ModelScore struct {
	MFLID    string
	Position string
	model.Value
	Inputs []string
}

// NewModelRun is a model run to write; the fields mean what they do on NewRun.
type NewModelRun struct {
	Season     int
	AsOf       time.Time
	Params     ParamSet
	Engine     string
	InputsHash string
	Scores     []ModelScore
}

// WriteModelRun stores a model run and its scores, unless the latest model run for the season
// matches it in param set, engine, inputs, scores and missing measures.
func (s *Store) WriteModelRun(ctx context.Context, nr NewModelRun) (Run, bool, error) {
	switch {
	case len(nr.Scores) == 0:
		return Run{}, false, fmt.Errorf("history: model run for season %d has no scores", nr.Season)
	case nr.InputsHash == "" || nr.Engine == "":
		return Run{}, false, fmt.Errorf("history: model run for season %d needs its inputs hash and engine", nr.Season)
	}
	hash, err := validModelScores(nr.Scores)
	if err != nil {
		return Run{}, false, err
	}
	p, err := s.pending(ctx, NewRun{Kind: RunModel, Season: nr.Season, AsOf: nr.AsOf, Params: nr.Params,
		Engine: nr.Engine, InputsHash: nr.InputsHash}, hash)
	if err != nil {
		return Run{}, false, err
	}
	return s.writeRun(ctx, p, func(tx *sql.Tx, runID int64) error {
		for _, sc := range nr.Scores {
			inputs, err := json.Marshal(sc.Inputs)
			if err != nil {
				return fmt.Errorf("history: encode inputs of %s: %w", sc.MFLID, err)
			}
			v := sc.Value
			if _, err := tx.ExecContext(ctx, `INSERT INTO model_scores (run_id, `+modelCols+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, runID, sc.MFLID, sc.Position, v.Now, v.NowPPG,
				v.Dynasty, v.DynastyPPG, v.Prior, v.PastGames, v.SeasonGames, v.ZPast, v.ZNow, v.OnField,
				string(inputs)); err != nil {
				return fmt.Errorf("history: write model score %s in run %d: %w", sc.MFLID, runID, err)
			}
		}
		return nil
	})
}

// validModelScores rejects a malformed or repeated id and any non-finite field, and returns the
// sha256 of the scores in id order.
func validModelScores(scores []ModelScore) (string, error) {
	seen := make(map[string]bool, len(scores))
	for _, sc := range scores {
		id, err := playerid.New(sc.MFLID)
		if err != nil || id.String() != sc.MFLID {
			return "", fmt.Errorf("history: model score id %q is not a canonical MFL id", sc.MFLID)
		}
		if seen[sc.MFLID] {
			return "", fmt.Errorf("history: %s is scored twice in one model run", sc.MFLID)
		}
		seen[sc.MFLID] = true
		v := sc.Value
		if !numeric.Finite(v.Now, v.NowPPG, v.Dynasty, v.DynastyPPG, v.Prior, v.PastGames, v.SeasonGames,
			v.ZPast, v.ZNow, v.OnField) {
			return "", fmt.Errorf("history: %s has a non-finite model field", sc.MFLID)
		}
	}
	sorted := slices.Clone(scores)
	slices.SortFunc(sorted, func(a, b ModelScore) int { return strings.Compare(a.MFLID, b.MFLID) })
	enc, err := json.Marshal(sorted)
	if err != nil {
		return "", fmt.Errorf("history: encode model scores: %w", err)
	}
	sum := sha256.Sum256(enc)
	return hex.EncodeToString(sum[:]), nil
}

const modelCols = `mfl_id, position, now, now_ppg, dynasty, dynasty_ppg, prior,
	past_games, season_games, z_past, z_now, on_field, inputs`

// ModelScores returns a model run's scores, highest dynasty value first.
func (s *Store) ModelScores(ctx context.Context, runID int64) ([]ModelScore, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT `+modelCols+` FROM model_scores
WHERE run_id = ? ORDER BY dynasty_ppg DESC, mfl_id ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("history: model scores: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []ModelScore
	for rows.Next() {
		var sc ModelScore
		var inputs string
		v := &sc.Value
		if err := rows.Scan(&sc.MFLID, &sc.Position, &v.Now, &v.NowPPG, &v.Dynasty, &v.DynastyPPG, &v.Prior,
			&v.PastGames, &v.SeasonGames, &v.ZPast, &v.ZNow, &v.OnField, &inputs); err != nil {
			return nil, fmt.Errorf("history: scan model score: %w", err)
		}
		if err := json.Unmarshal([]byte(inputs), &sc.Inputs); err != nil {
			return nil, fmt.Errorf("history: model score %s inputs: %w", sc.MFLID, err)
		}
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: model scores rows: %w", err)
	}
	return out, nil
}
