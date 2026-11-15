package commissioner

import (
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// MarshalJSON keeps an unrecorded contract blank, distinct from a recorded zero salary.
func (p NativePlayer) MarshalJSON() ([]byte, error) {
	type player NativePlayer
	if !p.ContractMissing {
		body, err := json.Marshal(player(p))
		if err != nil {
			return nil, fmt.Errorf("write recorded player: %w", err)
		}
		return body, nil
	}
	body, err := json.Marshal(struct {
		player
		Salary     *domain.Money `json:"salaryCents"`
		Expiration *int          `json:"expirationYear"`
	}{player: player(p)})
	if err != nil {
		return nil, fmt.Errorf("write player with missing contract: %w", err)
	}
	return body, nil
}

// CheckContracts protects monetary consumers without preventing roster membership readings.
func (r *RawNativeSnapshot) CheckContracts(franchise string) error {
	if words := r.missingContractWords(franchise); words != "" {
		return fmt.Errorf("%s", words)
	}
	return nil
}

func (r *RawNativeSnapshot) missingContractWords(franchise string) string {
	for _, p := range r.Players {
		if p.ContractMissing && (franchise == "" || p.Franchise == franchise) {
			return fmt.Sprintf("record the missing contract for player %s on %s before judging salary", p.ID, p.Franchise)
		}
	}
	return ""
}

func missingContractJobs(assets NativeAssets) []Job {
	if !assets.Seeded {
		return nil
	}
	var jobs []Job
	for _, p := range assets.Snapshot.Players {
		if !p.ContractMissing {
			continue
		}
		name := p.Name
		if name == "" {
			name = "Player " + p.ID
		}
		jobs = append(jobs, Job{ID: "contract." + p.Franchise + "." + p.ID,
			Name: "Contract from import — fill it in", State: JobLive,
			Words:    name + " on " + franchiseName(assets.Snapshot.Franchises, p.Franchise) + ": no contract recorded. Set salary and years with a contract correction.",
			Controls: []string{"CM-034"}})
	}
	return jobs
}

func (p NativePlayer) validateContract() error {
	if p.ContractMissing && (p.Salary != 0 || p.ExpirationYear != 0 || p.ContractStatus != "") {
		return fmt.Errorf("a missing contract cannot carry invented terms")
	}
	return nil
}

func (a NativeAssets) CheckContracts(franchise string) error {
	if !a.Seeded {
		return nil
	}
	return a.Snapshot.CheckContracts(franchise)
}
