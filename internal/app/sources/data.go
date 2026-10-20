package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/college"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type DataScope struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Note     string `json:"note"`
	Action   string `json:"action"`
	Progress string `json:"progress"`
}

func (a Service) FactsInput(ctx context.Context) (FactsInput, error) {
	in := FactsInput{Zone: time.Local}
	data, err := a.LabData()
	if err != nil {
		return in, fmt.Errorf("app: lab data: %w", err)
	}
	in.LabBuilt, in.LabThrough = data.Built, data.ArchiveThrough
	if in.Gaps, err = lab.FrozenGaps(); err != nil {
		return in, fmt.Errorf("app: frozen input gaps: %w", err)
	}
	rec, ok, err := a.Facts.LatestManifest(ctx, LabFrozenInputs)
	if err != nil || !ok {
		return in, err //nolint:wrapcheck // the store names the manifest
	}
	if in.Manifest, err = manifest.Decode(rec.Name, rec.SHA256, rec.Entries); err != nil {
		return in, err //nolint:wrapcheck // names the manifest
	}
	in.Published, in.PublishedAt = true, rec.PublishedAt
	shas := make([]string, len(in.Manifest.Entries))
	for i, e := range in.Manifest.Entries {
		shas[i] = e.SHA256
	}
	if in.Held, err = a.Facts.HeldBodies(ctx, shas); err != nil {
		return in, fmt.Errorf("app: checked facts: %w", err)
	}
	attempts, err := a.Facts.Snapshots(ctx, LabFrozenInputs)
	if err != nil {
		return in, fmt.Errorf("app: checked facts: %w", err)
	}
	in.Status = manifest.StatusOf(SnapshotAttempts(attempts), manifest.Want{ManifestSHA: rec.SHA256,
		Compiler: FactsCompiler, RulesHash: lab.CheckExceptions().SHA256})
	if len(attempts) > 0 {
		in.Latest = &attempts[0]
		if in.Latest.Checks != "" {
			if err := json.Unmarshal([]byte(in.Latest.Checks), &in.Results); err != nil {
				return in, fmt.Errorf("app: checked facts: stored check results: %w", err)
			}
		}
	}
	return in, nil
}

func SnapshotAttempts(rs []history.SnapshotRecord) []manifest.Attempt {
	out := make([]manifest.Attempt, len(rs))
	for i, r := range rs {
		out[i] = manifest.Attempt{ManifestSHA: r.ManifestSHA, Compiler: r.Compiler, RulesHash: r.RulesHash,
			ResultSHA: r.ResultSHA, Error: r.Error, BuiltAt: r.BuiltAt}
	}
	return out
}

func HistoryManifest(m manifest.Manifest, entries []byte) history.ManifestRecord {
	return history.ManifestRecord{SHA256: m.SHA256, Name: m.Name, Entries: entries}
}

func SignalsSummary(r SignalsReport) string {
	counts := map[string]int{}
	var added int
	for _, l := range r.Loads {
		added += l.Added
		counts[l.Status]++
	}
	summary := fmt.Sprintf("Looked at %s files: %s loaded, %s already current, %s skipped, %s failed; %s new values.",
		Thousands(len(r.Loads)), Thousands(counts[LoadDone]), Thousands(counts[LoadCurrent]),
		Thousands(counts[LoadSkipped]), Thousands(counts[LoadFailed]), Thousands(added))
	if r.CrosswalkNote != "" {
		summary += " " + r.CrosswalkNote
	}
	return summary
}

func DataWords() map[string]string {
	return map[string]string{
		"title": "Data and sources", "refresh": "Data refresh",
		"folder": "League archive folder (raw)", "setFolder": "Set folder",
		"read": "Read held records", "readProgress": "Refreshing…",
		"readResult": "Held records read; source health, signals, checked facts, player links and engine parameters updated.",
		"how":        "How this is counted", "crosswalkHow": "How players are linked",
		"parameters": "Engine parameters", "permission": "Changes require the admin role. Running as GM: values are read-only.",
		"find": "Find a setting", "empty": "No settings match.", "league": "League",
		"setting": "Setting and position", "value": "Value", "default": "Default", "override": "Override",
		"range": "Allowed", "overridden": "Overridden", "fitted": "Fitted", "handSet": "Hand-set",
		"apply": "Apply", "reset": "Reset", "clear": "Clear the Admin value and follow the shipped default",
		"kept":   "The last reading is kept.",
		"photos": "Player photos", "photosNote": "From ESPN, then Sleeper, cached on this computer for personal use. " +
			"Off means no photo is requested.", "photosOn": "Turn photos on", "photosOff": "Turn photos off",
	}
}

func DataScopes() []DataScope {
	return []DataScope{
		{"league", "League from MFL", "Rules, rosters and salary adjustments for the current season.", "Refresh League from MFL", "Refreshing…"},
		{"signals", "Signals", "The nflverse, CFBD and contract files that are due.", "Refresh Signals", "Refreshing…"},
		{"crosswalk", "Player crosswalk", "DynastyProcess's player ids, linked to the league's players.", "Refresh Player crosswalk", "Refreshing…"},
		{"archive", "League archive", "Files new to the 2013–2026 archive, then the Lab's pinned inputs.", "Refresh League archive", "Refreshing…"},
	}
}

func (a Service) HealthInput(ctx context.Context, photos *PhotoHealth) (HealthInput, error) {
	in := HealthInput{Reg: a.Health.Registry(), Now: time.Now(), Zone: time.Local}
	if a.CollegeKey == "" {
		in.Unset = map[string]string{college.Source: "needs a free CollegeFootballData key in " + CFBDEnvVar}
	}
	in.Photos = photos
	var err error
	if in.Sources, err = a.Health.SourceHealth(ctx); err != nil {
		return in, fmt.Errorf("app: source health: %w", err)
	}
	if in.MFL, err = a.Health.FetchHealth(ctx, "mfl"); err != nil {
		return in, fmt.Errorf("app: source health: %w", err)
	}
	engine, held, err := a.Health.LatestRun(ctx, a.Season, history.RunModel)
	if err != nil {
		return in, fmt.Errorf("app: engine health: %w", err)
	}
	if held {
		in.EngineAt = engine.AsOf
	}
	return in, nil
}

func Thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// View is the Sources reading without the root-owned settings and database checks.
type View struct {
	Health  report.Report
	Signals report.Report
	Facts   report.Report
	Checks  report.Report
	Scopes  []DataScope
	Words   map[string]string
}

// Read preserves the read order while the caller owns locks around held-state callbacks.
func (a Service) Read(ctx context.Context, photos *PhotoHealth, positions func() map[string]string, last func() SignalsReport) (View, error) {
	health, err := a.HealthInput(ctx, photos)
	if err != nil {
		return View{}, err
	}
	facts, err := a.FactsInput(ctx)
	if err != nil {
		return View{}, err
	}
	signals, err := a.SignalsView(ctx, positions())
	if err != nil {
		return View{}, err
	}
	held := last()
	return View{Health: HealthReport(health), Signals: SignalFilesReport(signals, held, time.Local), Facts: FactsReport(facts), Checks: ChecksReport(facts), Scopes: DataScopes(), Words: DataWords()}, nil
}
