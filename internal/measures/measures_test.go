package measures_test

import (
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/agetrajectory"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pfrcoverage"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/schooltier"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

//nolint:gochecknoglobals // a test flag must be registered before go test parses flags
var update = flag.Bool("update", false, "rewrite the Measure Dictionary from the registry")

const (
	measuresHeader = "measure,grain,unit,positions,meaning\n"
	sourcesHeader  = "source,name,status,max_age_days,host,path_prefix\n"
	fieldsHeader   = "source,field,measure,priority\n"
	feedsHeader    = "feed,source,url,first_season,id_column,id_type,season_column,week_column,filter\n"
)

// registryFS builds a registry from CSV bodies (header rows added).
func registryFS(ms, srcs, fields string) fstest.MapFS {
	return fstest.MapFS{
		"measures.csv":      {Data: []byte(measuresHeader + ms)},
		"sources.csv":       {Data: []byte(sourcesHeader + srcs)},
		"source_fields.csv": {Data: []byte(fieldsHeader + fields)},
		"feeds.csv":         {Data: []byte(feedsHeader)},
	}
}

const (
	okMeasure = "outcome.tackles_solo,week,count,DE DT LB CB S,Solo tackles.\n"
	okSource  = "pfr,Pro Football Reference,active,7,pro-football-reference.com,\n"
	okField   = "pfr,Solo,outcome.tackles_solo,1\n"
)

func TestEmbeddedRegistryLoads(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := reg.Measure("outcome.fantasy_points")
	if !ok || m.Grain != measures.GrainSeason || m.Family != measures.FamilyOutcome {
		t.Fatalf("outcome.fantasy_points = %+v, %v", m, ok)
	}
	if f, ok := reg.Field("mfl", playerscores.Field); !ok || f.Measure != m.Name {
		t.Fatalf("mfl %s maps to %+v, %v", playerscores.Field, f, ok)
	}
}

func TestLoadRejects(t *testing.T) {
	cases := []struct {
		name, measures, sources, fields, want string
	}{
		{"name without family", "tackles,week,count,all,x\n", okSource, "", "not family.name"},
		{"unknown family", "defense.tackles,week,count,all,x\n", okSource, "", "unknown family"},
		{"bad grain", "outcome.tackles,game,count,all,x\n", okSource, "", "grain"},
		{"two-word unit", "outcome.tackles,week,per game,all,x\n", okSource, "", "unit"},
		{"no meaning", "outcome.tackles,week,count,all, \n", okSource, "", "no meaning"},
		{"measure twice", okMeasure + okMeasure, okSource, "", "listed twice"},
		{"unknown position", "outcome.tackles,week,count,EDGE,x\n", okSource, "", "unknown or repeated"},
		{"source twice", okMeasure, okSource + okSource, "", "listed twice"},
		{"bad status", okMeasure, "pfr,P,lost,7,pfr.com,\n", "", "want active or retired"},
		{"zero max age", okMeasure, "pfr,P,active,0,pfr.com,\n", "", "max_age_days"},
		{"upper-case host", okMeasure, "pfr,P,active,7,PFR.com,\n", "", "lower-case host"},
		{"relative path", okMeasure, "pfr,P,active,7,pfr.com,data/\n", "", "must start with /"},
		{"unknown source", okMeasure, okSource, "espn,Solo,outcome.tackles_solo,1\n", "unknown source"},
		{"unknown measure", okMeasure, okSource, "pfr,Solo,outcome.sacks,1\n", "unknown measure"},
		{"zero priority", okMeasure, okSource, "pfr,Solo,outcome.tackles_solo,0\n", "priority"},
		{"field twice", okMeasure, okSource, okField + okField, "mapped twice"},
		{"two fields one measure", okMeasure, okSource, okField + "pfr,Tkl,outcome.tackles_solo,2\n", "one source, one field"},
		{"shared priority", okMeasure, okSource + "espn,ESPN,active,7,espn.com,\n",
			okField + "espn,soloTackles,outcome.tackles_solo,1\n", "share priority"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := measures.Load(registryFS(c.measures, c.sources, c.fields))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Load error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestFeeds(t *testing.T) {
	const (
		weekMeasure   = "outcome.sacks,week,count,DE DT LB CB S,Sacks.\n"
		playerMeasure = "prior.forty,player,seconds,all,40 time.\n"
		source        = "nflverse,nflverse,active,14,github.com,/nflverse/\n"
		fields        = "nflverse,stats.def_sacks,outcome.sacks,1\nnflverse,combine.forty,prior.forty,1\n"
		statsFeed     = "stats,nflverse,https://github.com/nflverse/s_{season}.csv,2021,player_id,gsis,season,week,season_type=REG&position!=OL|LS\n"
		combineFeed   = "combine,nflverse,https://github.com/nflverse/combine.csv,,pfr_id,pfr,,,\n"
	)
	load := func(feeds string) (*measures.Registry, error) {
		fsys := registryFS(weekMeasure+playerMeasure, source, fields)
		fsys["feeds.csv"] = &fstest.MapFile{Data: []byte(feedsHeader + feeds)}
		return measures.Load(fsys)
	}
	reg, err := load(statsFeed + combineFeed)
	if err != nil {
		t.Fatal(err)
	}
	stats, _ := reg.Feed("stats")
	if stats.Grain() != measures.GrainWeek || stats.URLFor(2024) != "https://github.com/nflverse/s_2024.csv" {
		t.Errorf("stats feed = %+v", stats)
	}
	row := map[string]string{"season_type": "REG", "position": "DE"}
	if !stats.Keep(func(c string) string { return row[c] }) {
		t.Error("a REG DE row was dropped")
	}
	for col, v := range map[string]string{"season_type": "POST", "position": "OL"} {
		r := map[string]string{"season_type": "REG", "position": "DE", col: v}
		if stats.Keep(func(c string) string { return r[c] }) {
			t.Errorf("a row with %s=%s was kept", col, v)
		}
	}
	if got := reg.Columns(stats); strings.Join(got, ",") != "def_sacks,player_id,position,season,season_type,week" {
		t.Errorf("Columns = %v", got)
	}
	combine, _ := reg.Feed("combine")
	if combine.Grain() != measures.GrainPlayer || len(reg.FeedFields(combine)) != 1 {
		t.Errorf("combine feed = %+v", combine)
	}

	for name, c := range map[string]struct{ feeds, want string }{
		"season token without first season": {strings.Replace(statsFeed, ",2021,", ",,", 1) + combineFeed, "first_season"},
		"URL outside its source":            {statsFeed + strings.Replace(combineFeed, "github.com/nflverse", "example.com", 1), "not under source"},
		"MFL id type":                       {statsFeed + strings.Replace(combineFeed, ",pfr,", ",mfl,", 1), "non-MFL id type"},
		"week without season":               {strings.Replace(statsFeed, ",season,week,", ",,week,", 1) + combineFeed, "no season column"},
		"grain mismatch":                    {statsFeed + strings.Replace(combineFeed, ",pfr,,,", ",pfr,season,,", 1), "season-grain feed"},
		"feed without fields":               {statsFeed + combineFeed + strings.Replace(combineFeed, "combine,", "orphan,", 1), "no source_fields rows"},
		"bad filter":                        {strings.Replace(statsFeed, "season_type=REG", "season_type", 1) + combineFeed, "not column=values"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := load(c.feeds); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Load error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestValidPeriod(t *testing.T) {
	for _, c := range []struct {
		grain        measures.Grain
		season, week int
		want         bool
	}{
		{measures.GrainPlayer, 0, 0, true}, {measures.GrainPlayer, 2024, 0, false},
		{measures.GrainSeason, 2024, 0, true}, {measures.GrainSeason, 2024, 3, false}, {measures.GrainSeason, 0, 0, false},
		{measures.GrainWeek, 2024, 3, true}, {measures.GrainWeek, 2024, 0, false}, {measures.GrainWeek, 0, 3, false},
	} {
		if got := (measures.Measure{Grain: c.grain}).ValidPeriod(c.season, c.week); got != c.want {
			t.Errorf("%s ValidPeriod(%d, %d) = %v", c.grain, c.season, c.week, got)
		}
	}
}

func TestLoadRejectsWrongHeader(t *testing.T) {
	fsys := registryFS(okMeasure, okSource, "")
	fsys["sources.csv"] = &fstest.MapFile{Data: []byte("id,name,status,max_age_days,host,path_prefix\n" + okSource)}
	if _, err := measures.Load(fsys); err == nil || !strings.Contains(err.Error(), "header must be") {
		t.Fatalf("Load error = %v, want a header complaint", err)
	}
}

func TestFieldsFeedingIsInPriorityOrder(t *testing.T) {
	reg, err := measures.Load(registryFS(okMeasure, okSource+"espn,ESPN,active,7,espn.com,\n",
		"espn,soloTackles,outcome.tackles_solo,2\n"+okField))
	if err != nil {
		t.Fatal(err)
	}
	got := reg.FieldsFeeding("outcome.tackles_solo")
	if len(got) != 2 || got[0].Source != "pfr" || got[1].Source != "espn" {
		t.Fatalf("FieldsFeeding = %+v, want pfr then espn", got)
	}
}

func TestEveryFetchedURLHasASource(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"https://api.myfantasyleague.com/2026/export?TYPE=league":    "mfl",
		"https://www47.myfantasyleague.com/2026/export?TYPE=rosters": "mfl",
		ras.SourceURL:               "nflverse",
		pfrcoverage.SourceURL:       "nflverse",
		agetrajectory.SourceURL:     "nflverse",
		crosswalk.SourceURL:         "dynastyprocess",
		collegeshare.SeasonStatsURL: "cfbd",
		schooltier.TeamsURL:         "cfbd",
		madden.RatingsURL:           "madden",
	}
	for raw, want := range cases {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got, ok := reg.SourceFor(u); !ok || got != want {
			t.Errorf("SourceFor(%s) = %q, %v; want %q", raw, got, ok, want)
		}
	}
	other, _ := url.Parse("https://github.com/someone/else/file.csv")
	if got, ok := reg.SourceFor(other); ok {
		t.Errorf("an unrelated GitHub URL matched source %q", got)
	}
}

// TestDictionaryIsCurrent fails when the checked-in Measure Dictionary differs from the
// registry. `make measure-dictionary` regenerates it.
func TestDictionaryIsCurrent(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", measures.DictionaryPath)
	want := reg.Dictionary()
	if *update {
		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run make measure-dictionary)", err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale: run make measure-dictionary", measures.DictionaryPath)
	}
}
