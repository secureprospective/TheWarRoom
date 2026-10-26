package control

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Host runs Control Room's reads and writes against the shared stores. The params store, the key
// store, the season feeds and the week stay behind it with their original locks; every method
// returns detached values and keeps the binding's original refusal order and text.
type Host interface {
	// Ready waits for startup and returns its failure, if any.
	Ready() error
	RoomHost
	ParamsHost
	KeyHost
	SeasonHost
	ClockHost
}

// RoomHost reads the league room: the scoring config, MFL's rule names and site news, the rulebook
// and the Accounting link. The two MFL reads go live with the archive as fallback; the API holds
// what they return in Held.
type RoomHost interface {
	ScoringConfig() league.RawConfig
	FetchRuleNames() (map[string]string, domain.Freshness, error)
	FetchSiteNews() ([]sitenews.RawItem, domain.Freshness, error)
	// Zone is the time zone the site news' as-of time is written in.
	Zone() *time.Location
	OpenMFLAccounting() error
	RulebookMarkdown() []byte
}

// SeasonHost reads the held season feeds and the alert tray's held inputs.
type SeasonHost interface {
	// SeasonFeeds copies the transactions, live-scoring lineups (and the week they were asked
	// for) and pending trades under the season lock. Nil stays nil.
	SeasonFeeds() (report.HeldFeed[[]leaguefeed.Transaction], report.HeldFeed[leaguefeed.Lineups], int,
		report.HeldFeed[[]leaguefeed.PendingTrade])
	// Alerts gathers the IR, lineup, bye and trade alerts for one franchise, unsorted.
	Alerts(franchiseID string) (AlertReading, error)
}

// ClockHost reads source health and the held NFL week, and writes a league window.
type ClockHost interface {
	// SourceHealth is the status strip's health state and label.
	SourceHealth() (state, label string, err error)
	Week() WeekSnapshot
	Now() time.Time
	// SetWindow writes one validated league date through the coordinator, after startup.
	SetWindow(kind, at string) error
}

// ParamsHost reads and writes the Engine Admin parameters.
type ParamsHost interface {
	Params() ParamsResult
	SetParam(key, position string, value float64) SetParamResult
	ResetParam(key, position string) SetParamResult
}

// KeyHost reads, replaces and forgets the MFL key.
type KeyHost interface {
	KeyStatus() (MFLKeyStatus, error)
	SetKey(key string) (MFLKeyStatus, error)
	DeleteKey() (MFLKeyStatus, error)
	// KeyUnavailable is the status SetMFLKey reports after recovering a panic.
	KeyUnavailable(detail string) MFLKeyStatus
}

// API is Control Room's binding: League settings, What's New, Accounting, By-laws, the admin
// parameters, the MFL key, the alert tray, the season feeds, the status strip and league dates.
// Its only state is Held, the private rule names and site news.
type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

// siteNewsFreshFor is how long a held site-news read serves before the next open refreshes it.
const siteNewsFreshFor = time.Hour

