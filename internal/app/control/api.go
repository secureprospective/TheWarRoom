package control

import (
	"log"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

// Host runs Control Room's reads and writes against the shared stores. The held rule names and
// site news, the params store and the key store stay behind it with their original locks; every
// method returns detached values and keeps the binding's original refusal order and text.
type Host interface {
	RoomHost
	ParamsHost
	KeyHost
}

// RoomHost reads the league room: rule names, site news, the rulebook and the Accounting link.
type RoomHost interface {
	LeagueSettings() (report.Report, error)
	SiteNews() (report.Report, error)
	OpenMFLAccounting() error
	RulebookMarkdown() []byte
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
// parameters and the MFL key. It holds no state of its own.
type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

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
	State      string `json:"state"`
	League     string `json:"league"`
	Season     int    `json:"season"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// TargetLeagueSettings is Control Room › League › League settings (M-091, M-097 merged in).
func (a *API) TargetLeagueSettings() (report.Report, error) {
	return a.host.LeagueSettings() //nolint:wrapcheck // the host keeps the binding's refusal text
}

// TargetSiteNews is Control Room › App › What's New (M-141), refreshed when an hour old.
func (a *API) TargetSiteNews() (report.Report, error) {
	return a.host.SiteNews() //nolint:wrapcheck // the host keeps the binding's refusal text
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

// Commands records the existing bindings without inventing publication metadata. Opening MFL's
// accounting page opens a browser, so it is desktop-only; the key lives in the OS keyring.
func Commands() []catalog.Command {
	c := func(method, availability string) catalog.Command {
		return catalog.Command{ID: catalog.Unset, Method: method, Version: catalog.Unset, Role: catalog.Unset,
			Gravity: catalog.Unset, Availability: availability}
	}
	return []catalog.Command{
		c("DeleteMFLKey", "desktop"), c("GetParams", "both"), c("MFLKeyStatus", "desktop"), c("ResetParam", "both"),
		c("SetMFLKey", "desktop"), c("SetParam", "both"), c("TargetAccounting", "both"), c("TargetBylaws", "both"),
		c("TargetLeagueSettings", "both"), c("TargetOpenMFLAccounting", "desktop"), c("TargetSiteNews", "both"),
	}
}
