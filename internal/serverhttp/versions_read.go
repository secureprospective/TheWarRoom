package serverhttp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func (h Hosted) workflowReplyData(r *http.Request, service *leagueservice.Service, raw RawWorkflow, league string, tags *ReadTags) (json.RawMessage, error) {
	if raw.Command != "versions.read" {
		return h.workflowData(leagueservice.WithFootballDigest(r.Context(), raw.Football), service, raw, league)
	}
	return h.versionsReading(r, service, league, tags)
}

func (h Hosted) versionsReading(r *http.Request, service *leagueservice.Service, league string, tags *ReadTags) (json.RawMessage, error) {
	//nolint:contextcheck // The service host supplies this request's context and admission.
	directory, err := service.TargetNativeDirectory(league)
	if err != nil {
		return nil, fmt.Errorf("versions league: %w", err)
	}
	generation, err := h.workflowAuthGeneration(r, league)
	if err != nil {
		return nil, err
	}
	pin := h.pinFor(r.Context(), league)
	//nolint:contextcheck // The service host supplies this request's context and admission.
	_, next, ok, err := service.ReadingEpoch(league, pin.Football.Kickoffs())
	if err != nil {
		return nil, fmt.Errorf("versions boundary: %w", err)
	}
	boundary := ""
	if ok && !next.IsZero() {
		boundary = next.UTC().Format(time.RFC3339Nano)
	}
	reading := catalog.VersionsReading{AuthGeneration: generation, League: league, Season: directory.RosterSeason,
		Topics: map[string]string{}, Football: pin.Digest, ServerTime: h.Store.Now().UTC().Format(time.RFC3339Nano), NextBoundary: boundary}
	if tags != nil {
		reading.ServerEpoch, reading.Topics = tags.ServerEpoch, tags.League
		reading.Football, reading.Live, reading.Values = tags.Football, tags.Live, tags.Values
	}
	body, err := json.Marshal(reading)
	if err != nil {
		return nil, fmt.Errorf("encode versions: %w", err)
	}
	return body, nil
}

func (h Hosted) workflowAuthGeneration(r *http.Request, league string) (string, error) {
	account, _ := r.Context().Value(replyPrincipalKey{}).(string)
	host := leagueservice.StoredHost{Store: h.pinFor(r.Context(), league).Store}
	records, err := host.CommissionerRecords(r.Context(), league)
	if err != nil {
		return "", fmt.Errorf("versions membership: %w", err)
	}
	principal := commissioner.PrincipalFor(records, account)
	people, err := commissioner.PeopleReading(league, principal, records)
	if err != nil {
		return "", fmt.Errorf("versions roles: %w", err)
	}
	memberships := []string{}
	for _, seat := range people.Seats {
		if seat.Owner == principal {
			memberships = append(memberships, "owner:"+seat.Franchise)
		}
		if seat.Caretaker == principal {
			memberships = append(memberships, "caretaker:"+seat.Franchise)
		}
	}
	for _, role := range people.Roles {
		if role.Person == principal {
			memberships = append(memberships, "role:"+role.Role)
		}
	}
	slices.Sort(memberships)
	// The session's domain-separated fingerprint is never a reusable credential.
	session := sha256.Sum256([]byte("workflow-session\x00" + strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
	identity := account + "\x00" + league + "\x00" + hex.EncodeToString(session[:]) + "\x00" + strings.Join(memberships, "\x00")
	generation := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(generation[:]), nil
}
