package commissioner

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// twin32Facts names the one rookie the proof drafts. No draft year is held for him, so he is in every pool.
func twin32Facts() nativecheck.Facts {
	return nativecheck.Facts{"1000": {Name: "Twin Player", Position: leaguerules.WR}}
}

// twin32PerSide is the players.pools value JSON for a pool per side.
func twin32PerSide() string { return fmt.Sprintf("%q", leaguerules.PerSide) }

// twin32Start creates a 32-team Default league with the given name, and sets its player pools when pools is
// not empty through the ordinary rules-change command. The refusal, if any, is the test's failure.
func twin32Start(t *testing.T, league, pools string) *ledgerBuilder {
	t.Helper()
	create, err := RawCommand{League: league, Command: Create, Args: RawArgs{Name: "Ours",
		Preset: PresetDefault32, Source: SourceNative, Season: 2026, FranchiseCount: 32}, Request: "create",
		Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	if pools != "" {
		raw := ruleRaw("players.pools", pools, "now", "Agreed")
		raw.League, raw.Request = league, "pools"
		if err := CheckRulesAppend(raw, b.records, seasonMoment(2)); err != nil {
			t.Fatalf("players.pools change refused: %v", err)
		}
		b.add(raw, seasonMoment(2))
	}
	seed, err := NativeStart(league, b.records, seasonMoment(2))
	if err != nil {
		t.Fatalf("starting snapshot refused: %v", err)
	}
	b.add(RawCommand{League: league, Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(2))
	return b
}

// twin32Open records the league's startup draft opening, the way the store does: prepare, then check.
func twin32Open(b *ledgerBuilder, league string, at time.Time) error {
	raw := RawCommand{League: league, Command: DraftStartupOpen, Args: RawArgs{Draft: &RawDraftArgs{Season: 2026,
		Startup: true}}, Request: "open", Effective: "now", Visibility: "league"}
	if err := PrepareDraftAppend(&raw, b.records, at, twin32Facts(), NoDraftStandings{}); err != nil {
		return err
	}
	if err := CheckDraftAppend(raw, b.records, at, NoDraftStandings{}); err != nil {
		return err
	}
	b.add(raw, at)
	return nil
}

// twin32Pick plans a startup-draft pick by the franchise, and records it only when the planner accepts it.
func twin32Pick(b *ledgerBuilder, league, franchise, player string, at time.Time) error {
	raw, err := PlanDraftPick(DraftPickInput{League: league, Season: 2026, Franchise: franchise, Player: player,
		Request: fmt.Sprintf("pick%d", len(b.records)), Expected: int64(len(b.records)), Startup: true},
		b.records, twin32Facts(), at, time.UTC, NoDraftStandings{})
	if err != nil {
		return err
	}
	b.add(raw, at)
	return nil
}

// twin32OnClock is the franchise on the clock of the league's startup draft for the franchise's side.
func twin32OnClock(t *testing.T, b *ledgerBuilder, league, franchise string,
	at time.Time) (NativeDraft, string) {
	t.Helper()
	reading, err := NativeDraftReading(RawNativeDraftScope{League: league, Season: 2026, Franchise: franchise,
		At: at, Startup: true}, b.records, time.UTC, NoDraftStandings{}, twin32Facts())
	if err != nil {
		t.Fatalf("read startup draft for %s: %v", franchise, err)
	}
	if reading.Draft.OnClock == nil {
		t.Fatalf("no franchise on the clock for %s; words: %q", franchise, reading.Words)
	}
	return reading, reading.Draft.OnClock.Owner
}

// twin32Sides are the franchises on each side of a league, in the league's order.
func twin32Sides(t *testing.T, b *ledgerBuilder) map[string][]string {
	t.Helper()
	sides, err := LeagueSides(b.records)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, f := range LeagueFranchises(b.records) {
		out[sides[f]] = append(out[sides[f]], f)
	}
	return out
}

// twin32Copies is the player's copies on the rosters of the league, in the ledger's order.
func twin32Copies(t *testing.T, b *ledgerBuilder, league, player string) []string {
	t.Helper()
	assets, err := NativeAssetsReading(league, b.records, seasonMoment(20))
	if err != nil {
		t.Fatalf("read rosters of %s: %v", league, err)
	}
	var out []string
	for _, p := range assets.Snapshot.Players {
		if p.ID == player {
			out = append(out, p.Franchise)
		}
	}
	return out
}

// TestTwin32SettingChange is case 1: the 32-Team Default takes players.pools = per_side before any games.
func TestTwin32SettingChange(t *testing.T) {
	const league = "native.twin32"
	before := twin32Start(t, league, "")
	rules, err := RulesNow(before.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("before: teams=%d pools=%q franchises=%d", rules.League.Teams.Value, rules.League.Pools.Value,
		len(LeagueFranchises(before.records)))
	if leaguerules.PoolsPerSide(rules) {
		t.Fatalf("the Default32 starts with a pool per side: %q", rules.League.Pools.Value)
	}

	b := twin32Start(t, league, twin32PerSide())
	rules, err = RulesNow(b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	sides, err := LeagueSides(b.records)
	if err != nil {
		t.Fatal(err)
	}
	names := RawNativeSnapshot{Sides: sides}.SideNames()
	bySide := twin32Sides(t, b)
	t.Logf("after: teams=%d pools=%q league sides=%v", rules.League.Teams.Value, rules.League.Pools.Value, names)
	for _, side := range names {
		t.Logf("side %s: %d franchises", side, len(bySide[side]))
	}
	if !slices.Equal(names, []string{"AFC", "NFC"}) {
		t.Fatalf("league sides %v, want [AFC NFC]", names)
	}
	if len(bySide["AFC"]) != 16 || len(bySide["NFC"]) != 16 {
		t.Fatalf("franchises per side AFC=%d NFC=%d, want 16 each", len(bySide["AFC"]), len(bySide["NFC"]))
	}
}

// TestTwin32StartupDraftPerSide is case 2: one opening, one startup draft per side, and X picked once
// per side.
func TestTwin32StartupDraftPerSide(t *testing.T) {
	const league = "native.twin32"
	b := twin32Start(t, league, twin32PerSide())
	bySide := twin32Sides(t, b)
	afc1, nfc1 := bySide["AFC"][0], bySide["NFC"][0]

	openAt := seasonMoment(3)
	if err := twin32Open(b, league, openAt); err != nil {
		t.Fatalf("startup draft open refused: %v", err)
	}
	at := openAt.Add(time.Second)
	afcReading, afcOwner := twin32OnClock(t, b, league, afc1, at)
	nfcReading, nfcOwner := twin32OnClock(t, b, league, nfc1, at)
	t.Logf("AFC reading: Side=%q Startup=%v order rounds=%d first owner=%s", afcReading.Side, afcReading.Startup,
		len(afcReading.Order), afcOwner)
	t.Logf("NFC reading: Side=%q Startup=%v order rounds=%d first owner=%s", nfcReading.Side, nfcReading.Startup,
		len(nfcReading.Order), nfcOwner)
	if afcReading.Side != "AFC" || nfcReading.Side != "NFC" {
		t.Fatalf("drafts read sides %q and %q, want AFC and NFC", afcReading.Side, nfcReading.Side)
	}
	if !slices.Contains(bySide["AFC"], afcOwner) || !slices.Contains(bySide["NFC"], nfcOwner) {
		t.Fatalf("first owners %s (AFC draft) and %s (NFC draft) are not on their side", afcOwner, nfcOwner)
	}

	if err := twin32Pick(b, league, afcOwner, "1000", at); err != nil {
		t.Fatalf("AFC first pick of 1000 refused: %v", err)
	}
	t.Log("AFC first pick of 1000: accepted")
	at = at.Add(time.Second)
	if err := twin32Pick(b, league, nfcOwner, "1000", at); err != nil {
		t.Fatalf("NFC first pick of 1000 refused: %v", err)
	}
	t.Log("NFC first pick of 1000: accepted")

	copies := twin32Copies(t, b, league, "1000")
	if !slices.Equal(copies, []string{afcOwner, nfcOwner}) {
		t.Fatalf("copies of 1000 after the two first picks %v, want [%s %s]", copies, afcOwner, nfcOwner)
	}
	t.Logf("copies of 1000 after the two first picks: %v", copies)

	at = at.Add(time.Second)
	_, secondAFC := twin32OnClock(t, b, league, afc1, at)
	if secondAFC == afcOwner {
		t.Fatalf("the AFC clock did not move to a second franchise: %s", secondAFC)
	}
	err := twin32Pick(b, league, secondAFC, "1000", at)
	if err == nil {
		t.Fatalf("second AFC pick of 1000 accepted by %s", secondAFC)
	}
	t.Logf("second AFC pick of 1000 by %s refused: %v", secondAFC, err)
	if copies := twin32Copies(t, b, league, "1000"); !slices.Equal(copies, []string{afcOwner, nfcOwner}) {
		t.Fatalf("refused pick changed the copies of 1000: %v", copies)
	}
}

// TestTwin32TwoLeagues is case 3: a 32-Team Default with one pool and a 32-Team Default with a pool per side,
// each its own ledger, both drafting the same player. Each league must read only its own roster.
func TestTwin32TwoLeagues(t *testing.T) {
	one := twin32Start(t, "native.twin32one", "")
	two := twin32Start(t, "native.twin32two", twin32PerSide())
	if err := twin32Open(one, "native.twin32one", seasonMoment(3)); err != nil {
		t.Fatalf("league one open refused: %v", err)
	}
	if err := twin32Open(two, "native.twin32two", seasonMoment(3)); err != nil {
		t.Fatalf("league two open refused: %v", err)
	}
	at := seasonMoment(3).Add(time.Second)

	oneTeam := LeagueFranchises(one.records)[0]
	_, oneOwner := twin32OnClock(t, one, "native.twin32one", oneTeam, at)
	if err := twin32Pick(one, "native.twin32one", oneOwner, "1000", at); err != nil {
		t.Fatalf("league one pick of 1000 refused: %v", err)
	}
	t.Logf("league one (OnePool): 1000 picked by %s: accepted", oneOwner)

	twoAFC := twin32Sides(t, two)["AFC"][0]
	_, twoOwner := twin32OnClock(t, two, "native.twin32two", twoAFC, at)
	if err := twin32Pick(two, "native.twin32two", twoOwner, "1000", at); err != nil {
		t.Fatalf("league two pick of 1000 refused: %v", err)
	}
	t.Logf("league two (PerSide): 1000 picked by %s: accepted", twoOwner)

	oneAssets, err := NativeAssetsReading("native.twin32one", one.records)
	if err != nil {
		t.Fatal(err)
	}
	twoAssets, err := NativeAssetsReading("native.twin32two", two.records)
	if err != nil {
		t.Fatal(err)
	}
	oneCopies := twin32Copies(t, one, "native.twin32one", "1000")
	twoCopies := twin32Copies(t, two, "native.twin32two", "1000")
	t.Logf("league one copies of 1000: %v; league two copies of 1000: %v", oneCopies, twoCopies)
	if !slices.Equal(oneCopies, []string{oneOwner}) {
		t.Fatalf("league one copies %v, want [%s]", oneCopies, oneOwner)
	}
	if !slices.Equal(twoCopies, []string{twoOwner}) {
		t.Fatalf("league two copies %v, want [%s]", twoCopies, twoOwner)
	}
	if len(oneAssets.Snapshot.Players) != 1 || len(twoAssets.Snapshot.Players) != 1 {
		t.Fatalf("roster counts one=%d two=%d, want 1 each", len(oneAssets.Snapshot.Players),
			len(twoAssets.Snapshot.Players))
	}
}
