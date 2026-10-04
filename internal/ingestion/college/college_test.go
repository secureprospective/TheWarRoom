package college

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/measures"
)

const body = `[
{"season":2023,"playerId":"1","team":"Ohio State","conference":"Big Ten","category":"receiving","statType":"YDS","stat":"1200"},
{"season":2023,"playerId":"2","team":"Ohio State","conference":"Big Ten","category":"receiving","statType":"YDS","stat":"300"},
{"season":2023,"playerId":"1","team":"Ohio State","conference":"Big Ten","category":"receiving","statType":"LONG","stat":"75"},
{"season":2023,"playerId":"3","team":"Akron","conference":"MAC","category":"receiving","statType":"YDS","stat":"500"}
]`

func TestFetchKeepsDirectoryPlayersWithTheirTeamTotals(t *testing.T) {
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Query().Get("year") != "2023" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	b, err := Fetch(context.Background(), srv.Client(), srv.URL, reg, "k", 2023, func(id string) bool { return id == "1" })
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range b.Facts {
		if f.IDType != IDType || f.ID != "1" || f.Season != 2023 {
			t.Errorf("fact %+v", f)
		}
		got = append(got, f.Field+"="+f.Raw)
	}
	slices.Sort(got)
	want := "player_season.conference=Big Ten,player_season.receiving.YDS=1200,player_season.team=Ohio State,team_season.receiving.YDS=1500"
	if strings.Join(got, ",") != want {
		t.Errorf("facts = %v\nwant %s", got, want)
	}
	if auth != "Bearer k" || b.Source != Source || len(b.BodySHA256) != 64 {
		t.Errorf("auth %q source %q sha %q", auth, b.Source, b.BodySHA256)
	}
}
