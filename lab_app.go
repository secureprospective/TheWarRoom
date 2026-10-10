package main

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type ReadingStatus string

const (
	ReadingOK       ReadingStatus = "ok"
	ReadingThin     ReadingStatus = "thin"
	ReadingFailed   ReadingStatus = "failed"
	ReadingNotWired ReadingStatus = "not_wired"
)

type Reading[T any] struct {
	snapshot.Sourced[T]
	Status        ReadingStatus `json:"status"`
	AbsenceReason *string       `json:"absenceReason"`
	Count         int           `json:"count"`
	Threshold     int           `json:"threshold"`
	Baseline      string        `json:"baseline"`
	Cutoff        string        `json:"cutoff"`
	Method        string        `json:"method"`
	InputSHA      string        `json:"inputSHA"`
	BuildStamp    string        `json:"buildStamp"`
	HowCounted    string        `json:"howCounted"`
}

type RawMarketScope struct {
	From   int      `json:"from"`
	To     int      `json:"to"`
	Phases []string `json:"phases"`
	Assets []string `json:"assets"`
	ByAge  bool     `json:"byAge"`
}

func (r RawMarketScope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To {
		return fmt.Errorf("market scope: invalid season range")
	}
	allowed := marketPhases()
	seen := make(map[string]bool)
	for _, p := range r.Phases {
		if !slices.Contains(allowed, p) || seen[p] {
			return fmt.Errorf("market scope: invalid or duplicate phase %q", p)
		}
		seen[p] = true
	}
	seen = make(map[string]bool)
	for _, asset := range r.Assets {
		if !slices.Contains(marketAssets(), asset) || seen[asset] {
			return fmt.Errorf("market scope: invalid or duplicate asset %q", asset)
		}
		seen[asset] = true
	}
	return nil
}

func (a *App) TargetMarket(raw RawMarketScope) Reading[lab.Market] {
	if err := raw.Validate(); err != nil {
		return failedMarket(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedMarket(err)
	}
	return marketReading(d, lab.Scope{From: raw.From, To: raw.To, Phases: slices.Clone(raw.Phases), ByAge: raw.ByAge})
}

// lab returns the App's shared decode, or a fresh one for an App built without NewApp.
func (a *App) lab() (*lab.Data, error) {
	if a.labData == nil {
		d, err := lab.Load()
		if err != nil {
			return nil, fmt.Errorf("load lab facts: %w", err)
		}
		return d, nil
	}
	return a.labData()
}

func failedMarket(err error) Reading[lab.Market] {
	reason := err.Error()
	return Reading[lab.Market]{Status: ReadingFailed, AbsenceReason: &reason, Threshold: lab.MinWhole, Method: "ridge market fit; frozen compiled facts", InputSHA: lab.DataSHA}
}
func marketReading(d *lab.Data, s lab.Scope) Reading[lab.Market] {
	value := lab.LeagueMarket(d, s)
	r := Reading[lab.Market]{Sourced: snapshot.Sourced[lab.Market]{Value: value, Provenance: snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: d.Built, Note: "Frozen historical input; not current league truth"}}}, Status: ReadingOK, Count: value.Trades, Threshold: lab.MinWhole, Baseline: "Median traded position or pick cost per delivered win; age splits keep this yardstick", Cutoff: d.ArchiveThrough, Method: "ridge least-squares with partial-pivot Gauss and sandwich covariance", InputSHA: lab.DataSHA, BuildStamp: buildLabel() + " " + buildDate, HowCounted: "Valued two-sided trades in the selected seasons and league-year stretches. Prices are anchored to a next-draft first-round pick. At least 80 trades for the whole view, 60 per half or window, and 8 delivery records with at least 0.5 forecast wins. Delivered wins are compared with the forecast at trade time. Ranges carry price and delivery uncertainty. The going rate is the asset-count-weighted median across positions and picks, never a live quote."}
	if value.Rows == nil {
		reason := "Fewer than 80 valued trades in this scope"
		r.Status = ReadingThin
		r.AbsenceReason = &reason
	}
	return r
}
