package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedPulseReads(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer, proof := f.directCommissioner(t)
	reads := []struct{ command, args string }{
		{"native.directory.read", `{"league":"native.hosted"}`},
		{"schedule.read", `{"league":"native.hosted"}`},
		{"week.read", `{"league":"native.hosted","franchise":"franchise.01","season":2026,"localZone":"UTC"}`},
		{"projections.read", `{"league":"native.hosted","players":"0100,0099","season":2026,"week":1}`},
		{"bracket.read", `{"league":"native.hosted","season":2026}`},
	}
	for _, read := range reads {
		t.Run(read.command, func(t *testing.T) {
			raw := workflowRequest(t, read.command, "read", read.args, "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
			if status != 200 {
				t.Fatalf("status %d: %s", status, result)
			}
			var response serverhttp.WorkflowResponse
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
			var data struct {
				Schedule struct {
					League string `json:"league"`
				} `json:"schedule"`
				League  string `json:"league"`
				Players []struct {
					Player string `json:"player"`
				} `json:"players"`
			}
			if err := json.Unmarshal(response.Data, &data); err != nil {
				t.Fatal(err)
			}
			if data.League != "native.hosted" && (read.command != "week.read" || data.Schedule.League != "native.hosted") {
				t.Fatalf("wrong reading: %s", response.Data)
			}
			if read.command == "projections.read" && (len(data.Players) != 2 || data.Players[0].Player != "0099" || data.Players[1].Player != "0100") {
				t.Fatalf("players not normalized: %s", response.Data)
			}
		})
	}
	t.Run("forbidden league", func(t *testing.T) {
		raw := workflowRequest(t, "native.directory.read", "read", `{"league":"native.other"}`, "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		status, _ := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
		if status == 200 {
			t.Fatal("unauthorized reading accepted")
		}
	})
	for _, read := range []struct{ command, args string }{
		{"week.read", `{"league":"native.hosted","franchise":"franchise.01","season":2026,"localZone":"Local"}`},
		{"week.read", `{"league":"native.hosted","franchise":"franchise.01","season":2026,"localZone":"Not/AZone"}`},
		{"week.read", `{"league":"native.hosted","franchise":"franchise.01","season":2026,"localZone":"UTC","at":"bad"}`},
		{"projections.read", `{"league":"native.hosted","players":"99","season":2026,"week":1}`},
		{"projections.read", `{"league":"native.hosted","players":"0099,,0100","season":2026,"week":1}`},
		{"projections.read", `{"league":"native.hosted","players":"` + strings.Repeat("0099,", 120) + `0100","season":2026,"week":1}`},
	} {
		t.Run("bad/"+read.command+"/"+read.args, func(t *testing.T) {
			raw := workflowRequest(t, read.command, "read", read.args, "")
			if err := raw.Validate(); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestHostedNFLGamesReadsTheServerBoardForAMemberOnly(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer, proof := f.directCommissioner(t)
	read := func(league string) (int, leagueservice.NFLGames) {
		t.Helper()
		raw := workflowRequest(t, "nfl.games.read", "read", `{"league":"`+league+`"}`, "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
		var games leagueservice.NFLGames
		if status == 200 {
			var response serverhttp.WorkflowResponse
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(response.Data, &games); err != nil {
				t.Fatal(err)
			}
		}
		return status, games
	}
	if status, games := read("native.hosted"); status != 200 || games.League != "native.hosted" || len(games.Board.Games) != 0 || games.Board.Games == nil {
		t.Fatalf("empty board: status %d %+v", status, games)
	}
	f.games.Set([]livegame.Game{{ID: "401", Status: "in", Home: livegame.Team{ID: "12", Abbreviation: "KC", Score: 14}, Away: livegame.Team{ID: "2", Abbreviation: "BUF", Score: 10}}}, time.Now(), "ESPN")
	status, games := read("native.hosted")
	if status != 200 || len(games.Board.Games) != 1 || games.Board.Games[0].Home.Abbreviation != "KC" || games.Board.Source != "ESPN" || games.Board.Stale {
		t.Fatalf("held board not read: status %d %+v", status, games)
	}
	if status, _ := read("native.other"); status == 200 {
		t.Fatal("a league the account does not belong to read the board")
	}
}

func TestHostedLeagueWeekAdmission(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer, proof := f.directCommissioner(t)
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "outsider")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code, Username: "outsider", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	outsider := f.login(t, "outsider", "private-password")
	owner := f.login(t, "owner", "private-password")
	for _, tc := range []struct {
		name, bearer, proof, franchise string
		status                         int
	}{
		{"commissioner league-only", bearer, proof, "", 200},
		{"owner league-only", owner, "", "", 200},
		{"owner franchise", owner, "", "franchise.02", 200},
		{"signed in without league role", outsider, "", "", 403},
		{"anonymous", "", "", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := fmt.Sprintf(`{"league":"native.hosted","franchise":%q,"season":2026,"localZone":"UTC"}`, tc.franchise)
			raw := workflowRequest(t, "week.read", "read", args, "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, result := f.send(t, serverhttp.WorkflowPath, string(body), tc.bearer, tc.proof)
			if status != tc.status {
				t.Fatalf("status %d, want %d: %s", status, tc.status, result)
			}
			if status != 200 {
				return
			}
			var response serverhttp.WorkflowResponse
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
			var got commissioner.NativeWeek
			if err := json.Unmarshal(response.Data, &got); err != nil {
				t.Fatal(err)
			}
			if got.Current == nil || got.Current.Week != 1 || got.Schedule.Franchise != tc.franchise {
				t.Fatalf("wrong current week: %s", response.Data)
			}
			if tc.franchise == "" {
				for _, game := range append(got.Schedule.Games, *got.Current) {
					if game.Game != "" || game.Opponent != "" || game.Venue != "" || game.Kickoff == nil {
						t.Fatalf("private or missing timing: %+v", game)
					}
				}
			}
		})
	}
}
