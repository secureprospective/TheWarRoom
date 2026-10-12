package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// DataActResult is the outcome of one Control Room data Act, in plain words.
type DataActResult struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

// DataScope is one part of the app's data a refresh Act can renew.
type DataScope struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

// DataSources is Control Room › Data and sources: source health and the signal files (C-07), the
// checked facts and the checks that will prove them (L-34), and the refresh scopes (S-09).
type DataSources struct {
	Health     Report      `json:"health"`
	Signals    Report      `json:"signals"`
	Facts      Report      `json:"facts"`
	Checks     Report      `json:"checks"`
	ArchiveDir string      `json:"archiveDir"`
	Scopes     []DataScope `json:"scopes"`
}

// dataScopes lists the refresh Acts in the order they are offered.
func dataScopes() []DataScope {
	return []DataScope{
		{"league", "League from MFL", "Rules, rosters and salary adjustments for the current season."},
		{"signals", "Signals", "The nflverse, CFBD and contract files that are due."},
		{"crosswalk", "Player crosswalk", "DynastyProcess's player ids, linked to the league's players."},
		{"archive", "League archive", "Files new to the 2013–2026 archive, then the Lab's pinned inputs."},
	}
}

// TargetDataRefresh renews one scope of the app's data and says what it did.
func (a *App) TargetDataRefresh(scope string) DataActResult {
	if err := a.ready(); err != nil {
		return DataActResult{Error: err.Error()}
	}
	summary, err := a.refreshScope(scope)
	if err != nil {
		return DataActResult{Error: err.Error()}
	}
	return DataActResult{OK: true, Summary: summary}
}

func (a *App) refreshScope(scope string) (string, error) {
	switch scope {
	case "league":
		r := a.RefreshLeague()
		if !r.OK {
			return "", fmt.Errorf("league refresh failed: %s", r.Error)
		}
		return leagueRefreshSummary(r), nil
	case "signals":
		ctx, cancel := context.WithTimeout(a.ctx, signalsTimeout)
		defer cancel()
		r := a.loadSignals(ctx)
		if r.Error != "" {
			return "", fmt.Errorf("signals refresh: %s; %s", r.Error, signalsSummary(r))
		}
		for _, l := range r.Loads {
			if l.Status == loadFailed {
				return "", fmt.Errorf("signals refresh: %s; %s", l.Error, signalsSummary(r))
			}
		}
		return signalsSummary(r), nil
	case "crosswalk":
		r := a.LoadCrosswalk()
		if !r.OK {
			return "", fmt.Errorf("crosswalk load failed: %s", r.Error)
		}
		return fmt.Sprintf("%s links; rostered players matched %s of %s, free agents %s of %s.", thousands(r.Links),
			thousands(r.Total.RosteredMatched), thousands(r.Total.Rostered), thousands(r.Total.FreeAgentsMatched),
			thousands(r.Total.FreeAgents)), nil
	case "archive":
		ctx, cancel := context.WithTimeout(a.ctx, archiveImportTimeout)
		defer cancel()
		return a.importLeagueArchive(ctx)
	}
	return "", fmt.Errorf("unknown data scope %q", scope)
}

func leagueRefreshSummary(r RefreshResult) string {
	changed, rules := "nothing new on MFL", "rules unchanged"
	if r.Changed {
		changed = "rosters and salary adjustments updated"
	}
	if r.RulesChanged {
		rules = "rules changed"
	}
	return fmt.Sprintf("Season %d: %s; %s.", r.Season, changed, rules)
}

func signalsSummary(r SignalsReport) string {
	counts := map[string]int{}
	var added int
	for _, l := range r.Loads {
		added += l.Added
		counts[l.Status]++
	}
	return fmt.Sprintf("Looked at %s files: %s loaded, %s already current, %s skipped, %s failed; %s new values.",
		thousands(len(r.Loads)), thousands(counts[loadDone]), thousands(counts[loadCurrent]),
		thousands(counts[loadSkipped]), thousands(counts[loadFailed]), thousands(added))
}

