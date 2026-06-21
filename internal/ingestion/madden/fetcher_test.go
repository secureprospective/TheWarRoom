package madden

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// doc builds one EA-shaped record with a couple of "_rating" attributes plus the
// ignored "_diff"/identity noise, so the parser is exercised on a realistic shape.
func doc(first, last, team, pos, birth string, overall, speed, cov int) string {
	return fmt.Sprintf(`{"firstName":%q,"lastName":%q,"team":%q,"position":%q,"plyrBirthdate":%q,
		"age":25,"plyrAssetname":"x_1","overall_rating":%d,"overall_diff":0,
		"speed_rating":%d,"speed_diff":-1,"manCoverage_rating":%d}`,
		first, last, team, pos, birth, overall, speed, cov)
}

// servePaged returns a client+URL that paginates a fixed doc list by the offset query,
// regardless of the limit, so the offset loop in fetchAll is exercised with small pages.
func servePaged(t *testing.T, docs []string, pageSize int) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		end := offset + pageSize
		if end > len(docs) {
			end = len(docs)
		}
		var slice []string
		if offset < len(docs) {
			slice = docs[offset:end]
		}
		body := "{\"count\":" + strconv.Itoa(len(docs)) + ",\"docs\":[" + join(slice) + "]}"
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}

// resolver maps the fixture players by name+birthdate; "Ghost Player" is intentionally
// absent (unresolvable).
func resolver(name, birth string) (string, bool) {
	m := map[string]string{
		"Aaron Banks|9/3/1997": "00-0000001",
		"Bo Corner|1/2/1999":   "00-0000002",
		"Caleb Deep|5/6/2000":  "00-0000003",
	}
	g, ok := m[name+"|"+birth]
	return g, ok
}

func TestFetch_ParsesPagesAndKeys(t *testing.T) {
	t.Parallel()
	docs := []string{
		doc("Aaron", "Banks", "49ers", "LG", "9/3/1997", 72, 61, 12),
		doc("Bo", "Corner", "Jets", "CB", "1/2/1999", 80, 91, 88),
		doc("Caleb", "Deep", "Rams", "QB", "5/6/2000", 77, 84, 5),
		doc("Ghost", "Player", "FA", "WR", "7/7/1990", 60, 70, 40), // unresolved -> skipped
	}
	client, url := servePaged(t, docs, 2) // force two pages (2 + 2)
	out, err := Fetch(context.Background(), client, url, resolver)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(out) != 3 {
		t.Fatalf("emitted %d records, want 3 (the unresolved Ghost Player is skipped)", len(out))
	}
	banks := out["00-0000001"]
	if banks.Overall != 72 || banks.FullName != "Aaron Banks" || banks.Position != "LG" {
		t.Errorf("Banks = {overall %d, name %q, pos %q}, want {72, Aaron Banks, LG}", banks.Overall, banks.FullName, banks.Position)
	}
	// Dynamic "_rating" attributes are captured (suffix-stripped); overall is NOT in the map.
	if banks.Attributes["speed"] != 61 || banks.Attributes["manCoverage"] != 12 {
		t.Errorf("Banks attributes = %v, want speed 61 / manCoverage 12", banks.Attributes)
	}
	if _, leaked := banks.Attributes["overall"]; leaked {
		t.Error("overall must be a named field, not duplicated into Attributes")
	}
}

func TestFetch_UnresolvedAllSkippedIsEmpty(t *testing.T) {
	t.Parallel()
	docs := []string{doc("No", "Match", "FA", "WR", "1/1/1980", 50, 50, 50)}
	client, url := servePaged(t, docs, 10)
	if _, err := Fetch(context.Background(), client, url, resolver); err == nil {
		t.Fatal("expected errEmpty when no player resolves")
	}
}

// Two distinct Madden players resolving to one gsis is ambiguous and the gsis is
// dropped, not mis-attributed.
func TestFetch_DuplicateGSISDropped(t *testing.T) {
	t.Parallel()
	docs := []string{
		doc("Aaron", "Banks", "49ers", "LG", "9/3/1997", 72, 61, 12),
		doc("Bo", "Corner", "Jets", "CB", "1/2/1999", 80, 91, 88),
	}
	client, url := servePaged(t, docs, 10)
	collide := func(_, _ string) (string, bool) { return "00-0000009", true } // both -> same gsis
	if _, err := Fetch(context.Background(), client, url, collide); err == nil {
		t.Fatal("expected errEmpty: the only gsis is poisoned by the collision and dropped")
	}
}

// A non-integer NUMBER in a rating field is real corruption and fails loud; a string-
// valued rating field (EA's descriptive ratings, like runningStyle) is skipped, not
// failed — covered by TestFetch_StringRatingSkipped.
func TestFetch_MalformedRatingFailsLoud(t *testing.T) {
	t.Parallel()
	bad := `{"firstName":"X","lastName":"Y","plyrBirthdate":"1/1/1990","overall_rating":72.5}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"docs":[` + bad + `]}`))
	}))
	t.Cleanup(srv.Close)
	always := func(_, _ string) (string, bool) { return "00-0000001", true }
	if _, err := Fetch(context.Background(), srv.Client(), srv.URL, always); err == nil {
		t.Fatal("expected a fail-loud error on a non-integer numeric rating")
	}
}

// EA's descriptive string ratings (runningStyle_rating = "Default Stride Awkward") are
// skipped as non-numeric attributes, NOT failed loud, and do not pollute Attributes.
func TestFetch_StringRatingSkipped(t *testing.T) {
	t.Parallel()
	rec := `{"firstName":"Aaron","lastName":"Banks","plyrBirthdate":"9/3/1997",
		"overall_rating":72,"speed_rating":61,"runningStyle_rating":"Default Stride Awkward"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"docs":[` + rec + `]}`))
	}))
	t.Cleanup(srv.Close)
	out, err := Fetch(context.Background(), srv.Client(), srv.URL, resolver)
	if err != nil {
		t.Fatalf("a string-valued rating must be skipped, not error: %v", err)
	}
	r := out["00-0000001"]
	if r.Overall != 72 || r.Attributes["speed"] != 61 {
		t.Errorf("numeric ratings should still parse: overall %d speed %d", r.Overall, r.Attributes["speed"])
	}
	if _, present := r.Attributes["runningStyle"]; present {
		t.Error("a string-valued rating must not appear in Attributes")
	}
}

func TestFetch_Non200FailsLoud(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // mimic the live m26 500
	}))
	t.Cleanup(srv.Close)
	if _, err := Fetch(context.Background(), srv.Client(), srv.URL, resolver); err == nil {
		t.Fatal("expected an error on a 500 (the current-season slug behaviour)")
	}
}
