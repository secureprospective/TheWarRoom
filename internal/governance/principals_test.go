package governance

import "testing"

func TestCanonicalAccountPrincipal(t *testing.T) {
	t.Parallel()
	id := "12345678-1234-4234-8234-123456789abc"
	got, err := AccountPrincipal(id)
	if err != nil || got != Principal("acct."+id) {
		t.Fatalf("account: %q %v", got, err)
	}
	for _, bad := range []string{"", "host-admin", "12345678-1234-4234-8234-123456789ABC", "00000000-0000-0000-0000-000000000000", "12345678-1234-4234-8234-123456789abz", "12345678123442348234123456789abc"} {
		if _, err := AccountPrincipal(bad); err == nil {
			t.Fatalf("accepted account %q", bad)
		}
	}
}

func TestVerifiedAliasesAreImmutableAndLeagueScoped(t *testing.T) {
	t.Parallel()
	const account Principal = "acct.12345678-1234-4234-8234-123456789abc"
	const other Principal = "acct.87654321-1234-4234-8234-123456789abc"
	aliases := []VerifiedAlias{{League: "native.one", Alias: "old", Account: account}, {League: "native.one", Alias: "second", Account: account}}
	p, err := NewPrincipals("native.one", aliases)
	if err != nil {
		t.Fatal(err)
	}
	aliases[0].Account = other
	if p.Resolve("old") != account || p.Resolve("second") != account || p.Resolve(other) != other || p.Resolve("unknown") != "unknown" {
		t.Fatal("resolution changed or invented an alias")
	}
	for _, bad := range [][]VerifiedAlias{
		{{League: "native.two", Alias: "old", Account: account}},
		{{League: "native.one", Alias: "old", Account: account}, {League: "native.one", Alias: "old", Account: other}},
		{{League: "native.one", Alias: other, Account: account}},
		{{League: "native.one", Alias: "old", Account: "host-admin"}},
		{{League: "native.one", Alias: " old ", Account: account}},
	} {
		if _, err := NewPrincipals("native.one", bad); err == nil {
			t.Fatalf("accepted aliases: %+v", bad)
		}
	}
}
