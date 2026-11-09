package feeds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func registry(t *testing.T, url string) *measures.Registry {
	t.Helper()
	reg, err := measures.Load(fstest.MapFS{
		"measures.csv": {Data: []byte("measure,grain,unit,positions,meaning\n" +
			"outcome.sacks,week,count,DE DT LB CB S,Sacks.\n" +
			"outcome.solo_tackles,week,count,DE DT LB CB S,Solo tackles.\n" +
			"context.team,week,text,all,Team.\n" +
			"prior.forty,player,seconds,all,40 time.\n")},
		"sources.csv": {Data: []byte("source,name,status,max_age_days,host,path_prefix\n" +
			"nflverse,nflverse,active,14,127.0.0.1,\n")},
		"source_fields.csv": {Data: []byte("source,field,measure,priority\n" +
			"nflverse,stats.def_sacks,outcome.sacks,1\n" +
			"nflverse,stats.def_tackles_solo,outcome.solo_tackles,1\n" +
			"nflverse,stats.team,context.team,1\n" +
			"nflverse,combine.forty,prior.forty,1\n")},
		"feeds.csv": {Data: []byte("feed,source,url,first_season,id_column,id_type,season_column,week_column,filter\n" +
			"stats," + "nflverse," + url + "/stats_{season}.csv,2021,player_id,gsis,season,week,season_type=REG\n" +
			"combine,nflverse," + url + "/combine.csv,,pfr_id,pfr,,,\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func serve(t *testing.T, files map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func facts(b measures.Batch) string {
	var out []string
	for _, f := range b.Facts {
		out = append(out, strings.Join([]string{f.IDType, f.ID, itoa(f.Season), itoa(f.Week), f.Field, f.Raw}, " "))
	}
	return strings.Join(out, "\n")
}

func itoa(i int) string { return strconv.Itoa(i) }

func TestReadWeekFeed(t *testing.T) {
	url := serve(t, map[string]string{"/stats_2024.csv": "player_id,season,week,season_type,def_sacks,team\n" +
		"00-1,2024,3,REG,1.5,BUF\n" +
		"00-2,2024,3,REG,0,MIA\n" +
		"NA,2024,3,REG,2,MIA\n" +
		"00-1,2024,19,POST,1,BUF\n"})
	reg := registry(t, url)
	f, _ := reg.Feed("stats")
	res, err := Read(context.Background(), http.DefaultClient, reg, f, 2024)
	if err != nil {
		t.Fatal(err)
	}
	want := "gsis 00-1 2024 3 stats.def_sacks 1.5\ngsis 00-1 2024 3 stats.team BUF\ngsis 00-2 2024 3 stats.team MIA"
	if got := facts(res.Batch); got != want {
		t.Errorf("facts =\n%s\nwant\n%s", got, want)
	}
	if res.Rows != 2 || len(res.Missing) != 1 || res.Missing[0] != "def_tackles_solo" {
		t.Errorf("rows %d, missing %v", res.Rows, res.Missing)
	}
	if res.Batch.Scope == nil || res.Batch.Scope.Seasons[0] != 2024 || strings.Join(res.Batch.Scope.Measures, ",") != "outcome.sacks" {
		t.Errorf("scope = %+v; a missing column must not be in it", res.Batch.Scope)
	}
	if res.Batch.Source != "nflverse" || len(res.Batch.BodySHA256) != 64 {
		t.Errorf("batch source %q sha %q", res.Batch.Source, res.Batch.BodySHA256)
	}
}

func TestReadPlayerFeedKeepsTheLaterRow(t *testing.T) {
	url := serve(t, map[string]string{"/combine.csv": "season,pfr_id,forty\n2020,SmitJo00,4.61\n2021,SmitJo00,4.55\n2021,DoeJa00,NA\n"})
	reg := registry(t, url)
	f, _ := reg.Feed("combine")
	res, err := Read(context.Background(), http.DefaultClient, reg, f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := facts(res.Batch); got != "pfr SmitJo00 0 0 combine.forty 4.55" {
		t.Errorf("facts = %q", got)
	}
	if res.Batch.Scope != nil {
		t.Error("a player-grain file has no week scope")
	}
}

func TestReadFailsLoudly(t *testing.T) {
	url := serve(t, map[string]string{
		"/stats_2023.csv": "player,season,week,season_type\nx,2023,1,REG\n",
		"/stats_2022.csv": "player_id,season,week,season_type\nx,2022,wk1,REG\n",
	})
	reg := registry(t, url)
	f, _ := reg.Feed("stats")
	for season, want := range map[int]string{2023: `no "player_id" column`, 2022: "week", 2021: "answered 404"} {
		if _, err := Read(context.Background(), http.DefaultClient, reg, f, season); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("season %d: err = %v, want %q", season, err, want)
		}
	}
}
