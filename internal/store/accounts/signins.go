package accounts

import (
	"context"
	"fmt"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// SignInFacts reads, for every account and every identity linked to it, whether its authenticator is
// confirmed and when one of its sessions last acted. It reads only; nothing here decides a sign-in.
func (s *Store) SignInFacts(ctx context.Context) ([]identity.SignInFact, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
WITH facts AS (
 SELECT a.id,
  EXISTS(SELECT 1 FROM account_totp t WHERE t.account_id = a.id AND t.confirmed = 1) AS totp,
  COALESCE((SELECT MAX(last_used_at) FROM account_sessions s WHERE s.account_id = a.id), 0) AS seen
 FROM accounts a)
SELECT 'acct.' || id, totp, seen FROM facts
UNION ALL
SELECT l.alias, f.totp, f.seen FROM account_aliases l JOIN facts f ON f.id = l.account_id
ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("accounts: read sign-in facts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []identity.SignInFact{}
	for rows.Next() {
		var principal string
		var totp bool
		var seen int64
		if err := rows.Scan(&principal, &totp, &seen); err != nil {
			return nil, fmt.Errorf("accounts: read sign-in fact: %w", err)
		}
		fact := identity.SignInFact{Principal: governance.Principal(principal), Authenticator: totp}
		if seen > 0 {
			fact.LastSeen = time.Unix(0, seen).UTC().Format(time.RFC3339)
		}
		out = append(out, fact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: read sign-in facts: %w", err)
	}
	return out, nil
}
