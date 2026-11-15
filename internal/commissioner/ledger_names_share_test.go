package commissioner

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

type ownerRoleAuth struct{}

func (ownerRoleAuth) Allow(_, principal, _ string) (string, error) {
	if principal == "commish" {
		return "commissioner", nil
	}
	return "owner", nil
}

// namesLedger is visibilityLedger with a native snapshot first and roster corrections, league and private,
// mixed in, so the shared name index is read by entries each viewer may or may not see.
func namesLedger(rng *rand.Rand, n int) []Record {
	seed := Record{Principal: "commish", Command: RawCommand{League: "native.test", Command: NativeSeed, Visibility: "league",
		Args: RawArgs{Native: &RawNativeSnapshot{
			Players:    []NativePlayer{{ID: "0003", Name: "Three"}, {ID: "0005", Name: "Five"}},
			Franchises: []NativeFranchise{{ID: "franchise.01", Name: "Ones"}, {ID: "franchise.02", Name: "Twos"}}}}}}
	records := []Record{seed}
	for _, r := range visibilityLedger(rng, n) {
		records = append(records, r)
		if rng.IntN(3) == 0 {
			records = append(records, Record{Principal: []string{"owner-a", "owner-b", "commish"}[rng.IntN(3)],
				Command: RawCommand{League: "native.test", Command: CorrectRoster, Visibility: []string{"league", "private"}[rng.IntN(2)],
					Args: RawArgs{Correction: &RawCorrection{Player: []string{"0003", "0005", "0009"}[rng.IntN(3)],
						Franchise: []string{"franchise.01", "franchise.02"}[rng.IntN(2)], Action: []string{"add", "remove"}[rng.IntN(2)]}}}})
		}
	}
	for i := range records {
		records[i].Receipt.Entry, records[i].Receipt.Revision = fmt.Sprintf("n%03d", i), int64(i+1)
		records[i].Canonical = fmt.Sprint(i)
	}
	return records
}

func withoutMemo(records []Record) []Record {
	out := slices.Clone(records)
	out[0].replay, out[0].govern = nil, nil
	return out
}

func readingJSON(t *testing.T, principal string, records []Record) string {
	t.Helper()
	got, err := Reading("native.test", principal, records, ownerRoleAuth{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// One name index per held revision, shared by every viewer, must give each viewer the reading the
// per-entry build gave; and an owner never receives another owner's private entries.
func TestLedgerSharedNamesMatchPerViewerBuild(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1109, 0x1ed6e7)) //nolint:gosec // a fixed test grid, not a secret
	corrections := 0
	for trial := range 600 {
		plain := namesLedger(rng, 2+rng.IntN(14))
		held := WithReplayCache(plain)
		principals := []string{"owner-a", "owner-b", "commish"}
		rng.Shuffle(len(principals), func(i, j int) { principals[i], principals[j] = principals[j], principals[i] })
		for _, principal := range principals {
			if got, want := readingJSON(t, principal, held), readingJSON(t, principal, plain); got != want {
				t.Fatalf("trial %d %s: shared %s\nper-viewer %s", trial, principal, got, want)
			}
		}
		// A later revision renames a player: the held index must not leak into it.
		renamed := append(slices.Clone(held), Record{Principal: "commish", Receipt: Receipt{Entry: "late", Revision: 999},
			Command: RawCommand{League: "native.test", Command: CorrectRoster, Visibility: "league",
				Args: RawArgs{Correction: &RawCorrection{Player: "0009", Franchise: "franchise.01", Action: "add"}}}},
			Record{Receipt: Receipt{Entry: "snap", Revision: 1000}, Command: RawCommand{Command: NativeSeed,
				Args: RawArgs{Native: &RawNativeSnapshot{Players: []NativePlayer{{ID: "0009", Name: "Nine"}}}}}})
		if got, want := readingJSON(t, "owner-a", renamed), readingJSON(t, "owner-a", withoutMemo(renamed)); got != want {
			t.Fatalf("trial %d: extended revision differs: %s\n%s", trial, got, want)
		}
		byID := map[string]Record{}
		for _, r := range plain {
			byID[r.Receipt.Entry] = r
			if r.Command.Command == CorrectRoster {
				corrections++
			}
		}
		for _, owner := range []string{"owner-a", "owner-b"} {
			reading, err := Reading("native.test", owner, held, ownerRoleAuth{})
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range reading.Entries {
				r := byID[e.ID]
				if r.Principal != owner && (r.Command.Visibility == "private" || OwnerBidAction(r.Command.Command) != "" ||
					r.Command.Command == WaiverClaim) {
					t.Fatalf("trial %d: %s received %s's private %s (%s)", trial, owner, r.Principal, r.Command.Command, e.ID)
				}
			}
		}
	}
	if corrections == 0 {
		t.Fatal("grid produced no corrections")
	}
}
