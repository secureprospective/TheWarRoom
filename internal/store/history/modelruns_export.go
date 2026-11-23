package history

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ModelRunFile is one model run in portable form: the run, the parameters it scored with and its
// scores. Player ids are canonical MFL ids, the same ids the draft pool and DynastyDraftRanking use.
type ModelRunFile struct {
	Format     string       `json:"format"`
	RunID      int64        `json:"runId"`
	Season     int          `json:"season"`
	AsOf       time.Time    `json:"asOf"`
	Engine     string       `json:"engine"`
	InputsHash string       `json:"inputsHash"`
	Params     ParamSet     `json:"params"`
	Scores     []ModelScore `json:"scores"`
}

// ModelRunFileFormat names the file layout so a later change can refuse an older file.
const ModelRunFileFormat = "twr-model-run/1"

// ExportModelRun reads model run runID, or the newest model run when runID is 0, for writing to a file.
func (s *Store) ExportModelRun(ctx context.Context, runID int64) (ModelRunFile, error) {
	var run Run
	var ok bool
	var err error
	if runID == 0 {
		run, ok, err = s.latestModelRun(ctx)
	} else {
		run, ok, err = s.run(ctx, `WHERE run_id = ? AND kind = ?`, runID, string(RunModel))
	}
	if err != nil {
		return ModelRunFile{}, err
	}
	if !ok {
		return ModelRunFile{}, errors.New("history: no such model run to export")
	}
	params, ok, err := s.RunParams(ctx, run.ParamSetID)
	if err != nil {
		return ModelRunFile{}, err
	}
	if !ok {
		return ModelRunFile{}, fmt.Errorf("history: model run %d has no stored param set", run.ID)
	}
	scores, err := s.ModelScores(ctx, run.ID)
	if err != nil {
		return ModelRunFile{}, err
	}
	return ModelRunFile{Format: ModelRunFileFormat, RunID: run.ID, Season: run.Season, AsOf: run.AsOf,
		Engine: run.Engine, InputsHash: run.InputsHash, Params: params, Scores: scores}, nil
}

// ImportModelRun writes an exported run through WriteModelRun. held is true, and nothing is written,
// when this store already holds a model run with the same season, engine and inputs.
func (s *Store) ImportModelRun(ctx context.Context, f ModelRunFile) (run Run, held bool, err error) {
	if f.Format != ModelRunFileFormat {
		return Run{}, false, fmt.Errorf("history: model run file format %q is not %q", f.Format, ModelRunFileFormat)
	}
	prev, ok, err := s.run(ctx, `WHERE kind = ? AND season = ? AND engine = ? AND inputs_hash = ? ORDER BY run_id DESC LIMIT 1`,
		string(RunModel), f.Season, f.Engine, f.InputsHash)
	if err != nil {
		return Run{}, false, err
	}
	if ok {
		return prev, true, nil
	}
	run, wrote, err := s.WriteModelRun(ctx, NewModelRun{Season: f.Season, AsOf: f.AsOf, Params: f.Params,
		Engine: f.Engine, InputsHash: f.InputsHash, Scores: f.Scores})
	return run, !wrote, err
}
