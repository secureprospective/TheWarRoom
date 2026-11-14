package offsite

import (
	"bytes"
	"testing"

	"filippo.io/age"
)

// A bucket that already holds a capture sealed to another key (the 11-14 drill left one) must not stand
// in for this server's own copy: the server uploads its own, and its own restore key opens it.
func TestRawCaptureSealedToAnotherKeyIsNotReused(t *testing.T) {
	r2, fake := testR2(t)
	other, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	own, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("compressed raw capture")
	if err := NewEvidence(r2, other.Recipient().String()).Put(t.Context(), "abc", body); err != nil {
		t.Fatal(err)
	}
	if err := NewEvidence(r2, own.Recipient().String()).Put(t.Context(), "abc", body); err != nil {
		t.Fatal(err)
	}
	if len(fake.objects) != 2 {
		t.Fatalf("%d objects; want one per key", len(fake.objects))
	}
	for _, identity := range []*age.X25519Identity{own, other} {
		object, err := r2.OpenEvidence(t.Context(), identity.Recipient().String(), "abc")
		if err != nil {
			t.Fatal(err)
		}
		got, err := UnsealEvidence(object, identity)
		_ = object.Close()
		if err != nil || !bytes.Equal(got, body) {
			t.Fatalf("key %s: %q, %v", identity.Recipient(), got, err)
		}
	}
}

// A capture an earlier release left under its sha256 alone still restores.
func TestRawCaptureLegacyNameStillRestores(t *testing.T) {
	r2, fake := testR2(t)
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("capture from release 8")
	sealed, err := SealEvidence(body, identity.Recipient().String())
	if err != nil {
		t.Fatal(err)
	}
	fake.objects[LegacyEvidenceName("abc")] = sealed
	object, err := r2.OpenEvidence(t.Context(), identity.Recipient().String(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = object.Close() }()
	got, err := UnsealEvidence(object, identity)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("legacy capture %q: %v", got, err)
	}
	if _, err := r2.OpenEvidence(t.Context(), identity.Recipient().String(), "missing"); err == nil {
		t.Fatal("a missing capture opened")
	}
}
