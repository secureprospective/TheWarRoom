package serverhttp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
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

type fixedNFLInjuryFeed struct{ feed leagueservice.NFLInjuryFeed }

func (f fixedNFLInjuryFeed) ReadNFLInjuryFeed(context.Context) (leagueservice.NFLInjuryFeed, error) {
	return f.feed, nil
}

func (fixedNFLInjuryFeed) ResolveNFLReturnWeeks(rows []leaguefeed.Injury, _ []leagueweek.Week) []leaguefeed.Injury {
	return rows
}

func TestHostedNFLPlayerStatusIsAReportedMemberRead(t *testing.T) {
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	fetched := time.Now().UTC().Add(-time.Minute)
	feed := leagueservice.NFLInjuryFeed{Season: 2026, FetchedAt: fetched, LastAttemptAt: fetched, Available: true,
		Injuries: []leaguefeed.Injury{{ID: id, Status: "Out", Details: "Ankle", ExpReturn: "Oct 15, 2026"}}}
	f := newHostedFixtureWithNFLInjuryFeed(t, fixedNFLInjuryFeed{feed: feed}, true, true)
	commissioner, _ := f.directCommissioner(t)
	member := f.login(t, "owner", "private-password")
	status, body := sendNFLPlayerStatusRead(t, f, member, "", "native.hosted", "")
	if status != 200 {
		t.Fatalf("member status read: %d: %s", status, body)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	var data struct {
		League    string     `json:"league"`
		State     string     `json:"state"`
		FetchedAt *time.Time `json:"fetchedAt"`
		Players   []struct {
			Player      string `json:"player"`
			Designation string `json:"designation"`
			BodyPart    string `json:"bodyPart"`
		} `json:"players"`
	}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.League != "native.hosted" || data.State != "available" || data.FetchedAt == nil || !data.FetchedAt.Equal(fetched) ||
		len(data.Players) != 1 || data.Players[0].Player != "0099" || data.Players[0].Designation != "Out" || data.Players[0].BodyPart != "Ankle" ||
		strings.Contains(string(response.Data), "Oct 15, 2026") {
		t.Fatalf("unexpected public status facts: %s", response.Data)
	}
	if status, body := sendNFLPlayerStatusRead(t, f, member, "", "native.other", ""); status == 200 {
		t.Fatalf("non-member read succeeded: %s", body)
	}
	if status, body := sendAgentNFLPlayerStatusRead(t, f, commissioner); status != 200 || !strings.Contains(string(body), `"designation":"Out"`) {
		t.Fatalf("scoped agent status read: %d: %s", status, body)
	}
}

func sendNFLPlayerStatusRead(t *testing.T, f *hostedFixture, bearer, proof, league, football string) (int, []byte) {
	t.Helper()
	raw := workflowRequest(t, "nfl.player-status.read", "read", `{"league":"`+league+`"}`, football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	return f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
}

func sendAgentNFLPlayerStatusRead(t *testing.T, f *hostedFixture, bearer string) (int, []byte) {
	t.Helper()
	token := f.agentToken(t, bearer, []string{"nfl.player-status.read"}, "read")
	assets := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	assetsBody, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &assets})
	if err != nil {
		t.Fatal(err)
	}
	status, result := f.send(t, serverhttp.WorkflowPath, string(assetsBody), token.Bearer, "")
	if status != 200 {
		t.Fatalf("agent pin read: %d: %s", status, result)
	}
	var pin serverhttp.WorkflowResponse
	if err := json.Unmarshal(result, &pin); err != nil {
		t.Fatal(err)
	}
	return sendNFLPlayerStatusRead(t, f, token.Bearer, "", "native.hosted", pin.Football)
}

