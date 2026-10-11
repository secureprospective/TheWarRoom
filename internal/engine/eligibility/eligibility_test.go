package eligibility

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// answerKeyCSV is the committed capture fixture; embedding keeps the engine pure — no
// filesystem access, even in its tests.
//
//go:embed testdata/answer-key.csv
var answerKeyCSV string

func liveFeed() domain.Freshness { return domain.Freshness{State: domain.FreshLive} }

// oneFranchise builds the input roster for a single-franchise league.
func oneFranchise(players ...RosterPlayer) []Franchise {
	return []Franchise{{ID: "0025", Players: players}}
}

func rosterPlayer(id string, status domain.RosterStatus) RosterPlayer {
	return RosterPlayer{ID: id, Status: status}
}

func active(id string) RosterPlayer { return rosterPlayer(id, domain.RosterActive) }

func fullInput(players ...RosterPlayer) Input {
	return Input{
		Season: 2026, IR: Capacity{Slots: 12, Held: true}, Taxi: Capacity{Slots: 8, Held: true},
		Rosters:  oneFranchise(players...),
		Injuries: Injuries{Status: map[string]string{}, Fresh: liveFeed()},
		Players: Players{
			DraftYear: map[string]DraftYear{}, Fresh: liveFeed(),
		},
	}
}

func verdictOf(t *testing.T, in Input, id string) PlayerVerdicts {
	t.Helper()
	got := Evaluate(in)
	v, ok := got[id]
	if !ok {
		t.Fatalf("player %s missing from verdicts", id)
	}
	return v
}

func TestToIRDesignations(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   Verdict
		reason string
	}{
		{"IR", Eligible, ""},
		{"IR-R", Eligible, ""},
		{"IR-PUP", Eligible, ""},
		{"IR-NFI", Eligible, ""},
		{"Out", Ineligible, "not on NFL injured reserve"},
		{"Questionable", Ineligible, "not on NFL injured reserve"},
		{"Doubtful", Ineligible, "not on NFL injured reserve"},
		{"RETIRED", Ineligible, "not on NFL injured reserve"},
		{"Reserve/COVID-19", Ineligible, "not on NFL injured reserve"},
		{"Suspended", Ineligible, "suspended"},
		{"Holdout", Ineligible, "holdout"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			in := fullInput(active("10249"))
			if tc.status != "" {
				in.Injuries.Status["10249"] = tc.status
			}
			d := verdictOf(t, in, "10249").IR
			if d.Direction != ToIR || d.Verdict != tc.want || d.Reason != tc.reason ||
				d.Source != SourceInjuries {
				t.Fatalf("IR decision = %+v, want %s %q from %s", d, tc.want, tc.reason, SourceInjuries)
			}
		})
	}
}

func TestToIRStaleFeedIsUnknownNeverGuess(t *testing.T) {
	for _, state := range []string{domain.FreshStale, domain.FreshFail} {
		in := fullInput(active("10249"))
		in.Injuries.Status["10249"] = "IR"
		in.Injuries.Fresh = domain.Freshness{State: state}
		d := verdictOf(t, in, "10249").IR
		if d.Verdict != Unknown || d.Reason != staleReason || d.Source != SourceInjuries {
			t.Fatalf("stale injuries (%s) = %+v, want unknown %q", state, d, staleReason)
		}
	}
}

func TestToIRCapacity(t *testing.T) {
	players := []RosterPlayer{active("10249")}
	for i := range 12 {
		players = append(players, rosterPlayer(fmt.Sprintf("9%03d", i), domain.RosterIR))
	}
	in := fullInput(players...)
	in.Injuries.Status["10249"] = "IR"
	d := verdictOf(t, in, "10249").IR
	if d.Verdict != Ineligible || d.Reason != "IR full (12/12)" || d.Source != SourceRosters {
		t.Fatalf("full IR = %+v, want ineligible IR full (12/12)", d)
	}
	in.IR = Capacity{Held: false}
	d = verdictOf(t, in, "10249").IR
	if d.Verdict != Unknown || d.Reason != "IR capacity not held" {
		t.Fatalf("unheld IR capacity = %+v, want unknown", d)
	}
}

func TestToTaxiExperience(t *testing.T) {
	for _, tc := range []struct {
		year   int
		held   bool
		want   Verdict
		reason string
	}{
		{2026, true, Eligible, ""},
		{2025, true, Eligible, ""},
		{2024, true, Eligible, ""},
		{2023, true, Ineligible, "3 or more years of NFL experience (drafted 2023)"},
		{2017, true, Ineligible, "3 or more years of NFL experience (drafted 2017)"},
		{0, false, Unknown, "no draft year held"},
		{1970, true, Unknown, "no draft year held"},
	} {
		t.Run(strconv.Itoa(tc.year)+"/"+strconv.FormatBool(tc.held), func(t *testing.T) {
			in := fullInput(active("10249"))
			in.Players.DraftYear["10249"] = DraftYear{Year: tc.year, Held: tc.held}
			d := verdictOf(t, in, "10249").Taxi
			if d.Direction != ToTaxi || d.Verdict != tc.want || d.Reason != tc.reason ||
				d.Source != SourcePlayers {
				t.Fatalf("taxi decision = %+v, want %s %q", d, tc.want, tc.reason)
			}
		})
	}
}

