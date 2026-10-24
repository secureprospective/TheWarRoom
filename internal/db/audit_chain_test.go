package db

import "testing"

func TestAuditHashV1(t *testing.T) {
	digest := auditDigest("payload")
	got := auditHash("league_event", 1, "event.1", digest, "")
	if got != "d6c092670a1371083941d983dbb3951a870e6879893fbbbc33e263023c881517" {
		t.Fatalf("v1 checkpoint format changed: %s", got)
	}
	for _, changed := range []string{
		auditHash("account_security_audit", 1, "event.1", digest, ""),
		auditHash("league_event", 2, "event.1", digest, ""),
		auditHash("league_event", 1, "event.2", digest, ""),
		auditHash("league_event", 1, "event.1", auditDigest("changed"), ""),
		auditHash("league_event", 1, "event.1", digest, got),
	} {
		if changed == got {
			t.Fatal("hash did not bind every field")
		}
	}
	if auditHash("a\x00b", 1, "c", digest, "") == auditHash("a", 1, "b\x00c", digest, "") {
		t.Fatal("ambiguous hash field framing")
	}
}