// ParamView is one calibration parameter as the admin panel shows it: its definition and the
// value in effect. Position is empty for a league-wide parameter.
type ParamView struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Group       string  `json:"group"`
	Position    string  `json:"position"`
	Description string  `json:"description"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Value       float64 `json:"value"`
	Calibrated  bool    `json:"calibrated"` // the default came from the fit tool, not a hand setting
	Overridden  bool    `json:"overridden"` // set in Admin: pinned against later shipped defaults
}

// ParamsResult is the admin panel payload.
type ParamsResult struct {
	OK     bool        `json:"ok"`
	Error  string      `json:"error"`
	Params []ParamView `json:"params"`
}

// SetParamResult is the admin-write payload.
type SetParamResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// MFLKeyStatus exposes presence, never any portion of the credential. State is one of
// absent | connected | rejected | unreachable | unavailable. Every one of them is returned with
// a nil error: Wails rejects the JS promise on any error and drops the status with it.
type MFLKeyStatus struct {
	MyFranchise       string `json:"myFranchise,omitempty"`
	TeamActionRefusal string `json:"teamActionRefusal"`
	State             string `json:"state"`
	League            string `json:"league"`
	Season            int    `json:"season"`
	VerifiedAt        string `json:"verifiedAt,omitempty"`
	Detail            string `json:"detail,omitempty"`
}

// TargetLeagueSettings is Control Room › League › League settings (M-091, M-097 merged in).
func (a *API) TargetLeagueSettings() (report.Report, error) {
	if err := a.host.Ready(); err != nil {
		return report.Report{}, fmt.Errorf("league settings: startup: %w", err)
	}
	held := a.held.SnapshotRuleNames()
	if held.Value == nil {
		names, fresh, err := a.host.FetchRuleNames()
		held = a.held.PublishRuleNames(names, fresh, err)
	}
	note := ""
	if held.Value == nil {
		note = "MFL's names for its scoring codes could not be read, so codes are shown: " + held.RefreshError
	}
	return LeagueSettingsReport(a.host.ScoringConfig(), held.Value, note), nil
}

// TargetSiteNews is Control Room › App › What's New (M-141), refreshed when an hour old.
func (a *API) TargetSiteNews() (report.Report, error) {
	if err := a.host.Ready(); err != nil {
		return report.Report{}, fmt.Errorf("site news: startup: %w", err)
	}
	held := a.held.SnapshotSiteNews()
	if held.RefreshError != "" || time.Since(held.FetchedAt) >= siteNewsFreshFor {
		rows, fresh, err := a.host.FetchSiteNews()
		held = a.held.PublishSiteNews(rows, fresh, err)
	}
	items, err := sitenews.Read(held.Value)
	if err != nil {
		return report.Report{}, err //nolint:wrapcheck // names the item
	}
	zone := a.host.Zone()
	asOf := "As of: not fetched yet"
	if !held.FetchedAt.IsZero() {
		asOf = "As of: " + report.Time(held.FetchedAt, zone)
	}
	return SiteNewsReport(items, asOf, held.RefreshError, zone), nil
}

// TargetAccounting is Control Room › League › Accounting (M-098, M-197).
func (a *API) TargetAccounting() report.Report { return AccountingReport() }

// TargetOpenMFLAccounting opens MFL's accounting page for the league. The app only opens it.
func (a *API) TargetOpenMFLAccounting() error {
	return a.host.OpenMFLAccounting() //nolint:wrapcheck // the host keeps the binding's refusal text
}

// TargetBylaws is Control Room › League › By-laws (M-185): the league's own rulebook, by section.
func (a *API) TargetBylaws() []BylawSection { return BylawSections(a.host.RulebookMarkdown()) }

// GetParams returns every calibration parameter with the value in effect.
func (a *API) GetParams() ParamsResult { return a.host.Params() }

// SetParam applies an admin override to a parameter; position is empty for a league-wide one.
// The params store validates the range.
func (a *API) SetParam(key, position string, value float64) SetParamResult {
	return a.host.SetParam(key, position, value)
}

// ResetParam clears a parameter's admin override, so it follows its shipped default again.
func (a *API) ResetParam(key, position string) SetParamResult {
	return a.host.ResetParam(key, position)
}

func (a *API) MFLKeyStatus() (MFLKeyStatus, error) {
	return a.host.KeyStatus() //nolint:wrapcheck // the host keeps the binding's refusal text
}

func (a *API) SetMFLKey(key string) (status MFLKeyStatus, err error) {
	// Wails recovers a binding panic and logs the raw IPC message, key included, at ERROR level
	// (on in production). Recover here first, in the bound method, and log only the panic's type.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SetMFLKey: recovered panic (%T)", r)
			status, err = a.host.KeyUnavailable("internal error while connecting; check the status"), nil
		}
	}()
	return a.host.SetKey(key) //nolint:wrapcheck // the host keeps the binding's refusal text
}

func (a *API) DeleteMFLKey() (MFLKeyStatus, error) {
	return a.host.DeleteKey() //nolint:wrapcheck // the host keeps the binding's refusal text
}

// AlertReading is the alert tray: what needs the GM's attention and what could not be checked.
type AlertReading struct {
	Alerts      []Alert       `json:"alerts"`
	Unavailable []Unavailable `json:"unavailable"`
}

// Unavailable names a check the tray could not run and why.
type Unavailable struct {
	Kind string `json:"kind"`
	Note string `json:"note"`
}

// TargetAlerts reads held state only; waivers are absent by league rule (gap-closure §8).
func (a *API) TargetAlerts(franchiseID string) (AlertReading, error) {
	r, err := a.host.Alerts(franchiseID)
	if err != nil {
		return AlertReading{}, fmt.Errorf("target alerts: %w", err)
	}
	slices.SortFunc(r.Alerts, CompareAlerts)
	return r, nil
}

// SeasonReading is the three season feeds as held, each with its own provenance.
type SeasonReading struct {
	Transactions  snapshot.Sourced[[]leaguefeed.Transaction]  `json:"transactions"`
	Lineups       snapshot.Sourced[leaguefeed.Lineups]        `json:"lineups"`
	PendingTrades snapshot.Sourced[[]leaguefeed.PendingTrade] `json:"pendingTrades"`
}

// TargetSeason reads held state only: no network and never refreshMu, so it answers while a
// refresh runs. Its only error is "not ready".
func (a *API) TargetSeason() (SeasonReading, error) {
	if err := a.host.Ready(); err != nil {
		return SeasonReading{}, fmt.Errorf("target season: startup: %w", err)
	}
	transactions, lineups, week, pending := a.host.SeasonFeeds()
	source := "liveScoring default week"
	if week != 0 {
		source = "liveScoring W=" + strconv.Itoa(week)
	}
	// Copies serialize as [] rather than null before the first fetch.
	txs := append([]leaguefeed.Transaction{}, transactions.Value...)
	trades := append([]leaguefeed.PendingTrade{}, pending.Value...)
	held := lineups.Value
	held.Franchises = append([]leaguefeed.Lineup{}, held.Franchises...)
	held.Matchups = append([]leaguefeed.Matchup{}, held.Matchups...)
	for i := range held.Franchises {
		row := &held.Franchises[i]
		row.Players = append([]leaguefeed.PlayerScore{}, row.Players...)
	}
	return SeasonReading{
		Transactions: snapshot.Sourced[[]leaguefeed.Transaction]{
			Value: txs, Provenance: transactions.Provenance("transactions", "transactions"),
		},
		Lineups: snapshot.Sourced[leaguefeed.Lineups]{
			Value: held, Provenance: lineups.Provenance(source, "live scores"),
		},
		PendingTrades: snapshot.Sourced[[]leaguefeed.PendingTrade]{
			Value: trades, Provenance: pending.Provenance("pendingTrades", "pending trades"),
		},
	}, nil
}

// ShellStatus is the status strip: source health and the gameday line.
type ShellStatus struct {
	Health  string `json:"health"`
	State   string `json:"state"`
	Gameday string `json:"gameday"`
}

func (a *API) TargetShellStatus() (ShellStatus, error) {
	if err := a.host.Ready(); err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: startup: %w", err)
	}
	state, label, err := a.host.SourceHealth()
	if err != nil {
		return ShellStatus{}, fmt.Errorf("shell status: %w", err)
	}
	gameday := Gameday(a.host.Week(), a.host.Now())
	return ShellStatus{Health: label, State: state, Gameday: gameday}, nil
}

// RawWindowDate is TargetSetWindow's request: a league window kind and an RFC 3339 time, or an
// empty time to follow the rulebook again.
type RawWindowDate struct{ Kind, At string }

func (r RawWindowDate) Validate() error {
	for _, rule := range leagueclock.Rules(0) {
		if string(rule.Kind) != r.Kind {
			continue
		}
		if r.At == "" {
			return nil
		}
		if _, err := time.Parse(time.RFC3339, r.At); err != nil {
			return fmt.Errorf("set a date and time with a timezone: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unknown league window %q", r.Kind)
}

// TargetSetWindow sets or resets one league date. The request is checked before startup.
func (a *API) TargetSetWindow(kind, at string) error {
	if err := (RawWindowDate{Kind: kind, At: at}).Validate(); err != nil {
		return err
	}
	if err := a.host.Ready(); err != nil {
		return fmt.Errorf("set league date: startup: %w", err)
	}
	return a.host.SetWindow(kind, at) //nolint:wrapcheck // the host keeps the binding's refusal text
}

// Commands records the existing bindings without inventing publication metadata. Opening MFL's
// accounting page opens a browser, so it is desktop-only; the key lives in the OS keyring. League
// dates copy the desktop window.set command's roles and gravity; the alert tray, season feeds and
// status strip have no registry verb.
func Commands() []catalog.Command { return catalog.DesktopCommands("control", nil) }
