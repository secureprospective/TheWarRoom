package facts

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// Exception is one pinned difference.
type Exception struct {
	Check, Key, Class, Reason string
}

// Exceptions are the pinned differences, by check and key.
type Exceptions struct {
	byKey map[[2]string]Exception
	SHA   string
}

// ParseExceptions reads an exceptions file (check,key,class,reason). It pins every known difference
// between the facts and MFL's own figures by identity: the check, the difference's key, whether it
// is explained or listed for review, and why. A count never stands in for an identity, so a new
// difference fails the compile even if an old one went away.
func ParseExceptions(body []byte) (Exceptions, error) {
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		return Exceptions{}, fmt.Errorf("exceptions: %w", err)
	}
	if len(rows) == 0 || !slices.Equal(rows[0], []string{"check", "key", "class", "reason"}) {
		return Exceptions{}, fmt.Errorf("exceptions: invalid columns")
	}
	sum := sha256.Sum256(body)
	out := Exceptions{byKey: make(map[[2]string]Exception, len(rows)-1), SHA: hex.EncodeToString(sum[:])}
	for i, r := range rows[1:] {
		e := Exception{Check: r[0], Key: r[1], Class: r[2], Reason: r[3]}
		if e.Check == "" || e.Key == "" || strings.TrimSpace(e.Reason) == "" || (e.Class != ClassExplained && e.Class != ClassReview) {
			return Exceptions{}, fmt.Errorf("exceptions row %d: needs a check, key, class (explained or review) and reason", i+2)
		}
		k := [2]string{e.Check, e.Key}
		if _, dup := out.byKey[k]; dup {
			return Exceptions{}, fmt.Errorf("exceptions row %d: %s %s pinned twice", i+2, e.Check, e.Key)
		}
		out.byKey[k] = e
	}
	return out, nil
}

func (x Exceptions) find(check, key string) (Exception, bool) {
	e, ok := x.byKey[[2]string{check, key}]
	return e, ok
}
