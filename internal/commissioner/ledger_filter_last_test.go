package commissioner

import "testing"

func TestLedgerFilterLastKeepsNewest(t *testing.T) {
	b := newLedger(t, PresetEmpty)
	for range 5 {
		b.revert(b.records[len(b.records)-1].Receipt.Entry, seasonMoment(2))
	}
	auth := CreatorAuthorizer{Creator: "creator"}
	full, err := Reading("native.test", "creator", b.records, auth)
	if err != nil {
		t.Fatal(err)
	}
	page, err := FilteredReading("native.test", "creator", b.records, auth, RawLedgerFilter{Last: 2})
	if err != nil || len(page.Entries) != 2 {
		t.Fatalf("last 2: %d entries, %v", len(page.Entries), err)
	}
	for i, entry := range page.Entries {
		if entry.ID != full.Entries[len(full.Entries)-2+i].ID {
			t.Fatalf("entry %d is %s, not one of the newest two", i, entry.ID)
		}
	}
	if page.Revision != full.Revision {
		t.Fatalf("page revision %d, full %d", page.Revision, full.Revision)
	}
	all, err := FilteredReading("native.test", "creator", b.records, auth, RawLedgerFilter{Last: 99})
	if err != nil || len(all.Entries) != len(full.Entries) {
		t.Fatalf("last 99: %d of %d, %v", len(all.Entries), len(full.Entries), err)
	}
	if _, err := DecodeLedgerFilter(`{"person":"","kind":"","from":"","before":"","last":-1}`); err == nil {
		t.Fatal("negative last was accepted")
	}
}
