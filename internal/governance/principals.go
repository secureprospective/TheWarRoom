package governance

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// VerifiedAlias is supplied by trusted account verification, never by a league command.
type VerifiedAlias struct {
	League         string
	Alias, Account Principal
}

// Principals is an immutable, league-scoped identity map. It grants no roles.
type Principals struct {
	league   string
	accounts map[Principal]Principal
}

func AccountPrincipal(id string) (Principal, error) {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' || strings.ToLower(id) != id {
		return "", fmt.Errorf("choose a canonical account UUID")
	}
	compact := strings.ReplaceAll(id, "-", "")
	decoded, err := hex.DecodeString(compact)
	if err != nil || len(decoded) != 16 || compact == strings.Repeat("0", 32) {
		return "", fmt.Errorf("choose a canonical account UUID")
	}
	return Principal("acct." + id), nil
}

func NewPrincipals(league string, aliases []VerifiedAlias) (*Principals, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league {
		return nil, fmt.Errorf("choose an app league for verified aliases")
	}
	p := &Principals{league: league, accounts: make(map[Principal]Principal)}
	for _, alias := range aliases {
		account, err := AccountPrincipal(strings.TrimPrefix(string(alias.Account), "acct."))
		if err != nil || account != alias.Account || alias.League != league || alias.Alias == "" ||
			strings.TrimSpace(string(alias.Alias)) != string(alias.Alias) || strings.HasPrefix(string(alias.Alias), "acct.") {
			return nil, fmt.Errorf("invalid or cross-league verified alias")
		}
		if prior, ok := p.accounts[alias.Alias]; ok && prior != account {
			return nil, fmt.Errorf("verified alias belongs to another account")
		}
		p.accounts[alias.Alias] = account
	}
	return p, nil
}

func (p *Principals) League() string { return p.league }
func (p *Principals) Resolve(principal Principal) Principal {
	if p != nil {
		if account, ok := p.accounts[principal]; ok {
			return account
		}
	}
	return principal
}
