package leaguemigration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// prepareInvites lists one invitation per franchise from MFL's owners, matched by the MFL franchise id
// the starting snapshot keeps. A franchise whose seat is held shows as claimed; one with an invitation
// already sent shows as sent; one MFL lists no usable email for is listed, never guessed.
func prepareInvites(league string, export Export, app AppLeague, state governance.State) Invites {
	owners := map[string]Owner{}
	for _, o := range export.Owners {
		owners[o.Franchise] = o
	}
	out := Invites{League: league, List: []Invite{}}
	for _, f := range app.Snapshot.Franchises {
		seat, _ := state.Franchise(governance.FranchiseID(f.ID))
		owner := owners[f.HeldID]
		email := strings.TrimSpace(owner.Email)
		invite := Invite{Franchise: f.ID, Team: f.Name, Owner: strings.TrimSpace(owner.Name), Email: email}
		if seat.Name != "" {
			invite.Team = seat.Name
		}
		switch {
		case seat.Seat != "":
			invite.Claimed = true
			invite.Words = fmt.Sprintf("Claimed by %s", seat.Seat)
		case email == "":
			invite.Words = "MFL holds no email for this owner; invite them by hand from the people screen"
		case !validEmail(email):
			invite.Words = fmt.Sprintf("MFL's email for this owner (%q) is not an email address; "+
				"invite them by hand from the people screen", email)
		default:
			invite.ID = inviteID(league, f.ID, email)
			if sent, ok := state.Invite(invite.ID); ok && !sent.Revoked {
				invite.Words = fmt.Sprintf("Invitation sent to %s; waiting for them to accept", email)
				break
			}
			invite.Ready = true
			out.Ready++
			invite.Words = fmt.Sprintf("Ready to send to %s", email)
		}
		out.List = append(out.List, invite)
	}
	body, _ := json.Marshal(out.List) //nolint:errchkjson // plain strings and bools always encode
	sum := sha256.Sum256(body)
	out.Digest = hex.EncodeToString(sum[:8])
	return out
}

func validEmail(raw string) bool {
	address, err := mail.ParseAddress(raw)
	return err == nil && address.Address == raw && address.Name == ""
}