func TestHostedNFLGamesReadsTheServerBoardForAMemberOnly(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer, proof := f.directCommissioner(t)
	read := func(league string) (int, leagueservice.NFLGames, []byte) {
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
		return status, games, result
	}
	if status, games, _ := read("native.hosted"); status != 200 || games.League != "native.hosted" || len(games.Board.Games) != 0 || games.Board.Games == nil {
		t.Fatalf("empty board: status %d %+v", status, games)
	}
	summaryBody, err := os.ReadFile("../ingestion/livegame/testdata/espn-summary-401873061-kickoff.json")
	if err != nil {
		t.Fatal(err)
	}
	summary, err := livegame.ParseSummary(summaryBody)
	if err != nil {
		t.Fatal(err)
	}
	boardGame := summary
	boardGame.Home.Record, boardGame.Away.Record = "", ""
	f.games.Set([]livegame.Game{boardGame}, time.Now(), "ESPN")
	status, withoutRecords, before := read("native.hosted")
	if status != 200 || len(withoutRecords.Board.Games) != 1 || withoutRecords.Board.Games[0].Home.Record != "" ||
		strings.Contains(string(before), `"record"`) {
		t.Fatalf("NFL board filled records absent from its held source: status %d %s", status, before)
	}
	f.games.SetDetails(map[string]leagueservice.NFLGame{summary.ID: {Game: &summary, At: time.Now()}})
	status, games, body := read("native.hosted")
	if status != 200 || len(games.Board.Games) != 1 || games.Board.Games[0].Home.Record != "3-6" ||
		games.Board.Games[0].Away.Record != "5-4" || games.Board.Source != "ESPN" || games.Board.Stale {
		t.Fatalf("held NFL records not read from the summary: status %d %+v", status, games)
	}
	if strings.Contains(string(body), `"pace"`) {
		t.Fatalf("live NFL board invented a pace field: %s", body)
	}
	if status, _, _ := read("native.other"); status == 200 {
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

func TestHostedFranchiseScheduleReadAcceptsCatalogScopeForOwner(t *testing.T) {
	f := newHostedFixture(t, true, true)
	owner := f.login(t, "owner", "private-password")
	args := `{"league":"native.hosted","franchise":"franchise.02","season":2026,"localZone":"UTC"}`
	raw := workflowRequest(t, "schedule.franchise.read", "read", args, "")
	if err := raw.Validate(); err != nil {
		t.Fatalf("valid catalog scope rejected: %v", err)
	}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	status, result := f.send(t, serverhttp.WorkflowPath, string(body), owner, "")
	if status != 200 {
		t.Fatalf("owner schedule read status %d: %s", status, result)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatal(err)
	}
	var schedule commissioner.FranchiseSchedule
	if err := json.Unmarshal(response.Data, &schedule); err != nil {
		t.Fatal(err)
	}
	if schedule.League != "native.hosted" || schedule.Franchise != "franchise.02" || schedule.Season != 2026 {
		t.Fatalf("wrong owner schedule reading: %+v", schedule)
	}

	for _, invalid := range []string{
		`{"league":"native.hosted","franchise":"franchise.02","season":2026}`,
		`{"league":"native.hosted","franchise":"franchise.02","season":2026,"localZone":"UTC","at":"now"}`,
	} {
		request := workflowRequest(t, "schedule.franchise.read", "read", invalid, "")
		if err := request.Validate(); err == nil {
			t.Errorf("invalid schedule scope accepted: %s", invalid)
		}
	}
}

func TestHostedFranchiseScheduleReadStillRequiresLeagueMembership(t *testing.T) {
	f := newHostedFixture(t, true, true)
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "outsider")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code,
		Username: "outsider", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	outsider := f.login(t, "outsider", "private-password")
	raw := workflowRequest(t, "schedule.franchise.read", "read",
		`{"league":"native.hosted","franchise":"franchise.02","season":2026,"localZone":"UTC"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, bearer string
		status       int
	}{{"signed in without league role", outsider, 403}, {"anonymous", "", 401}} {
		t.Run(tc.name, func(t *testing.T) {
			status, result := f.send(t, serverhttp.WorkflowPath, string(body), tc.bearer, "")
			if status != tc.status {
				t.Fatalf("status %d, want %d: %s", status, tc.status, result)
			}
		})
	}
}

func TestHostedNFLGameRead(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer := f.login(t, "owner", "private-password")
	game := livegame.Game{ID: "401", Status: "in", State: &livegame.State{Possession: "12", Down: 2, Distance: 7, ToEndzone: 40}, Plays: []livegame.Play{{ID: "1", Text: "Live pass"}}}
	f.games.SetDetails(map[string]leagueservice.NFLGame{"401": {Game: &game, At: time.Now()}})
	for _, tc := range []struct {
		league, game string
		found        bool
		status       int
	}{
		{"native.hosted", "401", true, 200}, {"native.hosted", "999", false, 200}, {"native.other", "401", false, 403},
	} {
		t.Run(tc.league+tc.game, func(t *testing.T) {
			raw := workflowRequest(t, "nfl.game.read", "read", fmt.Sprintf(`{"league":%q,"game":%q}`, tc.league, tc.game), "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, "")
			if status != tc.status {
				t.Fatalf("status %d: %s", status, result)
			}
			if status != 200 {
				return
			}
			var response serverhttp.WorkflowResponse
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
			var detail leagueservice.NFLGame
			if err := json.Unmarshal(response.Data, &detail); err != nil {
				t.Fatal(err)
			}
			if detail.League != tc.league || (detail.Game != nil) != tc.found {
				t.Fatalf("wrong detail: %+v", detail)
			}
			if tc.found {
				if detail.Game.State.Down != 2 || len(detail.Game.Plays) != 1 || detail.Stale {
					t.Fatalf("missing live detail: %+v", detail)
				}
			} else if !strings.Contains(detail.Words, "No live detail") {
				t.Fatalf("missing empty words: %+v", detail)
			}
		})
	}
}

// Franchise HQ's money (cap strip, contract schedules, cut costs) comes from these two reads; the
// hosted app could not reach either, so every figure said "Not available yet".
func TestHostedNFLGameReadDoesNotInventUnavailableTimeouts(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer := f.login(t, "owner", "private-password")
	body, err := os.ReadFile("../ingestion/livegame/testdata/espn-summary-401873061-kickoff.json")
	if err != nil {
		t.Fatal(err)
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		t.Fatal(err)
	}
	f.games.SetDetails(map[string]leagueservice.NFLGame{game.ID: {Game: &game, At: time.Now()}})
	raw := workflowRequest(t, "nfl.game.read", "read",
		fmt.Sprintf(`{"league":"native.hosted","game":%q}`, game.ID), "")
	request, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	status, result := f.send(t, serverhttp.WorkflowPath, string(request), bearer, "")
	if status != 200 {
		t.Fatalf("NFL detail status %d: %s", status, result)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatal(err)
	}
	var detail leagueservice.NFLGame
	if err := json.Unmarshal(response.Data, &detail); err != nil || detail.Game == nil {
		t.Fatalf("NFL detail: %v %+v", err, detail)
	}
	gameJSON, err := json.Marshal(detail.Game)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(gameJSON, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["timeouts"]; ok {
		t.Fatalf("NFL timeout counts were invented without held source data: %s", gameJSON)
	}
}

func TestHostedNFLGamePregameReadAndLegacyWireShape(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer := f.login(t, "owner", "private-password")
	temp := 48
	nowAt := time.Date(2026, 11, 22, 16, 0, 0, 0, time.UTC)
	preGame := livegame.Game{
		ID: "402", Status: "pre", Kickoff: nowAt.Add(2 * time.Hour),
		Home:      livegame.Team{ID: "2", Abbreviation: "BUF"},
		Away:      livegame.Team{ID: "15", Abbreviation: "MIA"},
		Predictor: &livegame.Predictor{Home: 59.1, Away: 40.6},
		Venue:     &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY"},
		Weather:   &livegame.Weather{Text: "Partly cloudy", TempF: &temp, Source: "Open-Meteo"},
	}
	liveGame := livegame.Game{
		ID: "401", Status: "in",
		Predictor: &livegame.Predictor{Home: 55, Away: 45},
		Venue:     &livegame.Venue{Name: "Arrowhead Stadium"},
	}
	f.games.SetDetails(map[string]leagueservice.NFLGame{
		"401": {Game: &liveGame, At: nowAt},
		"402": {Game: &preGame, At: nowAt},
	})

	// nfl.game.read must never serialize predictor, venue, or weather for pre or live games.
	for _, id := range []string{"401", "402"} {
		raw := workflowRequest(t, "nfl.game.read", "read",
			fmt.Sprintf(`{"league":"native.hosted","game":%q}`, id), "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, "")
		if status != 200 {
			t.Fatalf("nfl.game.read %s status %d: %s", id, status, result)
		}
		var resp serverhttp.WorkflowResponse
		if err := json.Unmarshal(result, &resp); err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Game map[string]json.RawMessage `json:"game"`
		}
		if err := json.Unmarshal(resp.Data, &envelope); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"predictor", "venue", "weather"} {
			if _, ok := envelope.Game[forbidden]; ok {
				t.Fatalf("nfl.game.read(%s) leaked %q: %s", id, forbidden, resp.Data)
			}
		}
	}

	// nfl.game.pregame.read serves cached pre-game fields and enforces league admission.
	for _, tc := range []struct {
		league, game string
		wantData     bool
		status       int
	}{
		{"native.hosted", "402", true, 200},
		{"native.hosted", "401", false, 200},
		{"native.hosted", "999", false, 200},
		{"native.other", "402", false, 403},
	} {
		t.Run(tc.league+"/"+tc.game, func(t *testing.T) {
			raw := workflowRequest(t, "nfl.game.pregame.read", "read",
				fmt.Sprintf(`{"league":%q,"game":%q}`, tc.league, tc.game), "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, "")
			if status != tc.status {
				t.Fatalf("status %d, want %d: %s", status, tc.status, result)
			}
			if status != 200 {
				return
			}
			var resp serverhttp.WorkflowResponse
			if err := json.Unmarshal(result, &resp); err != nil {
				t.Fatal(err)
			}
			var got leagueservice.NFLGamePregame
			if err := json.Unmarshal(resp.Data, &got); err != nil {
				t.Fatal(err)
			}
			if got.League != tc.league || got.Game != tc.game {
				t.Fatalf("wrong scope: %+v", got)
			}
			if tc.wantData {
				if got.Predictor == nil || got.Predictor.Home != 59.1 || got.Venue == nil ||
					got.Venue.Name != "Highmark Stadium" || got.Weather == nil ||
					*got.Weather.TempF != 48 {
					t.Fatalf("missing cached pregame fields: %+v", got)
				}
			} else if got.Predictor != nil || got.Venue != nil || got.Weather != nil {
				t.Fatalf("unexpected pregame fields: %+v", got)
			}
		})
	}
}

func TestHostedFranchiseHQMoneyReads(t *testing.T) {
	f := newHostedFixture(t, true, true)
	bearer, proof := f.directCommissioner(t)
	raw := workflowRequest(t, "native.cap.check", "read", `{"league":"native.hosted","viewerZone":"UTC"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
	if status != 200 {
		t.Fatalf("cap check status %d: %s", status, result)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatal(err)
	}
	var reading struct {
		League     string            `json:"league"`
		Franchises []json.RawMessage `json:"franchises"`
	}
	if err := json.Unmarshal(response.Data, &reading); err != nil || reading.League != "native.hosted" || len(reading.Franchises) == 0 {
		t.Fatalf("cap check reading %s (%v)", response.Data, err)
	}
	quote := workflowRequest(t, "contract.quote", "read", `{"league":"native.hosted","franchise":"franchise.01",`+
		`"player":"0100","action":"cut","viewerZone":"UTC","years":0,"level":0,"moveCents":0,"amountCents":0,"reason":""}`, "")
	if league, err := quote.League(); err != nil || league != "native.hosted" {
		t.Fatalf("contract quote not routed: %q %v", league, err)
	}
	for _, bad := range []struct{ command, args string }{
		{"native.cap.check", `{"league":"native.hosted","viewerZone":"Local"}`},
		{"native.cap.check", `{"league":"native.hosted","viewerZone":"UTC","extra":1}`},
		{"contract.quote", `{"league":"native.hosted","player":"0100","action":"cut","viewerZone":"Local"}`},
	} {
		raw := workflowRequest(t, bad.command, "read", bad.args, "")
		if err := raw.Validate(); err == nil {
			t.Fatalf("invalid %s accepted: %s", bad.command, bad.args)
		}
		if _, err := raw.League(); err == nil {
			t.Fatalf("invalid %s routed: %s", bad.command, bad.args)
		}
	}
}

func TestHostedNFLGamesReadableByAgentToken(t *testing.T) {
	f := newHostedFixture(t, true, true)
	human, _ := f.directCommissioner(t)
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: human}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01", Modes: []string{"read"},
		Commands: []string{"native.directory.read", "nfl.games.read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	summaryBody, err := os.ReadFile("../ingestion/livegame/testdata/espn-summary-401873061-kickoff.json")
	if err != nil {
		t.Fatal(err)
	}
	summary, err := livegame.ParseSummary(summaryBody)
	if err != nil {
		t.Fatal(err)
	}
	f.games.Set([]livegame.Game{summary}, time.Now(), "ESPN")
	raw := workflowRequest(t, "nfl.games.read", "read", `{"league":"native.hosted"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	reply := f.post(t, serverhttp.WorkflowPath, string(body), token.Bearer, "", 200)
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(reply, &response); err != nil {
		t.Fatal(err)
	}
	var games leagueservice.NFLGames
	if err := json.Unmarshal(response.Data, &games); err != nil {
		t.Fatal(err)
	}
	if len(games.Board.Games) != 1 || games.Board.Games[0].ID != summary.ID {
		t.Fatalf("agent token did not read the server's NFL board: %+v", games)
	}
}
