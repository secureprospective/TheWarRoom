package commissioner

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestMissingContractCorrection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		salary domain.Money
	}{{"recorded zero", 0}, {"recorded paid", 100}} {
		t.Run(tc.name, func(t *testing.T) {
			b := rolloverLedger(t)
			original, err := json.Marshal(b.records)
			if err != nil {
				t.Fatal(err)
			}
			raw := RawCommand{League: rolloverLeague, Command: CorrectRoster, Request: "unknown", Reason: "Record an unrecorded acquisition",
				Effective: "now", Visibility: "league", Args: RawArgs{Correction: &RawCorrection{Player: "999999", Franchise: "franchise.02", Action: "add"}}}
			if err := raw.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := CheckCorrectionAppend(raw, b.records, seasonMoment(5)); err != nil {
				t.Fatal(err)
			}
			b.add(raw, seasonMoment(5))
			assets, err := NativeAssetsReading(rolloverLeague, b.records)
			if err != nil {
				t.Fatal(err)
			}
			i := assets.Snapshot.PlayerOf("franchise.02", "999999")
			if i < 0 {
				t.Fatal("missing contract prevented membership")
			}
			p := assets.Snapshot.Players[i]
			if !p.ContractMissing || p.Salary != 0 || p.ExpirationYear != 0 || p.ContractStatus != "" {
				t.Fatalf("invented contract: %+v", p)
			}
			body, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(body, []byte(`"salaryCents":null`)) || !bytes.Contains(body, []byte(`"expirationYear":null`)) {
				t.Fatalf("contract is not blank: %s", body)
			}
			var roundtrip NativePlayer
			if err := json.Unmarshal(body, &roundtrip); err != nil || !reflect.DeepEqual(p, roundtrip) {
				t.Fatalf("wire lost missing state: %+v %v", roundtrip, err)
			}
			jobs, err := JobsReading(rolloverLeague, b.records, seasonMoment(5))
			if err != nil {
				t.Fatal(err)
			}
			hasAlert := func(j Job) bool {
				return j.ID == "contract.franchise.02.999999" && j.Name == "Contract from import — fill it in"
			}
			if !slices.ContainsFunc(jobs.Jobs, hasAlert) {
				t.Fatal("missing contract has no commissioner job")
			}
			capCheck, err := NativeCapCheckReading(RawNativeCapScope{League: rolloverLeague, ViewerZone: "UTC"}, b.records, seasonMoment(5))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range capCheck.Franchises {
				if f.Franchise == p.Franchise {
					if f.RoomCents != nil || f.CommittedCents != nil {
						t.Fatal("missing salary counted as zero")
					}
					for _, year := range f.Runway {
						if year.RoomCents != nil || year.CommittedCents != nil {
							t.Fatal("missing salary counted in runway")
						}
					}
				}
			}
			raw.Command, raw.Request = ContractEdit, "fill"
			raw.Args = RawArgs{Override: &RawNativeOverride{Player: p.ID, Franchise: p.Franchise, Salary: &tc.salary, Years: new(2)}}
			if err := CheckOverrideAppend(raw, b.records, seasonMoment(6)); err != nil {
				t.Fatal(err)
			}
			entry := b.add(raw, seasonMoment(6))
			filled, err := NativeAssetsReading(rolloverLeague, b.records)
			if err != nil {
				t.Fatal(err)
			}
			got := filled.Snapshot.Players[filled.Snapshot.PlayerOf(p.Franchise, p.ID)]
			if got.ContractMissing || got.Salary != tc.salary || got.ExpirationYear != 2027 {
				t.Fatalf("contract correction did not settle missing state: %+v", got)
			}
			jobs, err = JobsReading(rolloverLeague, b.records, seasonMoment(6))
			if err != nil || slices.ContainsFunc(jobs.Jobs, hasAlert) {
				t.Fatalf("settled alert remains: %v", err)
			}
			b.revert(entry, seasonMoment(7))
			jobs, err = JobsReading(rolloverLeague, b.records, seasonMoment(7))
			if err != nil || !slices.ContainsFunc(jobs.Jobs, hasAlert) {
				t.Fatalf("compensation lost missing alert: %v", err)
			}
			after, err := json.Marshal(b.records[:len(b.records)-3])
			if err != nil || !bytes.Equal(original, after) {
				t.Fatalf("correction rewrote original ledger: %v", err)
			}
		})
	}
}

func TestMissingContractValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		player NativePlayer
		bad    bool
	}{
		{"missing", NativePlayer{ContractMissing: true}, false},
		{"explicit zero", NativePlayer{}, false},
		{"missing with salary", NativePlayer{ContractMissing: true, Salary: 1}, true},
		{"missing with year", NativePlayer{ContractMissing: true, ExpirationYear: 2026}, true},
		{"missing with status", NativePlayer{ContractMissing: true, ContractStatus: domain.CStatusUFA}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, _ := rolloverInputs()
			p := tc.player
			p.ID, p.Franchise, p.Status = "0001", "f01", domain.RosterActive
			state.Assets.Players = []NativePlayer{p}
			err := state.Assets.Validate()
			if (err != nil) != tc.bad {
				t.Fatalf("bad=%t, got %v", tc.bad, err)
			}
			if p.ContractMissing && state.Assets.CheckContracts("f01") == nil {
				t.Fatal("missing contract monetary guard absent")
			}
			if state.Assets.CheckContracts("f02") != nil {
				t.Fatal("unrelated franchise blocked")
			}
			if !p.ContractMissing {
				body, err := json.Marshal(p)
				if err != nil || !strings.Contains(string(body), `"salaryCents":0`) {
					t.Fatalf("recorded zero became blank: %s %v", body, err)
				}
			}
		})
	}
}
