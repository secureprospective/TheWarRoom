package league

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// RuleDictionaryExport is MFL's dictionary of scoring events, served league-less from the api host.
const RuleDictionaryExport = "allRules"

type mflText struct {
	T string `json:"$t"`
}

// RawRuleCode is one scoring event in MFL's dictionary: its code and MFL's own short name.
type RawRuleCode struct {
	Code  mflText `json:"abbreviation"`
	Short mflText `json:"shortDescription"`
}

// FetchRuleDictionary reads MFL's scoring-event dictionary.
func FetchRuleDictionary(ctx context.Context, c *mfl.Client, year string) ([]RawRuleCode, error) {
	resp, err := c.Do(ctx, mfl.Request{Type: RuleDictionaryExport, Year: year})
	if err != nil {
		return nil, fmt.Errorf("league: fetch rule dictionary: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("league: rule dictionary unexpected status %d", resp.StatusCode)
	}
	return ParseRuleDictionary(resp.Body)
}

// ParseRuleDictionary reads the allRules export; every entry needs a code.
func ParseRuleDictionary(body []byte) ([]RawRuleCode, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("league: rule dictionary: %w", err)
	}
	var env struct {
		All struct {
			Rules ingestion.MFLList[RawRuleCode] `json:"rule"`
		} `json:"allRules"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("league: rule dictionary: %w", err)
	}
	for _, r := range env.All.Rules {
		if strings.TrimSpace(r.Code.T) == "" {
			return nil, fmt.Errorf("league: rule dictionary entry without a code")
		}
	}
	return env.All.Rules, nil
}

// RuleNames maps each event code to MFL's short name.
func RuleNames(codes []RawRuleCode) map[string]string {
	out := make(map[string]string, len(codes))
	for _, r := range codes {
		out[strings.TrimSpace(r.Code.T)] = strings.TrimSpace(r.Short.T)
	}
	return out
}
