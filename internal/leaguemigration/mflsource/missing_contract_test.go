package mflsource_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestImportMissingContractStaysBlank(t *testing.T) {
	for _, tc := range []struct {
		name, salary, year, status string
		missing                    bool
	}{
		{"missing", "", "", "", true},
		{"recorded zero", "0", "2026", "UFA", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			export := migrationtest.Synthetic("Contracts", migrationtest.Specs()["8 teams"])
			export.Rosters[0].Salary, export.Rosters[0].ContractYear, export.Rosters[0].ContractStatus = tc.salary, tc.year, tc.status
			plan, err := mflsource.Prepare(export)
			if err != nil {
				t.Fatal(err)
			}
			var found bool
			for _, p := range plan.Seed.Players {
				if p.ID != export.Rosters[0].Player {
					continue
				}
				found = true
				if p.ContractMissing != tc.missing {
					t.Fatalf("missing state lost: %+v", p)
				}
				if tc.missing && (p.Salary != 0 || p.ExpirationYear != 0 || p.ContractStatus != "") {
					t.Fatalf("invented terms: %+v", p)
				}
			}
			if !found {
				t.Fatal("missing contract dropped the roster player")
			}
		})
	}
}
