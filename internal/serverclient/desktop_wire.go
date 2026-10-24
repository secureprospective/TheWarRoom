package serverclient

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type loginBody struct {
	Protocol int    `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
	Device   string `json:"device"`
}
type enrollmentCredentials struct {
	Code     string `json:"code"`
	Username string `json:"username"`
	Password string `json:"password"`
}
type enrollmentBody struct {
	Protocol   int                   `json:"protocol"`
	Enrollment enrollmentCredentials `json:"enrollment"`
}
type desktopPayload interface {
	loginBody | enrollmentBody | serverhttp.RawAuthenticator | serverhttp.RawNativeRequest
}

type rawDesktopLogin serverhttp.LoginResponse

func (r rawDesktopLogin) Validate() error {
	if r.Protocol != 1 || (accounts.RawSession{Bearer: r.Bearer}).Validate() != nil || r.Session.ExpiresAt.IsZero() {
		return fmt.Errorf("invalid session response")
	}
	return validateDesktopAccount(r.Session.Account)
}

type rawDesktopEnrollment struct {
	Protocol int              `json:"protocol"`
	Account  accounts.Account `json:"account"`
}

func (r rawDesktopEnrollment) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("invalid enrollment response")
	}
	return validateDesktopAccount(r.Account)
}
func validateDesktopAccount(account accounts.Account) error {
	if _, err := account.Principal(); err != nil {
		return fmt.Errorf("invalid account response: %w", err)
	}
	if _, err := accounts.CanonicalUsername(account.Username); err != nil {
		return fmt.Errorf("invalid username response: %w", err)
	}
	return nil
}

type rawDesktopAuthenticator serverhttp.AuthenticatorResponse

func (r rawDesktopAuthenticator) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported authenticator response")
	}
	if r.Seed != "" {
		seed, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(r.Seed)
		if err != nil || len(seed) != 20 {
			return fmt.Errorf("invalid authenticator seed")
		}
	}
	if r.RecentAuth != "" && (accounts.RawSession{Bearer: r.RecentAuth}).Validate() != nil {
		return fmt.Errorf("invalid recent authentication proof")
	}
	return nil
}

type rawDesktopPeople commissioner.People

func (r rawDesktopPeople) Validate() error {
	if r.League == "" || r.Name == "" || r.YourRole == "" {
		return fmt.Errorf("invalid league response")
	}
	if r.YourFranchise != "" {
		for _, seat := range r.Seats {
			if seat.Franchise == r.YourFranchise && seat.Name != "" {
				return nil
			}
		}
		return fmt.Errorf("assigned franchise missing from league response")
	}
	return nil
}

func decodeDesktop[T interface{ Validate() error }](body []byte, raw T) error {
	if err := json.Unmarshal(body, raw); err != nil {
		return fmt.Errorf("decode native response: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("check native response: %w", err)
	}
	return nil
}

func (d *Desktop) enter(ctx context.Context, view DesktopView) (DesktopView, error) {
	if d.bearer == "" {
		return view, fmt.Errorf("sign in before opening your franchise")
	}
	body, err := postDesktop(ctx, d, serverhttp.PeoplePath, serverhttp.RawNativeRequest{Protocol: 1, League: d.league})
	if err != nil {
		return view, err
	}
	var people rawDesktopPeople
	if err := decodeDesktop(body, &people); err != nil {
		return view, err
	}
	if people.League != d.league {
		return view, fmt.Errorf("server returned a different league")
	}
	view.Name, view.Role, view.Franchise = people.Name, people.YourRole, people.YourFranchise
	for _, seat := range people.Seats {
		if seat.Franchise == people.YourFranchise {
			view.FranchiseName = seat.Name
		}
	}
	if view.Franchise == "" {
		view.Message = "No franchise is assigned to this account. Ask your commissioner to check your seat; do not create another account."
	}
	return view, nil
}

func (d *Desktop) authenticator(ctx context.Context, raw RawDesktop, view DesktopView) (DesktopView, error) {
	paths := map[string]string{"totp-begin": serverhttp.TOTPBeginPath, "totp-confirm": serverhttp.TOTPConfirmPath, "totp-step-up": serverhttp.TOTPStepUpPath}
	body, err := postDesktop(ctx, d, paths[raw.Action], serverhttp.RawAuthenticator{Protocol: 1, Username: raw.Username, Password: raw.Password, Code: raw.Code})
	if err != nil {
		return view, err
	}
	var response rawDesktopAuthenticator
	if err := decodeDesktop(body, &response); err != nil {
		return view, err
	}
	switch raw.Action {
	case "totp-begin":
		if response.Seed == "" {
			return view, fmt.Errorf("server did not return an authenticator setup key")
		}
		view.Seed = response.Seed
		view.Message = "Add this setup key to your authenticator as a time-based account, then confirm a code. Keep the key private."
	case "totp-confirm":
		view.Message = "Authenticator confirmed. Wait for a new code, then verify it to open privileged access."
	case "totp-step-up":
		if response.RecentAuth == "" {
			return view, fmt.Errorf("server did not verify recent authentication")
		}
		d.proof = response.RecentAuth
		view.Message = "Authenticator verified for five minutes. Open your franchise now."
	}
	return view, nil
}

func desktopRefusal(path string, status int) error {
	switch path {
	case serverhttp.EnrollmentPath:
		return fmt.Errorf("enrollment refused: the code may be expired, used, or for a different username. Ask your commissioner for a new personally handed code; codes last 48 hours")
	case serverhttp.LoginPath:
		return fmt.Errorf("sign-in refused. Check your username and password, or open account recovery")
	case serverhttp.PeoplePath:
		if status == 401 {
			return fmt.Errorf("your session expired or was revoked. Sign in again")
		}
		return fmt.Errorf("league access refused. Commissioners must set up and verify their authenticator; owners should ask the commissioner to check their seat")
	default:
		return fmt.Errorf("authenticator refused. Check your password and phone clock, then use a fresh code. If the phone is lost, open account recovery")
	}
}
