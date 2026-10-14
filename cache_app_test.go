package main

import (
	"strings"
	"testing"
)

func TestEveryHeldCopyIsWiredToARefreshTheAppAlreadyHas(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range cacheScopes() {
		if c.run == nil || c.ID == "" || seen[c.ID] || c.Label == "" || c.Note == "" || c.Action == "" ||
			c.Progress == "" {
			t.Fatalf("incomplete held copy: %+v", c)
		}
		seen[c.ID] = true
		for _, word := range []string{"cache", "force", "endpoint", "MFL"} {
			if strings.Contains(strings.ToLower(c.Label+c.Note+c.Action), strings.ToLower(word)) {
				t.Errorf("%s shows an internal word %q", c.ID, word)
			}
		}
	}
	app := commissionerTestApp(t)
	listed := app.TargetCacheScopes()
	if len(listed) != len(cacheScopes()) || len(listed) < 5 {
		t.Fatalf("the screen lists what Go can read again: %+v", listed)
	}
}

func TestReadingAnUnknownHeldCopyAgainIsRefusedWithoutAskingAnySource(t *testing.T) {
	app := commissionerTestApp(t)
	for _, scope := range []string{"", "everything", "league"} {
		result := app.TargetCacheClear(scope)
		if result.OK || !strings.Contains(result.Error, "unknown held copy") {
			t.Fatalf("%q: %+v", scope, result)
		}
	}
}