// TargetDataSources reads what history holds about every source and the checked facts.
func (a *App) TargetDataSources() (DataSources, error) {
	if err := a.ready(); err != nil {
		return DataSources{}, fmt.Errorf("data and sources: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	settings, err := a.readSettings()
	if err != nil {
		return DataSources{}, err
	}
	health, err := a.healthInput(ctx)
	if err != nil {
		return DataSources{}, err
	}
	facts, err := a.factsInput(ctx)
	if err != nil {
		return DataSources{}, err
	}
	signals, err := a.signalsView(ctx)
	if err != nil {
		return DataSources{}, err
	}
	a.signalsMu.Lock()
	last := a.signals
	a.signalsMu.Unlock()
	return DataSources{
		Health: healthReport(health), Signals: signalFilesReport(signals, last, time.Local),
		Facts: factsReport(facts), Checks: checksReport(facts.checks),
		ArchiveDir: settings.LeagueArchiveDir, Scopes: dataScopes(),
	}, nil
}

func (a *App) healthInput(ctx context.Context) (healthInput, error) {
	in := healthInput{reg: a.history.Registry(), now: time.Now(), zone: time.Local}
	var err error
	if in.sources, err = a.history.SourceHealth(ctx); err != nil {
		return in, fmt.Errorf("app: source health: %w", err)
	}
	if in.mfl, err = a.history.FetchHealth(ctx, "mfl"); err != nil {
		return in, fmt.Errorf("app: source health: %w", err)
	}
	run, ok, err := a.history.LatestRun(ctx, a.season, history.RunBoard)
	if err != nil {
		return in, fmt.Errorf("app: source health: %w", err)
	}
	if ok {
		in.boardRun, in.missing = true, run.MissingMeasures
	}
	return in, nil
}

func (a *App) factsInput(ctx context.Context) (factsInput, error) {
	in := factsInput{zone: time.Local}
	data, err := a.labData()
	if err != nil {
		return in, fmt.Errorf("app: lab data: %w", err)
	}
	in.labBuilt, in.labThrough = data.Built, data.ArchiveThrough
	if in.checks, err = labSourceChecks(data.Raw.Source); err != nil {
		return in, err
	}
	rec, ok, err := a.history.LatestManifest(ctx, labFrozenInputs)
	if err != nil || !ok {
		return in, err //nolint:wrapcheck // the store names the manifest
	}
	if in.manifest, err = manifest.Decode(rec.Name, rec.SHA256, rec.Entries); err != nil {
		return in, err //nolint:wrapcheck // names the manifest
	}
	in.published, in.publishedAt = true, rec.PublishedAt
	shas := make([]string, len(in.manifest.Entries))
	for i, e := range in.manifest.Entries {
		shas[i] = e.SHA256
	}
	if in.held, err = a.history.HeldBodies(ctx, shas); err != nil {
		return in, fmt.Errorf("app: checked facts: %w", err)
	}
	attempts, err := a.history.Snapshots(ctx, labFrozenInputs)
	if err != nil {
		return in, fmt.Errorf("app: checked facts: %w", err)
	}
	in.status = manifest.StatusOf(snapshotAttempts(attempts), manifest.Want{ManifestSHA: rec.SHA256})
	return in, nil
}

func snapshotAttempts(rs []history.SnapshotRecord) []manifest.Attempt {
	out := make([]manifest.Attempt, len(rs))
	for i, r := range rs {
		out[i] = manifest.Attempt{ManifestSHA: r.ManifestSHA, Compiler: r.Compiler, RulesHash: r.RulesHash,
			ResultSHA: r.ResultSHA, Error: r.Error, BuiltAt: r.BuiltAt}
	}
	return out
}

func historyManifest(m manifest.Manifest, entries []byte) history.ManifestRecord {
	return history.ManifestRecord{SHA256: m.SHA256, Name: m.Name, Entries: entries}
}

func (a *App) TargetCrosswalk() Report {
	c := a.GetCrosswalkReport()
	r := Report{ID: "C-08", Title: "Player crosswalk", Status: "ok", Source: "Source: DynastyProcess", AsOf: "As of: this launch",
		Columns: []string{"Position", "Rostered matched", "Free agents matched"}, Rows: []ReportRow{},
		How: "DynastyProcess links MFL players to NFL ids. Unmatched rostered players are listed with the reason; their facts wait for a link."}
	if !c.OK {
		r.Status, r.Note = "empty", "No crosswalk loaded since launch. Refresh the Player crosswalk scope to load it."
		if c.Error != "" {
			r.Status, r.Note = "failed", c.Error
		}
		return r
	}
	r.AsOf = "Loaded: " + labTime(c.LoadedAt, time.Local)
	r.Note = fmt.Sprintf("%s links; %s waiting facts resolved.", thousands(c.Links), thousands(c.Promoted))
	for _, rate := range append(c.Rates, c.Total) {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{rate.Position, crosswalkShare(rate.RosteredMatched, rate.Rostered), crosswalkShare(rate.FreeAgentsMatched, rate.FreeAgents)}})
	}
	for _, m := range c.Unmatched {
		r.Rows = append(r.Rows, ReportRow{Group: "Unmatched rostered players", Cells: []string{m.Name + " · " + m.MFLID, m.Position + " · " + m.Franchise, m.Reason}})
	}
	return r
}

func crosswalkShare(matched, total int) string {
	if total == 0 {
		return "None"
	}
	return fmt.Sprintf("%s of %s · %.1f%%", thousands(matched), thousands(total), 100*float64(matched)/float64(total))
}
