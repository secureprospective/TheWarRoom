package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
)

func inputFor(file, body string) ResolvedInput {
	sum := sha256.Sum256([]byte(body))
	return ResolvedInput{File: file, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(body)), Body: []byte(body)}
}

func compileFor(t *testing.T, inputs []ResolvedInput) Facts {
	t.Helper()
	facts, err := Compile(t.Context(), inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	return facts
}

func TestPlayerIdentityAndFallback(t *testing.T) {
	inputs := []ResolvedInput{
		inputFor("2020/players_DETAILS1.json", `{"players":{"player":{"id":"0151","name":"Old","position":"S","college":"School","draft_year":"0","birthdate":"951282000"}}}`),
		inputFor("2021/players_DETAILS1.json", `{"error":"not available"}`),
		inputFor("2021/players.json", `{"players":{"player":[{"id":"0151","name":"New","position":"CB","birthdate":"951282000"}]}}`),
		inputFor("2026/players_DETAILS1.json", `{"players":{"player":{"id":"0151","position":"S","draft_round":"7"}}}`),
	}
	facts := compileFor(t, inputs)
	p := facts.Player
	if len(p.Pid) != 1 || p.Pid[0] != "0151" || p.Name[0] != "New" || value(p.College[0]) != "School" || value(p.Position[0]) != "S" {
		t.Fatalf("identity merge: %+v", p)
	}
	if p.DraftYear[0] != nil || *p.DraftRound[0] != 7 || *p.FirstYear[0] != 2020 || *p.LastYear[0] != 2026 {
		t.Fatal("year or null semantics")
	}
	if len(facts.PlayerConflict.Pid) != 1 || facts.PlayerConflict.Year[0] != 2020 || value(facts.PlayerConflict.Value[0]) != "Old" || value(facts.PlayerConflict.Kept[0]) != "New" {
		t.Fatal("lost historical disagreement")
	}
	if len(facts.PlayerSeason.Pid) != 3 || len(facts.SourceFile.Src) != 4 {
		t.Fatal("fallback or error provenance lost")
	}
	reversed := slices.Clone(inputs)
	slices.Reverse(reversed)
	if !reflect.DeepEqual(facts, compileFor(t, reversed)) {
		t.Fatal("input order changes facts")
	}
}

func TestCompileRejectsInvalidInputs(t *testing.T) {
	valid := inputFor("2026/players.json", `{"players":{"player":{"id":"0151","name":"Name"}}}`)
	corrupt := valid
	corrupt.Body = []byte(`{}`)
	badSize := valid
	badSize.Bytes++
	cases := [][]ResolvedInput{
		{corrupt}, {badSize}, {valid, valid},
		{inputFor("../players.json", `{}`)}, {inputFor("2026/players.json", `[]`)},
		{inputFor("2026/players.json", `{"players":{"player":{"name":"Name"}}}`)},
		{inputFor("2026/players.json", `{"players":{"player":{"id":151}}}`)},
		{inputFor("2026/players.json", `{`)},
		{inputFor("2026/weeklyResults_WYTD.json", `{"allWeeklyResults":{"weeklyResults":{"matchup":{"franchise":{"id":"0025","score":"1"}},"franchise":{"id":"0025","score":"2"}}}}`)},
	}
	for i, inputs := range cases {
		if _, err := Compile(t.Context(), inputs, nil); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Compile(ctx, []ResolvedInput{valid}, nil); err == nil {
		t.Fatal("cancelled compile published")
	}
}

func TestRawFormattingAndBirthdates(t *testing.T) {
	raw, err := decodeRaw([]byte(`{"x":"é😀\n","a":[1,null,"a\\b"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := raw.dump(); got != `{"x": "\u00e9\ud83d\ude00\n", "a": [1, null, "a\\b"]}` {
		t.Fatalf("Python serialization: %s", got)
	}
	eastern, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if got := value(birthdate(ptr("946684800"), eastern)); got != "1999-12-31" {
		t.Fatal(got)
	}
	if birthdate(ptr("0"), eastern) != nil || birthdate(ptr("bad"), eastern) != nil {
		t.Fatal("missing birthdate became date")
	}
}

func TestCuratedValidationAndReplay(t *testing.T) {
	for _, body := range []string{"", "txn_id\n1\n", "txn_id,from_fid,to_fid,kind,pick_year,pick_round,pick_slot,pick_orig,cap_m,cap_seasons,player,clarity,reading\n1,A,B,pick,2020,,1,A,,,,clear,test\n"} {
		if _, err := ParseCurated([]byte(body)); err == nil {
			t.Fatal("malformed curated CSV accepted")
		}
	}
	key := pickKey{2020, 1, "A"}
	dep := noteDependencies{users: map[pickKey]string{key: "C"}, moves: map[pickKey][]transfer{key: {{1, "A", "B", -1}}}}
	notes := map[pickKey][]transfer{key: {{2, "B", "C", 0}, {3, "A", "D", 1}}}
	verdict := noteVerdicts(dep, notes)
	if value(verdict[0]) != "agrees" || value(verdict[1]) != "sender did not hold it" {
		t.Fatal("pick replay")
	}
	dep.users[key] = "D"
	if value(noteVerdicts(dep, notes)[0]) != "chain ends with another team" {
		t.Fatal("final holder proof")
	}
	delete(dep.users, key)
	if len(noteVerdicts(dep, notes)) != 0 {
		t.Fatal("unrecorded draft got a verdict")
	}
	r := RawNoteReading{TxnID: 1, Kind: "none", Clarity: "clear", Reading: "No assets"}
	if _, err := Compile(t.Context(), nil, []RawNoteReading{r}); err == nil {
		t.Fatal("absent curated transaction accepted")
	}
}

func TestFullCompileMeasurement(t *testing.T) {
	if os.Getenv("TWR_FACTS_MEASURE") != "1" {
		t.Skip("full compile measurement is opt-in")
	}
	root := os.Getenv("TWR_FACTS_ARCHIVE")
	if root == "" {
		t.Fatal("set TWR_FACTS_ARCHIVE")
	}
	inputs := fixtureInputs(t)
	for i := range inputs {
		body, err := readWithin(root, inputs[i].File)
		if err != nil {
			t.Fatal(err)
		}
		inputs[i].Body = body
	}
	curated, err := os.ReadFile("../lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	facts, err := Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Player.Pid) != 7927 || len(facts.NoteReading.TxnID) != 469 {
		t.Fatal("full compile truncated")
	}
	t.Logf("full compile wall=%s inputs=%d players=%d", time.Since(start), len(inputs), len(facts.Player.Pid))
}

func TestInputFixtureFingerprint(t *testing.T) {
	body, err := os.ReadFile("../lab/testdata/facts/inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != "0b3d723a02cd36d7e8d220fa1518e675541d8961350ae2c7bbe3cb9ea4f2b38d" {
		t.Fatal("independent pinned-input fixture changed")
	}
}

func TestCompiledCuratedValuesAreOwned(t *testing.T) {
	input := inputFor("2020/transactions.json", `{"transactions":{"transaction":{"type":"TRADE","timestamp":"1"}}}`)
	year := int64(2020)
	note := RawNoteReading{TxnID: 1, Kind: "pick", Clarity: "clear", Reading: "A pick", PickYear: &year}
	facts, err := Compile(t.Context(), []ResolvedInput{input}, []RawNoteReading{note})
	if err != nil {
		t.Fatal(err)
	}
	year = 2026
	if *facts.NoteReading.PickYear[0] != 2020 {
		t.Fatal("caller mutation changed published facts")
	}
}

func TestFingerprintFollowsEveryCell(t *testing.T) {
	a := compileFor(t, []ResolvedInput{inputFor("2020/rosters.json", `{"rosters":{"franchise":{"id":"0001","player":{"id":"1","salary":"5"}}}}`)})
	b := compileFor(t, []ResolvedInput{inputFor("2020/rosters.json", `{"rosters":{"franchise":{"id":"0001","player":{"id":"1","salary":"6"}}}}`)})
	c := compileFor(t, []ResolvedInput{inputFor("2020/rosters.json", `{"rosters":{"franchise":{"id":"0001","player":{"id":"1"}}}}`)})
	if a.Fingerprint() == b.Fingerprint() || a.Fingerprint() == c.Fingerprint() || len(a.Fingerprint()) != 64 {
		t.Fatal("a changed value or a null must change the fingerprint")
	}
	again := compileFor(t, []ResolvedInput{inputFor("2020/rosters.json", `{"rosters":{"franchise":{"id":"0001","player":{"id":"1","salary":"5"}}}}`)})
	if a.Fingerprint() != again.Fingerprint() {
		t.Fatal("the same inputs must give the same fingerprint")
	}
}
