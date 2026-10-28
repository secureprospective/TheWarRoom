// Package serverversion is the league server's compatibility contract: which desktop protocol
// and durable command versions it serves, and which league files this build may open.
package serverversion

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Contract is one server release: the league file schema it writes, the protocol minor it
// speaks, the oldest minor it still serves after a security fix, and the durable command
// version its clients write.
type Contract struct {
	Release       int
	Schema        int
	ProtocolMinor int
	SecurityFloor int
	Commands      int
}

// releases is append-only. Add a row whenever any field but Release changes (TestCurrentContract
// fails until you do), and never edit a shipped row: a server refuses files a later row stamped.
func releases() []Contract {
	return []Contract{
		{Release: 1, Schema: 8, ProtocolMinor: 1, SecurityFloor: 0, Commands: 1},
		{Release: 2, Schema: 9, ProtocolMinor: 1, SecurityFloor: 0, Commands: 1},
		{Release: 3, Schema: 10, ProtocolMinor: 1, SecurityFloor: 0, Commands: 1},
		{Release: 4, Schema: 11, ProtocolMinor: 1, SecurityFloor: 0, Commands: 1},
		{Release: 5, Schema: 12, ProtocolMinor: 1, SecurityFloor: 0, Commands: 1},
	}
}

// ProtocolMajor changes only for a break no minor can bridge. OldestCommands is the oldest
// durable command version a client may still send.
const (
	ProtocolMajor  = 1
	OldestCommands = 1
)

// Current is this build's contract.
func Current() Contract { rows := releases(); return rows[len(rows)-1] }

// ClientProtocol is the protocol this build's clients declare, as "major.minor".
func ClientProtocol() string { return fmt.Sprintf("%d.%d", ProtocolMajor, Current().ProtocolMinor) }

// ClientCommands is the durable command version this build's clients declare.
func ClientCommands() string { return strconv.Itoa(Current().Commands) }

// Side names who must update before the app and server can work together.
type Side string

const (
	App    Side = "app"
	Server Side = "server"
)

const (
	appTooOld    = "This copy of The War Room is too old for your league's server. Install the latest update of The War Room, then open it again."
	appUnsafe    = "This copy of The War Room has a security problem that a newer update fixes. Install the latest update of The War Room, then open it again."
	serverTooOld = "Your league's server is older than this copy of The War Room. Ask your commissioner to update the league server, then try again."
)

// UpdateRequired refuses a client whose versions this server does not serve.
type UpdateRequired struct {
	Update  Side
	Message string
}

func (u *UpdateRequired) Error() string { return u.Message }

// ErrMalformed marks a version header that is not a plain number (commands) or "major.minor".
var ErrMalformed = errors.New("malformed version header")

// Admit checks a client's declared protocol and command version. Empty values come from clients
// built before version headers existed: protocol 1.0 writing command version 1.
func Admit(protocol, commands string) error {
	return admit(Current(), protocol, commands)
}

func admit(current Contract, protocol, commands string) error {
	major, minor, written := ProtocolMajor, 0, OldestCommands
	if protocol != "" {
		left, right, cut := strings.Cut(protocol, ".")
		var leftOK, rightOK bool
		major, leftOK = number(left)
		minor, rightOK = number(right)
		if !cut || !leftOK || !rightOK {
			return ErrMalformed
		}
	}
	if commands != "" {
		var ok bool
		if written, ok = number(commands); !ok {
			return ErrMalformed
		}
	}
	switch {
	case major > ProtocolMajor || (major == ProtocolMajor && minor > current.ProtocolMinor) || written > current.Commands:
		return &UpdateRequired{Update: Server, Message: serverTooOld}
	case major < ProtocolMajor || minor < current.ProtocolMinor-1 || written < OldestCommands:
		return &UpdateRequired{Update: App, Message: appTooOld}
	case minor < current.SecurityFloor:
		return &UpdateRequired{Update: App, Message: appUnsafe}
	}
	return nil
}

// number accepts one to four ASCII digits, nothing else.
func number(text string) (int, bool) {
	if len(text) == 0 || len(text) > 4 || strings.Trim(text, "0123456789") != "" {
		return 0, false
	}
	value, err := strconv.Atoi(text)
	return value, err == nil
}
