package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func decimalOf(n leaguerules.Decimal) *leaguerules.Decimal { return &n }

func TestStatItemIDRoundTripsAndRefusesAnythingElse(t *testing.T) {
	id := StatItemID{Season: 2026, Week: 6, Player: "0099", Seq: 3, Digest: "0123456789abcdef"}
	back, err := ParseStatItemID(id.String())
	if err != nil || back != id || id.String() != "stat.2026.6.0099.3.0123456789abcdef" {
		t.Fatalf("round trip: %+v %v", back, err)
	}
	for _, bad := range []string{"", "stat.2026.6.0099.3", "item.2026.6.0099.3.0123456789abcdef",
		"stat.2026.06.0099.3.0123456789abcdef", "stat.2026.6.99.3.0123456789abcdef",
		"stat.2026.6.0099.0.0123456789abcdef", "stat.2026.23.0099.3.0123456789abcdef",
		"stat.1919.6.0099.3.0123456789abcdef", "stat.2026.6.0099.3.0123456789ABCDEF",
		"stat.2026.6.0099.3.0123456789abcde", "stat.2026.6.0099.3.0123456789abcdef.x"} {
		if _, err := ParseStatItemID(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func correction(source string) RawArgs {
	return RawArgs{Stat: &RawStatChange{Player: "0099", Season: 2026, Week: 6, Stat: "passing_yards",
		Value: decimalOf(2450000), Source: source}}
}

func TestStatCorrectionNeedsASourceTypedByAPerson(t *testing.T) {
	if _, err := DecodeStatChange(correction("Gamebook, Sunday"), StatCorrect); err != nil {
		t.Fatalf("a sourced correction was refused: %v", err)
	}
	for _, source := range []string{"", "   ", " padded ", "line\nbreak", strings.Repeat("x", maxSourceLen+1)} {
		if _, err := DecodeStatChange(correction(source), StatCorrect); err == nil {
			t.Errorf("accepted the source %q", source)
		}
	}
}

func TestStatCommandsRefuseDetailsOfAnotherCommand(t *testing.T) {
	item := StatItemID{Season: 2026, Week: 6, Player: "0099", Seq: 1, Digest: "0123456789abcdef"}.String()
	refused := map[string]struct {
		command string
		args    RawArgs
	}{
		"approve with a value":        {StatApprove, RawArgs{Stat: &RawStatChange{Item: item, Value: decimalOf(1)}}},
		"reject with a source":        {StatReject, RawArgs{Stat: &RawStatChange{Item: item, Source: "me"}}},
		"approve without an item":     {StatApprove, RawArgs{Stat: &RawStatChange{}}},
		"approve with a made-up item": {StatApprove, RawArgs{Stat: &RawStatChange{Item: "stat.1"}}},
		"correction with an item":     {StatCorrect, RawArgs{Stat: &RawStatChange{Item: item}}},
		"approve with other details":  {StatApprove, RawArgs{Name: "x", Stat: &RawStatChange{Item: item}}},
		"no details":                  {StatApprove, RawArgs{}},
		"negative tackles":            {StatCorrect, RawArgs{Stat: &RawStatChange{Player: "0099", Season: 2026, Week: 6, Stat: "tackle", Value: decimalOf(-10000), Source: "x"}}},
		"no value":                    {StatCorrect, RawArgs{Stat: &RawStatChange{Player: "0099", Season: 2026, Week: 6, Stat: "tackle", Source: "x"}}},
		"a stat that is not a name":   {StatCorrect, RawArgs{Stat: &RawStatChange{Player: "0099", Season: 2026, Week: 6, Stat: "Passing Yards", Value: decimalOf(1), Source: "x"}}},
		"a short player id":           {StatCorrect, RawArgs{Stat: &RawStatChange{Player: "99", Season: 2026, Week: 6, Stat: "tackle", Value: decimalOf(1), Source: "x"}}},
	}
	for name, c := range refused {
		if _, err := DecodeStatChange(c.args, c.command); err == nil {
			t.Errorf("accepted: %s", name)
		}
	}
	negativeYards := RawArgs{Stat: &RawStatChange{Player: "0099", Season: 2026, Week: 6, Stat: "rushing_yards",
		Value: decimalOf(-30000), Source: "Gamebook"}}
	if _, err := DecodeStatChange(negativeYards, StatCorrect); err != nil {
		t.Errorf("a loss of yards is a real value: %v", err)
	}
}

func TestStatCommandsAreGravityTwoAndNeedNoReason(t *testing.T) {
	for _, command := range []string{StatApprove, StatReject, StatCorrect} {
		d, err := DefinitionFor(command)
		if err != nil || d.Gravity != "G2" {
			t.Fatalf("%s: %+v %v", command, d, err)
		}
	}
	raw := RawCommand{League: "native.test", Command: StatCorrect, Args: correction("Gamebook"), Request: "r",
		Effective: "now", Visibility: "league"}
	if err := raw.Validate(); err != nil {
		t.Fatalf("a G2 correction needs no reason: %v", err)
	}
	raw.Visibility = "private"
	if err := raw.Validate(); err == nil {
		t.Fatal("a statistic decision must be shown to the league")
	}
}

func TestApprovedStatWithoutHeldVersionsIsRefused(t *testing.T) {
	if _, err := countedVersion(nil, StatBaseline{Digest: "0123456789abcdef", Entry: "approval"}); err == nil {
		t.Fatal("approved statistics disappeared without an error")
	}
	if got, err := countedVersion(nil, StatBaseline{}); err != nil || got != nil {
		t.Fatalf("unapproved missing feed = %v, %v", got, err)
	}
}
