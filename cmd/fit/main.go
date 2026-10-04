// Command fit fits the model's parameters from a history database (plan Stage 6). It writes the
// fitted params the app ships with and the fit report:
//
//	go run ./cmd/fit -db ~/scratch/history.db
//
// Run it against a copy of history.db, never the live file: opening the store initializes it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/model/fit"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

func main() {
	path := flag.String("db", "", "path to a copy of history.db")
	first := flag.Int("first", 2021, "first NFL season to fit on")
	holdout := flag.Int("holdout", 2025, "the season every fit is scored on")
	out := flag.String("params", "internal/store/params/fitted.json", "where to write the fitted params")
	report := flag.String("report", "docs/fit/Fit_Report.md", "where to write the report")
	flag.Parse()
	if *path == "" {
		log.Fatal("fit: -db is required")
	}
	if err := run(context.Background(), *path, *first, *holdout, *out, *report); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, path string, first, holdout int, out, report string) error {
	obs, lastWeek, err := read(ctx, path, first, holdout)
	if err != nil {
		return err
	}
	results := fit.Run(model.Build(obs, lastWeek), first, holdout)
	file := params.FittedFile{Fitted: time.Now().UTC().Format("2006-01-02"), First: first, Holdout: holdout}
	for _, r := range results {
		for key, v := range r.Params.Values() {
			file.Params = append(file.Params, params.FittedValue{Key: key, Position: string(r.Position), Value: v})
		}
	}
	slices.SortFunc(file.Params, func(a, b params.FittedValue) int {
		if a.Position != b.Position {
			return compare(a.Position, b.Position)
		}
		return compare(a.Key, b.Key)
	})
	enc, err := json.MarshalIndent(file, "", " ")
	if err != nil {
		return fmt.Errorf("fit: encode params: %w", err)
	}
	if err := os.WriteFile(out, append(enc, '\n'), 0o600); err != nil {
		return fmt.Errorf("fit: write params: %w", err)
	}
	md := fit.Markdown(results, first, holdout, fmt.Sprintf("a history database holding %d values", len(obs)))
	if err := os.WriteFile(report, []byte(md), 0o600); err != nil {
		return fmt.Errorf("fit: write report: %w", err)
	}
	log.Printf("fit: %d positions, %d params, report %s", len(results), len(file.Params), report)
	return nil
}

// read loads every value the model reads: player facts, college seasons and NFL weeks. lastWeek
// is each season's last week of league scoring.
func read(ctx context.Context, path string, first, holdout int) ([]model.Obs, map[int]int, error) {
	pools, err := db.Open(ctx, path)
	if err != nil {
		return nil, nil, fmt.Errorf("fit: open %s: %w", path, err)
	}
	defer func() { _ = pools.Close() }()
	reg, err := measures.Embedded()
	if err != nil {
		return nil, nil, fmt.Errorf("fit: registry: %w", err)
	}
	h := history.New(pools, reg)
	if err := h.Initialize(ctx); err != nil {
		return nil, nil, fmt.Errorf("fit: open history: %w", err)
	}
	var obs []model.Obs
	lastWeek := map[int]int{}
	for season := range holdout + 1 {
		if season != 0 && season < first-8 {
			continue
		}
		feats, err := h.Features(ctx, history.FeatureQuery{AsOf: time.Now(), Season: season, Measures: model.Measures()})
		if err != nil {
			return nil, nil, fmt.Errorf("fit: read season %d: %w", season, err)
		}
		for _, f := range feats {
			obs = append(obs, model.Obs{Player: f.PlayerID, Season: f.Season, Week: f.Week, Measure: f.Measure, Value: f.Value, Text: f.Text})
			if f.Measure == "outcome.weekly_fantasy_points" {
				lastWeek[f.Season] = max(lastWeek[f.Season], f.Week)
			}
		}
	}
	return obs, lastWeek, nil
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