func TestToTaxiStaleSeasonAndCapacity(t *testing.T) {
	in := fullInput(active("10249"))
	in.Players.DraftYear["10249"] = DraftYear{Year: 2026, Held: true}
	in.Players.Fresh = domain.Freshness{State: domain.FreshStale}
	if d := verdictOf(t, in, "10249").Taxi; d.Verdict != Unknown || d.Reason != staleReason {
		t.Fatalf("stale players feed = %+v, want unknown stale", d)
	}
	in.Players.Fresh = liveFeed()
	in.Season = 0
	if d := verdictOf(t, in, "10249").Taxi; d.Verdict != Unknown || d.Reason != "season not held" {
		t.Fatalf("missing season = %+v, want unknown season not held", d)
	}
	in.Season = 2026
	players := []RosterPlayer{active("10249")}
	for i := range 8 {
		players = append(players, rosterPlayer(fmt.Sprintf("9%03d", i), domain.RosterTaxi))
	}
	in.Rosters = oneFranchise(players...)
	if d := verdictOf(t, in, "10249").Taxi; d.Verdict != Ineligible || d.Reason != "taxi full (8/8)" {
		t.Fatalf("full taxi = %+v, want ineligible taxi full (8/8)", d)
	}
	in.Taxi = Capacity{Held: false}
	if d := verdictOf(t, in, "10249").Taxi; d.Verdict != Unknown || d.Reason != "taxi capacity not held" {
		t.Fatalf("unheld taxi capacity = %+v, want unknown", d)
	}
}

func TestCrossStatusVerdictsMatchMflsPages(t *testing.T) {
	// The captured IR page lists taxi players only on the promote table, and the taxi page
	// lists IR players nowhere: both squads are reachable only from the active roster.
	in := fullInput(rosterPlayer("10249", domain.RosterIR))
	v := verdictOf(t, in, "10249")
	if v.IR.Direction != OffIR || v.IR.Verdict != Eligible || v.IR.Source != SourceRosters {
		t.Fatalf("on-IR IR slot = %+v, want eligible activation", v.IR)
	}
	if v.Taxi.Verdict != Ineligible || v.Taxi.Reason != "on injured reserve" {
		t.Fatalf("on-IR taxi slot = %+v, want ineligible on injured reserve", v.Taxi)
	}
	in = fullInput(rosterPlayer("10249", domain.RosterTaxi))
	v = verdictOf(t, in, "10249")
	if v.Taxi.Direction != OffTaxi || v.Taxi.Verdict != Eligible || v.Taxi.Source != SourceRosters {
		t.Fatalf("on-taxi taxi slot = %+v, want eligible promotion", v.Taxi)
	}
	if v.IR.Verdict != Ineligible || v.IR.Reason != "on the taxi squad" {
		t.Fatalf("on-taxi IR slot = %+v, want ineligible on the taxi squad", v.IR)
	}
}

func TestNotCheckedNamesMflAsTheAuthority(t *testing.T) {
	got := NotChecked(staleReason)
	if got != "Not checked: stale; MFL will decide" {
		t.Fatalf("note = %q", got)
	}
}

// TestAnswerKeyReproducesMfl is the gate: the rules must reproduce MFL's own verdict for every
// row captured from the owner pages on 2026-10-10, with no wrong verdicts. The fixture holds
// only each row's identity, MFL's verdict and the two feed fields — no other page content.
func TestAnswerKeyReproducesMfl(t *testing.T) {
	rows, err := csv.NewReader(strings.NewReader(answerKeyCSV)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 80 {
		t.Fatalf("answer key holds %d rows, want the full capture", len(rows))
	}
	input := Input{
		Season: 2026, IR: Capacity{Slots: 12, Held: true}, Taxi: Capacity{Slots: 8, Held: true},
		Rosters:  []Franchise{{ID: "0025"}},
		Injuries: Injuries{Status: map[string]string{}, Fresh: liveFeed()},
		Players:  Players{DraftYear: map[string]DraftYear{}, Fresh: liveFeed()},
	}
	type expectation struct {
		page, player, table, mfl string
	}
	var want []expectation
	seen := map[string]bool{}
	for _, row := range rows[1:] {
		if len(row) != 6 {
			t.Fatalf("answer key row %v: want 6 columns", row)
		}
		e := expectation{page: row[0], player: row[1], table: row[2], mfl: row[3]}
		status := domain.RosterActive
		switch {
		case e.page == "O18" && e.table == "on":
			status = domain.RosterIR
		case e.page == "O98" && e.table == "on":
			status = domain.RosterTaxi
		}
		key := e.player + "/" + string(status)
		if !seen[key] {
			seen[key] = true
			input.Rosters[0].Players = append(input.Rosters[0].Players, RosterPlayer{ID: e.player, Status: status})
		}
		if row[4] != "" {
			input.Injuries.Status[e.player] = row[4]
		}
		if row[5] != "" {
			year, perr := strconv.Atoi(row[5])
			if perr != nil {
				t.Fatalf("answer key draft year %q: %v", row[5], perr)
			}
			input.Players.DraftYear[e.player] = DraftYear{Year: year, Held: true}
		}
		want = append(want, e)
	}
	verdicts := Evaluate(input)
	for _, e := range want {
		v, ok := verdicts[e.player]
		if !ok {
			t.Errorf("%s %s: player missing from verdicts", e.page, e.player)
			continue
		}
		var d Decision
		if e.page == "O18" {
			d = v.IR
		} else {
			d = v.Taxi
		}
		wantVerdict := Ineligible
		if e.mfl == "offered" {
			wantVerdict = Eligible
		}
		if d.Verdict != wantVerdict {
			t.Errorf("%s %s (table %s): MFL %s, rule %s %q — a wrong verdict",
				e.page, e.player, e.table, e.mfl, d.Verdict, d.Reason)
		}
	}
	if t.Failed() {
		return
	}
	t.Logf("answer key: %d rows, %d/%d reproduced", len(want), len(want), len(want))
}
